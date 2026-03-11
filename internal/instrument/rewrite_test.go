package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Test helpers
// =============================================================================

const testRandomID = "test123"

// instrumentTestFile instruments a file from testdata/ and returns the result.
func instrumentTestFile(t *testing.T, filename string) ([]byte, *FileInstrumentation) {
	t.Helper()
	path := filepath.Join("testdata", filename)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	rewritten, inst, err := InstrumentFile(src, path, "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatalf("instrument %s: %v", path, err)
	}
	return rewritten, inst
}

// assertParseable checks that the instrumented source is valid Go.
func assertParseable(t *testing.T, filename string, src []byte) {
	t.Helper()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		// Show first few lines around the error for context
		lines := strings.Split(string(src), "\n")
		start := 0
		end := len(lines)
		if end > 30 {
			end = 30
		}
		for i := start; i < end; i++ {
			t.Logf("%3d: %s", i+1, lines[i])
		}
		t.Fatalf("instrumented %s is not valid Go:\n%v", filename, err)
	}
}

// assertBlockPositionsInOriginalRange checks that all reported block positions
// fall within the original source file line range.
func assertBlockPositionsInOriginalRange(t *testing.T, filename string, origSrc []byte, inst *FileInstrumentation) {
	t.Helper()
	origLines := strings.Count(string(origSrc), "\n") + 1

	for i, b := range inst.Blocks {
		if b.StartLine < 1 || b.StartLine > origLines {
			t.Errorf("block %d: StartLine %d out of range [1, %d] in %s", i, b.StartLine, origLines, filename)
		}
		if b.EndLine < 1 || b.EndLine > origLines {
			t.Errorf("block %d: EndLine %d out of range [1, %d] in %s", i, b.EndLine, origLines, filename)
		}
		if b.StartLine > b.EndLine {
			t.Errorf("block %d: StartLine %d > EndLine %d in %s", i, b.StartLine, b.EndLine, filename)
		}
		if b.StartLine == b.EndLine && b.StartCol > b.EndCol {
			t.Errorf("block %d: on same line but StartCol %d > EndCol %d in %s", i, b.StartCol, b.EndCol, filename)
		}
		if b.NumStmts < 0 {
			t.Errorf("block %d: negative NumStmts %d in %s", i, b.NumStmts, filename)
		}
	}
}

// assertNoOverlappingBlocks checks that no two blocks overlap.
func assertNoOverlappingBlocks(t *testing.T, filename string, inst *FileInstrumentation) {
	t.Helper()
	for i := 0; i < len(inst.Blocks); i++ {
		for j := i + 1; j < len(inst.Blocks); j++ {
			bi, bj := inst.Blocks[i], inst.Blocks[j]
			// Convert to linear positions for comparison
			iStart := bi.StartLine*10000 + bi.StartCol
			iEnd := bi.EndLine*10000 + bi.EndCol
			jStart := bj.StartLine*10000 + bj.StartCol
			jEnd := bj.EndLine*10000 + bj.EndCol

			if iStart < jEnd && jStart < iEnd {
				// Overlapping is expected for nested blocks (e.g. a for loop inside
				// a function). The Go cover tool allows this. We only check for
				// identical blocks.
				if iStart == jStart && iEnd == jEnd {
					t.Errorf("blocks %d and %d are identical: [%d:%d - %d:%d] in %s",
						i, j, bi.StartLine, bi.StartCol, bi.EndLine, bi.EndCol, filename)
				}
			}
		}
	}
}

// assertCounterInjected checks that counter statements appear in the output.
func assertCounterInjected(t *testing.T, src []byte, expectedBlocks int) {
	t.Helper()
	content := string(src)
	counterPattern := fmt.Sprintf("GococoCov_%s_", testRandomID)
	emitPattern := fmt.Sprintf("GococoEmit_%s(", testRandomID)

	counterCount := strings.Count(content, counterPattern)
	emitCount := strings.Count(content, emitPattern)

	if counterCount != expectedBlocks {
		t.Errorf("expected %d counter injections, got %d", expectedBlocks, counterCount)
	}
	if emitCount != expectedBlocks {
		t.Errorf("expected %d emit injections, got %d", expectedBlocks, emitCount)
	}
}

// =============================================================================
// Layer 1: Instrumented code is valid Go + block positions correct
// =============================================================================

func TestInstrument_AllTestdataFiles(t *testing.T) {
	files, err := filepath.Glob("testdata/*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no testdata files found")
	}

	for _, f := range files {
		filename := filepath.Base(f)
		t.Run(filename, func(t *testing.T) {
			origSrc, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			rewritten, inst, err := InstrumentFile(origSrc, f, "test/pkg", "GoCov_0", testRandomID, 0)
			if err != nil {
				t.Fatalf("instrument failed: %v", err)
			}

			// 1. Instrumented code must be valid Go
			assertParseable(t, filename, rewritten)

			// 2. Block positions must be in original source range
			assertBlockPositionsInOriginalRange(t, filename, origSrc, inst)

			// 3. No duplicate blocks
			assertNoOverlappingBlocks(t, filename, inst)

			// 4. Every block should have counter+emit injected
			if len(inst.Blocks) > 0 {
				assertCounterInjected(t, rewritten, len(inst.Blocks))
			}

			t.Logf("%s: %d blocks", filename, len(inst.Blocks))
		})
	}
}

// =============================================================================
// Layer 2: Position preservation — block metadata matches original source
// =============================================================================

func TestPositionPreservation_Basic(t *testing.T) {
	src := []byte(`package main

func main() {
	x := 1
	if x > 0 {
		x++
	}
}
`)
	rewritten, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}

	assertParseable(t, "main.go", rewritten)

	// There should be 3 blocks:
	// Block 0: main body start (lines 4-5 area, before the if)
	// Block 1: if-true body (line 6)
	// Block 2: after-if (empty or continuation)
	if len(inst.Blocks) < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", len(inst.Blocks))
	}

	// The if-true block must start on line 6 (where x++ is)
	// and must refer to the ORIGINAL source positions
	found := false
	for _, b := range inst.Blocks {
		if b.StartLine == 6 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a block starting at line 6 (the x++ inside if)")
		for i, b := range inst.Blocks {
			t.Logf("  block %d: L%d:%d - L%d:%d (%d stmts)", i, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmts)
		}
	}
}

func TestPositionPreservation_IfElseChain(t *testing.T) {
	src := []byte(`package main

func classify(x int) string {
	if x > 100 {
		return "big"
	} else if x > 10 {
		return "medium"
	} else {
		return "small"
	}
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Should have blocks for: function body start, if-true (L5), else-if-true (L7), else (L9)
	if len(inst.Blocks) < 4 {
		t.Errorf("expected at least 4 blocks for if-else-if-else, got %d", len(inst.Blocks))
	}

	// Verify specific blocks exist at expected lines
	expectedLines := map[int]bool{5: false, 7: false, 9: false}
	for _, b := range inst.Blocks {
		if _, ok := expectedLines[b.StartLine]; ok {
			expectedLines[b.StartLine] = true
		}
	}
	for line, found := range expectedLines {
		if !found {
			t.Errorf("no block starts at line %d", line)
		}
	}
}

func TestPositionPreservation_Switch(t *testing.T) {
	src := []byte(`package main

func check(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Each case clause gets a block
	if len(inst.Blocks) < 3 {
		t.Errorf("expected at least 3 blocks (one per case), got %d", len(inst.Blocks))
	}
}

func TestPositionPreservation_ForRange(t *testing.T) {
	src := []byte(`package main

func sum(s []int) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Block for function body (before for), block for for-body
	if len(inst.Blocks) < 2 {
		t.Errorf("expected at least 2 blocks, got %d", len(inst.Blocks))
	}

	// The for body block should start at line 6 (total += v)
	found := false
	for _, b := range inst.Blocks {
		if b.StartLine == 6 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected block at line 6 (for body)")
		for i, b := range inst.Blocks {
			t.Logf("  block %d: L%d:%d - L%d:%d", i, b.StartLine, b.StartCol, b.EndLine, b.EndCol)
		}
	}
}

// =============================================================================
// Layer 3: Block count expectations for specific patterns
// =============================================================================

func TestBlockCount_EmptyFunc(t *testing.T) {
	src := []byte(`package main
func empty() {}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Empty function body should still get one block (with 0 stmts)
	if len(inst.Blocks) != 1 {
		t.Errorf("expected 1 block for empty func, got %d", len(inst.Blocks))
	}
	if len(inst.Blocks) > 0 && inst.Blocks[0].NumStmts != 0 {
		t.Errorf("empty func block should have 0 stmts, got %d", inst.Blocks[0].NumStmts)
	}
}

func TestBlockCount_SimpleSequential(t *testing.T) {
	src := []byte(`package main
func f() {
	a := 1
	b := 2
	_ = a + b
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// All sequential → one block, 3 statements
	if len(inst.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(inst.Blocks))
	}
	if len(inst.Blocks) > 0 && inst.Blocks[0].NumStmts != 3 {
		t.Errorf("expected 3 stmts, got %d", inst.Blocks[0].NumStmts)
	}
}

func TestBlockCount_IfElse(t *testing.T) {
	src := []byte(`package main
func f(x int) int {
	if x > 0 {
		return x
	} else {
		return -x
	}
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Blocks: 1(before if) + 1(if-true) + 1(else) = 3
	if len(inst.Blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(inst.Blocks))
		for i, b := range inst.Blocks {
			t.Logf("  block %d: L%d:%d - L%d:%d (%d stmts)", i, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmts)
		}
	}
}

func TestBlockCount_SelectCases(t *testing.T) {
	src := []byte(`package main
func f(a, b <-chan int) int {
	select {
	case v := <-a:
		return v
	case v := <-b:
		return v
	}
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Blocks: 1(func body before select) + 2(case clauses) = 3
	if len(inst.Blocks) < 3 {
		t.Errorf("expected at least 3 blocks, got %d", len(inst.Blocks))
	}
}

func TestBlockCount_EmptySwitch(t *testing.T) {
	src := []byte(`package main
func f(x int) {
	switch x {
	}
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Empty switch: just the function body block
	if len(inst.Blocks) != 1 {
		t.Errorf("expected 1 block for empty switch, got %d", len(inst.Blocks))
	}
}

func TestBlockCount_ForBreakContinue(t *testing.T) {
	src := []byte(`package main
func f(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		if i == 3 {
			continue
		}
		if i == 7 {
			break
		}
		sum += i
	}
	return sum
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Multiple blocks from break/continue
	if len(inst.Blocks) < 6 {
		t.Errorf("expected at least 6 blocks, got %d", len(inst.Blocks))
		for i, b := range inst.Blocks {
			t.Logf("  block %d: L%d:%d - L%d:%d (%d stmts)", i, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmts)
		}
	}
}

func TestBlockCount_Closure(t *testing.T) {
	src := []byte(`package main
func f() int {
	x := func() int {
		return 42
	}()
	return x
}
`)
	rewritten, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertParseable(t, "main.go", rewritten)
	// Blocks: outer func body (before func lit boundary) + closure body + outer after closure
	if len(inst.Blocks) < 2 {
		t.Errorf("expected at least 2 blocks, got %d", len(inst.Blocks))
	}
}

func TestBlockCount_Panic(t *testing.T) {
	src := []byte(`package main
func f(x int) int {
	if x < 0 {
		panic("negative")
	}
	return x
}
`)
	_, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}
	// panic breaks the block → blocks for: before-if, if-true(panic), after-if
	if len(inst.Blocks) < 3 {
		t.Errorf("expected at least 3 blocks (panic breaks block), got %d", len(inst.Blocks))
	}
}

// =============================================================================
// Layer 4: Global coverage variable declaration
// =============================================================================

func TestBuildGlobalCoverVarDecl(t *testing.T) {
	files := []*FileInstrumentation{
		{
			VarName:  "GoCov_0",
			FilePath: "test/pkg/main.go",
			Blocks: []BlockInfo{
				{StartLine: 3, StartCol: 2, EndLine: 5, EndCol: 2, NumStmts: 2},
				{StartLine: 6, StartCol: 3, EndLine: 7, EndCol: 2, NumStmts: 1},
			},
		},
		{
			VarName:  "GoCov_1",
			FilePath: "test/pkg/util.go",
			Blocks: []BlockInfo{
				{StartLine: 10, StartCol: 2, EndLine: 12, EndCol: 2, NumStmts: 3},
			},
		},
	}

	decl := BuildGlobalCoverVarDecl(files, testRandomID)

	// Must be valid Go
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "coverdef.go", decl, parser.AllErrors)
	if err != nil {
		t.Logf("Generated code:\n%s", decl)
		t.Fatalf("generated coverdef is not valid Go: %v", err)
	}

	// Must contain expected symbols
	expectedSymbols := []string{
		"GococoBlock_" + testRandomID,
		"GococoEmit_" + testRandomID,
		"GococoCov_" + testRandomID,
		"SetEnabled_" + testRandomID,
		"EventChan_" + testRandomID,
		"BlockMeta_" + testRandomID,
	}
	for _, sym := range expectedSymbols {
		if !strings.Contains(decl, sym) {
			t.Errorf("missing symbol %q in generated coverdef", sym)
		}
	}

	// Check metadata values
	if !strings.Contains(decl, `"test/pkg/main.go"`) {
		t.Error("missing file path in metadata")
	}
	if !strings.Contains(decl, `"test/pkg/util.go"`) {
		t.Error("missing file path in metadata")
	}
}

// =============================================================================
// Layer 5: Instrumented source preserves original AST structure
// =============================================================================

// TestOriginalFunctionsPreserved verifies that all functions in the original
// source still exist in the instrumented output.
func TestOriginalFunctionsPreserved(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.go")
	for _, f := range files {
		filename := filepath.Base(f)
		t.Run(filename, func(t *testing.T) {
			origSrc, _ := os.ReadFile(f)
			fset := token.NewFileSet()
			origAST, err := parser.ParseFile(fset, f, origSrc, parser.ParseComments)
			if err != nil {
				t.Skipf("skip unparseable: %v", err)
			}

			// Collect original function names
			origFuncs := make(map[string]bool)
			for _, decl := range origAST.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					origFuncs[fn.Name.Name] = true
				}
			}

			// Instrument
			rewritten, _, err := InstrumentFile(origSrc, f, "test/pkg", "GoCov_0", testRandomID, 0)
			if err != nil {
				t.Fatal(err)
			}

			// Parse instrumented
			fset2 := token.NewFileSet()
			newAST, err := parser.ParseFile(fset2, f, rewritten, parser.AllErrors)
			if err != nil {
				t.Fatalf("instrumented code not parseable: %v", err)
			}

			// Check all original functions are still present
			newFuncs := make(map[string]bool)
			for _, decl := range newAST.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					newFuncs[fn.Name.Name] = true
				}
			}

			for name := range origFuncs {
				if !newFuncs[name] {
					t.Errorf("function %q disappeared after instrumentation", name)
				}
			}
		})
	}
}

// TestInstrumentedCodeCompiles verifies that adding the import and line directive
// produces code that can at least be parsed (compilation requires the full project).
func TestInstrumentedCodeCompiles(t *testing.T) {
	src := []byte(`package main

import "fmt"

func main() {
	fmt.Println("hello")
	if true {
		fmt.Println("branch")
	}
}
`)
	rewritten, inst, err := InstrumentFile(src, "main.go", "test/pkg", "GoCov_0", testRandomID, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(inst.Blocks) == 0 {
		t.Fatal("expected blocks")
	}

	// Add import + line directive like instrument.go does
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "main.go", rewritten, parser.ParseComments)
	rewritten = addImport(rewritten, fset, f, "test/pkg/gococodef", ".")
	rewritten = append([]byte(lineDirective("main.go")), rewritten...)

	// Must still be parseable
	fset2 := token.NewFileSet()
	_, err = parser.ParseFile(fset2, "main.go", rewritten, parser.AllErrors)
	if err != nil {
		t.Logf("Source:\n%s", string(rewritten))
		t.Fatalf("final instrumented code is not parseable: %v", err)
	}
}
