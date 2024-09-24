package nodesubnet

import (
	"context"
	"log"
	"net/netip"
	"time"

	"github.com/Azure/azure-container-networking/nmagent"
	"github.com/Azure/azure-container-networking/refreshticker"
	"github.com/pkg/errors"
)

const (
	// Default minimum time between secondary IP fetches
	DefaultMinRefreshInterval = 4 * time.Second
	// Default maximum time between secondary IP fetches
	DefaultMaxRefreshInterval = 1024 * time.Second
)

var ErrRefreshSkipped = errors.New("refresh skipped due to throttling")

// InterfaceRetriever is an interface is implemented by the NMAgent Client, and also a mock client for testing.
type InterfaceRetriever interface {
	GetInterfaceIPInfo(ctx context.Context) (nmagent.Interfaces, error)
}

// IPConsumer is an interface implemented by whoever consumes the secondary IPs fetched in nodesubnet
type IPConsumer interface {
	UpdateIPsForNodeSubnet(netip.Addr, []netip.Addr) error
}

// IPFetcher fetches secondary IPs from NMAgent at regular intervals. The
// interval will vary within the range of minRefreshInterval and
// maxRefreshInterval. When no diff is observed after a fetch, the interval
// doubles (subject to the maximum interval). When a diff is observed, the
// interval resets to the minimum.
type IPFetcher struct {
	// Node subnet config
	intfFetcherClient  InterfaceRetriever
	ticker             refreshticker.TickProvider
	tickerInterval     time.Duration
	consumer           IPConsumer
	minRefreshInterval time.Duration
	maxRefreshInterval time.Duration
}

// NewIPFetcher creates a new IPFetcher. If minInterval is 0, it will default to 4 seconds.
// If maxInterval is 0, it will default to 1024 seconds (or minInterval, if it is higher).
func NewIPFetcher(
	client InterfaceRetriever,
	consumer IPConsumer,
	minInterval time.Duration,
	maxInterval time.Duration,
) *IPFetcher {
	if minInterval == 0 {
		minInterval = DefaultMinRefreshInterval
	}

	if maxInterval == 0 {
		maxInterval = DefaultMaxRefreshInterval
	}

	maxInterval = max(maxInterval, minInterval)

	return &IPFetcher{
		intfFetcherClient:  client,
		consumer:           consumer,
		minRefreshInterval: minInterval,
		maxRefreshInterval: maxInterval,
		tickerInterval:     minInterval,
	}
}

// UpdateFetchIntervalForNoObservedDiff informs IPFetcher that no diff was observed in the last fetch.
// In the current design, this doubles the fetch interval, subject to the maximum interval.
func (c *IPFetcher) UpdateFetchIntervalForNoObservedDiff() {
	c.tickerInterval = min(c.tickerInterval*2, c.maxRefreshInterval) //nolint:gomnd // doubling interval

	if c.ticker != nil {
		c.ticker.Reset(c.tickerInterval)
	}
}

// UpdateFetchIntervalForNoObservedDiff informs IPFetcher that a diff was observed in the last fetch.
// In the current design, this resets the fetch interval to the minimum.
func (c *IPFetcher) UpdateFetchIntervalForObservedDiff() {
	c.tickerInterval = c.minRefreshInterval

	if c.ticker != nil {
		c.ticker.Reset(c.tickerInterval)
	}
}

// Start the IPFetcher.
func (c *IPFetcher) Start(ctx context.Context) {
	go func() {
		// Do an initial fetch
		err := c.RefreshSecondaryIPs(ctx)
		if err != nil {
			log.Printf("Error refreshing secondary IPs: %v", err)
		}

		if c.ticker == nil {
			c.ticker = refreshticker.NewTimedTickProvider(c.tickerInterval)
		}

		defer c.ticker.Stop()

		for {
			select {
			case <-c.ticker.C():
				err := c.RefreshSecondaryIPs(ctx)
				if err != nil {
					log.Printf("Error refreshing secondary IPs: %v", err)
				}
			case <-ctx.Done():
				log.Printf("IPFetcher stopped")
				return
			}
		}
	}()
}

// Fetch IPs from NMAgent and pass to the consumer
func (c *IPFetcher) RefreshSecondaryIPs(ctx context.Context) error {
	response, err := c.intfFetcherClient.GetInterfaceIPInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "getting interface IPs")
	}

	if len(response.Entries) == 0 {
		return errors.New("no interfaces found in response from NMAgent")
	}

	primaryIP, secondaryIPs := flattenIPListFromResponse(&response)
	err = c.consumer.UpdateIPsForNodeSubnet(primaryIP, secondaryIPs)
	if err != nil {
		return errors.Wrap(err, "updating secondary IPs")
	}

	return nil
}

// Get the list of secondary IPs from fetched Interfaces
func flattenIPListFromResponse(resp *nmagent.Interfaces) (primary netip.Addr, secondaryIPs []netip.Addr) {
	var primaryIP netip.Addr
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
					primaryIP = netip.Addr(a.Address)
					continue
				}

				secondaryIPs = append(secondaryIPs, netip.Addr(a.Address))
				addressCount++
			}
			log.Printf("Got %d addresses from subnet %s", addressCount, s.Prefix)
		}
	}

	return primaryIP, secondaryIPs
}
