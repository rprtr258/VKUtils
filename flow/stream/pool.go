package stream

import "sync"

// Pool is a pipe capable of running tasks in parallel.
type Pool[A any] func(Stream[func() A]) Stream[A]

// NewPool creates an execution pool that will execute tasks concurrently.
// Simultaneously there could be as many as size executions.
func NewPool[A any](size int) Pool[A] {
	tasks := make(chan func() A)
	results := make(chan A)
	var wg sync.WaitGroup
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			for task := range tasks {
				results <- task()
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return func(xs Stream[func() A]) Stream[A] {
		return FromPairOfChannels(xs, tasks, results)
	}
}
