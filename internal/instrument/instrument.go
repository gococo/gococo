// Package instrument implements the core build-time instrumentation for gococo.
//
// It copies a Go project to a temp directory, rewrites all source files to inject
// coverage counters and event emitters, injects a runtime agent, and builds the
// modified project.
package instrument

import (
	"crypto/sha256"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Options configures the instrumentation.
type Options struct {
	Host      string   // gococo server address (e.g. "127.0.0.1:7778")
	Packages  []string // packages to build (e.g. "." or "./cmd/myapp")
	GoFlags   []string // additional flags to pass to `go build`
	OutputDir string   // where to place the built binary (-o)
	Debug     bool
}

// Run performs the full instrument-and-build pipeline.
func Run(opts Options) error {
	// 1. Determine project root and module info
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	modPath, modDir, err := findModuleInfo(wd)
	if err != nil {
		return fmt.Errorf("find module: %w", err)
	}

	fmt.Printf("[gococo] module: %s at %s\n", modPath, modDir)

	// 2. List packages
	patterns := opts.Packages
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	pkgs, err := ListPackages(wd, patterns)
	if err != nil {
		return fmt.Errorf("list packages: %w", err)
	}

	// 3. Create temp directory
	tmpDir, err := os.MkdirTemp("", "gococo-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if !opts.Debug {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Printf("[gococo] temp dir: %s\n", tmpDir)
	}

	randomID := filepath.Base(tmpDir)
	// Sanitize: replace hyphens for valid Go identifiers
	randomID = strings.ReplaceAll(randomID, "-", "_")

	// 4. Copy project to temp
	tmpProject := filepath.Join(tmpDir, filepath.Base(modDir))
	if err := copyDir(modDir, tmpProject); err != nil {
		return fmt.Errorf("copy project: %w", err)
	}
	fmt.Println("[gococo] project copied to temp directory")

	// 5. Create the global coverage definition package
	coverDefPkgName := "gococodef"
	coverDefImportPath := modPath + "/" + coverDefPkgName
	coverDefDir := filepath.Join(tmpProject, coverDefPkgName)
	if err := os.MkdirAll(coverDefDir, 0o755); err != nil {
		return fmt.Errorf("mkdir coverdef: %w", err)
	}

	// 6. Instrument all project source files
	var allInstrumentations []*FileInstrumentation
	fileIdx := 0

	mains := FindMainPackages(pkgs)
	// Collect all project packages (main + deps)
	projectPkgs := make(map[string]*Package)
	for _, mp := range mains {
		projectPkgs[mp.ImportPath] = mp
		for _, dep := range mp.Deps {
			if p, ok := pkgs[dep]; ok && IsProjectPackage(p, modPath) {
				projectPkgs[p.ImportPath] = p
			}
		}
	}

	for _, pkg := range projectPkgs {
		pkgTmpDir := translateDir(pkg.Dir, modDir, tmpProject)
		allFiles := append(pkg.GoFiles, pkg.CgoFiles...)

		for _, goFile := range allFiles {
			filePath := filepath.Join(pkgTmpDir, goFile)
			src, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read %s: %w", filePath, err)
			}

			varName := GenerateCoverVarName(pkg.ImportPath, fileIdx)
			rewritten, inst, err := InstrumentFile(src, filePath, pkg.ImportPath, varName, randomID, fileIdx)
			if err != nil {
				return fmt.Errorf("instrument %s: %w", filePath, err)
			}

			if len(inst.Blocks) > 0 {
				// Add import for the coverdef package (dot import so counters are accessible)
				fset := token.NewFileSet()
				f, _ := parser.ParseFile(fset, filePath, rewritten, parser.ParseComments)
				rewritten = addImport(rewritten, fset, f, coverDefImportPath, ".")

				// Add line directive
				rewritten = append([]byte(lineDirective(filePath)), rewritten...)
			}

			if err := os.WriteFile(filePath, rewritten, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", filePath, err)
			}

			allInstrumentations = append(allInstrumentations, inst)
			fileIdx++
		}
	}
	fmt.Printf("[gococo] instrumented %d files (%d blocks total)\n", fileIdx, countBlocks(allInstrumentations))

	// 7. Write global coverage variable file
	coverSrc := BuildGlobalCoverVarDecl(allInstrumentations, randomID)
	if err := os.WriteFile(filepath.Join(coverDefDir, "coverdef.go"), []byte(coverSrc), 0o644); err != nil {
		return fmt.Errorf("write coverdef: %w", err)
	}

	// 8. Inject agent into each main package
	for _, mp := range mains {
		mainTmpDir := translateDir(mp.Dir, modDir, tmpProject)
		if err := injectAgent(mainTmpDir, mp.ImportPath, coverDefImportPath, randomID, opts.Host, allInstrumentations); err != nil {
			return fmt.Errorf("inject agent: %w", err)
		}
		fmt.Printf("[gococo] injected agent into %s\n", mp.ImportPath)
	}

	// 9. Build the instrumented project
	return buildProject(tmpProject, wd, opts)
}

func injectAgent(mainDir string, mainImportPath string, coverDefImportPath string, randomID string, host string, files []*FileInstrumentation) error {
	agentPkgName := "gococo_agent_" + randomID
	agentDir := filepath.Join(mainDir, agentPkgName)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	// Write bridge file in main package
	bridgeTmpl := template.Must(template.New("bridge").Parse(bridgeTemplate))
	bridgePath := filepath.Join(mainDir, "gococo_bridge_"+randomID+".go")
	bf, err := os.Create(bridgePath)
	if err != nil {
		return err
	}
	defer bf.Close()

	agentImportPath := mainImportPath + "/" + agentPkgName
	if err := bridgeTmpl.Execute(bf, map[string]string{
		"AgentImportPath": agentImportPath,
	}); err != nil {
		return err
	}

	// Build file metadata for template
	type fileMeta struct {
		FileIdx    int
		BlockCount int
	}
	var metas []fileMeta
	for i, fi := range files {
		if len(fi.Blocks) > 0 {
			metas = append(metas, fileMeta{FileIdx: i, BlockCount: len(fi.Blocks)})
		}
	}

	// Write agent file
	agentTmpl := template.Must(template.New("agent").Parse(agentTemplate))
	agentPath := filepath.Join(agentDir, "agent.go")
	af, err := os.Create(agentPath)
	if err != nil {
		return err
	}
	defer af.Close()

	return agentTmpl.Execute(af, map[string]interface{}{
		"PackageName":        agentPkgName,
		"CoverDefImportPath": coverDefImportPath,
		"Host":               host,
		"RandomID":           randomID,
		"FileMetas":          metas,
	})
}

func buildProject(tmpProject string, originalWd string, opts Options) error {
	goflags := make([]string, len(opts.GoFlags))
	copy(goflags, opts.GoFlags)

	hasOutput := false
	for _, f := range goflags {
		if f == "-o" {
			hasOutput = true
			break
		}
	}
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = originalWd
	}
	if !hasOutput {
		goflags = append(goflags, "-o", outputDir)
	}

	packages := opts.Packages
	if len(packages) == 0 {
		packages = []string{"."}
	}
	goflags = append(goflags, packages...)

	args := append([]string{"build"}, goflags...)
	cmd := exec.Command("go", args...)
	cmd.Dir = tmpProject
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[gococo] go build %s\n", strings.Join(args[1:], " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Println("[gococo] build complete")
	return nil
}

func findModuleInfo(dir string) (modPath string, modDir string, err error) {
	cmd := exec.Command("go", "list", "-m", "-json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("go list -m: %w", err)
	}

	// Simple parsing: find Path and Dir fields
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"Path":`) {
			modPath = strings.Trim(strings.TrimPrefix(line, `"Path":`), ` ",`)
		}
		if strings.HasPrefix(line, `"Dir":`) {
			modDir = strings.Trim(strings.TrimPrefix(line, `"Dir":`), ` ",`)
		}
	}
	if modPath == "" || modDir == "" {
		return "", "", fmt.Errorf("could not determine module path/dir from go list -m output")
	}
	return modPath, modDir, nil
}

func translateDir(pkgDir, origRoot, tmpRoot string) string {
	rel, err := filepath.Rel(origRoot, pkgDir)
	if err != nil {
		return pkgDir
	}
	return filepath.Join(tmpRoot, rel)
}

func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-a", src, dst)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func countBlocks(files []*FileInstrumentation) int {
	n := 0
	for _, f := range files {
		n += len(f.Blocks)
	}
	return n
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:8])
}

// Embedded templates
const bridgeTemplate = `// Code generated by gococo. DO NOT EDIT.
package main

import _ "{{.AgentImportPath}}"
`

const agentTemplate = `// Code generated by gococo. DO NOT EDIT.
package {{.PackageName}}

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	_cov "{{.CoverDefImportPath}}"
)

func init() {
	host := "{{.Host}}"
	if env := os.Getenv("GOCOCO_HOST"); env != "" {
		host = env
	}

	// Synchronous registration: block until connected or fail fast.
	agentID := registerAgent(host)
	registerBlocks(host, agentID)
	log.Printf("[gococo] agent ready, streaming events")

	// Start async event streaming and counter snapshot.
	go runStreaming(host, agentID)
}

func registerAgent(host string) string {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	cmdline := strings.Join(os.Args, " ")

	v := url.Values{}
	v.Set("hostname", hostname)
	v.Set("pid", fmt.Sprintf("%d", pid))
	v.Set("cmdline", cmdline)

	const maxRetries = 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/api/internal/register?%s", host, v.Encode()))
		if err != nil {
			log.Printf("[gococo] register failed (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(1 * time.Second)
			continue
		}
		buf := make([]byte, 256)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		if resp.StatusCode == 200 {
			agentID := strings.TrimSpace(string(buf[:n]))
			log.Printf("[gococo] registered as agent %s", agentID)
			return agentID
		}
		log.Printf("[gococo] register returned %d (attempt %d/%d)", resp.StatusCode, i+1, maxRetries)
		time.Sleep(1 * time.Second)
	}

	fmt.Fprintf(os.Stderr, "[gococo] fatal: could not connect to server at %s after %d attempts\n", host, maxRetries)
	os.Exit(1)
	return ""
}

func runStreaming(host string, agentID string) {
	// Wait briefly for main() and other init() to finish startup,
	// then send a counter snapshot to capture their coverage.
	time.Sleep(500 * time.Millisecond)
	sendCounterSnapshot(host, agentID)

	for {
		err := streamEvents(host, agentID)
		if err != nil {
			log.Printf("[gococo] stream error: %v, reconnecting...", err)
		}
		_cov.SetEnabled_{{.RandomID}}(false)
		time.Sleep(2 * time.Second)
		_cov.SetEnabled_{{.RandomID}}(true)
	}
}

func registerBlocks(host string, agentID string) {
	var sb strings.Builder
	{{- range .FileMetas}}
	for bi := 0; bi < {{.BlockCount}}; bi++ {
		file, sl, sc, el, ec, stmts := _cov.BlockMeta_{{$.RandomID}}({{.FileIdx}}, bi)
		fmt.Fprintf(&sb, "%s|%d|%d|%d|%d|%d|%d\n", file, bi, sl, sc, el, ec, stmts)
	}
	{{- end}}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/internal/register-blocks?agent_id=%s", host, agentID),
		"text/plain",
		strings.NewReader(sb.String()))
	if err != nil {
		log.Printf("[gococo] register blocks failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("[gococo] registered block metadata with server")
}

func sendCounterSnapshot(host string, agentID string) {
	entries := _cov.CounterSnapshot_{{.RandomID}}()
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "%s|%d|%d|%d|%d|%d|%d|%d\n",
			e.File, e.BlockIdx, e.Count, e.SL, e.SC, e.EL, e.EC, e.Stmts)
	}
	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/internal/counters?agent_id=%s", host, agentID),
		"text/plain",
		strings.NewReader(sb.String()))
	if err != nil {
		log.Printf("[gococo] send counter snapshot failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("[gococo] sent counter snapshot (%d blocks)", len(entries))
}

func streamEvents(host string, agentID string) error {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		bw := bufio.NewWriter(pw)
		var seq uint64
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case block := <-_cov.EventChan_{{.RandomID}}():
				if block == nil {
					return
				}
				seq++
				gid := getGoroutineID()
				ts := time.Now().UnixNano()
				fi := block.FileIdx
				bi := block.BlockIdx
				file, sl, sc, el, ec, stmts := _cov.BlockMeta_{{.RandomID}}(fi, bi)
				fmt.Fprintf(bw, "%d|%d|%d|%s|%d|%d|%d|%d|%d|%d\n",
					seq, ts, gid, file, bi, sl, sc, el, ec, stmts)
			case <-ticker.C:
				bw.Flush()
			}
		}
	}()

	req, err := http.NewRequest("POST",
		fmt.Sprintf("http://%s/api/internal/events?agent_id=%s", host, agentID), pr)
	if err != nil {
		pw.Close()
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Transfer-Encoding", "chunked")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		pw.Close()
		return err
	}
	defer resp.Body.Close()
	return fmt.Errorf("server closed connection: %d", resp.StatusCode)
}

func getGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := string(buf[:n])
	s = s[len("goroutine "):]
	var id int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + int64(c-'0')
	}
	return id
}
`
