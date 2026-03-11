package server

import (
	"sync"
	"sync/atomic"

	"github.com/gococo/gococo/internal/event"
)

// Hub manages event broadcasting from agents to UI clients.
type Hub struct {
	ring    *event.RingBuffer
	clients sync.Map // clientID -> chan event.CoverEvent
	nextID  int64
}

// NewHub creates a new event hub with the given history capacity.
func NewHub(historySize int) *Hub {
	return &Hub{
		ring: event.NewRingBuffer(historySize),
	}
}

// Publish stores an event and broadcasts it to all connected clients.
func (h *Hub) Publish(e event.CoverEvent) {
	h.ring.Push(e)
	h.clients.Range(func(key, value interface{}) bool {
		ch := value.(chan event.CoverEvent)
		select {
		case ch <- e:
		default:
			// slow client, drop event
		}
		return true
	})
}

// Subscribe returns a channel that receives new events and a cancel function.
func (h *Hub) Subscribe(bufSize int) (<-chan event.CoverEvent, func()) {
	id := atomic.AddInt64(&h.nextID, 1)
	ch := make(chan event.CoverEvent, bufSize)
	h.clients.Store(id, ch)
	cancel := func() {
		h.clients.Delete(id)
		close(ch)
	}
	return ch, cancel
}

// History returns the most recent n events.
func (h *Hub) History(n int) []event.CoverEvent {
	return h.ring.Last(n)
}

// TotalEvents returns the count of all events ever received.
func (h *Hub) TotalEvents() int {
	return h.ring.Count()
}
