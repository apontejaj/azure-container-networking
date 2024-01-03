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
	componentFolders = []string{
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
			return fmt.Errorf("directory not found: %s\ncurrent working directory: %s", dir, cwd)
		}
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

	kubeProxyConfig := armcontainerservice.NetworkProfileKubeProxyConfig{
		Mode:    to.Ptr(armcontainerservice.ModeIPVS),
		Enabled: to.Ptr(false),
		IpvsConfig: to.Ptr(armcontainerservice.NetworkProfileKubeProxyConfigIpvsConfig{
			Scheduler:            to.Ptr(armcontainerservice.IpvsSchedulerLeastConnection),
			TCPTimeoutSeconds:    to.Ptr(int32(900)), //nolint:gomnd set by existing kube-proxy in hack/aks/kube-proxy.json
			TCPFinTimeoutSeconds: to.Ptr(int32(120)), //nolint:gomnd set by existing kube-proxy in hack/aks/kube-proxy.json
			UDPTimeoutSeconds:    to.Ptr(int32(300)), //nolint:gomnd set by existing kube-proxy in hack/aks/kube-proxy.json
		}),
	}

	fmt.Printf("using kube-proxy config:\n")
	printjson(kubeProxyConfig)

	ciliumCluster.Properties.NetworkProfile.KubeProxyConfig = to.Ptr(kubeProxyConfig)

	// Deploy cluster
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	log.Printf("creating cluster %s in resource group %s...", c.ClusterName, c.ResourceGroupName)

	poller, err := clientFactory.NewManagedClustersClient().BeginCreateOrUpdate(ctx, c.ResourceGroupName, c.ClusterName, ciliumCluster, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to create cluster: %v", err)
	}

	log.Printf("deploying cilium components to cluster %s in resource group %s...", c.ClusterName, c.ResourceGroupName)

	err = c.deployCiliumComponents()
	if err != nil {
		fmt.Errorf("failed to deploy cilium components: %v", err)
	}

	return err
}

func (c *CreateBYOCiliumCluster) deployCiliumComponents() error {
	// create temporary directory for kubeconfig, as we need access to deploy cilium things
	dir, err := os.MkdirTemp("", "cilium-e2e")
	if err != nil {
		log.Fatal(err)
	}

	kubeconfigpath := dir + "/kubeconfig"

	// reuse getKubeConfig job
	getKubeconfigJob := GetAKSKubeConfig{
		ClusterName:        c.ClusterName,
		SubscriptionID:     c.SubscriptionID,
		ResourceGroupName:  c.ResourceGroupName,
		Location:           c.Location,
		KubeConfigFilePath: dir + "/kubeconfig",
	}

	getKubeconfigJob.Run()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigpath)
	if err != nil {
		fmt.Println("error building kubeconfig: ", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("error creating Kubernetes client: ", err)
		return err
	}

	for _, dir := range componentFolders {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println("error:", err)
				return err
			}

			// Check if it's a regular file (not a directory)
			if !info.IsDir() {
				// Get the YAML file
				yamlFile, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("error reading YAML file: %s\n", err)
					return err
				}

				// Decode the YAML file into a Kubernetes object
				decode := scheme.Codecs.UniversalDeserializer().Decode
				obj, _, err := decode([]byte(yamlFile), nil, nil)
				if err != nil {
					fmt.Printf("error decoding YAML file: %s\n", err)
					return err
				}

				err = k8s.CreateResource(context.Background(), obj, clientset)
				if err != nil {
					return err
				}

				fmt.Printf("created resource: %s\n", path)
			}

			return nil
		})

		if err != nil {
			fmt.Println("Error walking the path:", err)
		}
	}
	return nil
}

func (c *CreateBYOCiliumCluster) ExpectError() bool {
	return false
}

func (c *CreateBYOCiliumCluster) SaveParametersToJob() bool {
	return true
}
