package nodesubnet_test

import (
	"context"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns/nodesubnet"
	"github.com/Azure/azure-container-networking/nmagent"
)

// Mock client that simply tracks if refresh has been called
type TestClient struct {
	refreshCount int32
}

// Mock refresh
func (c *TestClient) GetInterfaceIPInfo(_ context.Context) (nmagent.Interfaces, error) {
	atomic.AddInt32(&c.refreshCount, 1)
	return nmagent.Interfaces{}, nil
}

var _ nodesubnet.InterfaceRetriever = &TestClient{}

// Mock client that simply consumes fetched IPs
type TestConsumer struct {
	consumeCount int32
}

// Mock IP update
func (c *TestConsumer) UpdateIPsForNodeSubnet(_ netip.Addr, _ []netip.Addr) error {
	atomic.AddInt32(&c.consumeCount, 1)
	return nil
}

var _ nodesubnet.IPConsumer = &TestConsumer{}

// MockTickProvider is a mock implementation of the TickProvider interface
type MockTickProvider struct {
	tickChan        chan time.Time
	currentDuration time.Duration
}

// NewMockTickProvider creates a new MockTickProvider
func NewMockTickProvider() *MockTickProvider {
	return &MockTickProvider{
		tickChan: make(chan time.Time, 1),
	}
}

// C returns the channel on which ticks are delivered
func (m *MockTickProvider) C() <-chan time.Time {
	return m.tickChan
}

// Stop stops the ticker
func (m *MockTickProvider) Stop() {
	close(m.tickChan)
}

// Tick manually sends a tick to the channel
func (m *MockTickProvider) Tick() {
	m.tickChan <- time.Now()
}

func (m *MockTickProvider) Reset(d time.Duration) {
	m.currentDuration = d
}

var _ nodesubnet.TickProvider = &MockTickProvider{}

func TestRefresh(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := nodesubnet.NewIPFetcher(clientPtr, consumerPtr, 0, 0)
	ticker := NewMockTickProvider()
	fetcher.SetTicker(ticker)
	ctx, cancel := testContext(t)
	defer cancel()
	fetcher.Start(ctx)
	ticker.Tick() // Trigger a refresh
	ticker.Tick() // This tick will be read only after previous refresh is done
	ticker.Tick() // This call will block until the prevous tick is read

	// At least 2 refreshes - one initial and one after the first tick should be done
	if atomic.LoadInt32(&clientPtr.refreshCount) < 2 {
		t.Error("Not enough refreshes")
	}

	// At least 2 consumes - one initial and one after the first tick should be done
	if atomic.LoadInt32(&consumerPtr.consumeCount) < 2 {
		t.Error("Not enough consumes")
	}
}

func TestIntervalUpdate(t *testing.T) {
	clientPtr := &TestClient{}
	consumerPtr := &TestConsumer{}
	fetcher := nodesubnet.NewIPFetcher(clientPtr, consumerPtr, 0, 0)
	interval := fetcher.GetCurrentQueryInterval()
	ticker := NewMockTickProvider()
	fetcher.SetTicker(ticker)

	if interval != nodesubnet.DefaultMinRefreshInterval {
		t.Error("Default min interval not used")
	}

	for range 10 {
		fetcher.UpdateFetchIntervalForNoObservedDiff()
		exp := interval * 2
		if interval == nodesubnet.DefaultMaxRefreshInterval {
			exp = nodesubnet.DefaultMaxRefreshInterval
		}
		if fetcher.GetCurrentQueryInterval() != exp || ticker.currentDuration != exp {
			t.Error("Interval not updated correctly")
		} else {
			interval = exp
		}
	}

	fetcher.UpdateFetchIntervalForObservedDiff()

	if fetcher.GetCurrentQueryInterval() != nodesubnet.DefaultMinRefreshInterval || ticker.currentDuration != nodesubnet.DefaultMinRefreshInterval {
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
