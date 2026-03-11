// Package protocol defines the wire format between instrumented binaries and the gococo server.
//
// Each coverage event is encoded as a pipe-delimited line:
//
//	SEQ|TIMESTAMP|GID|FILE|BLOCK|START_LINE|START_COL|END_LINE|END_COL|NUM_STMTS
package protocol

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gococo/gococo/internal/event"
)

// EncodeCoverEvent encodes a CoverEvent to wire format.
func EncodeCoverEvent(e *event.CoverEvent) string {
	return fmt.Sprintf("%d|%d|%d|%s|%d|%d|%d|%d|%d|%d",
		e.Seq, e.Timestamp, e.GID, e.FileID, e.BlockIdx,
		e.StartLine, e.StartCol, e.EndLine, e.EndCol, e.NumStmts)
}

// DecodeCoverEvent decodes a wire format line into a CoverEvent.
func DecodeCoverEvent(line string) (event.CoverEvent, error) {
	parts := strings.Split(line, "|")
	if len(parts) != 10 {
		return event.CoverEvent{}, fmt.Errorf("invalid event line: expected 10 fields, got %d", len(parts))
	}

	var e event.CoverEvent
	var err error

	e.Seq, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return e, fmt.Errorf("invalid seq: %w", err)
	}
	e.Timestamp, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return e, fmt.Errorf("invalid timestamp: %w", err)
	}
	e.GID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return e, fmt.Errorf("invalid gid: %w", err)
	}
	e.FileID = parts[3]
	e.BlockIdx, err = strconv.Atoi(parts[4])
	if err != nil {
		return e, fmt.Errorf("invalid block: %w", err)
	}
	e.StartLine, err = strconv.Atoi(parts[5])
	if err != nil {
		return e, fmt.Errorf("invalid start_line: %w", err)
	}
	e.StartCol, err = strconv.Atoi(parts[6])
	if err != nil {
		return e, fmt.Errorf("invalid start_col: %w", err)
	}
	e.EndLine, err = strconv.Atoi(parts[7])
	if err != nil {
		return e, fmt.Errorf("invalid end_line: %w", err)
	}
	e.EndCol, err = strconv.Atoi(parts[8])
	if err != nil {
		return e, fmt.Errorf("invalid end_col: %w", err)
	}
	e.NumStmts, err = strconv.Atoi(parts[9])
	if err != nil {
		return e, fmt.Errorf("invalid num_stmts: %w", err)
	}
	return e, nil
}
