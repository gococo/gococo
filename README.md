# gococo — Go Coverage Collection Tools

**Real-time Go coverage visualization.**

gococo instruments your Go binary at build time, streams per-block execution events as they happen, and displays live coverage in a web UI. Unlike Go's built-in coverage (batch dump at exit), gococo shows you *which code is running right now*.

## How It Works

```
┌──────────────┐   instrument    ┌──────────────────┐
│  Go Project  │ ──────────────► │ Instrumented Bin  │
└──────────────┘   gococo build  └────────┬─────────┘
                                          │ events (HTTP stream)
                                          ▼
                                 ┌──────────────────┐   SSE
                                 │  gococo server   │ ──────► Web UI
                                 └──────────────────┘
```

1. **`gococo build`** — Parses Go source via AST, injects a counter increment + event emit at every basic block, builds the modified binary.
2. **Instrumented binary** — Runs normally. A background agent streams block-level events to the server via chunked HTTP POST.
3. **`gococo server`** — Receives events, tracks per-block hit counts, serves a real-time web UI via Server-Sent Events (SSE).

## Quick Start

### Install

```bash
go install github.com/gococo/gococo/cmd/gococo@latest
```

### Usage

```bash
# 1. Start the server (in your project directory for source display)
cd /path/to/your/project
gococo server --addr 127.0.0.1:7778

# 2. Instrument and build your project
gococo build --host 127.0.0.1:7778 -o ./myapp-instrumented .

# 3. Run the instrumented binary
./myapp-instrumented

# 4. Open the web UI
open http://127.0.0.1:7778
```

### Try the Demo

```bash
# Build gococo
go build -o gococo ./cmd/gococo/

# Start server pointed at demo source
./gococo server --root examples/demo &

# Instrument and run the demo (auto-exercises code every second)
cd examples/demo
../../gococo build --host 127.0.0.1:7778 -o /tmp/demo .
/tmp/demo

# Open http://127.0.0.1:7778 and watch coverage grow in real time
```

## Web UI Features

- **Source code view** — Real source with line-level coverage highlighting
- **Live coverage glow** — Recently hit lines pulse green, then fade
- **Timestamp per line** — See exactly when each line was last executed (e.g. `16:52:30 (3s ago)`)
- **Directory tree** — Collapsible file tree with per-file coverage percentage
- **Goroutine tracking** — See which goroutines executed which code
- **Execution flow** — Bottom panel shows latest block per goroutine with code snippet
- **Coverage stats** — Server-side stmt-level coverage (e.g. `85.2% (176/206 stmts)`)

## Architecture

### Instrumentation

For each basic block in the source, gococo injects:

```go
GococoCov_RAND_FILEIDX[blockIdx]++; GococoEmit_RAND(fileIdx, blockIdx);
```

- Counter arrays (`GococoCov_*`) — Always increment, never lost. Sent as a snapshot at agent startup.
- Event channel — Buffered (8192), non-blocking (`select/default`). Feeds the real-time stream.
- Dot import — Instrumented files use `import . "module/gococodef"` to access counters without prefix.

### Agent

Injected into `main` packages as an `init()` function:

1. **Synchronous registration** — Blocks until server is reachable (10 retries, then exit).
2. **Block metadata** — Sends all block positions so server knows total coverage.
3. **Counter snapshot** — Sent 500ms after startup to capture `init()` and `main()` coverage.
4. **Event streaming** — Chunked HTTP POST with `io.Pipe` + buffered writer. Auto-reconnects.

### Server

- `/api/internal/register` — Agent registration
- `/api/internal/register-blocks` — Block metadata (all blocks, including uncovered)
- `/api/internal/counters` — Counter snapshot (accurate hit counts)
- `/api/internal/events` — Chunked event stream from agent
- `/api/events/stream` — SSE to web UI clients
- `/api/coverage/summary` — Per-file coverage stats
- `/api/coverage/blocks` — Block-level coverage for a file
- `/api/source` — Source code from disk (resolved via go.mod module path)

## CLI Reference

```
gococo server [--addr HOST:PORT] [--root DIR]
    Start the relay server.
    --addr   Listen address (default: 127.0.0.1:7778)
    --root   Source code root for /api/source (default: current directory)

gococo build [--host HOST:PORT] [-o OUTPUT] [BUILD_FLAGS...] [PACKAGES]
    Instrument and build a Go project.
    --host   Server address for the agent to connect to (default: 127.0.0.1:7778)
    -o       Output binary path
    --debug  Keep temp directory for inspection

gococo version
    Show version.
```

Environment variable `GOCOCO_HOST` overrides the server address at runtime.

## Development

```bash
# Run all tests
make test

# Unit tests only (30 tests, 13 testdata fixtures)
go test -v ./internal/instrument/

# E2E tests (instrument, build, run, verify coverage)
go test -v -timeout 120s ./tests/e2e/

# Build
make build

# Web UI development
cd web && npm install && npm run dev
```

## License

MIT
