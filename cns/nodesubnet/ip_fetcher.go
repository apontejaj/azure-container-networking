package nodesubnet

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/Azure/azure-container-networking/nmagent"
	"github.com/pkg/errors"
)

var ErrorRefreshSkipped = errors.New("Refresh skipped due to throttling")

// This interface is implemented by the NMAgent Client, and also a mock client for testing
type InterfaceRetriever interface {
	GetInterfaceIPInfo(ctx context.Context) (nmagent.Interfaces, error)
}

type IPFetcher struct {
	// Node subnet state
	secondaryIPQueryInterval   time.Duration // Minimum time between secondary IP fetches
	secondaryIPLastRefreshTime time.Time     // Time of last secondary IP fetch

	ipFectcherClient InterfaceRetriever
}

func NewIPFetcher(nmaClient InterfaceRetriever, queryInterval time.Duration) *IPFetcher {
	return &IPFetcher{
		ipFectcherClient:         nmaClient,
		secondaryIPQueryInterval: queryInterval,
	}
}

// If secondaryIPQueryInterval has elapsed since the last fetch, fetch secondary IPs
func (c *IPFetcher) RefreshSecondaryIPsIfNeeded(ctx context.Context) (ips []net.IP, err error) {
	if time.Since(c.secondaryIPLastRefreshTime) < c.secondaryIPQueryInterval {
		return nil, ErrorRefreshSkipped
	}

	c.secondaryIPLastRefreshTime = time.Now()
	response, err := c.ipFectcherClient.GetInterfaceIPInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Getting Interface IPs")
	}

	res := flattenIPListFromResponse(&response)
	return res, nil
}

// Get the list of secondary IPs from fetched Interfaces
func flattenIPListFromResponse(resp *nmagent.Interfaces) (res []net.IP) {
	// For each interface...
	for _, intf := range resp.Entries {
		if !intf.IsPrimary {
			continue
		}

		// For each subnet on the interface...
		for _, s := range intf.InterfaceSubnets {
			addressCount := 0
			// For each address in the subnet...
			for _, a := range s.IPAddress {
				// Primary addresses are reserved for the host.
				if a.IsPrimary {
					continue
				}

				res = append(res, net.IP(a.Address))
				addressCount++
			}
			log.Printf("Got %d addresses from subnet %s", addressCount, s.Prefix)
		}
	}

	return res
}
