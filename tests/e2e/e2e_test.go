package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Test harness
// =============================================================================

var gococoBinary string

func TestMain(m *testing.M) {
	// Build gococo binary once for all tests
	tmp, err := os.MkdirTemp("", "gococo-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create tmp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	gococoBinary = filepath.Join(tmp, "gococo")
	projectRoot, _ := filepath.Abs("../..")

	cmd := exec.Command("go", "build", "-o", gococoBinary, "./cmd/gococo/")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build gococo: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// testEnv manages a gococo server + instrumented app for one test.
type testEnv struct {
	t          *testing.T
	serverAddr string
	serverCmd  *exec.Cmd
	appCmd     *exec.Cmd
	appAddr    string
	tmpDir     string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	tmp, err := os.MkdirTemp("", "gococo-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	return &testEnv{t: t, tmpDir: tmp}
}

func (e *testEnv) cleanup() {
	if e.appCmd != nil && e.appCmd.Process != nil {
		e.appCmd.Process.Kill()
		e.appCmd.Wait()
	}
	if e.serverCmd != nil && e.serverCmd.Process != nil {
		e.serverCmd.Process.Kill()
		e.serverCmd.Wait()
	}
	os.RemoveAll(e.tmpDir)
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// startServer starts a gococo server on a random port.
func (e *testEnv) startServer() {
	e.t.Helper()
	e.serverAddr = freePort(e.t)

	e.serverCmd = exec.Command(gococoBinary, "server", "--addr", e.serverAddr)
	e.serverCmd.Stderr = os.Stderr
	if err := e.serverCmd.Start(); err != nil {
		e.t.Fatalf("start server: %v", err)
	}

	// Wait for server to be ready
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://%s/api/agents", e.serverAddr))
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	e.t.Fatal("server did not start in time")
}

// instrumentAndBuild instruments a test project and builds it.
func (e *testEnv) instrumentAndBuild(projectDir string) string {
	e.t.Helper()
	absProject, _ := filepath.Abs(projectDir)
	binaryName := filepath.Base(absProject) + "-instrumented"
	outputPath := filepath.Join(e.tmpDir, binaryName)

	cmd := exec.Command(gococoBinary, "build", "--host", e.serverAddr, "-o", outputPath, ".")
	cmd.Dir = absProject
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		e.t.Fatalf("instrument+build %s: %v", projectDir, err)
	}
	return outputPath
}

// startApp starts the instrumented binary and waits for it to print its listen address.
func (e *testEnv) startApp(binaryPath string) {
	e.t.Helper()
	e.appCmd = exec.Command(binaryPath)
	e.appCmd.Env = append(os.Environ(), "PORT=0")

	stdout, err := e.appCmd.StdoutPipe()
	if err != nil {
		e.t.Fatal(err)
	}
	e.appCmd.Stderr = os.Stderr

	if err := e.appCmd.Start(); err != nil {
		e.t.Fatalf("start app: %v", err)
	}

	// Read until we see "LISTEN <addr>"
	scanner := bufio.NewScanner(stdout)
	done := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "LISTEN ") {
				e.appAddr = strings.TrimPrefix(line, "LISTEN ")
				done <- true
				go io.Copy(io.Discard, stdout)
				return
			}
		}
		done <- false
	}()

	select {
	case ok := <-done:
		if !ok {
			e.t.Fatal("app exited without printing LISTEN address")
		}
	case <-time.After(15 * time.Second):
		e.t.Fatal("app did not print LISTEN address within timeout")
	}
}

// hitEndpoint sends a GET to the app and returns the body.
func (e *testEnv) hitEndpoint(path string) string {
	e.t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://%s%s", e.appAddr, path))
	if err != nil {
		e.t.Fatalf("hit %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// getCoverageSummary fetches coverage data from the server.
func (e *testEnv) getCoverageSummary() coverageSummary {
	e.t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://%s/api/coverage/summary", e.serverAddr))
	if err != nil {
		e.t.Fatalf("get coverage: %v", err)
	}
	defer resp.Body.Close()

	var cs coverageSummary
	json.NewDecoder(resp.Body).Decode(&cs)
	return cs
}

// waitForEvents waits until the server has received at least n events.
func (e *testEnv) waitForEvents(n int, timeout time.Duration) {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			cs := e.getCoverageSummary()
			e.t.Fatalf("timeout waiting for %d events (got %d)", n, cs.TotalEvents)
		default:
			cs := e.getCoverageSummary()
			if cs.TotalEvents >= n {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

type coverageSummary struct {
	Files       []fileCoverage `json:"files"`
	TotalStmts  int            `json:"total_stmts"`
	HitStmts    int            `json:"hit_stmts"`
	OverallPct  float64        `json:"overall_pct"`
	TotalEvents int            `json:"total_events"`
}

type fileCoverage struct {
	File        string  `json:"file"`
	TotalBlocks int     `json:"total_blocks"`
	HitBlocks   int     `json:"hit_blocks"`
	TotalStmts  int     `json:"total_stmts"`
	HitStmts    int     `json:"hit_stmts"`
	Percentage  float64 `json:"percentage"`
}

func (e *testEnv) findFile(cs coverageSummary, suffix string) *fileCoverage {
	for i := range cs.Files {
		if strings.HasSuffix(cs.Files[i].File, suffix) {
			return &cs.Files[i]
		}
	}
	return nil
}

// =============================================================================
// E2E Tests
// =============================================================================

// TestE2E_SingleFile_PartialCoverage verifies that:
// 1. Only exercised branches appear in coverage
// 2. Unexercised code (neverCalled) is NOT covered
// 3. Coverage percentages are correct
func TestE2E_SingleFile_PartialCoverage(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	env.startServer()
	binary := env.instrumentAndBuild("testprojects/singlefile")
	env.startApp(binary)

	// Wait for agent connection
	time.Sleep(2 * time.Second)

	// Only hit branch-a endpoint
	body := env.hitEndpoint("/branch-a")
	if !strings.Contains(body, "a=") {
		t.Errorf("unexpected response: %s", body)
	}

	// Wait for events to be collected
	env.waitForEvents(1, 10*time.Second)
	time.Sleep(1 * time.Second) // let remaining events flush

	cs := env.getCoverageSummary()
	t.Logf("coverage: %.1f%% (%d/%d stmts), %d events",
		cs.OverallPct, cs.HitStmts, cs.TotalStmts, cs.TotalEvents)

	// Must have events
	if cs.TotalEvents == 0 {
		t.Fatal("no events received")
	}

	// Coverage must NOT be 100% — neverCalled and branchB not exercised
	if cs.OverallPct >= 100.0 {
		t.Errorf("expected partial coverage, got %.1f%%", cs.OverallPct)
	}

	// Coverage must be > 0%
	if cs.OverallPct <= 0 {
		t.Errorf("expected some coverage, got %.1f%%", cs.OverallPct)
	}

	// branchA (n>5 path) should be hit
	for _, f := range cs.Files {
		t.Logf("  %s: %d/%d blocks, %d/%d stmts (%.0f%%)",
			f.File, f.HitBlocks, f.TotalBlocks, f.HitStmts, f.TotalStmts, f.Percentage)
	}

	// Now hit branch-b and verify coverage increases
	prevHit := cs.HitStmts
	env.hitEndpoint("/branch-b")
	time.Sleep(1 * time.Second)

	cs2 := env.getCoverageSummary()
	t.Logf("after branch-b: %.1f%% (%d/%d stmts), %d events",
		cs2.OverallPct, cs2.HitStmts, cs2.TotalStmts, cs2.TotalEvents)

	if cs2.HitStmts <= prevHit {
		t.Errorf("expected more stmts hit after exercising branch-b: before=%d after=%d",
			prevHit, cs2.HitStmts)
	}

	// Still should not be 100% — neverCalled is unreachable
	if cs2.OverallPct >= 100.0 {
		t.Errorf("expected <100%% coverage (neverCalled exists), got %.1f%%", cs2.OverallPct)
	}
}

// TestE2E_MultiPackage verifies cross-package coverage collection.
func TestE2E_MultiPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	env.startServer()
	binary := env.instrumentAndBuild("testprojects/multipkg")
	env.startApp(binary)

	time.Sleep(2 * time.Second)

	// Hit /add → exercises calc.Add
	body := env.hitEndpoint("/add")
	if body != "7" {
		t.Errorf("expected 7, got %q", body)
	}

	env.waitForEvents(1, 10*time.Second)
	time.Sleep(1 * time.Second)

	cs := env.getCoverageSummary()
	t.Logf("after /add: %.1f%% (%d/%d stmts), %d events, %d files",
		cs.OverallPct, cs.HitStmts, cs.TotalStmts, cs.TotalEvents, len(cs.Files))

	// Should have coverage in at least 2 files (main.go and calc/calc.go)
	if len(cs.Files) < 2 {
		t.Errorf("expected coverage in at least 2 files, got %d", len(cs.Files))
	}

	// calc.go should have partial coverage (Add hit, Sub/Multiply/Divide not)
	calcFile := env.findFile(cs, "calc.go")
	if calcFile != nil {
		t.Logf("calc.go: %d/%d blocks hit", calcFile.HitBlocks, calcFile.TotalBlocks)
		if calcFile.HitBlocks >= calcFile.TotalBlocks {
			t.Errorf("calc.go should have partial coverage (Sub/Multiply/Divide not called)")
		}
		if calcFile.HitBlocks == 0 {
			t.Errorf("calc.go should have some coverage (Add was called)")
		}
	}

	// greeting.go should have NO coverage yet
	greetFile := env.findFile(cs, "greeting.go")
	if greetFile != nil && greetFile.HitBlocks > 0 {
		t.Errorf("greeting.go should not be covered yet, but has %d hit blocks", greetFile.HitBlocks)
	}

	// Now hit /greet → exercises greeting.Hello
	body = env.hitEndpoint("/greet")
	if body != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", body)
	}
	time.Sleep(1 * time.Second)

	cs2 := env.getCoverageSummary()
	greetFile2 := env.findFile(cs2, "greeting.go")
	if greetFile2 == nil || greetFile2.HitBlocks == 0 {
		t.Errorf("greeting.go should now have coverage after /greet")
	}

	// Goodbye should still be uncovered
	if greetFile2 != nil {
		t.Logf("greeting.go: %d/%d blocks hit", greetFile2.HitBlocks, greetFile2.TotalBlocks)
		if greetFile2.Percentage >= 100.0 {
			t.Errorf("greeting.go should not be 100%% (Goodbye not called)")
		}
	}

	for _, f := range cs2.Files {
		t.Logf("  %s: %.0f%%", f.File, f.Percentage)
	}
}

// TestE2E_InstrumentAndBuild_Compiles tests that instrumentation doesn't break compilation
// for various project structures.
func TestE2E_InstrumentAndBuild_Compiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in short mode")
	}

	projects := []string{
		"testprojects/singlefile",
		"testprojects/multipkg",
	}

	env := newTestEnv(t)
	defer env.cleanup()
	env.startServer()

	for _, proj := range projects {
		t.Run(filepath.Base(proj), func(t *testing.T) {
			binary := env.instrumentAndBuild(proj)
			// Verify binary exists and is executable
			info, err := os.Stat(binary)
			if err != nil {
				t.Fatalf("binary not found: %v", err)
			}
			if info.Size() == 0 {
				t.Fatal("binary is empty")
			}
			t.Logf("built %s: %d bytes", filepath.Base(binary), info.Size())
		})
	}
}
