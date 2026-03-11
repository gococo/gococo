package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gococo/gococo/internal/event"
)

// AgentRegistry manages connected instrumented processes.
type AgentRegistry struct {
	agents sync.Map // id -> *AgentState
	nextID int64
}

// AgentState tracks the state of a connected agent.
type AgentState struct {
	Info      event.AgentInfo
	Connected bool
	Since     time.Time
}

// NewAgentRegistry creates a new agent registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{}
}

// Register adds a new agent and returns its ID.
func (r *AgentRegistry) Register(hostname string, pid int, cmdline string, remoteIP string) string {
	id := atomic.AddInt64(&r.nextID, 1)
	idStr := itoa(id)

	state := &AgentState{
		Info: event.AgentInfo{
			ID:       idStr,
			Hostname: hostname,
			PID:      pid,
			CmdLine:  cmdline,
			RemoteIP: remoteIP,
		},
		Connected: true,
		Since:     time.Now(),
	}
	r.agents.Store(idStr, state)
	return idStr
}

// SetConnected updates the connection status of an agent.
func (r *AgentRegistry) SetConnected(id string, connected bool) {
	if raw, ok := r.agents.Load(id); ok {
		state := raw.(*AgentState)
		state.Connected = connected
	}
}

// List returns all registered agents.
func (r *AgentRegistry) List() []AgentState {
	var result []AgentState
	r.agents.Range(func(key, value interface{}) bool {
		state := value.(*AgentState)
		result = append(result, *state)
		return true
	})
	return result
}

// Remove deletes an agent from the registry.
func (r *AgentRegistry) Remove(id string) {
	r.agents.Delete(id)
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
