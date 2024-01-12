package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	k8s "github.com/Azure/azure-container-networking/test/e2e/kubernetes"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
)

var (
	ErrResourceNameTooLong = fmt.Errorf("resource name too long")
	componentFolders       = []string{
		"manifests/cilium/v1.14/cns",
		"manifests/cilium/v1.14/agent",
		"manifests/cilium/v1.14/ipmasq",
		"manifests/cilium/v1.14/operator",
	}
)

type CreateBYOCiliumCluster struct {
	SubscriptionID    string
	ResourceGroupName string
	Location          string
	ClusterName       string
	VnetName          string
	SubnetName        string
	PodCidr           string
	DNSServiceIP      string
	ServiceCidr       string
}

func printjson(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(data); err != nil {
		panic(err)
	}
}

func (c *CreateBYOCiliumCluster) Prevalidate() error {
	// get current working directory
	cwd, _ := os.Getwd()

	for _, dir := range componentFolders {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory not found: %s\ncurrent working directory: %s, err: %w", dir, cwd, err)
		}
	}

	if len(c.ResourceGroupName) > 80 { //nolint:gomnd // 80 is the max length for resource group names
		return fmt.Errorf("resource group name for nodes cannot exceed 80 characters: %w", ErrResourceNameTooLong)
	}

	return nil
}

func (c *CreateBYOCiliumCluster) Postvalidate() error {
	return nil
}

func (c *CreateBYOCiliumCluster) Run() error {
	// Start with default cluster template
	ciliumCluster := GetStarterClusterTemplate(c.Location)
	ciliumCluster.Properties.NetworkProfile.NetworkPlugin = to.Ptr(armcontainerservice.NetworkPluginNone)
	ciliumCluster.Properties.NetworkProfile.NetworkPluginMode = to.Ptr(armcontainerservice.NetworkPluginModeOverlay)
	ciliumCluster.Properties.NetworkProfile.PodCidr = to.Ptr(c.PodCidr)
	ciliumCluster.Properties.NetworkProfile.DNSServiceIP = to.Ptr(c.DNSServiceIP)
	ciliumCluster.Properties.NetworkProfile.ServiceCidr = to.Ptr(c.ServiceCidr)
	subnetkey := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", c.SubscriptionID, c.ResourceGroupName, c.VnetName, c.SubnetName)
	ciliumCluster.Properties.AgentPoolProfiles[0].VnetSubnetID = to.Ptr(subnetkey)

	// Set the kubeproxy config
	kubeProxyConfig := armcontainerservice.NetworkProfileKubeProxyConfig{
		Mode:    to.Ptr(armcontainerservice.ModeIPVS),
		Enabled: to.Ptr(false),
		IpvsConfig: to.Ptr(armcontainerservice.NetworkProfileKubeProxyConfigIpvsConfig{
			Scheduler:            to.Ptr(armcontainerservice.IpvsSchedulerLeastConnection),
			TCPTimeoutSeconds:    to.Ptr(int32(900)), //nolint:gomnd // set by existing kube-proxy in hack/aks/kube-proxy.json
			TCPFinTimeoutSeconds: to.Ptr(int32(120)), //nolint:gomnd // set by existing kube-proxy in hack/aks/kube-proxy.json
			UDPTimeoutSeconds:    to.Ptr(int32(300)), //nolint:gomnd // set by existing kube-proxy in hack/aks/kube-proxy.json
		}),
	}

	log.Printf("using kube-proxy config:\n")
	printjson(kubeProxyConfig)
	ciliumCluster.Properties.NetworkProfile.KubeProxyConfig = to.Ptr(kubeProxyConfig)

	// Deploy cluster
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return fmt.Errorf("failed to obtain a credential: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultClusterCreateTimeout)
	defer cancel()

	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create az client: %w", err)
	}

	log.Printf("creating cluster \"%s\" in resource group \"%s\"...", c.ClusterName, c.ResourceGroupName)

	poller, err := clientFactory.NewManagedClustersClient().BeginCreateOrUpdate(ctx, c.ResourceGroupName, c.ClusterName, ciliumCluster, nil)
	if err != nil {
		return fmt.Errorf("failed to finish the create cluster request: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// get kubeconfig
	log.Printf("getting kubeconfig for cluster \"%s\" in resource group \"%s\"...", c.ClusterName, c.ResourceGroupName)
	clientset, err := c.getKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig for cluster \"%s\": %w", c.ClusterName, err)
	}

	// Deploy the cilium components once the cluster is created
	log.Printf("deploying cilium components to cluster \"%s\" in resource group \"%s\"...", c.ClusterName, c.ResourceGroupName)
	err = c.deployCiliumComponents(clientset)
	if err != nil {
		return fmt.Errorf("failed to deploy cilium components: %w", err)
	}

	// wait for cilium pods to be ready
	k8s.WaitForPodReady(ctx, clientset, "kube-system", "k8s-app=cilium")

	return err
}

func (c *CreateBYOCiliumCluster) getKubeConfig() (*kubernetes.Clientset, error) {
	// create temporary directory for kubeconfig, as we need access to deploy cilium things
	dir, err := os.MkdirTemp("", "cilium-e2e")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory to deploy cilium components on cluster \"%s\": %w", c.ClusterName, err)
	}
	// reuse getKubeConfig job
	kubeconfigpath := dir + "/kubeconfig"
	getKubeconfigJob := GetAKSKubeConfig{
		ClusterName:        c.ClusterName,
		SubscriptionID:     c.SubscriptionID,
		ResourceGroupName:  c.ResourceGroupName,
		Location:           c.Location,
		KubeConfigFilePath: kubeconfigpath,
	}

	err = getKubeconfigJob.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig to deploy cilium components on cluster \"%s\": %w", c.ClusterName, err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigpath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig to deploy cilium components on cluster \"%s\": %w", c.ClusterName, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client to deploy cilium components on cluster \"%s\": %w", c.ClusterName, err)
	}
	return clientset, nil
}

func (c *CreateBYOCiliumCluster) deployCiliumComponents(clientset *kubernetes.Clientset) error {
	// traverse the predefined Cilium component folders
	for _, dir := range componentFolders {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error traversing the cilium components directory: %w", err)
			}

			// Skip directories, only care about yaml
			if !info.IsDir() {
				yamlFile, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading YAML file: %w", err)
				}

				// Decode the YAML file into a Kubernetes object
				decode := scheme.Codecs.UniversalDeserializer().Decode
				obj, _, err := decode([]byte(yamlFile), nil, nil)
				if err != nil {
					return fmt.Errorf("error decoding YAML file: %w", err)
				}

				// create the resource
				err = k8s.CreateResource(context.Background(), obj, clientset)
				if err != nil {
					return fmt.Errorf("error creating resource: %w", err)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking the cilium components directory: %w", err)
		}
	}

	return nil
}
