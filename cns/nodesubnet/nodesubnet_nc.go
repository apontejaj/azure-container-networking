package nodesubnet

import (
	"net/netip"
	"strconv"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
)

const (
	// ID for fake NC that we create to store NodeSubnet IPS
	NodeSubnetNCID        = "NodeSubnetNC_IPv4"
	NodeSubnetNCVersion   = 0
	NodeSubnetHostVersion = "0"
	NodeSubnetNCStatus    = v1alpha.NCUpdateSuccess
)

// CreateNodeSubnetNCRequest generates a CreateNetworkContainerRequest that simply stores the static secondary IPs.
func CreateNodeSubnetNCRequest(primaryIP netip.Addr, secondaryIPs []netip.Addr) (*cns.CreateNetworkContainerRequest, error) {
	secondaryIPConfigs := map[string]cns.SecondaryIPConfig{}

	for _, secondaryIP := range secondaryIPs {
		// iterate through all secondary IP addresses add them to the request as secondary IPConfigs.
		secondaryIPConfigs[secondaryIP.String()] = cns.SecondaryIPConfig{
			IPAddress: secondaryIP.String(),
			NCVersion: NodeSubnetNCVersion,
		}
	}

	return &cns.CreateNetworkContainerRequest{
		HostPrimaryIP:        primaryIP.String(),
		SecondaryIPConfigs:   secondaryIPConfigs,
		NetworkContainerid:   NodeSubnetNCID,
		NetworkContainerType: cns.NodeSubnet,
		Version:              strconv.FormatInt(NodeSubnetNCVersion, 10), //nolint:gomnd // it's decimal
		IPConfiguration:      cns.IPConfiguration{},
		NCStatus:             NodeSubnetNCStatus,
	}, nil
}
