//go:build linux
// +build linux

package network

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"testing"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/network"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
	cniSkel "github.com/containernetworking/cni/pkg/skel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetNetworkOptions(t *testing.T) {
	tests := []struct {
		name             string
		cnsNwConfig      cns.GetNetworkContainerResponse
		nwInfo           network.EndpointInfo
		expectedVlanID   string
		expectedSnatBrIP string
	}{
		{
			name: "set network options multitenancy",
			cnsNwConfig: cns.GetNetworkContainerResponse{
				MultiTenancyInfo: cns.MultiTenancyInfo{
					ID: 1,
				},
				LocalIPConfiguration: cns.IPConfiguration{
					IPSubnet: cns.IPSubnet{
						IPAddress:    "169.254.0.4",
						PrefixLength: 17,
					},
					GatewayIPAddress: "169.254.0.1",
				},
			},
			nwInfo: network.EndpointInfo{
				Options: make(map[string]interface{}),
			},
			expectedVlanID:   "1",
			expectedSnatBrIP: "169.254.0.1/17",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setNetworkOptions(&tt.cnsNwConfig, &tt.nwInfo)
			require.Condition(t, assert.Comparison(func() bool {
				optMap := tt.nwInfo.Options[dockerNetworkOption]
				vlanID, ok := optMap.(map[string]interface{})[network.VlanIDKey]
				if !ok {
					return false
				}
				snatBridgeIP, ok := optMap.(map[string]interface{})[network.SnatBridgeIPKey]
				return ok && vlanID == tt.expectedVlanID && snatBridgeIP == tt.expectedSnatBrIP
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
			name: "set endpoint options multitenancy",
			cnsNwConfig: cns.GetNetworkContainerResponse{
				MultiTenancyInfo: cns.MultiTenancyInfo{
					ID: 1,
				},
				LocalIPConfiguration: cns.IPConfiguration{
					IPSubnet: cns.IPSubnet{
						IPAddress:    "169.254.0.4",
						PrefixLength: 17,
					},
					GatewayIPAddress: "169.254.0.1",
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
				vlanID := tt.epInfo.Data[network.VlanIDKey]
				localIP := tt.epInfo.Data[network.LocalIPKey]
				snatBrIP := tt.epInfo.Data[network.SnatBridgeIPKey]

				return tt.epInfo.AllowInboundFromHostToNC == true &&
					tt.epInfo.AllowInboundFromNCToHost == false &&
					tt.epInfo.NetworkContainerID == "abcd" &&
					vlanID == 1 &&
					localIP == "169.254.0.4/17" &&
					snatBrIP == "169.254.0.1/17"
			}))
		})
	}
}

func TestAddDefaultRoute(t *testing.T) {
	tests := []struct {
		name   string
		gwIP   string
		epInfo network.EndpointInfo
		result network.InterfaceInfo
	}{
		{
			name: "add default route multitenancy",
			gwIP: "192.168.0.1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			addDefaultRoute(tt.gwIP, &tt.epInfo, &tt.result)
			require.Condition(t, assert.Comparison(func() bool {
				return len(tt.epInfo.Routes) == 1 &&
					len(tt.result.Routes) == 1 &&
					tt.epInfo.Routes[0].DevName == snatInterface &&
					tt.epInfo.Routes[0].Gw.String() == "192.168.0.1"
			}))
		})
	}
}

func TestAddSnatForDns(t *testing.T) {
	tests := []struct {
		name   string
		gwIP   string
		epInfo network.EndpointInfo
		result network.InterfaceInfo
	}{
		{
			name: "add snat for dns",
			gwIP: "192.168.0.1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			addSnatForDNS(tt.gwIP, &tt.epInfo, &tt.result)
			require.Condition(t, assert.Comparison(func() bool {
				return len(tt.epInfo.Routes) == 1 &&
					len(tt.result.Routes) == 1 &&
					tt.epInfo.Routes[0].DevName == snatInterface &&
					tt.epInfo.Routes[0].Gw.String() == "192.168.0.1" &&
					tt.epInfo.Routes[0].Dst.String() == "168.63.129.16/32"
			}))
		})
	}
}

// Happy path scenario for add and delete
func TestPluginLinuxAdd(t *testing.T) {
	resources := GetTestResources()
	localNwCfg := cni.NetworkConfig{
		CNIVersion:                 "0.3.0",
		Name:                       "mulnet",
		MultiTenancy:               true,
		EnableExactMatchForPodName: true,
		Master:                     "eth0",
	}
	type endpointEntry struct {
		epInfo    *network.EndpointInfo
		epIDRegex string
	}

	tests := []struct {
		name   string
		plugin *NetPlugin
		args   *cniSkel.CmdArgs
		want   []endpointEntry
		match  func(*network.EndpointInfo, *network.EndpointInfo) bool
	}{
		{
			// in swiftv1 linux multitenancy, we only get 1 response from cns at a time
			name: "Add Happy Path Swiftv1 Multitenancy",
			plugin: &NetPlugin{
				Plugin:             resources.Plugin,
				nm:                 network.NewMockNetworkmanager(network.NewMockEndpointClient(nil)),
				tb:                 &telemetry.TelemetryBuffer{},
				report:             &telemetry.CNIReport{},
				multitenancyClient: NewMockMultitenancy(false, []*cns.GetNetworkContainerResponse{GetTestCNSResponse3()}),
			},
			args: &cniSkel.CmdArgs{
				StdinData:   localNwCfg.Serialize(),
				ContainerID: "test-container",
				Netns:       "bc526fae-4ba0-4e80-bc90-ad721e5850bf",
				Args:        fmt.Sprintf("K8S_POD_NAME=%v;K8S_POD_NAMESPACE=%v", "test-pod", "test-pod-ns"),
				IfName:      eth0IfName,
			},
			match: func(ei1, ei2 *network.EndpointInfo) bool {
				return ei1.NetworkContainerID == ei2.NetworkContainerID
			},
			want: []endpointEntry{
				// should match with GetTestCNSResponse3
				{
					epInfo: &network.EndpointInfo{
						ContainerID: "test-container",
						Data: map[string]interface{}{
							"VlanID":       1, // Vlan ID used here
							"localIP":      "168.254.0.4/17",
							"snatBridgeIP": "168.254.0.1/17",
							"vethname":     "mulnettest-containereth0",
						},
						Routes: []network.RouteInfo{
							{
								Dst: *parseCIDR("192.168.0.4/24"),
								Gw:  net.ParseIP("192.168.0.1"),
								// interface to use is NOT propagated to ep info
							},
						},
						AllowInboundFromHostToNC: true,
						EnableSnatOnHost:         true,
						EnableMultiTenancy:       true,
						EnableSnatForDns:         true,
						PODName:                  "test-pod",
						PODNameSpace:             "test-pod-ns",
						NICType:                  cns.InfraNIC,
						MasterIfName:             eth0IfName,
						NetworkContainerID:       "Swift_74b34111-6e92-49ee-a82a-8881c850ce0e",
						NetworkID:                "mulnet",
						NetNsPath:                "bc526fae-4ba0-4e80-bc90-ad721e5850bf",
						NetNs:                    "bc526fae-4ba0-4e80-bc90-ad721e5850bf",
						HostSubnetPrefix:         "20.240.0.0/24",
						Options: map[string]interface{}{
							dockerNetworkOption: map[string]interface{}{
								"VlanID":       "1", // doesn't seem to be used in linux
								"snatBridgeIP": "168.254.0.1/17",
							},
						},
						// matches with cns ip configuration
						IPAddresses: []net.IPNet{
							{
								IP:   net.ParseIP("20.0.0.10"),
								Mask: getIPNetWithString("20.0.0.10/24").Mask,
							},
						},
						NATInfo: nil,
						// ip config pod ip + mask(s) from cns > interface info > subnet info
						Subnets: []network.SubnetInfo{
							{
								Family: platform.AfINET,
								// matches cns ip configuration (20.0.0.1/24 == 20.0.0.0/24)
								Prefix: *getIPNetWithString("20.0.0.0/24"),
								// matches cns ip configuration gateway ip address
								Gateway: net.ParseIP("20.0.0.1"),
							},
						},
					},
					epIDRegex: `test-con-eth0`,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Add(tt.args)
			require.NoError(t, err)
			allEndpoints, _ := tt.plugin.nm.GetAllEndpoints("")
			require.Len(t, allEndpoints, len(tt.want))
			for _, wantedEndpointEntry := range tt.want {
				epId := "none"
				for _, endpointInfo := range allEndpoints {
					log.Printf("%v", endpointInfo.NetworkID)
					if tt.match(wantedEndpointEntry.epInfo, endpointInfo) {
						// save the endpoint id before removing it
						epId = endpointInfo.EndpointID
						require.Regexp(t, regexp.MustCompile(wantedEndpointEntry.epIDRegex), epId)

						// omit endpoint id and ifname fields as they are nondeterministic
						endpointInfo.EndpointID = ""
						endpointInfo.IfName = ""

						require.Equal(t, wantedEndpointEntry.epInfo, endpointInfo)
						break
					}
				}
				if epId == "none" {
					t.Fail()
				}
				tt.plugin.nm.DeleteEndpoint("", epId, nil)
			}

		})
	}
}
