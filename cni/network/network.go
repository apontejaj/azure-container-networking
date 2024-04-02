// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cni/api"
	"github.com/Azure/azure-container-networking/cni/util"
	"github.com/Azure/azure-container-networking/cns"
	cnscli "github.com/Azure/azure-container-networking/cns/client"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/iptables"
	"github.com/Azure/azure-container-networking/netio"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/network"
	"github.com/Azure/azure-container-networking/network/policy"
	"github.com/Azure/azure-container-networking/platform"
	nnscontracts "github.com/Azure/azure-container-networking/proto/nodenetworkservice/3.302.0.744"
	"github.com/Azure/azure-container-networking/store"
	"github.com/Azure/azure-container-networking/telemetry"
	cniSkel "github.com/containernetworking/cni/pkg/skel"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/100"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	dockerNetworkOption = "com.docker.network.generic"
	OpModeTransparent   = "transparent"
	// Supported IP version. Currently support only IPv4
	ipamV6                = "azure-vnet-ipamv6"
	defaultRequestTimeout = 15 * time.Second
	ipv4FullMask          = 32
	ipv6FullMask          = 128
)

// CNI Operation Types
const (
	CNI_ADD    = "ADD"
	CNI_DEL    = "DEL"
	CNI_UPDATE = "UPDATE"
)

const (
	// URL to query NMAgent version and determine whether we snat on host
	nmAgentSupportedApisURL = "http://168.63.129.16/machine/plugins/?comp=nmagent&type=GetSupportedApis"
	// Only SNAT support (no DNS support)
	nmAgentSnatSupportAPI = "NetworkManagementSnatSupport"
	// SNAT and DNS are both supported
	nmAgentSnatAndDnsSupportAPI = "NetworkManagementDNSSupport"
)

// temporary consts related func determineSnat() which is to be deleted after
// a baking period with newest NMAgent changes
const (
	jsonFileExtension = ".json"
)

// NetPlugin represents the CNI network plugin.
type NetPlugin struct {
	*cni.Plugin
	nm                 network.NetworkManager
	ipamInvoker        IPAMInvoker
	report             *telemetry.CNIReport
	tb                 *telemetry.TelemetryBuffer
	nnsClient          NnsClient
	multitenancyClient MultitenancyClient
}

type PolicyArgs struct {
	nwInfo    *network.NetworkInfo
	nwCfg     *cni.NetworkConfig
	ipconfigs []*network.IPConfig
}

// client for node network service
type NnsClient interface {
	// Do network port programming for the pod via node network service.
	// podName - name of the pod as received from containerD
	// nwNamesapce - network namespace name as received from containerD
	AddContainerNetworking(ctx context.Context, podName, nwNamespace string) (*nnscontracts.ConfigureContainerNetworkingResponse, error)

	// Undo or delete network port programming for the pod via node network service.
	// podName - name of the pod as received from containerD
	// nwNamesapce - network namespace name as received from containerD
	DeleteContainerNetworking(ctx context.Context, podName, nwNamespace string) (*nnscontracts.ConfigureContainerNetworkingResponse, error)
}

// snatConfiguration contains a bool that determines whether CNI enables snat on host and snat for dns
type snatConfiguration struct {
	EnableSnatOnHost bool
	EnableSnatForDns bool
}

// NewPlugin creates a new NetPlugin object.
func NewPlugin(name string,
	config *common.PluginConfig,
	client NnsClient,
	multitenancyClient MultitenancyClient,
) (*NetPlugin, error) {
	// Setup base plugin.
	plugin, err := cni.NewPlugin(name, config.Version)
	if err != nil {
		return nil, err
	}

	nl := netlink.NewNetlink()
	// Setup network manager.
	nm, err := network.NewNetworkManager(nl, platform.NewExecClient(logger), &netio.NetIO{}, network.NewNamespaceClient(), iptables.NewClient())
	if err != nil {
		return nil, err
	}

	config.NetApi = nm

	return &NetPlugin{
		Plugin:             plugin,
		nm:                 nm,
		nnsClient:          client,
		multitenancyClient: multitenancyClient,
	}, nil
}

func (plugin *NetPlugin) SetCNIReport(report *telemetry.CNIReport, tb *telemetry.TelemetryBuffer) {
	plugin.report = report
	plugin.tb = tb
}

// Starts the plugin.
func (plugin *NetPlugin) Start(config *common.PluginConfig) error {
	// Initialize base plugin.
	err := plugin.Initialize(config)
	if err != nil {
		logger.Error("Failed to initialize base plugin", zap.Error(err))
		return err
	}

	// Log platform information.
	logger.Info("Plugin Info",
		zap.String("name", plugin.Name),
		zap.String("version", plugin.Version))

	// Initialize network manager. rehyrdration not required on reboot for cni plugin
	err = plugin.nm.Initialize(config, false)
	if err != nil {
		logger.Error("Failed to initialize network manager", zap.Error(err))
		return err
	}

	logger.Info("Plugin started")

	return nil
}

func sendEvent(plugin *NetPlugin, msg string) {
	eventMsg := fmt.Sprintf("[%d] %s", os.Getpid(), msg)
	plugin.report.Version = plugin.Version
	plugin.report.EventMessage = eventMsg
	telemetry.SendCNIEvent(plugin.tb, plugin.report)
}

func (plugin *NetPlugin) GetAllEndpointState(networkid string) (*api.AzureCNIState, error) {
	st := api.AzureCNIState{
		ContainerInterfaces: make(map[string]api.PodNetworkInterfaceInfo),
	}

	eps, err := plugin.nm.GetAllEndpoints(networkid)
	if err == store.ErrStoreEmpty {
		logger.Error("failed to retrieve endpoint state", zap.Error(err))
	} else if err != nil {
		return nil, err
	}

	for _, ep := range eps {
		id := ep.Id
		info := api.PodNetworkInterfaceInfo{
			PodName:       ep.PODName,
			PodNamespace:  ep.PODNameSpace,
			PodEndpointId: ep.Id,
			ContainerID:   ep.ContainerID,
			IPAddresses:   ep.IPAddresses,
		}

		st.ContainerInterfaces[id] = info
	}

	return &st, nil
}

// Stops the plugin.
func (plugin *NetPlugin) Stop() {
	plugin.nm.Uninitialize()
	plugin.Uninitialize()
	logger.Info("Plugin stopped")
}

// findInterfaceByMAC returns the name of the master interface
func (plugin *NetPlugin) findInterfaceByMAC(macAddress string) string {
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		// find master interface by macAddress for Swiftv2
		if iface.HardwareAddr.String() == macAddress {
			return iface.Name
		}
	}
	// Failed to find a suitable interface.
	return ""
}

// findMasterInterfaceBySubnet returns the name of the master interface.
func (plugin *NetPlugin) findMasterInterfaceBySubnet(nwCfg *cni.NetworkConfig, subnetPrefix *net.IPNet) string {
	// An explicit master configuration wins. Explicitly specifying a master is
	// useful if host has multiple interfaces with addresses in the same subnet.
	if nwCfg.Master != "" {
		return nwCfg.Master
	}

	// Otherwise, pick the first interface with an IP address in the given subnet.
	subnetPrefixString := subnetPrefix.String()
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			_, ipnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if subnetPrefixString == ipnet.String() {
				return iface.Name
			}
		}
	}

	// Failed to find a suitable interface.
	return ""
}

// GetEndpointID returns a unique endpoint ID based on the CNI args.
func GetEndpointID(args *cniSkel.CmdArgs) string {
	infraEpId, _ := network.ConstructEndpointID(args.ContainerID, args.Netns, args.IfName)
	return infraEpId
}

// getPodInfo returns POD info by parsing the CNI args.
func (plugin *NetPlugin) getPodInfo(args string) (name, ns string, err error) {
	podCfg, err := cni.ParseCniArgs(args)
	if err != nil {
		logger.Error("Error while parsing CNI Args", zap.Error(err))
		return "", "", err
	}

	k8sNamespace := string(podCfg.K8S_POD_NAMESPACE)
	if len(k8sNamespace) == 0 {
		errMsg := "Pod Namespace not specified in CNI Args"
		logger.Error(errMsg)
		return "", "", plugin.Errorf(errMsg)
	}

	k8sPodName := string(podCfg.K8S_POD_NAME)
	if len(k8sPodName) == 0 {
		errMsg := "Pod Name not specified in CNI Args"
		logger.Error(errMsg)
		return "", "", plugin.Errorf(errMsg)
	}

	return k8sPodName, k8sNamespace, nil
}

func SetCustomDimensions(cniMetric *telemetry.AIMetric, nwCfg *cni.NetworkConfig, err error) {
	if cniMetric == nil {
		logger.Error("Unable to set custom dimension. Report is nil")
		return
	}

	if err != nil {
		cniMetric.Metric.CustomDimensions[telemetry.StatusStr] = telemetry.FailedStr
	} else {
		cniMetric.Metric.CustomDimensions[telemetry.StatusStr] = telemetry.SucceededStr
	}

	if nwCfg != nil {
		if nwCfg.MultiTenancy {
			cniMetric.Metric.CustomDimensions[telemetry.CNIModeStr] = telemetry.MultiTenancyStr
		} else {
			cniMetric.Metric.CustomDimensions[telemetry.CNIModeStr] = telemetry.SingleTenancyStr
		}

		cniMetric.Metric.CustomDimensions[telemetry.CNINetworkModeStr] = nwCfg.Mode
	}
}

func (plugin *NetPlugin) setCNIReportDetails(nwCfg *cni.NetworkConfig, opType, msg string) {
	plugin.report.OperationType = opType
	plugin.report.SubContext = fmt.Sprintf("%+v", nwCfg)
	plugin.report.EventMessage = msg
	plugin.report.BridgeDetails.NetworkMode = nwCfg.Mode
	plugin.report.InterfaceDetails.SecondaryCAUsedCount = plugin.nm.GetNumberOfEndpoints("", nwCfg.Name)
}

func addNatIPV6SubnetInfo(nwCfg *cni.NetworkConfig,
	resultV6 *cniTypesCurr.Result,
	nwInfo *network.NetworkInfo,
) {
	if nwCfg.IPV6Mode == network.IPV6Nat {
		ipv6Subnet := resultV6.IPs[0].Address
		ipv6Subnet.IP = ipv6Subnet.IP.Mask(ipv6Subnet.Mask)
		ipv6SubnetInfo := network.SubnetInfo{
			Family:  platform.AfINET6,
			Prefix:  ipv6Subnet,
			Gateway: resultV6.IPs[0].Gateway,
		}
		logger.Info("ipv6 subnet info",
			zap.Any("ipv6SubnetInfo", ipv6SubnetInfo))
		nwInfo.Subnets = append(nwInfo.Subnets, ipv6SubnetInfo)
	}
}

func (plugin *NetPlugin) addIpamInvoker(ipamAddConfig IPAMAddConfig) (IPAMAddResult, error) {
	ipamAddResult, err := plugin.ipamInvoker.Add(ipamAddConfig)
	if err != nil {
		return IPAMAddResult{}, err
	}
	sendEvent(plugin, fmt.Sprintf("Allocated IPAddress from ipam interfaces: %+v", ipamAddResult.interfaceInfo))
	return ipamAddResult, nil
}

// get network
func (plugin *NetPlugin) getNetworkID(netNs string, interfaceInfo *network.InterfaceInfo, nwCfg *cni.NetworkConfig) (string, error) {
	networkID, err := plugin.getNetworkName(netNs, interfaceInfo, nwCfg)
	if err != nil {
		return "", err
	}
	return networkID, nil
}

// get network info for legacy
func (plugin *NetPlugin) getNetworkInfo(netNs string, interfaceInfo *network.InterfaceInfo, nwCfg *cni.NetworkConfig) network.EndpointInfo {
	networkID, _ := plugin.getNetworkID(netNs, interfaceInfo, nwCfg)
	nwInfo, _ := plugin.nm.GetNetworkInfo(networkID)

	return nwInfo
}

// CNI implementation
// https://github.com/containernetworking/cni/blob/master/SPEC.md

// Add handles CNI add commands.
func (plugin *NetPlugin) Add(args *cniSkel.CmdArgs) error {
	var (
		ipamAddResult    IPAMAddResult
		ipamAddResults   []IPAMAddResult
		azIpamResult     *cniTypesCurr.Result
		enableInfraVnet  bool
		enableSnatForDNS bool
		k8sPodName       string
		cniMetric        telemetry.AIMetric
	)

	startTime := time.Now()

	logger.Info("Processing ADD command",
		zap.String("containerId", args.ContainerID),
		zap.String("netNS", args.Netns),
		zap.String("ifName", args.IfName),
		zap.Any("args", args.Args),
		zap.String("path", args.Path),
		zap.ByteString("stdinData", args.StdinData))
	sendEvent(plugin, fmt.Sprintf("[cni-net] Processing ADD command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData))

	// Parse network configuration from stdin.
	nwCfg, err := cni.ParseNetworkConfig(args.StdinData)
	if err != nil {
		err = plugin.Errorf("Failed to parse network configuration: %v.", err)
		return err
	}

	iptables.DisableIPTableLock = nwCfg.DisableIPTableLock
	plugin.setCNIReportDetails(nwCfg, CNI_ADD, "")

	defer func() {
		operationTimeMs := time.Since(startTime).Milliseconds()
		cniMetric.Metric = aitelemetry.Metric{
			Name:             telemetry.CNIAddTimeMetricStr,
			Value:            float64(operationTimeMs),
			AppVersion:       plugin.Version,
			CustomDimensions: make(map[string]string),
		}
		SetCustomDimensions(&cniMetric, nwCfg, err)
		telemetry.SendCNIMetric(&cniMetric, plugin.tb)

		// Add Interfaces to result.
		for _, intfInfo := range ipamAddResult.interfaceInfo {
			cniResult := convertInterfaceInfoToCniResult(intfInfo, args.IfName)
			addSnatInterface(nwCfg, cniResult)
			// Convert result to the requested CNI version.
			res, vererr := cniResult.GetAsVersion(nwCfg.CNIVersion)
			if vererr != nil {
				logger.Error("GetAsVersion failed", zap.Error(vererr))
				plugin.Error(vererr)
			}

			if err == nil && res != nil {
				// Output the result to stdout.
				res.Print()
			}

			logger.Info("ADD command completed for",
				zap.String("pod", k8sPodName),
				zap.Any("IPs", cniResult.IPs),
				zap.Error(err))
		}
	}()

	// Parse Pod arguments.
	k8sPodName, k8sNamespace, err := plugin.getPodInfo(args.Args)
	if err != nil {
		return err
	}

	plugin.report.ContainerName = k8sPodName + ":" + k8sNamespace

	k8sContainerID := args.ContainerID
	if len(k8sContainerID) == 0 {
		errMsg := "Container ID not specified in CNI Args"
		logger.Error(errMsg)
		return plugin.Errorf(errMsg)
	}

	k8sIfName := args.IfName
	if len(k8sIfName) == 0 {
		errMsg := "Interfacename not specified in CNI Args"
		logger.Error(errMsg)
		return plugin.Errorf(errMsg)
	}

	platformInit(nwCfg)
	if nwCfg.ExecutionMode == string(util.Baremetal) {
		var res *nnscontracts.ConfigureContainerNetworkingResponse
		logger.Info("Baremetal mode. Calling vnet agent for ADD")
		res, err = plugin.nnsClient.AddContainerNetworking(context.Background(), k8sPodName, args.Netns)

		if err == nil {
			ipamAddResult.interfaceInfo = append(ipamAddResult.interfaceInfo, network.InterfaceInfo{
				IPConfigs: convertNnsToIPConfigs(res, args.IfName, k8sPodName, "AddContainerNetworking"),
				NICType:   cns.InfraNIC,
			})
		}

		return err
	}

	for _, ns := range nwCfg.PodNamespaceForDualNetwork {
		if k8sNamespace == ns {
			logger.Info("Enable infravnet for pod",
				zap.String("pod", k8sPodName),
				zap.String("namespace", k8sNamespace))
			enableInfraVnet = true
			break
		}
	}

	cnsClient, err := cnscli.New(nwCfg.CNSUrl, defaultRequestTimeout)
	if err != nil {
		return fmt.Errorf("failed to create cns client with error: %w", err)
	}

	options := make(map[string]any)
	ipamAddConfig := IPAMAddConfig{nwCfg: nwCfg, args: args, options: options}

	if nwCfg.MultiTenancy {
		plugin.report.Context = "AzureCNIMultitenancy"
		plugin.multitenancyClient.Init(cnsClient, AzureNetIOShim{})

		// Temporary if block to determining whether we disable SNAT on host (for multi-tenant scenario only)
		if enableSnatForDNS, nwCfg.EnableSnatOnHost, err = plugin.multitenancyClient.DetermineSnatFeatureOnHost(
			snatConfigFileName, nmAgentSupportedApisURL); err != nil {
			return fmt.Errorf("%w", err)
		}

		ipamAddResult, err = plugin.multitenancyClient.GetAllNetworkContainers(context.TODO(), nwCfg, k8sPodName, k8sNamespace, args.IfName)
		if err != nil {
			err = fmt.Errorf("GetAllNetworkContainers failed for podname %s namespace %s. error: %w", k8sPodName, k8sNamespace, err)
			logger.Error("GetAllNetworkContainers failed",
				zap.String("pod", k8sPodName),
				zap.String("namespace", k8sNamespace),
				zap.Error(err))
			return err
		}

		if !plugin.isDualNicFeatureSupported(args.Netns) {
			errMsg := fmt.Sprintf("received multiple NC results %+v from CNS while dualnic feature is not supported", ipamAddResults)
			logger.Error("received multiple NC results from CNS while dualnic feature is not supported",
				zap.Any("results", ipamAddResult))
			return plugin.Errorf(errMsg)
		}
	} else {
		if plugin.ipamInvoker == nil {
			switch nwCfg.IPAM.Type {
			case network.AzureCNS:
				plugin.ipamInvoker = NewCNSInvoker(k8sPodName, k8sNamespace, cnsClient, util.ExecutionMode(nwCfg.ExecutionMode), util.IpamMode(nwCfg.IPAM.Mode))
			default:
				// legacy
				nwInfo := plugin.getNetworkInfo(args.Netns, nil, nwCfg)
				plugin.ipamInvoker = NewAzureIpamInvoker(plugin, &nwInfo)
			}
		}

		ipamAddResult, err = plugin.addIpamInvoker(ipamAddConfig)
		if err != nil {
			return fmt.Errorf("IPAM Invoker Add failed with error: %w", err)
		}
		//TODO: remove
		ifIndex, _ := findDefaultInterface(ipamAddResult)

		if ifIndex == -1 {
			ifIndex, _ = findDefaultInterface(ipamAddResult)
		}
		// This proably needs to be changed as we return all interfaces...
		sendEvent(plugin, fmt.Sprintf("Allocated IPAddress from ipam DefaultInterface: %+v, SecondaryInterfaces: %+v", ipamAddResult.interfaceInfo[ifIndex], ipamAddResult.interfaceInfo))
	}

	ifIndex, _ := findDefaultInterface(ipamAddResult)
	endpointID := plugin.nm.GetEndpointID(args.ContainerID, args.IfName)
	policies := cni.GetPoliciesFromNwCfg(nwCfg.AdditionalArgs)

	// if !nwCfg.MultiTenancy && nwCfg.IPAM.Type == network.AzureCNS {
	// 	nwInfo, nwInfoErr = plugin.getNetworkInfo(args.Netns, ipamAddResult, nwCfg)
	// }

	// Check whether the network already exists.

	// Handle consecutive ADD calls for infrastructure containers.
	// This is a temporary work around for issue #57253 of Kubernetes.
	// We can delete this if statement once they fix it.
	// Issue link: https://github.com/kubernetes/kubernetes/issues/57253

	// if nwInfoErr == nil {
	// 	logger.Info("Found network with subnet",
	// 		zap.String("network", networkID),
	// 		zap.String("subnet", nwInfo.Subnets[0].Prefix.String()))
	// 	nwInfo.IPAMType = nwCfg.IPAM.Type
	// 	options = nwInfo.Options

	// 	var resultSecondAdd *cniTypesCurr.Result
	// 	resultSecondAdd, err = plugin.handleConsecutiveAdd(args, endpointID, networkID, &nwInfo, nwCfg)
	// 	if err != nil {
	// 		logger.Error("handleConsecutiveAdd failed", zap.Error(err))
	// 		return err
	// 	}

	// 	if resultSecondAdd != nil {
	// 		ifIndex, err = findDefaultInterface(ipamAddResult)
	// 		if err != nil {
	// 			ipamAddResult.interfaceInfo = append(ipamAddResult.interfaceInfo, network.InterfaceInfo{})
	// 			ifIndex = len(ipamAddResult.interfaceInfo) - 1
	// 		}
	// 		ipamAddResult.interfaceInfo[ifIndex] = convertCniResultToInterfaceInfo(resultSecondAdd)
	// 		return nil
	// 	}
	// }

	defer func() { //nolint:gocritic
		if err != nil {
			// for multi-tenancies scenario, CNI is not supposed to invoke CNS for cleaning Ips
			if !(nwCfg.MultiTenancy && nwCfg.IPAM.Type == network.AzureCNS) {
				plugin.cleanupAllocationOnError(ipamAddResult.interfaceInfo[ifIndex].IPConfigs, nwCfg, args, options)
			}
		}
	}()

	// TODO: is it possible nwInfo is not populated? YES it seems so, in which case the nwInfo is an empty struct!
	// TODO: This will mean when we try to do nwInfo.Id or nwInfo.Subnets in createEpInfo, we'll have problems!
	epInfos := []*network.EndpointInfo{}
	for _, ifInfo := range ipamAddResult.interfaceInfo {
		// TODO: hopefully I can get natInfo here?
		natInfo := getNATInfo(nwCfg, options[network.SNATIPKey], enableSnatForDNS)
		networkID, _ := plugin.getNetworkID(args.Netns, &ifInfo, nwCfg)
		createEpInfoOpt := createEpInfoOpt{
			nwCfg:            nwCfg,
			cnsNetworkConfig: ifInfo.NCResponse,
			ipamAddResult:    ipamAddResult,
			azIpamResult:     azIpamResult,
			args:             args,
			policies:         policies,
			endpointID:       endpointID,
			k8sPodName:       k8sPodName,
			k8sNamespace:     k8sNamespace,
			enableInfraVnet:  enableInfraVnet,
			enableSnatForDNS: enableSnatForDNS,
			natInfo:          natInfo,
			networkID:        networkID,

			ifInfo:        &ifInfo,
			ipamAddConfig: &ipamAddConfig,
			ipv6Enabled:   ipamAddResult.ipv6Enabled,
		}
		var epInfo *network.EndpointInfo
		epInfo, err = plugin.createEpInfo(&createEpInfoOpt)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		epInfos = append(epInfos, epInfo)
		// TODO: should this statement be based on the current iteration instead of the constant ifIndex?
		sendEvent(plugin, fmt.Sprintf("CNI ADD succeeded: IP:%+v, VlanID: %v, podname %v, namespace %v numendpoints:%d",
			ipamAddResult.interfaceInfo[ifIndex].IPConfigs, epInfo.Data[network.VlanIDKey], k8sPodName, k8sNamespace, plugin.nm.GetNumberOfEndpoints("", nwCfg.Name)))

	}
	cnsclient, err := cnscli.New(nwCfg.CNSUrl, defaultRequestTimeout)
	if err != nil {
		return errors.Wrap(err, "failed to create cns client")
	}
	plugin.nm.EndpointCreate(cnsclient, epInfos)

	return nil

}

// loop over each interface info and call createEndpoint with it
// you don't have access to a list of interface infos here
// no need for default index because we only trigger the default index code when the type is the infra nic
/**
Overview:
Create endpoint info
Pass pointer to endpoint info to a new function to populate endpoint info fields (right now we don't separate the populating code into functions)
Pass pointer to endpoint info to another new function to populate network info fields (or we can just do it in one big switch statement)
  To populate network info, use Paul's code which should get the network info fields from various things passed in (we should pass those things in here too)
  Potentially problematic getting the network id because plugin.getNetworkName seems to just always return nwCfg.Name (NEED ASSISTANCE)
  But if we do get the network id correctly, just pass to Paul's code to get the network info fields which we can populate epInfo with
After both sections are done, we can call createNetworkInternal (pass in epInfo) and createEndpointInternal (pass in epInfo)
  Will need to modify those two functions as code from createEndpointInternal, for example, was pulled out into this function below
*/
// maybe create a createEndpointOpt struct for all the things we need to pass in

func (plugin *NetPlugin) findMasterInterface(opt *createEpInfoOpt) string {
	switch opt.ifInfo.NICType {
	case cns.InfraNIC:
		return plugin.findMasterInterfaceBySubnet(opt.ipamAddConfig.nwCfg, &opt.ifInfo.HostSubnetPrefix)
	case cns.DelegatedVMNIC:
		return plugin.findInterfaceByMAC(opt.ifInfo.MacAddress.String())
	default:
		return ""
	}
}

type createEpInfoOpt struct {
	nwCfg            *cni.NetworkConfig
	cnsNetworkConfig *cns.GetNetworkContainerResponse
	ipamAddResult    IPAMAddResult
	azIpamResult     *cniTypesCurr.Result
	args             *cniSkel.CmdArgs
	policies         []policy.Policy
	endpointID       string
	k8sPodName       string
	k8sNamespace     string
	enableInfraVnet  bool
	enableSnatForDNS bool
	natInfo          []policy.NATInfo
	networkID        string

	ifInfo        *network.InterfaceInfo
	ipamAddConfig *IPAMAddConfig
	ipv6Enabled   bool
}

func (plugin *NetPlugin) createEpInfo(opt *createEpInfoOpt) (*network.EndpointInfo, error) { // you can modify to pass in whatever else you need
	var (
		epInfo network.EndpointInfo
		nwInfo network.NetworkInfo
	)

	// populate endpoint info section
	masterIfName := plugin.findMasterInterface(opt)
	if masterIfName == "" {
		err := plugin.Errorf("Failed to find the master interface")
		return nil, err
	}
	logger.Info("Found master interface", zap.String("ifname", masterIfName))
	if err := plugin.addExternalInterface(masterIfName, opt.ifInfo.HostSubnetPrefix.String()); err != nil {
		return nil, err
	}

	nwDNSInfo, err := plugin.getNetworkDNSSettings(opt.ipamAddConfig.nwCfg, opt.ifInfo.DNS)
	if err != nil {
		return nil, err
	}

	nwInfo = network.NetworkInfo{
		Id:                            opt.networkID,
		Mode:                          opt.ipamAddConfig.nwCfg.Mode,
		MasterIfName:                  masterIfName,
		AdapterName:                   opt.ipamAddConfig.nwCfg.AdapterName,
		BridgeName:                    opt.ipamAddConfig.nwCfg.Bridge,
		EnableSnatOnHost:              opt.ipamAddConfig.nwCfg.EnableSnatOnHost, // (unused) overridden by endpoint info value NO CONFLICT: Confirmed same value as field with same name in epInfo above
		DNS:                           nwDNSInfo,                                // (unused) overridden by endpoint info value POSSIBLE CONFLICT TODO: Check what takes precedence
		Policies:                      opt.policies,                             // not present in non-infra
		NetNs:                         opt.ipamAddConfig.args.Netns,
		Options:                       opt.ipamAddConfig.options,
		DisableHairpinOnHostInterface: opt.ipamAddConfig.nwCfg.DisableHairpinOnHostInterface,
		IPV6Mode:                      opt.ipamAddConfig.nwCfg.IPV6Mode,     // not present in non-infra TODO: check if IPV6Mode field can be deprecated // (unused) overridden by endpoint info value NO CONFLICT: Confirmed same value as field with same name in epInfo above
		IPAMType:                      opt.ipamAddConfig.nwCfg.IPAM.Type,    // not present in non-infra
		ServiceCidrs:                  opt.ipamAddConfig.nwCfg.ServiceCidrs, // (unused) overridden by endpoint info value NO CONFLICT: Confirmed same value as field with same name in epInfo above
		IsIPv6Enabled:                 opt.ipv6Enabled,                      // not present in non-infra
		NICType:                       string(opt.ifInfo.NICType),
	}

	if err := addSubnetToNetworkInfo(*opt.ifInfo, &nwInfo); err != nil {
		logger.Info("Failed to add subnets to networkInfo", zap.Error(err))
		return nil, err
	}
	setNetworkOptions(opt.ifInfo.NCResponse, &nwInfo)

	// populate endpoint info
	// taken from createEndpointInternal function
	// populate endpoint info fields code is below
	// did not move cns client code because we don't need it here

	// taken from create endpoint internal function
	// TODO: does this change the behavior (we moved the infra nic code into here rather than above the switch statement)? (it shouldn't)
	// at this point you are 100% certain that the interface passed in is the infra nic, and thus, the default interface info
	defaultInterfaceInfo := opt.ifInfo
	epDNSInfo, err := getEndpointDNSSettings(opt.nwCfg, defaultInterfaceInfo.DNS, opt.k8sNamespace) // Probably won't panic if given bad values
	if err != nil {
		// TODO: will it error out if we have a secondary endpoint that has blank DNS though? If so, problem!
		err = plugin.Errorf("Failed to getEndpointDNSSettings: %v", err)
		return nil, err
	}
	policyArgs := PolicyArgs{
		// pass podsubnet info etc. part of epinfo
		nwInfo:    &nwInfo, // TODO: (1/2 opt.nwInfo) we do not have the full nw info created yet-- is this okay? getEndpointPolicies requires nwInfo.Subnets only (checked)
		nwCfg:     opt.nwCfg,
		ipconfigs: defaultInterfaceInfo.IPConfigs,
	}
	endpointPolicies, err := getEndpointPolicies(policyArgs)
	if err != nil {
		// TODO: should only error out if we have an ip config and it is not readable
		logger.Error("Failed to get endpoint policies", zap.Error(err))
		return nil, err
	}

	opt.policies = append(opt.policies, endpointPolicies...)

	vethName := fmt.Sprintf("%s.%s", opt.k8sNamespace, opt.k8sPodName)
	if opt.nwCfg.Mode != OpModeTransparent {
		// this mechanism of using only namespace and name is not unique for different incarnations of POD/container.
		// IT will result in unpredictable behavior if API server decides to
		// reorder DELETE and ADD call for new incarnation of same POD.
		vethName = fmt.Sprintf("%s%s%s", nwInfo.Id, opt.args.ContainerID, opt.args.IfName) // TODO: (2/2 opt.nwInfo) The dummy nwInfo has the Id, right?
	}

	// for secondary
	var addresses []net.IPNet
	for _, ipconfig := range opt.ifInfo.IPConfigs {
		addresses = append(addresses, ipconfig.Address)
	}

	epInfo = network.EndpointInfo{
		Id:                 opt.endpointID,
		ContainerID:        opt.args.ContainerID,
		NetNsPath:          opt.args.Netns,
		IfName:             opt.args.IfName,
		Data:               make(map[string]interface{}),
		EndpointDNS:        epDNSInfo,
		Policies:           opt.policies,
		IPsToRouteViaHost:  opt.nwCfg.IPsToRouteViaHost,
		EnableSnatOnHost:   opt.nwCfg.EnableSnatOnHost,
		EnableMultiTenancy: opt.nwCfg.MultiTenancy,
		EnableInfraVnet:    opt.enableInfraVnet,
		EnableSnatForDns:   opt.enableSnatForDNS,
		PODName:            opt.k8sPodName,
		PODNameSpace:       opt.k8sNamespace,
		SkipHotAttachEp:    false, // Hot attach at the time of endpoint creation
		IPV6Mode:           opt.nwCfg.IPV6Mode,
		VnetCidrs:          opt.nwCfg.VnetCidrs,
		ServiceCidrs:       opt.nwCfg.ServiceCidrs,
		NATInfo:            opt.natInfo,
		NICType:            opt.ifInfo.NICType,
		SkipDefaultRoutes:  opt.ifInfo.SkipDefaultRoutes, // we know we are the default interface at this point
		Routes:             defaultInterfaceInfo.Routes,
		// added the following for delegated vm nic
		IPAddresses: addresses,
		MacAddress:  opt.ifInfo.MacAddress,
	}

	epPolicies, err := getPoliciesFromRuntimeCfg(opt.nwCfg, opt.ipamAddResult.ipv6Enabled) // not specific to delegated or infra
	if err != nil {
		logger.Error("failed to get policies from runtime configurations", zap.Error(err))
		return nil, plugin.Errorf(err.Error())
	}
	epInfo.Policies = append(epInfo.Policies, epPolicies...)

	// Populate addresses.
	for _, ipconfig := range defaultInterfaceInfo.IPConfigs {
		epInfo.IPAddresses = append(epInfo.IPAddresses, ipconfig.Address)
	}

	if opt.ipamAddResult.ipv6Enabled { // not specific to this particular interface
		epInfo.IPV6Mode = string(util.IpamMode(opt.nwCfg.IPAM.Mode)) // TODO: check IPV6Mode field can be deprecated and can we add IsIPv6Enabled flag for generic working
	}

	if opt.azIpamResult != nil && opt.azIpamResult.IPs != nil {
		epInfo.InfraVnetIP = opt.azIpamResult.IPs[0].Address
	}

	if opt.nwCfg.MultiTenancy {
		plugin.multitenancyClient.SetupRoutingForMultitenancy(opt.nwCfg, opt.cnsNetworkConfig, opt.azIpamResult, &epInfo, defaultInterfaceInfo)
	}

	setEndpointOptions(opt.cnsNetworkConfig, &epInfo, vethName)

	// populate endpoint info with network info fields
	// TODO: maybe instead of creating a network info, directly modify epInfo, but some helper methods rely on networkinfo...
	epInfo.MasterIfName = nwInfo.MasterIfName
	epInfo.AdapterName = nwInfo.AdapterName
	epInfo.NetworkId = nwInfo.Id
	epInfo.Mode = nwInfo.Mode
	epInfo.Subnets = nwInfo.Subnets
	epInfo.PodSubnet = nwInfo.PodSubnet
	epInfo.BridgeName = nwInfo.BridgeName
	epInfo.NetNs = nwInfo.NetNs
	epInfo.Options = nwInfo.Options
	epInfo.DisableHairpinOnHostInterface = nwInfo.DisableHairpinOnHostInterface
	epInfo.IPAMType = nwInfo.IPAMType
	epInfo.IsIPv6Enabled = nwInfo.IsIPv6Enabled
	epInfo.NetworkDNS = nwInfo.DNS

	// now our ep info should have the full combined information from both the network and endpoint structs
	return &epInfo, nil
}

// cleanup allocated ipv4 and ipv6 addresses if they exist
func (plugin *NetPlugin) cleanupAllocationOnError(
	result []*network.IPConfig,
	nwCfg *cni.NetworkConfig,
	args *cniSkel.CmdArgs,
	options map[string]interface{},
) {
	if result != nil {
		for i := 0; i < len(result); i++ {
			if er := plugin.ipamInvoker.Delete(&result[i].Address, nwCfg, args, options); er != nil {
				logger.Error("Failed to cleanup ip allocation on failure", zap.Error(er))
			}
		}
	}
}

// Copied from paul's commit
// Add the master as an external interface
func (plugin *NetPlugin) addExternalInterface(masterIfName, hostSubnetPrefix string) error {
	err := plugin.nm.AddExternalInterface(masterIfName, hostSubnetPrefix)
	if err != nil {
		err = plugin.Errorf("Failed to add external interface: %v", err)
		return err
	}
	return nil
}

func (plugin *NetPlugin) getNetworkDNSSettings(nwCfg *cni.NetworkConfig, dns network.DNSInfo) (network.DNSInfo, error) {
	nwDNSInfo, err := getNetworkDNSSettings(nwCfg, dns)
	if err != nil {
		err = plugin.Errorf("Failed to getDNSSettings: %v", err)
		return network.DNSInfo{}, err
	}
	logger.Info("DNS Info", zap.Any("info", nwDNSInfo))
	return nwDNSInfo, nil
}

// func (plugin *NetPlugin) createNetworkInternal(
// 	networkID string,
// 	policies []policy.Policy,
// 	ipamAddConfig IPAMAddConfig,
// 	ipamAddResult IPAMAddResult,
// 	ifIndex int,
// ) (network.NetworkInfo, error) {
// 	nwInfo := network.NetworkInfo{}
// 	ipamAddResult.hostSubnetPrefix.IP = ipamAddResult.hostSubnetPrefix.IP.Mask(ipamAddResult.hostSubnetPrefix.Mask)
// 	ipamAddConfig.nwCfg.IPAM.Subnet = ipamAddResult.hostSubnetPrefix.String()
// 	// Find the master interface.
// 	masterIfName := plugin.findMasterInterface(ipamAddConfig.nwCfg, &ipamAddResult.hostSubnetPrefix)
// 	if masterIfName == "" {
// 		err := plugin.Errorf("Failed to find the master interface")
// 		return nwInfo, err
// 	}
// 	logger.Info("Found master interface", zap.String("ifname", masterIfName))

// 	// Add the master as an external interface.
// 	err := plugin.nm.AddExternalInterface(masterIfName, ipamAddResult.hostSubnetPrefix.String())
// 	if err != nil {
// 		err = plugin.Errorf("Failed to add external interface: %v", err)
// 		return nwInfo, err
// 	}

// 	nwDNSInfo, err := getNetworkDNSSettings(ipamAddConfig.nwCfg, ipamAddResult.interfaceInfo[ifIndex].DNS)
// 	if err != nil {
// 		err = plugin.Errorf("Failed to getDNSSettings: %v", err)
// 		return nwInfo, err
// 	}

// 	logger.Info("DNS Info", zap.Any("info", nwDNSInfo))

// 	// Create the network.
// 	nwInfo = network.NetworkInfo{
// 		Id:                            networkID,
// 		Mode:                          ipamAddConfig.nwCfg.Mode,
// 		MasterIfName:                  masterIfName,
// 		AdapterName:                   ipamAddConfig.nwCfg.AdapterName,
// 		BridgeName:                    ipamAddConfig.nwCfg.Bridge,
// 		EnableSnatOnHost:              ipamAddConfig.nwCfg.EnableSnatOnHost,
// 		DNS:                           nwDNSInfo,
// 		Policies:                      policies,
// 		NetNs:                         ipamAddConfig.args.Netns,
// 		Options:                       ipamAddConfig.options,
// 		DisableHairpinOnHostInterface: ipamAddConfig.nwCfg.DisableHairpinOnHostInterface,
// 		IPV6Mode:                      ipamAddConfig.nwCfg.IPV6Mode, // TODO: check if IPV6Mode field can be deprecated
// 		IPAMType:                      ipamAddConfig.nwCfg.IPAM.Type,
// 		ServiceCidrs:                  ipamAddConfig.nwCfg.ServiceCidrs,
// 		IsIPv6Enabled:                 ipamAddResult.ipv6Enabled,
// 	}

// 	if err = addSubnetToNetworkInfo(ipamAddResult, &nwInfo, ifIndex); err != nil {
// 		logger.Info("Failed to add subnets to networkInfo",
// 			zap.Error(err))
// 		return nwInfo, err
// 	}
// 	setNetworkOptions(ipamAddResult.interfaceInfo[0].NCResponse, &nwInfo) // Alex will fix this

// 	err = plugin.nm.CreateNetwork(&nwInfo)
// 	if err != nil {
// 		err = plugin.Errorf("createNetworkInternal: Failed to create network: %v", err)
// 	}

// 	return nwInfo, err
// }

// construct network info with ipv4/ipv6 subnets
func addSubnetToNetworkInfo(interfaceInfo network.InterfaceInfo, nwInfo *network.NetworkInfo) error {
	for _, ipConfig := range interfaceInfo.IPConfigs {
		ip, podSubnetPrefix, err := net.ParseCIDR(ipConfig.Address.String())
		if err != nil {
			return fmt.Errorf("Failed to ParseCIDR for pod subnet prefix: %w", err)
		}

		subnet := network.SubnetInfo{
			Family:  platform.AfINET,
			Prefix:  *podSubnetPrefix,
			Gateway: ipConfig.Gateway,
		}
		if ip.To4() == nil {
			subnet.Family = platform.AfINET6
		}

		nwInfo.Subnets = append(nwInfo.Subnets, subnet)
	}

	return nil
}

// type createEndpointInternalOpt struct {
// 	nwCfg            *cni.NetworkConfig
// 	cnsNetworkConfig *cns.GetNetworkContainerResponse
// 	ipamAddResult    IPAMAddResult
// 	azIpamResult     *cniTypesCurr.Result
// 	args             *cniSkel.CmdArgs
// 	nwInfo           *network.NetworkInfo
// 	policies         []policy.Policy
// 	endpointID       string
// 	k8sPodName       string
// 	k8sNamespace     string
// 	enableInfraVnet  bool
// 	enableSnatForDNS bool
// 	natInfo          []policy.NATInfo
// }

// func (plugin *NetPlugin) createEndpointInternal(opt *createEndpointInternalOpt, ifIndex int) (network.EndpointInfo, error) {
// 	epInfo := network.EndpointInfo{}

// 	ifInfo := opt.ipamAddResult.interfaceInfo[ifIndex]
// 	epDNSInfo, err := getEndpointDNSSettings(opt.nwCfg, ifInfo.DNS, opt.k8sNamespace)
// 	if err != nil {
// 		err = plugin.Errorf("Failed to getEndpointDNSSettings: %v", err)
// 		return epInfo, err
// 	}
// 	policyArgs := PolicyArgs{
// 		nwInfo:    opt.nwInfo,
// 		nwCfg:     opt.nwCfg,
// 		ipconfigs: ifInfo.IPConfigs,
// 	}
// 	endpointPolicies, err := getEndpointPolicies(policyArgs)
// 	if err != nil {
// 		logger.Error("Failed to get endpoint policies", zap.Error(err))
// 		return epInfo, err
// 	}

// 	opt.policies = append(opt.policies, endpointPolicies...)

// 	vethName := fmt.Sprintf("%s.%s", opt.k8sNamespace, opt.k8sPodName)
// 	if opt.nwCfg.Mode != OpModeTransparent {
// 		// this mechanism of using only namespace and name is not unique for different incarnations of POD/container.
// 		// IT will result in unpredictable behavior if API server decides to
// 		// reorder DELETE and ADD call for new incarnation of same POD.
// 		vethName = fmt.Sprintf("%s%s%s", opt.nwInfo.Id, opt.args.ContainerID, opt.args.IfName)
// 	}

// 	epInfo = network.EndpointInfo{
// 		Id:                 opt.endpointID,
// 		ContainerID:        opt.args.ContainerID,
// 		NetNsPath:          opt.args.Netns,
// 		IfName:             opt.args.IfName,
// 		Data:               make(map[string]interface{}),
// 		DNS:                epDNSInfo,
// 		Policies:           opt.policies,
// 		IPsToRouteViaHost:  opt.nwCfg.IPsToRouteViaHost,
// 		EnableSnatOnHost:   opt.nwCfg.EnableSnatOnHost,
// 		EnableMultiTenancy: opt.nwCfg.MultiTenancy,
// 		EnableInfraVnet:    opt.enableInfraVnet,
// 		EnableSnatForDns:   opt.enableSnatForDNS,
// 		PODName:            opt.k8sPodName,
// 		PODNameSpace:       opt.k8sNamespace,
// 		SkipHotAttachEp:    false, // Hot attach at the time of endpoint creation
// 		IPV6Mode:           opt.nwCfg.IPV6Mode,
// 		VnetCidrs:          opt.nwCfg.VnetCidrs,
// 		ServiceCidrs:       opt.nwCfg.ServiceCidrs,
// 		NATInfo:            opt.natInfo,
// 		NICType:            cns.InfraNIC,
// 		SkipDefaultRoutes:  opt.ipamAddResult.interfaceInfo[ifIndex].SkipDefaultRoutes,
// 		Routes:             ifInfo.Routes,
// 	}

// 	epPolicies, err := getPoliciesFromRuntimeCfg(opt.nwCfg, opt.ipamAddResult.ipv6Enabled)
// 	if err != nil {
// 		logger.Error("failed to get policies from runtime configurations", zap.Error(err))
// 		return epInfo, plugin.Errorf(err.Error())
// 	}
// 	epInfo.Policies = append(epInfo.Policies, epPolicies...)

// 	// Populate addresses.
// 	for _, ipconfig := range ifInfo.IPConfigs {
// 		epInfo.IPAddresses = append(epInfo.IPAddresses, ipconfig.Address)
// 	}

// 	if opt.ipamAddResult.ipv6Enabled {
// 		epInfo.IPV6Mode = string(util.IpamMode(opt.nwCfg.IPAM.Mode)) // TODO: check IPV6Mode field can be deprecated and can we add IsIPv6Enabled flag for generic working
// 	}

// 	if opt.azIpamResult != nil && opt.azIpamResult.IPs != nil {
// 		epInfo.InfraVnetIP = opt.azIpamResult.IPs[0].Address
// 	}

// 	if opt.nwCfg.MultiTenancy {
// 		plugin.multitenancyClient.SetupRoutingForMultitenancy(opt.nwCfg, opt.cnsNetworkConfig, opt.azIpamResult, &epInfo, &ifInfo)
// 	}

// 	setEndpointOptions(opt.cnsNetworkConfig, &epInfo, vethName)

// 	cnsclient, err := cnscli.New(opt.nwCfg.CNSUrl, defaultRequestTimeout)
// 	if err != nil {
// 		logger.Error("failed to initialized cns client", zap.String("url", opt.nwCfg.CNSUrl),
// 			zap.String("error", err.Error()))
// 		return epInfo, plugin.Errorf(err.Error())
// 	}

// 	epInfos := []*network.EndpointInfo{&epInfo}
// 	epIndex := 0 // epInfo index for InfraNIC
// 	// get secondary interface info
// 	for i := 0; i < len(opt.ipamAddResult.interfaceInfo); i++ {
// 		switch opt.ipamAddResult.interfaceInfo[i].NICType {
// 		case cns.DelegatedVMNIC:
// 			// secondary
// 			var addresses []net.IPNet
// 			for _, ipconfig := range opt.ipamAddResult.interfaceInfo[i].IPConfigs {
// 				addresses = append(addresses, ipconfig.Address)
// 			}

// 			epInfos = append(epInfos,
// 				&network.EndpointInfo{
// 					ContainerID:       epInfo.ContainerID,
// 					NetNsPath:         epInfo.NetNsPath,
// 					IPAddresses:       addresses,
// 					Routes:            opt.ipamAddResult.interfaceInfo[i].Routes,
// 					MacAddress:        opt.ipamAddResult.interfaceInfo[i].MacAddress,
// 					NICType:           opt.ipamAddResult.interfaceInfo[i].NICType,
// 					SkipDefaultRoutes: opt.ipamAddResult.interfaceInfo[i].SkipDefaultRoutes,
// 				})
// 		case cns.BackendNIC:
// 			// todo
// 		case cns.InfraNIC:
// 			epIndex = i
// 			continue
// 		default:
// 			// Error catch for unsupported NICType?
// 		}
// 	}

// 	// Create the endpoint.
// 	logger.Info("Creating endpoint", zap.String("endpointInfo", epInfo.PrettyString()))
// 	sendEvent(plugin, fmt.Sprintf("[cni-net] Creating endpoint %s.", epInfo.PrettyString()))
// 	err = plugin.nm.CreateEndpoint(cnsclient, opt.nwInfo.Id, epInfos, epIndex)
// 	if err != nil {
// 		err = plugin.Errorf("Failed to create endpoint: %v", err)
// 	}

// 	return epInfo, err
// }

// Get handles CNI Get commands.
func (plugin *NetPlugin) Get(args *cniSkel.CmdArgs) error {
	var (
		result    cniTypesCurr.Result
		err       error
		nwCfg     *cni.NetworkConfig
		epInfo    *network.EndpointInfo
		iface     *cniTypesCurr.Interface
		networkID string
	)

	logger.Info("Processing GET command",
		zap.String("container", args.ContainerID),
		zap.String("netns", args.Netns),
		zap.String("ifname", args.IfName),
		zap.String("args", args.Args),
		zap.String("path", args.Path))

	defer func() {
		// Add Interfaces to result.
		iface = &cniTypesCurr.Interface{
			Name: args.IfName,
		}
		result.Interfaces = append(result.Interfaces, iface)

		// Convert result to the requested CNI version.
		res, vererr := result.GetAsVersion(nwCfg.CNIVersion)
		if vererr != nil {
			logger.Error("GetAsVersion failed", zap.Error(vererr))
			plugin.Error(vererr)
		}

		if err == nil && res != nil {
			// Output the result to stdout.
			res.Print()
		}

		logger.Info("GET command completed", zap.Any("result", result),
			zap.Error(err))
	}()

	// Parse network configuration from stdin.
	if nwCfg, err = cni.ParseNetworkConfig(args.StdinData); err != nil {
		err = plugin.Errorf("Failed to parse network configuration: %v.", err)
		return err
	}

	logger.Info("Read network configuration", zap.Any("config", nwCfg))

	iptables.DisableIPTableLock = nwCfg.DisableIPTableLock

	// Initialize values from network config.
	if networkID, err = plugin.getNetworkName(args.Netns, nil, nwCfg); err != nil {
		// TODO: Ideally we should return from here only.
		logger.Error("Failed to extract network name from network config",
			zap.Error(err))
	}

	endpointID := GetEndpointID(args)

	// Query the network.
	if _, err = plugin.nm.GetNetworkInfo(networkID); err != nil {
		logger.Error("Failed to query network", zap.Error(err))
		return err
	}

	// Query the endpoint.
	if epInfo, err = plugin.nm.GetEndpointInfo(networkID, endpointID); err != nil {
		logger.Error("Failed to query endpoint", zap.Error(err))
		return err
	}

	for _, ipAddresses := range epInfo.IPAddresses {
		ipConfig := &cniTypesCurr.IPConfig{
			Interface: &epInfo.IfIndex,
			Address:   ipAddresses,
		}

		if epInfo.Gateways != nil {
			ipConfig.Gateway = epInfo.Gateways[0]
		}

		result.IPs = append(result.IPs, ipConfig)
	}

	for _, route := range epInfo.Routes {
		result.Routes = append(result.Routes, &cniTypes.Route{Dst: route.Dst, GW: route.Gw})
	}

	result.DNS.Nameservers = epInfo.EndpointDNS.Servers
	result.DNS.Domain = epInfo.EndpointDNS.Suffix

	return nil
}

// Delete handles CNI delete commands.
func (plugin *NetPlugin) Delete(args *cniSkel.CmdArgs) error {
	var (
		err          error
		nwCfg        *cni.NetworkConfig
		k8sPodName   string
		k8sNamespace string
		networkID    string
		nwInfo       network.EndpointInfo
		epInfo       *network.EndpointInfo
		cniMetric    telemetry.AIMetric
	)

	startTime := time.Now()

	logger.Info("Processing DEL command",
		zap.String("containerId", args.ContainerID),
		zap.String("netNS", args.Netns),
		zap.String("ifName", args.IfName),
		zap.Any("args", args.Args),
		zap.String("path", args.Path),
		zap.ByteString("stdinData", args.StdinData))
	sendEvent(plugin, fmt.Sprintf("[cni-net] Processing DEL command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v, StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData))

	defer func() {
		logger.Info("DEL command completed",
			zap.String("pod", k8sPodName),
			zap.Error(err))
	}()

	// Parse network configuration from stdin.
	if nwCfg, err = cni.ParseNetworkConfig(args.StdinData); err != nil {
		err = plugin.Errorf("[cni-net] Failed to parse network configuration: %v", err)
		return err
	}

	// Parse Pod arguments.
	if k8sPodName, k8sNamespace, err = plugin.getPodInfo(args.Args); err != nil {
		logger.Error("Failed to get POD info", zap.Error(err))
	}

	plugin.setCNIReportDetails(nwCfg, CNI_DEL, "")
	plugin.report.ContainerName = k8sPodName + ":" + k8sNamespace

	iptables.DisableIPTableLock = nwCfg.DisableIPTableLock

	sendMetricFunc := func() {
		operationTimeMs := time.Since(startTime).Milliseconds()
		cniMetric.Metric = aitelemetry.Metric{
			Name:             telemetry.CNIDelTimeMetricStr,
			Value:            float64(operationTimeMs),
			AppVersion:       plugin.Version,
			CustomDimensions: make(map[string]string),
		}
		SetCustomDimensions(&cniMetric, nwCfg, err)
		telemetry.SendCNIMetric(&cniMetric, plugin.tb)
	}

	platformInit(nwCfg)

	logger.Info("Execution mode", zap.String("mode", nwCfg.ExecutionMode))
	if nwCfg.ExecutionMode == string(util.Baremetal) {

		logger.Info("Baremetal mode. Calling vnet agent for delete container")

		// schedule send metric before attempting delete
		defer sendMetricFunc()
		_, err = plugin.nnsClient.DeleteContainerNetworking(context.Background(), k8sPodName, args.Netns)
		if err != nil {
			return fmt.Errorf("nnsClient.DeleteContainerNetworking failed with err %w", err)
		}
	}

	if plugin.ipamInvoker == nil {
		switch nwCfg.IPAM.Type {
		case network.AzureCNS:
			cnsClient, cnsErr := cnscli.New("", defaultRequestTimeout)
			if cnsErr != nil {
				logger.Error("failed to create cns client", zap.Error(cnsErr))
				return errors.Wrap(cnsErr, "failed to create cns client")
			}
			plugin.ipamInvoker = NewCNSInvoker(k8sPodName, k8sNamespace, cnsClient, util.ExecutionMode(nwCfg.ExecutionMode), util.IpamMode(nwCfg.IPAM.Mode))

		default:
			plugin.ipamInvoker = NewAzureIpamInvoker(plugin, &nwInfo)
		}
	}

	// Loop through all the networks that are created for the given Netns. In case of multi-nic scenario ( currently supported
	// scenario is dual-nic ), single container may have endpoints created in multiple networks. As all the endpoints are
	// deleted, getNetworkName will return error of the type NetworkNotFoundError which will result in nil error as compliance
	// with CNI SPEC as mentioned below.

	numEndpointsToDelete := 1
	// only get number of endpoints if it's multitenancy mode
	if nwCfg.MultiTenancy {
		numEndpointsToDelete = plugin.nm.GetNumEndpointsByContainerID(args.ContainerID)
	}

	logger.Info("Endpoints to be deleted", zap.Int("count", numEndpointsToDelete))
	for i := 0; i < numEndpointsToDelete; i++ {
		// Initialize values from network config.
		networkID, err = plugin.getNetworkName(args.Netns, nil, nwCfg)
		if err != nil {
			// If error is not found error, then we ignore it, to comply with CNI SPEC.
			if network.IsNetworkNotFoundError(err) {
				err = nil
				return err
			}

			logger.Error("Failed to extract network name from network config", zap.Error(err))
			err = plugin.Errorf("Failed to extract network name from network config. error: %v", err)
			return err
		}
		// Query the network.
		if nwInfo, err = plugin.nm.GetNetworkInfo(networkID); err != nil {
			if !nwCfg.MultiTenancy {
				logger.Error("Failed to query network",
					zap.String("network", networkID),
					zap.Error(err))
				// Log the error but return success if the network is not found.
				// if cni hits this, mostly state file would be missing and it can be reboot scenario where
				// container runtime tries to delete and create pods which existed before reboot.
				// this condition will not apply to stateless CNI since the network struct will be crated on each call
				err = nil
				if !plugin.nm.IsStatelessCNIMode() {
					return err
				}
			}
		}

		endpointID := plugin.nm.GetEndpointID(args.ContainerID, args.IfName)
		// Query the endpoint.
		if epInfo, err = plugin.nm.GetEndpointInfo(networkID, endpointID); err != nil {
			logger.Info("GetEndpoint",
				zap.String("endpoint", endpointID),
				zap.Error(err))
			if !nwCfg.MultiTenancy {
				// attempt to release address associated with this Endpoint id
				// This is to ensure clean up is done even in failure cases

				logger.Error("Failed to query endpoint",
					zap.String("endpoint", endpointID),
					zap.Error(err))
				logger.Error("Release ip by ContainerID (endpoint not found)",
					zap.String("containerID", args.ContainerID))
				sendEvent(plugin, fmt.Sprintf("Release ip by ContainerID (endpoint not found):%v", args.ContainerID))
				if err = plugin.ipamInvoker.Delete(nil, nwCfg, args, nwInfo.Options); err != nil {
					return plugin.RetriableError(fmt.Errorf("failed to release address(no endpoint): %w", err))
				}
			}
			// Log the error but return success if the endpoint being deleted is not found.
			err = nil
			return err
		}

		// schedule send metric before attempting delete
		defer sendMetricFunc() //nolint:gocritic
		logger.Info("Deleting endpoint",
			zap.String("endpointID", endpointID))
		sendEvent(plugin, fmt.Sprintf("Deleting endpoint:%v", endpointID))
		// Delete the endpoint.
		if err = plugin.nm.DeleteEndpoint(networkID, endpointID, epInfo); err != nil {
			// return a retriable error so the container runtime will retry this DEL later
			// the implementation of this function returns nil if the endpoint doens't exist, so
			// we don't have to check that here
			return plugin.RetriableError(fmt.Errorf("failed to delete endpoint: %w", err))
		}

		if !nwCfg.MultiTenancy {
			// Call into IPAM plugin to release the endpoint's addresses.
			for i := range epInfo.IPAddresses {
				logger.Info("Release ip", zap.String("ip", epInfo.IPAddresses[i].IP.String()))
				sendEvent(plugin, fmt.Sprintf("Release ip:%s", epInfo.IPAddresses[i].IP.String()))
				err = plugin.ipamInvoker.Delete(&epInfo.IPAddresses[i], nwCfg, args, nwInfo.Options)
				if err != nil {
					return plugin.RetriableError(fmt.Errorf("failed to release address: %w", err))
				}
			}
		} else if epInfo.EnableInfraVnet {
			nwCfg.IPAM.Subnet = nwInfo.Subnets[0].Prefix.String()
			nwCfg.IPAM.Address = epInfo.InfraVnetIP.IP.String()
			err = plugin.ipamInvoker.Delete(nil, nwCfg, args, nwInfo.Options)
			if err != nil {
				return plugin.RetriableError(fmt.Errorf("failed to release address: %w", err))
			}
		}
	}
	sendEvent(plugin, fmt.Sprintf("CNI DEL succeeded : Released ip %+v podname %v namespace %v", nwCfg.IPAM.Address, k8sPodName, k8sNamespace))

	return err
}

// Update handles CNI update commands.
// Update is only supported for multitenancy and to update routes.
func (plugin *NetPlugin) Update(args *cniSkel.CmdArgs) error {
	var (
		result              *cniTypesCurr.Result
		err                 error
		nwCfg               *cni.NetworkConfig
		existingEpInfo      *network.EndpointInfo
		podCfg              *cni.K8SPodEnvArgs
		orchestratorContext []byte
		targetNetworkConfig *cns.GetNetworkContainerResponse
		cniMetric           telemetry.AIMetric
	)

	startTime := time.Now()

	logger.Info("Processing UPDATE command",
		zap.String("netns", args.Netns),
		zap.String("args", args.Args),
		zap.String("path", args.Path))

	// Parse network configuration from stdin.
	if nwCfg, err = cni.ParseNetworkConfig(args.StdinData); err != nil {
		err = plugin.Errorf("Failed to parse network configuration: %v.", err)
		return err
	}

	logger.Info("Read network configuration", zap.Any("config", nwCfg))

	iptables.DisableIPTableLock = nwCfg.DisableIPTableLock
	plugin.setCNIReportDetails(nwCfg, CNI_UPDATE, "")

	defer func() {
		operationTimeMs := time.Since(startTime).Milliseconds()
		cniMetric.Metric = aitelemetry.Metric{
			Name:             telemetry.CNIUpdateTimeMetricStr,
			Value:            float64(operationTimeMs),
			AppVersion:       plugin.Version,
			CustomDimensions: make(map[string]string),
		}
		SetCustomDimensions(&cniMetric, nwCfg, err)
		telemetry.SendCNIMetric(&cniMetric, plugin.tb)

		if result == nil {
			result = &cniTypesCurr.Result{}
		}

		// Convert result to the requested CNI version.
		res, vererr := result.GetAsVersion(nwCfg.CNIVersion)
		if vererr != nil {
			logger.Error("GetAsVersion failed", zap.Error(vererr))
			plugin.Error(vererr)
		}

		if err == nil && res != nil {
			// Output the result to stdout.
			res.Print()
		}

		logger.Info("UPDATE command completed",
			zap.Any("result", result),
			zap.Error(err))
	}()

	// Parse Pod arguments.
	if podCfg, err = cni.ParseCniArgs(args.Args); err != nil {
		logger.Error("Error while parsing CNI Args during UPDATE",
			zap.Error(err))
		return err
	}

	k8sNamespace := string(podCfg.K8S_POD_NAMESPACE)
	if len(k8sNamespace) == 0 {
		errMsg := "Required parameter Pod Namespace not specified in CNI Args during UPDATE"
		logger.Error(errMsg)
		return plugin.Errorf(errMsg)
	}

	k8sPodName := string(podCfg.K8S_POD_NAME)
	if len(k8sPodName) == 0 {
		errMsg := "Required parameter Pod Name not specified in CNI Args during UPDATE"
		logger.Error(errMsg)
		return plugin.Errorf(errMsg)
	}

	// Initialize values from network config.
	networkID := nwCfg.Name

	// Query the network.
	if _, err = plugin.nm.GetNetworkInfo(networkID); err != nil {
		errMsg := fmt.Sprintf("Failed to query network during CNI UPDATE: %v", err)
		logger.Error(errMsg)
		return plugin.Errorf(errMsg)
	}

	// Query the existing endpoint since this is an update.
	// Right now, we do not support updating pods that have multiple endpoints.
	existingEpInfo, err = plugin.nm.GetEndpointInfoBasedOnPODDetails(networkID, k8sPodName, k8sNamespace, nwCfg.EnableExactMatchForPodName)
	if err != nil {
		plugin.Errorf("Failed to retrieve target endpoint for CNI UPDATE [name=%v, namespace=%v]: %v", k8sPodName, k8sNamespace, err)
		return err
	}

	logger.Info("Retrieved existing endpoint from state that may get update",
		zap.Any("info", existingEpInfo))

	// now query CNS to get the target routes that should be there in the networknamespace (as a result of update)
	logger.Info("Going to collect target routes from CNS",
		zap.String("pod", k8sPodName),
		zap.String("namespace", k8sNamespace))

	// create struct with info for target POD
	podInfo := cns.KubernetesPodInfo{
		PodName:      k8sPodName,
		PodNamespace: k8sNamespace,
	}
	if orchestratorContext, err = json.Marshal(podInfo); err != nil {
		logger.Error("Marshalling KubernetesPodInfo failed",
			zap.Error(err))
		return plugin.Errorf(err.Error())
	}

	cnsclient, err := cnscli.New(nwCfg.CNSUrl, defaultRequestTimeout)
	if err != nil {
		logger.Error("failed to initialized cns client",
			zap.String("url", nwCfg.CNSUrl),
			zap.String("error", err.Error()))
		return plugin.Errorf(err.Error())
	}

	if targetNetworkConfig, err = cnsclient.GetNetworkContainer(context.TODO(), orchestratorContext); err != nil {
		logger.Info("GetNetworkContainer failed",
			zap.Error(err))
		return plugin.Errorf(err.Error())
	}

	logger.Info("Network config received from cns",
		zap.String("pod", k8sPodName),
		zap.String("namespace", k8sNamespace),
		zap.Any("config", targetNetworkConfig))
	targetEpInfo := &network.EndpointInfo{}

	// get the target routes that should replace existingEpInfo.Routes inside the network namespace
	logger.Info("Going to collect target routes for from targetNetworkConfig",
		zap.String("pod", k8sPodName),
		zap.String("namespace", k8sNamespace))
	if targetNetworkConfig.Routes != nil && len(targetNetworkConfig.Routes) > 0 {
		for _, route := range targetNetworkConfig.Routes {
			logger.Info("Adding route from routes to targetEpInfo", zap.Any("route", route))
			_, dstIPNet, _ := net.ParseCIDR(route.IPAddress)
			gwIP := net.ParseIP(route.GatewayIPAddress)
			targetEpInfo.Routes = append(targetEpInfo.Routes, network.RouteInfo{Dst: *dstIPNet, Gw: gwIP, DevName: existingEpInfo.IfName})
			logger.Info("Successfully added route from routes to targetEpInfo", zap.Any("route", route))
		}
	}

	logger.Info("Going to collect target routes based on Cnetaddressspace from targetNetworkConfig",
		zap.String("pod", k8sPodName),
		zap.String("namespace", k8sNamespace))

	ipconfig := targetNetworkConfig.IPConfiguration
	for _, ipRouteSubnet := range targetNetworkConfig.CnetAddressSpace {
		logger.Info("Adding route from cnetAddressspace to targetEpInfo", zap.Any("subnet", ipRouteSubnet))
		dstIPNet := net.IPNet{IP: net.ParseIP(ipRouteSubnet.IPAddress), Mask: net.CIDRMask(int(ipRouteSubnet.PrefixLength), 32)}
		gwIP := net.ParseIP(ipconfig.GatewayIPAddress)
		route := network.RouteInfo{Dst: dstIPNet, Gw: gwIP, DevName: existingEpInfo.IfName}
		targetEpInfo.Routes = append(targetEpInfo.Routes, route)
		logger.Info("Successfully added route from cnetAddressspace to targetEpInfo", zap.Any("subnet", ipRouteSubnet))
	}

	logger.Info("Finished collecting new routes in targetEpInfo", zap.Any("route", targetEpInfo.Routes))
	logger.Info("Now saving existing infravnetaddress space if needed.")
	for _, ns := range nwCfg.PodNamespaceForDualNetwork {
		if k8sNamespace == ns {
			targetEpInfo.EnableInfraVnet = true
			targetEpInfo.InfraVnetAddressSpace = nwCfg.InfraVnetAddressSpace
			logger.Info("Saving infravnet address space",
				zap.String("space", targetEpInfo.InfraVnetAddressSpace),
				zap.String("namespace", existingEpInfo.PODNameSpace),
				zap.String("pod", existingEpInfo.PODName))
			break
		}
	}

	// Update the endpoint.
	logger.Info("Now updating existing endpoint with targetNetworkConfig",
		zap.String("endpoint", existingEpInfo.Id),
		zap.Any("config", targetNetworkConfig))
	if err = plugin.nm.UpdateEndpoint(networkID, existingEpInfo, targetEpInfo); err != nil {
		err = plugin.Errorf("Failed to update endpoint: %v", err)
		return err
	}

	msg := fmt.Sprintf("CNI UPDATE succeeded : Updated %+v podname %v namespace %v", targetNetworkConfig, k8sPodName, k8sNamespace)
	plugin.setCNIReportDetails(nwCfg, CNI_UPDATE, msg)

	return nil
}

func convertNnsToIPConfigs(
	netRes *nnscontracts.ConfigureContainerNetworkingResponse,
	ifName string,
	podName string,
	operationName string,
) []*network.IPConfig {
	// This function does not add interfaces to CNI result. Reason being CRI (containerD in baremetal case)
	// only looks for default interface named "eth0" and this default interface is added in the defer
	// method of ADD method
	var ipConfigs []*network.IPConfig

	if netRes.Interfaces != nil {
		for _, ni := range netRes.Interfaces {
			for _, ip := range ni.Ipaddresses {
				ipAddr := net.ParseIP(ip.Ip)

				prefixLength, err := strconv.Atoi(ip.PrefixLength)
				if err != nil {
					logger.Error("Error parsing prefix length while converting to cni result",
						zap.String("prefixLength", ip.PrefixLength),
						zap.String("operation", operationName),
						zap.String("pod", podName),
						zap.Error(err))
					continue
				}

				address := net.IPNet{
					IP:   ipAddr,
					Mask: net.CIDRMask(prefixLength, ipv6FullMask),
				}

				if ipAddr.To4() != nil {
					address.Mask = net.CIDRMask(prefixLength, ipv4FullMask)
				}

				gateway := net.ParseIP(ip.DefaultGateway)

				ipConfigs = append(ipConfigs, &network.IPConfig{
					Address: address,
					Gateway: gateway,
				})
			}
		}
	}

	return ipConfigs
}

func convertInterfaceInfoToCniResult(info network.InterfaceInfo, ifName string) *cniTypesCurr.Result {
	result := &cniTypesCurr.Result{
		Interfaces: []*cniTypesCurr.Interface{
			{
				Name: ifName,
			},
		},
		DNS: cniTypes.DNS{
			Domain:      info.DNS.Suffix,
			Nameservers: info.DNS.Servers,
		},
	}

	if len(info.IPConfigs) > 0 {
		for _, ipconfig := range info.IPConfigs {
			result.IPs = append(result.IPs, &cniTypesCurr.IPConfig{Address: ipconfig.Address, Gateway: ipconfig.Gateway})
		}

		for i := range info.Routes {
			result.Routes = append(result.Routes, &cniTypes.Route{Dst: info.Routes[i].Dst, GW: info.Routes[i].Gw})
		}
	}

	return result
}

func convertCniResultToInterfaceInfo(result *cniTypesCurr.Result) network.InterfaceInfo {
	interfaceInfo := network.InterfaceInfo{}

	if result != nil {
		for _, ipconfig := range result.IPs {
			interfaceInfo.IPConfigs = append(interfaceInfo.IPConfigs, &network.IPConfig{Address: ipconfig.Address, Gateway: ipconfig.Gateway})
		}

		for _, route := range result.Routes {
			interfaceInfo.Routes = append(interfaceInfo.Routes, network.RouteInfo{Dst: route.Dst, Gw: route.GW})
		}

		interfaceInfo.DNS = network.DNSInfo{
			Suffix:  result.DNS.Domain,
			Servers: result.DNS.Nameservers,
		}
	}

	return interfaceInfo
}

func findDefaultInterface(ipamAddResult IPAMAddResult) (int, error) {
	for i := 0; i < len(ipamAddResult.interfaceInfo); i++ {
		if ipamAddResult.interfaceInfo[i].NICType == cns.InfraNIC {
			return i, nil
		}
	}

	return -1, errors.New("no NIC was of type InfraNIC")
}
