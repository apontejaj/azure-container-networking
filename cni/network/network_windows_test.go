//go:build windows
// +build windows

package network

import (
	"fmt"
	"net"
	"testing"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/network"
	"github.com/Azure/azure-container-networking/network/hnswrapper"
	"github.com/Azure/azure-container-networking/network/policy"
	"github.com/Azure/azure-container-networking/telemetry"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	network.Hnsv2 = hnswrapper.NewHnsv2wrapperFake()
	network.Hnsv1 = hnswrapper.NewHnsv1wrapperFake()
}

// Test windows network policies is set
func TestAddWithRunTimeNetPolicies(t *testing.T) {
	_, ipnetv4, _ := net.ParseCIDR("10.240.0.0/12")
	_, ipnetv6, _ := net.ParseCIDR("fc00::/64")

	tests := []struct {
		name       string
		nwInfo     network.EndpointInfo
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "add ipv6 endpoint policy",
			nwInfo: network.EndpointInfo{
				Subnets: []network.SubnetInfo{
					{
						Gateway: net.ParseIP("10.240.0.1"),
						Prefix:  *ipnetv4,
					},
					{
						Gateway: net.ParseIP("fc00::1"),
						Prefix:  *ipnetv6,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p, err := getIPV6EndpointPolicy(tt.nwInfo.Subnets)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Condition(t, assert.Comparison(func() bool { return p.Type == policy.EndpointPolicy }))
			}
		})
	}
}

func TestPluginSecondAddSamePodWindows(t *testing.T) {
	plugin, _ := cni.NewPlugin("name", "0.3.0")

	tests := []struct {
		name       string
		methods    []string
		cniArgs    skel.CmdArgs
		plugin     *NetPlugin
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "CNI consecutive add already hot attached",
			methods: []string{"ADD", "ADD"},
			cniArgs: skel.CmdArgs{
				ContainerID: "test1-container",
				Netns:       "test1-container",
				StdinData:   nwCfg.Serialize(),
				Args:        fmt.Sprintf("K8S_POD_NAME=%v;K8S_POD_NAMESPACE=%v", "container1", "container1-ns"),
				IfName:      eth0IfName,
			},
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			wantErr: false,
		},
		{
			name:    "CNI consecutive add not hot attached",
			methods: []string{"ADD", "ADD"},
			cniArgs: skel.CmdArgs{
				ContainerID: "test1-container",
				Netns:       "test1-container",
				StdinData:   nwCfg.Serialize(),
				Args:        fmt.Sprintf("K8S_POD_NAME=%v;K8S_POD_NAMESPACE=%v", "container1", "container1-ns"),
				IfName:      eth0IfName,
			},
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var err error
			for _, method := range tt.methods {
				if method == "ADD" {
					err = tt.plugin.Add(&tt.cniArgs)
				}
			}

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				endpoints, _ := tt.plugin.nm.GetAllEndpoints(nwCfg.Name)
				require.Condition(t, assert.Comparison(func() bool { return len(endpoints) == 1 }), "Expected 2 but got %v", len(endpoints))
			}
		})
	}
}

func TestSetNetworkOptions(t *testing.T) {
	tests := []struct {
		name           string
		cnsNwConfig    cns.GetNetworkContainerResponse
		nwInfo         network.EndpointInfo
		expectedVlanID string
	}{
		{
			name: "set network options vlanid test",
			cnsNwConfig: cns.GetNetworkContainerResponse{
				MultiTenancyInfo: cns.MultiTenancyInfo{
					ID: 1,
				},
			},
			nwInfo: network.EndpointInfo{
				Options: make(map[string]interface{}),
			},
			expectedVlanID: "1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setNetworkOptions(&tt.cnsNwConfig, &tt.nwInfo)
			require.Condition(t, assert.Comparison(func() bool {
				vlanMap := tt.nwInfo.Options[dockerNetworkOption]
				value, ok := vlanMap.(map[string]interface{})[network.VlanIDKey]
				return ok && value == tt.expectedVlanID
			}))
		})
	}
}

func TestSetEndpointOptions(t *testing.T) {
	tests := []struct {
		name        string
		cnsNwConfig cns.GetNetworkContainerResponse
		epInfo      network.EndpointInfo
		vethName    string
	}{
		{
			name: "set network options vlanid test",
			cnsNwConfig: cns.GetNetworkContainerResponse{
				MultiTenancyInfo: cns.MultiTenancyInfo{
					ID: 1,
				},
				CnetAddressSpace: []cns.IPSubnet{
					{
						IPAddress:    "192.168.0.4",
						PrefixLength: 24,
					},
				},
				AllowHostToNCCommunication: true,
				AllowNCToHostCommunication: false,
				NetworkContainerID:         "abcd",
			},
			epInfo: network.EndpointInfo{
				Data: make(map[string]interface{}),
			},
			vethName: "azv1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setEndpointOptions(&tt.cnsNwConfig, &tt.epInfo, tt.vethName)
			require.Condition(t, assert.Comparison(func() bool {
				return tt.epInfo.AllowInboundFromHostToNC == true &&
					tt.epInfo.AllowInboundFromNCToHost == false &&
					tt.epInfo.NetworkContainerID == "abcd"
			}))
		})
	}
}

func TestSetPoliciesFromNwCfg(t *testing.T) {
	tests := []struct {
		name          string
		nwCfg         cni.NetworkConfig
		isIPv6Enabled bool
	}{
		{
			name: "Runtime network polices",
			nwCfg: cni.NetworkConfig{
				RuntimeConfig: cni.RuntimeConfig{
					PortMappings: []cni.PortMapping{
						{
							Protocol:      "tcp",
							HostIp:        "192.168.0.4",
							HostPort:      8000,
							ContainerPort: 80,
						},
					},
				},
			},
			isIPv6Enabled: false,
		},
		{
			name: "Runtime hostPort mapping polices",
			nwCfg: cni.NetworkConfig{
				RuntimeConfig: cni.RuntimeConfig{
					PortMappings: []cni.PortMapping{
						{
							Protocol:      "tcp",
							HostPort:      44000,
							ContainerPort: 80,
						},
					},
				},
			},
			isIPv6Enabled: false,
		},
		{
			name: "Runtime hostPort mapping polices with ipv6 hostIP",
			nwCfg: cni.NetworkConfig{
				RuntimeConfig: cni.RuntimeConfig{
					PortMappings: []cni.PortMapping{
						{
							Protocol:      "tcp",
							HostPort:      44000,
							ContainerPort: 80,
							HostIp:        "2001:2002:2003::1",
						},
					},
				},
			},
			isIPv6Enabled: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			policies, err := getPoliciesFromRuntimeCfg(&tt.nwCfg, tt.isIPv6Enabled)
			require.NoError(t, err)
			require.Condition(t, assert.Comparison(func() bool {
				return len(policies) > 0 && policies[0].Type == policy.EndpointPolicy
			}))
		})
	}
}

func TestDSRPolciy(t *testing.T) {
	tests := []struct {
		name      string
		args      PolicyArgs
		wantCount int
	}{
		{
			name: "test enable dsr policy",
			args: PolicyArgs{
				nwCfg: &cni.NetworkConfig{
					WindowsSettings: cni.WindowsSettings{
						EnableLoopbackDSR: true,
					},
				},
				subnetInfos: []network.SubnetInfo{},
				ipconfigs: []*network.IPConfig{
					{
						Address: func() net.IPNet {
							_, ipnet, _ := net.ParseCIDR("10.0.0.5/24")
							return *ipnet
						}(),
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "test disable dsr policy",
			args: PolicyArgs{
				nwCfg:       &cni.NetworkConfig{},
				subnetInfos: []network.SubnetInfo{},
				ipconfigs: []*network.IPConfig{
					{
						Address: func() net.IPNet {
							_, ipnet, _ := net.ParseCIDR("10.0.0.5/24")
							return *ipnet
						}(),
					},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			policies, err := getEndpointPolicies(tt.args)
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, len(policies))
		})
	}
}

func TestGetNetworkNameFromCNS(t *testing.T) {
	plugin, _ := cni.NewPlugin("name", "0.3.0")
	tests := []struct {
		name          string
		plugin        *NetPlugin
		netNs         string
		nwCfg         *cni.NetworkConfig
		interfaceInfo *network.InterfaceInfo
		want          string
		wantErr       bool
	}{
		{
			name: "Get Network Name from CNS with correct CIDR",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "net",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				Name:         "azure",
				MultiTenancy: true,
			},
			interfaceInfo: &network.InterfaceInfo{
				IPConfigs: []*network.IPConfig{
					{
						Address: net.IPNet{
							IP:   net.ParseIP("10.240.0.5"),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
				NCResponse: &cns.GetNetworkContainerResponse{
					MultiTenancyInfo: cns.MultiTenancyInfo{
						ID: 1,
					},
				},
			},
			want:    "azure-vlan1-10-240-0-0_24",
			wantErr: false,
		},
		{
			name: "Get Network Name from CNS with malformed CIDR #1",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "net",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				Name:         "azure",
				MultiTenancy: true,
			},
			interfaceInfo: &network.InterfaceInfo{
				IPConfigs: []*network.IPConfig{
					{
						Address: net.IPNet{
							IP:   net.ParseIP(""),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
				NCResponse: &cns.GetNetworkContainerResponse{
					MultiTenancyInfo: cns.MultiTenancyInfo{
						ID: 1,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get Network Name from CNS with malformed CIDR #2",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "net",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				Name:         "azure",
				MultiTenancy: true,
			},
			interfaceInfo: &network.InterfaceInfo{
				IPConfigs: []*network.IPConfig{
					{
						Address: net.IPNet{
							IP:   net.ParseIP("10.0.00.6"),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
				NCResponse: &cns.GetNetworkContainerResponse{
					MultiTenancyInfo: cns.MultiTenancyInfo{
						ID: 1,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get Network Name from CNS without NetNS",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				Name:         "azure",
				MultiTenancy: true,
			},
			interfaceInfo: &network.InterfaceInfo{
				IPConfigs: []*network.IPConfig{
					{
						Address: net.IPNet{
							IP:   net.ParseIP("10.0.0.6"),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
				NCResponse: &cns.GetNetworkContainerResponse{
					MultiTenancyInfo: cns.MultiTenancyInfo{
						ID: 1,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get Network Name from CNS without multitenancy",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, false, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "azure",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				Name:         "azure",
				MultiTenancy: false,
			},
			interfaceInfo: &network.InterfaceInfo{
				IPConfigs: []*network.IPConfig{
					{
						Address: net.IPNet{
							IP:   net.ParseIP("10.0.0.6"),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
				NCResponse: &cns.GetNetworkContainerResponse{},
			},
			want:    "azure",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			networkName, err := tt.plugin.getNetworkName(tt.netNs, tt.interfaceInfo, tt.nwCfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, networkName)
			}
		})
	}
}

func TestGetNetworkNameSwiftv2FromCNS(t *testing.T) {
	// TODO: Add IB and Accelnet NIC test to this test
	plugin, _ := cni.NewPlugin("name", "0.3.0")

	macAddress := "00:00:5e:00:53:01"
	parsedMacAddress, _ := net.ParseMAC(macAddress)
	swiftv2L1VHSecondaryInterfacesInfo := make(map[string]network.InterfaceInfo)

	swiftv2L1VHInterfaceInfo := network.InterfaceInfo{
		Name:       "swiftv2L1VHinterface",
		MacAddress: parsedMacAddress,
		NICType:    cns.DelegatedVMNIC,
	}
	swiftv2L1VHSecondaryInterfacesInfo[macAddress] = swiftv2L1VHInterfaceInfo

	tests := []struct {
		name          string
		plugin        *NetPlugin
		netNs         string
		nwCfg         *cni.NetworkConfig
		ipamAddResult *IPAMAddResult
		want          net.HardwareAddr
		wantErr       bool
	}{
		{
			name: "Get Network Name from CNS for swiftv2",
			plugin: &NetPlugin{
				Plugin:      plugin,
				nm:          network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				ipamInvoker: NewMockIpamInvoker(false, false, false, true, false),
				report:      &telemetry.CNIReport{},
				tb:          &telemetry.TelemetryBuffer{},
			},
			netNs: "azure",
			nwCfg: &cni.NetworkConfig{
				CNIVersion:   "0.3.0",
				MultiTenancy: false,
			},
			ipamAddResult: &IPAMAddResult{
				interfaceInfo: swiftv2L1VHSecondaryInterfacesInfo,
			},
			want:    parsedMacAddress,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.ipamAddResult.interfaceInfo)
			networkName, err := tt.plugin.getNetworkName(tt.netNs, &swiftv2L1VHInterfaceInfo, tt.nwCfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				expectedMacAddress := "azure-" + tt.want.String()
				require.NoError(t, err)
				require.Equal(t, expectedMacAddress, networkName)
			}
		})
	}
}
