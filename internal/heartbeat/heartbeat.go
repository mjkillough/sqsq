package heartbeat

import (
	"sync"
	"time"
)

// Run synchronously runs the provided fn(), running beat() every interval.
func Run(interval time.Duration, beat func(), fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go (func() {
		fn()
		wg.Done()
	})()

	ticker := time.NewTicker(interval)
	go (func() {
		for range ticker.C {
			beat()
		}
	})()

	wg.Wait()
	ticker.Stop()
}
