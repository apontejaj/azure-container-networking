package nodesubnet

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/Azure/azure-container-networking/nmagent"
	"github.com/pkg/errors"
)

const (
	// Minimum time between secondary IP fetches
	MinRefreshInterval = 4 * time.Second
	// Maximum time between secondary IP fetches
	MaxRefreshInterval = 1024 * time.Second
)

var ErrorRefreshSkipped = errors.New("refresh skipped due to throttling")

// This interface is implemented by the NMAgent Client, and also a mock client for testing
type InterfaceRetriever interface {
	GetInterfaceIPInfo(ctx context.Context) (nmagent.Interfaces, error)
}

// This interface is implemented by whoever consumes the secondary IPs fetched in nodesubnet
type SecondaryIPConsumer interface {
	UpdateSecondaryIPs([]net.IP) error
}

type IPFetcher struct {
	// Node subnet config
	ipFectcherClient InterfaceRetriever
	ticker           *time.Ticker
	tickerInterval   time.Duration
	consumer         SecondaryIPConsumer
}

func NewIPFetcher(nmaClient InterfaceRetriever, c SecondaryIPConsumer) *IPFetcher {
	return &IPFetcher{
		ipFectcherClient: nmaClient,
		consumer:         c,
	}
}

func (c *IPFetcher) updateFetchIntervalForNoObservedDiff() {
	c.tickerInterval = min(c.tickerInterval*2, MaxRefreshInterval)
	c.ticker.Reset(c.tickerInterval)
}

func (c *IPFetcher) updateFetchIntervalForObservedDiff() {
	c.tickerInterval = MinRefreshInterval
	c.ticker.Reset(c.tickerInterval)
}

func (c *IPFetcher) Start(ctx context.Context) {
	go func() {
		c.tickerInterval = MinRefreshInterval
		c.ticker = time.NewTicker(c.tickerInterval)
		defer c.ticker.Stop()

		for {
			select {
			case <-c.ticker.C:
				err := c.RefreshSecondaryIPs(ctx)
				if err != nil {
					log.Printf("Error refreshing secondary IPs: %v", err)
				}
			case <-ctx.Done():
				log.Println("IPFetcher stopped")
				return
			}
		}
	}()
}

// If secondaryIPQueryInterval has elapsed since the last fetch, fetch secondary IPs
func (c *IPFetcher) RefreshSecondaryIPs(ctx context.Context) error {
	response, err := c.ipFectcherClient.GetInterfaceIPInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "getting interface IPs")
	}

	res := flattenIPListFromResponse(&response)
	err = c.consumer.UpdateSecondaryIPs(res)
	if err != nil {
		return errors.Wrap(err, "updating secondary IPs")
	}

	return nil
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
