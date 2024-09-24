package nodesubnet_test

import (
	"context"
	"net/netip"
	"sync"
	"testing"

	"github.com/Azure/azure-container-networking/cns/nodesubnet"
	"github.com/Azure/azure-container-networking/nmagent"
	"github.com/Azure/azure-container-networking/refreshticker"
)

// Mock client that simply tracks if refresh has been called
type TestClient struct {
	refreshCount int32
	mu           sync.Mutex
}

// FetchRefreshCount atomically fetches the refresh count
func (c *TestClient) FetchRefreshCount() int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshCount
}

// UpdateRefreshCount atomically updates the refresh count
func (c *TestClient) UpdateRefreshCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshCount++
}

// Mock refresh
func (c *TestClient) GetInterfaceIPInfo(_ context.Context) (nmagent.Interfaces, error) {
	c.UpdateRefreshCount()
	return nmagent.Interfaces{}, nil
}

var _ nodesubnet.InterfaceRetriever = &TestClient{}

// Mock client that simply consumes fetched IPs
type TestConsumer struct {
	consumeCount int32
	mu           sync.Mutex
}

// FetchConsumeCount atomically fetches the consume count
func (c *TestConsumer) FetchConsumeCount() int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consumeCount
}

// UpdateConsumeCount atomically updates the consume count
func (c *TestConsumer) UpdateConsumeCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consumeCount++
}

// Mock IP update
func (c *TestConsumer) UpdateIPsForNodeSubnet(_ netip.Addr, _ []netip.Addr) error {
	c.UpdateConsumeCount()
	return nil
}

var _ nodesubnet.IPConsumer = &TestConsumer{}

func TestRefresh(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := nodesubnet.NewIPFetcher(clientPtr, consumerPtr, 0, 0)
	ticker := refreshticker.NewMockTickProvider()
	fetcher.SetTicker(ticker)
	ctx, cancel := testContext(t)
	defer cancel()
	fetcher.Start(ctx)
	ticker.Tick() // Trigger a refresh
	ticker.Tick() // This tick will be read only after previous refresh is done
	ticker.Tick() // This call will block until the prevous tick is read

	// At least 2 refreshes - one initial and one after the first tick should be done
	if clientPtr.FetchRefreshCount() < 2 {
		t.Error("Not enough refreshes")
	}

	// No consumes, since the responses are empty
	if consumerPtr.FetchConsumeCount() > 0 {
		t.Error("Consume called unexpectedly, shouldn't be called since responses are empty")
	}
}

func TestIntervalUpdate(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := nodesubnet.NewIPFetcher(clientPtr, consumerPtr, 0, 0)
	interval := fetcher.GetCurrentQueryInterval()
	ticker := refreshticker.NewMockTickProvider()
	fetcher.SetTicker(ticker)

	if interval != nodesubnet.DefaultMinRefreshInterval {
		t.Error("Default min interval not used")
	}

	for i := 1; i <= 10; i++ {
		fetcher.UpdateFetchIntervalForNoObservedDiff()
		exp := interval * 2
		if interval == nodesubnet.DefaultMaxRefreshInterval {
			exp = nodesubnet.DefaultMaxRefreshInterval
		}
		if fetcher.GetCurrentQueryInterval() != exp || ticker.GetCurrentDuration() != exp {
			t.Error("Interval not updated correctly")
		} else {
			interval = exp
		}
	}

	fetcher.UpdateFetchIntervalForObservedDiff()

	if fetcher.GetCurrentQueryInterval() != nodesubnet.DefaultMinRefreshInterval || ticker.GetCurrentDuration() != nodesubnet.DefaultMinRefreshInterval {
		t.Error("Observed diff update incorrect")
	}
}

// testContext creates a context from the provided testing.T that will be
// canceled if the test suite is terminated.
func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	if deadline, ok := t.Deadline(); ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithCancel(context.Background())
}
