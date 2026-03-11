package event

import "sync"

// RingBuffer is a fixed-size circular buffer for CoverEvents.
// It is safe for concurrent use.
type RingBuffer struct {
	mu    sync.RWMutex
	buf   []CoverEvent
	cap   int
	head  int // next write position
	count int // total written (monotonic)
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buf: make([]CoverEvent, capacity),
		cap: capacity,
	}
}

// Push appends an event to the buffer.
func (rb *RingBuffer) Push(e CoverEvent) {
	rb.mu.Lock()
	rb.buf[rb.head%rb.cap] = e
	rb.head++
	rb.count++
	rb.mu.Unlock()
}

// Last returns the most recent n events in chronological order.
func (rb *RingBuffer) Last(n int) []CoverEvent {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	available := rb.head
	if available > rb.cap {
		available = rb.cap
	}
	if n > available {
		n = available
	}
	if n == 0 {
		return nil
	}

	result := make([]CoverEvent, n)
	start := rb.head - n
	for i := 0; i < n; i++ {
		result[i] = rb.buf[(start+i)%rb.cap]
	}
	return result
}

// Count returns the total number of events ever written.
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
