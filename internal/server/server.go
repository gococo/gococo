// Package server implements the gococo relay server.
// It receives real-time coverage events from instrumented binaries
// and broadcasts them to connected UI clients via SSE.
package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gococo/gococo/internal/event"
	"github.com/gococo/gococo/internal/protocol"
)

// Server is the gococo relay server.
type Server struct {
	hub      *Hub
	agents   *AgentRegistry
	addr     string
	mux      *http.ServeMux
	sourceFS http.FileSystem // for serving embedded web UI

	// Coverage summary tracking
	mu          sync.RWMutex
	blockStates map[string]*blockState // "file:block" -> state
}

type blockState struct {
	File      string
	BlockIdx  int
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	NumStmts  int
	HitCount  uint64
	LastHitAt time.Time
}

// New creates a new gococo server.
func New(addr string, webFS http.FileSystem) *Server {
	s := &Server{
		hub:         NewHub(100000),
		agents:      NewAgentRegistry(),
		addr:        addr,
		mux:         http.NewServeMux(),
		sourceFS:    webFS,
		blockStates: make(map[string]*blockState),
	}
	s.routes()
	return s
}

// Run starts the server.
func (s *Server) Run() error {
	log.Printf("[gococo] server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes() {
	// Internal API (for instrumented binaries)
	s.mux.HandleFunc("/api/internal/register", s.handleRegister)
	s.mux.HandleFunc("/api/internal/register-blocks", s.handleRegisterBlocks)
	s.mux.HandleFunc("/api/internal/events", s.handleEvents)

	// Public API (for UI)
	s.mux.HandleFunc("/api/agents", s.handleListAgents)
	s.mux.HandleFunc("/api/events/stream", s.handleEventStream)
	s.mux.HandleFunc("/api/events/history", s.handleEventHistory)
	s.mux.HandleFunc("/api/coverage/summary", s.handleCoverageSummary)
	s.mux.HandleFunc("/api/coverage/blocks", s.handleCoverageBlocks)

	// Web UI
	if s.sourceFS != nil {
		s.mux.Handle("/", http.FileServer(s.sourceFS))
	}
}

// handleRegister registers a new agent.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	pidStr := r.URL.Query().Get("pid")
	cmdline := r.URL.Query().Get("cmdline")

	if hostname == "" || pidStr == "" {
		http.Error(w, "missing hostname or pid", http.StatusBadRequest)
		return
	}

	pid, _ := strconv.Atoi(pidStr)
	remoteIP := r.RemoteAddr

	id := s.agents.Register(hostname, pid, cmdline, remoteIP)
	log.Printf("[gococo] agent registered: id=%s hostname=%s pid=%d cmdline=%s", id, hostname, pid, cmdline)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, id)
}

// handleRegisterBlocks receives all block metadata from an agent at startup.
// This allows the server to know about ALL blocks (including uncovered ones).
// Format: file|blockIdx|startLine|startCol|endLine|endCol|numStmts per line.
func (s *Server) handleRegisterBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	scanner := bufio.NewScanner(r.Body)
	count := 0
	s.mu.Lock()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 7)
		if len(parts) != 7 {
			continue
		}
		file := parts[0]
		blockIdx, _ := strconv.Atoi(parts[1])
		sl, _ := strconv.Atoi(parts[2])
		sc, _ := strconv.Atoi(parts[3])
		el, _ := strconv.Atoi(parts[4])
		ec, _ := strconv.Atoi(parts[5])
		stmts, _ := strconv.Atoi(parts[6])

		key := fmt.Sprintf("%s:%d", file, blockIdx)
		if _, exists := s.blockStates[key]; !exists {
			s.blockStates[key] = &blockState{
				File:      file,
				BlockIdx:  blockIdx,
				StartLine: sl,
				StartCol:  sc,
				EndLine:   el,
				EndCol:    ec,
				NumStmts:  stmts,
			}
			count++
		}
	}
	s.mu.Unlock()

	log.Printf("[gococo] registered %d blocks from agent", count)
	w.WriteHeader(http.StatusOK)
}

// handleEvents receives a chunked stream of coverage events from an agent.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "missing agent_id", http.StatusBadRequest)
		return
	}

	s.agents.SetConnected(agentID, true)
	defer s.agents.SetConnected(agentID, false)

	log.Printf("[gococo] agent %s event stream connected", agentID)

	scanner := bufio.NewScanner(r.Body)
	scanner.Buffer(make([]byte, 4096), 64*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ev, err := protocol.DecodeCoverEvent(line)
		if err != nil {
			log.Printf("[gococo] decode error from agent %s: %v", agentID, err)
			continue
		}

		s.hub.Publish(ev)
		s.updateBlockState(&ev)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("[gococo] agent %s stream error: %v", agentID, err)
	}

	log.Printf("[gococo] agent %s event stream disconnected", agentID)
}

func (s *Server) updateBlockState(e *event.CoverEvent) {
	key := fmt.Sprintf("%s:%d", e.FileID, e.BlockIdx)
	s.mu.Lock()
	bs, ok := s.blockStates[key]
	if !ok {
		bs = &blockState{
			File:      e.FileID,
			BlockIdx:  e.BlockIdx,
			StartLine: e.StartLine,
			StartCol:  e.StartCol,
			EndLine:   e.EndLine,
			EndCol:    e.EndCol,
			NumStmts:  e.NumStmts,
		}
		s.blockStates[key] = bs
	}
	bs.HitCount++
	bs.LastHitAt = time.Now()
	s.mu.Unlock()
}

// handleListAgents returns all registered agents.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	agents := s.agents.List()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
	})
}

// handleEventStream sends real-time events to UI clients via SSE.
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch, cancel := s.hub.Subscribe(4096)
	defer cancel()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleEventHistory returns recent events.
func (s *Server) handleEventHistory(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	n := 1000
	if q := r.URL.Query().Get("last"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			n = v
		}
	}

	events := s.hub.History(n)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"total":  s.hub.TotalEvents(),
	})
}

// CoverageSummaryEntry describes coverage for a single file.
type CoverageSummaryEntry struct {
	File        string  `json:"file"`
	TotalBlocks int     `json:"total_blocks"`
	HitBlocks   int     `json:"hit_blocks"`
	TotalStmts  int     `json:"total_stmts"`
	HitStmts    int     `json:"hit_stmts"`
	Percentage  float64 `json:"percentage"`
}

// handleCoverageSummary returns per-file coverage stats.
func (s *Server) handleCoverageSummary(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	s.mu.RLock()
	// Group by file
	fileBlocks := make(map[string][]*blockState)
	for _, bs := range s.blockStates {
		fileBlocks[bs.File] = append(fileBlocks[bs.File], bs)
	}
	s.mu.RUnlock()

	var entries []CoverageSummaryEntry
	var totalStmts, hitStmts int

	for file, blocks := range fileBlocks {
		entry := CoverageSummaryEntry{File: file, TotalBlocks: len(blocks)}
		for _, bs := range blocks {
			entry.TotalStmts += bs.NumStmts
			totalStmts += bs.NumStmts
			if bs.HitCount > 0 {
				entry.HitBlocks++
				entry.HitStmts += bs.NumStmts
				hitStmts += bs.NumStmts
			}
		}
		if entry.TotalStmts > 0 {
			entry.Percentage = float64(entry.HitStmts) / float64(entry.TotalStmts) * 100
		}
		entries = append(entries, entry)
	}

	var overallPct float64
	if totalStmts > 0 {
		overallPct = float64(hitStmts) / float64(totalStmts) * 100
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":          entries,
		"total_stmts":    totalStmts,
		"hit_stmts":      hitStmts,
		"overall_pct":    overallPct,
		"total_events":   s.hub.TotalEvents(),
	})
}

// BlockDetail is the JSON shape for a single coverage block.
type BlockDetail struct {
	BlockIdx  int    `json:"block_idx"`
	StartLine int    `json:"sl"`
	StartCol  int    `json:"sc"`
	EndLine   int    `json:"el"`
	EndCol    int    `json:"ec"`
	NumStmts  int    `json:"stmts"`
	HitCount  uint64 `json:"hit_count"`
	LastHitAt int64  `json:"last_hit_ts"` // unix ms
}

// handleCoverageBlocks returns block-level coverage for a given file.
// Query param: file=<import_path/filename>
func (s *Server) handleCoverageBlocks(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	fileQuery := r.URL.Query().Get("file")

	s.mu.RLock()
	var blocks []BlockDetail
	for _, bs := range s.blockStates {
		if fileQuery != "" && bs.File != fileQuery {
			continue
		}
		blocks = append(blocks, BlockDetail{
			BlockIdx:  bs.BlockIdx,
			StartLine: bs.StartLine,
			StartCol:  bs.StartCol,
			EndLine:   bs.EndLine,
			EndCol:    bs.EndCol,
			NumStmts:  bs.NumStmts,
			HitCount:  bs.HitCount,
			LastHitAt: bs.LastHitAt.UnixMilli(),
		})
	}
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"blocks": blocks,
	})
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
