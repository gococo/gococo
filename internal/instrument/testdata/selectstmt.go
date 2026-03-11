package testdata

import "time"

// Simple select.
func SimpleSelect(ch1, ch2 <-chan int) int {
	select {
	case v := <-ch1:
		return v
	case v := <-ch2:
		return v
	}
}

// Select with default.
func SelectDefault(ch <-chan int) int {
	select {
	case v := <-ch:
		return v
	default:
		return -1
	}
}

// Select with timeout.
func SelectTimeout(ch <-chan int) int {
	select {
	case v := <-ch:
		return v
	case <-time.After(time.Millisecond):
		return -1
	}
}

// Select in loop.
func SelectInLoop(ch <-chan int, done <-chan struct{}) int {
	sum := 0
	for {
		select {
		case v := <-ch:
			sum += v
		case <-done:
			return sum
		}
	}
}

// Empty select (blocks forever).
// Note: we don't call this in tests, only verify instrumentation.
func EmptySelect() {
	select {}
}
