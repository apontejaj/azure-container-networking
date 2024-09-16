package nodesubnet

import (
	"context"
	"log"
	"time"

	"github.com/Azure/azure-container-networking/nmagent"
)

// This interface is implemented by the NMAgent Client, and also a mock client for testing
type ClientInterface interface {
	GetInterfaceIPInfo(ctx context.Context) (nmagent.Interfaces, error)
}

type IPFetcher struct {
	// Node subnet state
	secondaryIPQueryInterval   time.Duration // Minimum time between secondary IP fetches
	secondaryIPLastRefreshTime time.Time     // Time of last secondary IP fetch

	nmaClient ClientInterface
}

func NewIPFetcher(nmaClient ClientInterface, queryInterval time.Duration) *IPFetcher {
	return &IPFetcher{
		nmaClient:                nmaClient,
		secondaryIPQueryInterval: queryInterval,
	}
}

// Exposed for testing
func (c *IPFetcher) SetSecondaryIPQueryInterval(interval time.Duration) {
	c.secondaryIPQueryInterval = interval
}

// If secondaryIPQueryInterval has elapsed since the last fetch, fetch secondary IPs
func (c *IPFetcher) RefreshSecondaryIPsIfNeeded(ctx context.Context) (refreshNeeded bool, ips []string, err error) {
	if time.Since(c.secondaryIPLastRefreshTime) < c.secondaryIPQueryInterval {
		return false, nil, nil
	}

	c.secondaryIPLastRefreshTime = time.Now()
	response, err := c.nmaClient.GetInterfaceIPInfo(ctx)
	if err != nil {
		return true, nil, err
	}

	res, err := flattenIPListFromResponse(&response)
	return true, res, err
}

// Get the list of secondary IPs from fetched Interfaces
func flattenIPListFromResponse(resp *nmagent.Interfaces) (res []string, err error) {
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

				res = append(res, a.Address)
				addressCount++
			}
			log.Printf("Got %d addresses from subnet %s", addressCount, s.Prefix)
		}
	}

	return res, nil
}
