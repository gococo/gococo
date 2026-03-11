package testdata

import "sync"

// Simple goroutine.
func SimpleGoroutine() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = 42
	}()
	wg.Wait()
}

// Multiple goroutines.
func MultiGoroutine(n int) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = id * 2
		}(i)
	}
	wg.Wait()
}

// Goroutine with channel communication.
func GoroutineWithChannel() int {
	ch := make(chan int, 1)
	go func() {
		ch <- 42
	}()
	return <-ch
}

// Producer-consumer pattern.
func ProducerConsumer() int {
	ch := make(chan int, 10)
	var wg sync.WaitGroup

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()

	// Consumer
	sum := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range ch {
			sum += v
		}
	}()

	wg.Wait()
	return sum
}

// Goroutine launched from a named function.
func namedWorker(ch chan<- int, val int) {
	ch <- val * 2
}

func LaunchNamedWorker() int {
	ch := make(chan int, 1)
	go namedWorker(ch, 21)
	return <-ch
}
