// Package event defines the coverage event types used across gococo.
package event

// CoverEvent represents a single basic block execution event
// streamed from an instrumented binary to the gococo server.
type CoverEvent struct {
	Seq       uint64 `json:"seq"`
	Timestamp int64  `json:"ts"`
	GID       int64  `json:"gid"`
	FileID    string `json:"file"`
	BlockIdx  int    `json:"block"`
	StartLine int    `json:"sl"`
	StartCol  int    `json:"sc"`
	EndLine   int    `json:"el"`
	EndCol    int    `json:"ec"`
	NumStmts  int    `json:"stmts"`
}

// AgentInfo describes a connected instrumented process.
type AgentInfo struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	PID      int    `json:"pid"`
	CmdLine  string `json:"cmdline"`
	RemoteIP string `json:"remote_ip"`
}
