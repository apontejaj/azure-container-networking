package nodesubnet

import "time"

// TickProvider defines a wrapper for time.Ticker
type TickProvider interface {
	Stop()
	Reset(d time.Duration)
	C() <-chan time.Time
}

// TimedTickProvider wraps a time.Ticker to implement TickProvider
type TimedTickProvider struct {
	ticker *time.Ticker
}

var _ TickProvider = &TimedTickProvider{}

// NewTickerWrapper creates a new TickerWrapper
func NewTimedTickProvider(d time.Duration) *TimedTickProvider {
	return &TimedTickProvider{ticker: time.NewTicker(d)}
}

// Stop stops the ticker
func (tw *TimedTickProvider) Stop() {
	tw.ticker.Stop()
}

// Reset resets the ticker with a new duration
func (tw *TimedTickProvider) Reset(d time.Duration) {
	tw.ticker.Reset(d)
}

// C returns the ticker's channel
func (tw *TimedTickProvider) C() <-chan time.Time {
	return tw.ticker.C
}
