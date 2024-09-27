package refresh

import "time"

func (f *Fetcher[T]) GetCurrentInterval() time.Duration {
	return f.currentInterval
}

func (f *Fetcher[T]) SetTicker(t TickProvider) {
	f.ticker = t
}
