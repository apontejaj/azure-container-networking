package nodesubnet

import "time"

// These methods is in this file (_test.go) because they are helpers. They are
// built during tests, and are not part of the main code.

func (c *IPFetcher) GetCurrentQueryInterval() time.Duration {
	return c.tickerInterval
}

func (c *IPFetcher) SetTicker(tickProvider TickProvider) {
	c.ticker = tickProvider
}
