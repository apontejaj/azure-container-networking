package validate

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Azure/azure-container-networking/cns"
	restserver "github.com/Azure/azure-container-networking/cns/restserver"
	k8sutils "github.com/Azure/azure-container-networking/test/internal/k8sutils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	privilegedDaemonSetPath = "../manifests/load/privileged-daemonset.yaml"
	privilegedLabelSelector = "app=privileged-daemonset"
	privilegedNamespace     = "kube-system"

	cnsLabelSelector        = "k8s-app=azure-cns"
	ciliumLabelSelector     = "k8s-app=cilium"
	overlayClusterLabelName = "overlay"
	dualstackNodeStatusAddr = 3
)

var (
	restartNetworkCmd    = []string{"bash", "-c", "chroot /host /bin/bash -c 'systemctl restart systemd-networkd'"}
	cnsStateFileCmd      = []string{"bash", "-c", "cat /var/run/azure-cns/azure-endpoints.json"}
	azureCnsStateFileCmd = []string{"bash", "-c", "cat /var/run/azure-vnet.json"} // azure cni statefile is /var/run/azure-vnet.json
	ciliumStateFileCmd   = []string{"bash", "-c", "cilium endpoint list -o json"}
	cnsLocalCacheCmd     = []string{"curl", "localhost:10090/debug/ipaddresses", "-d", "{\"IPConfigStateFilter\":[\"Assigned\"]}"}
)

var dualstackoverlaynodelabel = map[string]string{
	"kubernetes.azure.com/podnetwork-type":   "overlay",
	"kubernetes.azure.com/podv6network-type": "overlay",
}

type stateFileIpsFunc func([]byte) (map[string]string, error)

type LinuxClient struct{}

type LinuxValidator struct {
	Validator
}

type CnsState struct {
	Endpoints map[string]restserver.EndpointInfo `json:"Endpoints"`
}

type CNSLocalCache struct {
	IPConfigurationStatus []cns.IPConfigurationStatus `json:"IPConfigurationStatus"`
}

type CiliumEndpointStatus struct {
	Status NetworkingStatus `json:"status"`
}

type NetworkingStatus struct {
	Networking NetworkingAddressing `json:"networking"`
}

type NetworkingAddressing struct {
	Addresses     []Address `json:"addressing"`
	InterfaceName string    `json:"interface-name"`
}

type Address struct {
	Addr string `json:"ipv4"`
}

// parse azure-vnet.json
// azure cni manages endpoint state
type AzureCniState struct {
	AzureCniState Network `json:"Network"`
}

type Network struct {
	Version            string                   `json:"Version"`
	TimeStamp          string                   `json:"TimeStamp"`
	ExternalInterfaces map[string]InterfaceInfo `json:"ExternalInterfaces"` // key: interface name; value: Interface Info
}

type InterfaceInfo struct {
	Name     string                 `json:"Name"`
	Networks map[string]NetworkInfo `json:"Networks"` // key: networkName, value: NetworkInfo
}

type AzureVnetInfo struct {
	Name     string
	Networks map[string]NetworkInfo // key: network name, value: NetworkInfo
}

type NetworkInfo struct {
	ID        string
	Mode      string
	Subnets   []Subnet
	Endpoints map[string]AzureVnetEndpointInfo // key: azure endpoint name, value: AzureVnetEndpointInfo
	PODName   string
}

type Subnet struct {
	Family    int
	Prefix    Prefix
	Gateway   string
	PrimaryIP string
}

type Prefix struct {
	IP   string
	Mask string
}

type AzureVnetEndpointInfo struct {
	IfName      string
	MacAddress  string
	IPAddresses []Prefix
	PODName     string
}

type check struct {
	name             string
	stateFileIps     func([]byte) (map[string]string, error)
	podLabelSelector string
	podNamespace     string
	cmd              []string
}

func (l *LinuxClient) CreateClient(ctx context.Context, clienset *kubernetes.Clientset, config *rest.Config, namespace, cni string, restartCase bool) IValidator {
	// deploy privileged pod
	privilegedDaemonSet, err := k8sutils.MustParseDaemonSet(privilegedDaemonSetPath)
	if err != nil {
		panic(err)
	}
	daemonsetClient := clienset.AppsV1().DaemonSets(privilegedNamespace)
	err = k8sutils.MustCreateDaemonset(ctx, daemonsetClient, privilegedDaemonSet)
	if err != nil {
		panic(err)
	}
	err = k8sutils.WaitForPodsRunning(ctx, clienset, privilegedNamespace, privilegedLabelSelector)
	if err != nil {
		panic(err)
	}
	return &LinuxValidator{
		Validator: Validator{
			ctx:         ctx,
			clientset:   clienset,
			config:      config,
			namespace:   namespace,
			cni:         cni,
			restartCase: restartCase,
		},
	}
}

func (v *LinuxValidator) ValidateStateFile() error {
	checkSet := make(map[string][]check) // key is cni type, value is a list of check

	// TODO: add cniv1 when adding related test cases
	checkSet["cilium"] = []check{
		{"cns", cnsStateFileIps, cnsLabelSelector, privilegedNamespace, cnsStateFileCmd},
		{"cilium", ciliumStateFileIps, ciliumLabelSelector, privilegedNamespace, ciliumStateFileCmd},
		{"cns cache", cnsCacheStateFileIps, cnsLabelSelector, privilegedNamespace, cnsLocalCacheCmd},
	}

	checkSet["cniv2"] = []check{
		{"cns cache", cnsCacheStateFileIps, cnsLabelSelector, privilegedNamespace, cnsLocalCacheCmd},
		{"azure cns noncilium", azureCNSNonCiliumStateFileIPs, privilegedLabelSelector, privilegedNamespace, azureCnsStateFileCmd},
	}

	for _, check := range checkSet[v.cni] {
		err := v.validate(check.stateFileIps, check.cmd, check.name, check.podNamespace, check.podLabelSelector)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *LinuxValidator) ValidateRestartNetwork() error {
	nodes, err := k8sutils.GetNodeList(v.ctx, v.clientset)
	if err != nil {
		return errors.Wrapf(err, "failed to get node list")
	}

	for index := range nodes.Items {
		// get the privileged pod
		pod, err := k8sutils.GetPodsByNode(v.ctx, v.clientset, privilegedNamespace, privilegedLabelSelector, nodes.Items[index].Name)
		if err != nil {
			return errors.Wrapf(err, "failed to get privileged pod")
		}

		privelegedPod := pod.Items[0]
		// exec into the pod to get the state file
		_, err = k8sutils.ExecCmdOnPod(v.ctx, v.clientset, privilegedNamespace, privelegedPod.Name, restartNetworkCmd, v.config)
		if err != nil {
			return errors.Wrapf(err, "failed to exec into privileged pod")
		}
		err = k8sutils.WaitForPodsRunning(v.ctx, v.clientset, "", "")
		if err != nil {
			return errors.Wrapf(err, "failed to wait for pods running")
		}
	}
	return nil
}

func cnsStateFileIps(result []byte) (map[string]string, error) {
	var cnsResult CnsState
	err := json.Unmarshal(result, &cnsResult)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal cns endpoint list")
	}

	cnsPodIps := make(map[string]string)
	for _, v := range cnsResult.Endpoints {
		for ifName, ip := range v.IfnameToIPMap {
			if ifName == "eth0" {
				ip := ip.IPv4[0].IP.String()
				cnsPodIps[ip] = v.PodName
			}
		}
	}
	return cnsPodIps, nil
}

func azureCNSNonCiliumStateFileIPs(result []byte) (map[string]string, error) {
	var azureCNSNonCiliumResult AzureCniState
	err := json.Unmarshal(result, &azureCNSNonCiliumResult)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal azure cni endpoint list")
	}

	azureCnsPodIps := make(map[string]string)
	for _, v := range azureCNSNonCiliumResult.AzureCniState.ExternalInterfaces {
		for _, networks := range v.Networks {
			for _, ip := range networks.Endpoints {
				pod := ip.PODName
				ipv4 := ip.IPAddresses[0].IP
				azureCnsPodIps[ipv4] = pod

				if len(ip.IPAddresses) > 1 {
					ipv6 := ip.IPAddresses[1].IP
					azureCnsPodIps[ipv6] = pod
				}
			}
		}
	}
	return azureCnsPodIps, nil
}

func ciliumStateFileIps(result []byte) (map[string]string, error) {
	var ciliumResult []CiliumEndpointStatus
	err := json.Unmarshal(result, &ciliumResult)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal cilium endpoint list")
	}

	ciliumPodIps := make(map[string]string)
	for _, v := range ciliumResult {
		for _, addr := range v.Status.Networking.Addresses {
			if addr.Addr != "" {
				ciliumPodIps[addr.Addr] = v.Status.Networking.InterfaceName
			}
		}
	}
	return ciliumPodIps, nil
}

func cnsCacheStateFileIps(result []byte) (map[string]string, error) {
	var cnsLocalCache CNSLocalCache

	err := json.Unmarshal(result, &cnsLocalCache)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal cns local cache")
	}

	cnsPodIps := make(map[string]string)
	for index := range cnsLocalCache.IPConfigurationStatus {
		cnsPodIps[cnsLocalCache.IPConfigurationStatus[index].IPAddress] = cnsLocalCache.IPConfigurationStatus[index].PodInfo.Name()
	}
	return cnsPodIps, nil
}

func (v *LinuxValidator) validate(stateFileIps stateFileIpsFunc, cmd []string, checkType, namespace, labelSelector string) error {
	log.Printf("Validating %s state file", checkType)
	nodes, err := k8sutils.GetNodeList(v.ctx, v.clientset)
	if err != nil {
		return errors.Wrapf(err, "failed to get node list")
	}

	for index := range nodes.Items {
		// get the privileged pod
		pod, err := k8sutils.GetPodsByNode(v.ctx, v.clientset, namespace, labelSelector, nodes.Items[index].Name)
		if err != nil {
			return errors.Wrapf(err, "failed to get privileged pod")
		}
		podName := pod.Items[0].Name
		// exec into the pod to get the state file
		result, err := k8sutils.ExecCmdOnPod(v.ctx, v.clientset, namespace, podName, cmd, v.config)
		if err != nil {
			return errors.Wrapf(err, "failed to exec into privileged pod")
		}
		filePodIps, err := stateFileIps(result)
		if err != nil {
			return errors.Wrapf(err, "failed to get pod ips from state file")
		}
		if len(filePodIps) == 0 && v.restartCase {
			log.Printf("No pods found on node %s", nodes.Items[index].Name)
			continue
		}
		// get the pod ips
		podIps := getPodIPsWithoutNodeIP(v.ctx, v.clientset, nodes.Items[index])

		check := compareIPs(filePodIps, podIps)

		if !check {
			return errors.Wrapf(errors.New("State file validation failed"), "for %s on node %s", checkType, nodes.Items[index].Name)
		}
	}
	log.Printf("State file validation for %s passed", checkType)
	return nil
}

func (v *LinuxValidator) ValidateDualStackNodeProperties() error {
	log.Print("Validating Dualstack Overlay Linux Node properties")
	nodes, err := k8sutils.GetNodeList(v.ctx, v.clientset)
	if err != nil {
		return errors.Wrapf(err, "failed to get node list")
	}

	for index := range nodes.Items {
		nodeName := nodes.Items[index].ObjectMeta.Name
		// check node status
		nodeConditions := nodes.Items[index].Status.Conditions
		if nodeConditions[len(nodeConditions)-1].Type != corev1.NodeReady {
			return errors.Wrapf(err, "node %s status is not ready", nodeName)
		}

		// get node labels
		nodeLabels := nodes.Items[index].ObjectMeta.GetLabels()
		for key := range nodeLabels {
			if value, ok := dualstackoverlaynodelabel[key]; ok {
				log.Printf("label %s is correctly shown on the node %+v", key, nodeName)
				if value != overlayClusterLabelName {
					return errors.Wrapf(err, "node %s overlay label name is wrong", nodeName)
				}
			}
		}

		// get node allocated IPs and check whether it includes ipv4 and ipv6 address
		// node status addresses object will return three objects; two of them are ip addresses object(one is ipv4 and one is ipv6)
		if len(nodes.Items[index].Status.Addresses) < dualstackNodeStatusAddr {
			return errors.Wrapf(err, "node %s is missing IPv6 internal IP", nodeName)
		}
	}

	return nil
}
