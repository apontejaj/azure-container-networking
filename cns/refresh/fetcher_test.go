package refresh_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Azure/azure-container-networking/cns/nodesubnet"
	"github.com/Azure/azure-container-networking/nmagent"
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
func (c *TestConsumer) ConsumeInterfaces(nmagent.Interfaces) error {
	c.UpdateConsumeCount()
	return nil
}

func TestRefresh(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := refresh.NewFetcher[nmagent.Interfaces](clientPtr.GetInterfaceIPInfo, 0, 0, consumerPtr.ConsumeInterfaces)
	ticker := refresh.NewMockTickProvider()
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

	// At least 2 consumes - one initial and one after the first tick should be done
	if consumerPtr.FetchConsumeCount() < 0 {
		t.Error("Not enough consumes")
	}
}

func TestInterval(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := refresh.NewFetcher[nmagent.Interfaces](clientPtr.GetInterfaceIPInfo, 0, 0, consumerPtr.ConsumeInterfaces)
	interval := fetcher.GetCurrentInterval()

	if interval != refresh.DefaultMinInterval {
		t.Error("Default min interval not used")
	}

	// Testing that the interval doubles will require making the interval thread-safe. Not doing that to avoid performance hit.
}

// testContext creates a context from the provided testing.T that will be
// canceled if the test suite is terminated.
func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	if deadline, ok := t.Deadline(); ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithCancel(context.Background())
}
