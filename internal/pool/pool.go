package pool

type Pool struct {
	semaphore   chan struct{}
	stop        chan struct{}
	concurrency int
}

func New(concurrency int, maxFetch int, fetch func(int) []func()) Pool {
	semaphore := make(chan struct{}, concurrency)
	stop := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		semaphore <- struct{}{}
	}

	go (func() {
		for {
			select {
			case _, _ = <-stop:
				return
			case <-semaphore:
				// We can run at least one job. See if there's more (up to
				// maxFetch), to allow fetching batches of jobs at once.
				num := 1
			Counter:
				for {
					select {
					case <-semaphore:
						num++
						if num == maxFetch {
							break Counter
						}
					default:
						break Counter
					}
				}

				for _, fn := range fetch(num) {
					go (func() {
						fn()
						semaphore <- struct{}{}
					})()

					num--
				}

				// We might have fetched fewer jobs than we had space for.
				for i := 0; i < num; i++ {
					semaphore <- struct{}{}
				}
			}
		}

	})()

	return Pool{stop, semaphore, concurrency}
}

func (p *Pool) Stop() {
	close(p.stop)

	// Wait for all workers to finish
	for i := 0; i < p.concurrency; i++ {
		<-p.semaphore
	}
}
