package retry

import "time"

func Do(f func() error, maxRuns, sleepMs int) error {
	var err error
	for i := 0; i < maxRuns; i++ {
		err = f()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
	return err
}
