package instrument

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// BlockInfo describes a single instrumented basic block.
type BlockInfo struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	NumStmts  int
}

// FileInstrumentation holds the result of instrumenting a single file.
type FileInstrumentation struct {
	VarName  string      // e.g. "gococo_0_a1b2c3"
	FilePath string      // import path + filename
	Blocks   []BlockInfo // all instrumented blocks
}

// InstrumentFile rewrites a Go source file to inject coverage counters.
// It returns the rewritten source and the instrumentation metadata.
//
// For each basic block, it injects:
//
//	gococo_cov.Count_RAND[i]++; gococo_cov.Emit_RAND(fileIdx, i)
//
// where RAND is a unique identifier derived from the temp directory name.
func InstrumentFile(src []byte, filename string, importPath string, varName string, randomID string, fileIdx int) ([]byte, *FileInstrumentation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	longName := path.Join(importPath, path.Base(filename))
	inst := &FileInstrumentation{
		VarName:  varName,
		FilePath: longName,
	}

	rw := &rewriter{
		fset:     fset,
		src:      src,
		blocks:   nil,
		varName:  varName,
		randomID: randomID,
		fileIdx:  fileIdx,
	}

	ast.Walk(rw, f)

	if len(rw.blocks) == 0 {
		return src, inst, nil
	}

	inst.Blocks = rw.blocks

	// Now apply edits: we re-parse and use AST manipulation.
	// Actually, we use a simpler approach: byte-level editing via offset tracking.
	// But for correctness with Go syntax, we use the printer after AST modification.
	//
	// Since direct AST insertion is complex, we use a buffer-based approach:
	// record insertion points and splice them in reverse order.
	edited := applyInsertions(src, rw.insertions)

	// Prepend the import for the coverage variable package.
	// We'll add this at the caller level (instrument.go) to avoid double imports.

	return edited, inst, nil
}

// insertion represents a text insertion at a byte offset.
type insertion struct {
	offset int
	text   string
}

// rewriter walks the AST and records where to insert counter statements.
type rewriter struct {
	fset       *token.FileSet
	src        []byte
	blocks     []BlockInfo
	insertions []insertion
	varName    string
	randomID   string
	fileIdx    int
}

func (rw *rewriter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockStmt:
		if len(n.List) > 0 {
			switch n.List[0].(type) {
			case *ast.CaseClause:
				for _, s := range n.List {
					clause := s.(*ast.CaseClause)
					rw.instrumentBlock(clause.Colon+1, clause.End(), clause.Body, false)
				}
				return nil
			case *ast.CommClause:
				for _, s := range n.List {
					clause := s.(*ast.CommClause)
					rw.instrumentBlock(clause.Colon+1, clause.End(), clause.Body, false)
				}
				return nil
			}
		}
		rw.instrumentBlock(n.Lbrace+1, n.Rbrace+1, n.List, true)

	case *ast.IfStmt:
		if n.Init != nil {
			ast.Walk(rw, n.Init)
		}
		ast.Walk(rw, n.Cond)
		ast.Walk(rw, n.Body)
		if n.Else != nil {
			ast.Walk(rw, n.Else)
		}
		return nil

	case *ast.SelectStmt:
		if n.Body == nil || len(n.Body.List) == 0 {
			return nil
		}

	case *ast.SwitchStmt:
		if n.Body == nil || len(n.Body.List) == 0 {
			if n.Init != nil {
				ast.Walk(rw, n.Init)
			}
			if n.Tag != nil {
				ast.Walk(rw, n.Tag)
			}
			return nil
		}

	case *ast.TypeSwitchStmt:
		if n.Body == nil || len(n.Body.List) == 0 {
			if n.Init != nil {
				ast.Walk(rw, n.Init)
			}
			ast.Walk(rw, n.Assign)
			return nil
		}
	}
	return rw
}

func (rw *rewriter) instrumentBlock(insertPos token.Pos, blockEnd token.Pos, stmts []ast.Stmt, extendToEnd bool) {
	if len(stmts) == 0 {
		rw.addCounter(insertPos, blockEnd, insertPos, 0)
		return
	}

	pos := stmts[0].Pos()
	for i := 0; i < len(stmts); {
		end := blockEnd
		j := i
		for ; j < len(stmts); j++ {
			s := stmts[j]
			if rw.breaksBlock(s) {
				end = rw.stmtBoundary(s)
				j++
				extendToEnd = false
				break
			}
			end = s.End()
		}
		if extendToEnd {
			end = blockEnd
		}

		ipos := pos
		if i == 0 {
			ipos = insertPos
		}
		rw.addCounter(pos, end, ipos, j-i)

		stmts = stmts[j:]
		i = 0
		if len(stmts) > 0 {
			pos = stmts[0].Pos()
		}
	}
}

func (rw *rewriter) addCounter(start, end, insertAt token.Pos, numStmts int) {
	idx := len(rw.blocks)
	startPos := rw.fset.Position(start)
	endPos := rw.fset.Position(end)

	rw.blocks = append(rw.blocks, BlockInfo{
		StartLine: startPos.Line,
		StartCol:  startPos.Column,
		EndLine:   endPos.Line,
		EndCol:    endPos.Column,
		NumStmts:  numStmts,
	})

	counter := fmt.Sprintf("GococoCov_%s[%d]++; GococoEmit_%s(%d, %d);",
		rw.randomID, idx, rw.randomID, rw.fileIdx, idx)

	offset := rw.fset.Position(insertAt).Offset
	rw.insertions = append(rw.insertions, insertion{offset: offset, text: counter})
}

func (rw *rewriter) breaksBlock(s ast.Stmt) bool {
	switch s.(type) {
	case *ast.BlockStmt, *ast.BranchStmt, *ast.ForStmt, *ast.IfStmt,
		*ast.LabeledStmt, *ast.RangeStmt, *ast.SwitchStmt,
		*ast.SelectStmt, *ast.TypeSwitchStmt:
		return true
	case *ast.ExprStmt:
		expr := s.(*ast.ExprStmt)
		if call, ok := expr.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				return true
			}
		}
	}
	if hasFunc, _ := containsFuncLit(s); hasFunc {
		return true
	}
	return false
}

func (rw *rewriter) stmtBoundary(s ast.Stmt) token.Pos {
	switch s := s.(type) {
	case *ast.BlockStmt:
		return s.Lbrace
	case *ast.IfStmt:
		return s.Body.Lbrace
	case *ast.ForStmt:
		return s.Body.Lbrace
	case *ast.RangeStmt:
		return s.Body.Lbrace
	case *ast.SwitchStmt:
		return s.Body.Lbrace
	case *ast.TypeSwitchStmt:
		return s.Body.Lbrace
	case *ast.SelectStmt:
		return s.Body.Lbrace
	case *ast.LabeledStmt:
		return rw.stmtBoundary(s.Stmt)
	}
	if hasFunc, pos := containsFuncLit(s); hasFunc {
		return pos
	}
	return s.End()
}

func containsFuncLit(n ast.Node) (bool, token.Pos) {
	if n == nil {
		return false, 0
	}
	var found token.Pos
	ast.Inspect(n, func(node ast.Node) bool {
		if found != 0 {
			return false
		}
		if fl, ok := node.(*ast.FuncLit); ok {
			found = fl.Body.Lbrace
			return false
		}
		return true
	})
	return found != 0, found
}

// applyInsertions inserts text snippets at given byte offsets, processing in reverse order
// so that earlier offsets remain valid.
func applyInsertions(src []byte, ins []insertion) []byte {
	// Sort by offset descending so we can insert from back to front.
	for i := 0; i < len(ins); i++ {
		for j := i + 1; j < len(ins); j++ {
			if ins[j].offset > ins[i].offset {
				ins[i], ins[j] = ins[j], ins[i]
			}
		}
	}

	buf := make([]byte, 0, len(src)*2)
	buf = append(buf, src...)

	for _, in := range ins {
		text := []byte(in.text)
		tail := make([]byte, len(buf)-in.offset)
		copy(tail, buf[in.offset:])
		buf = append(buf[:in.offset], text...)
		buf = append(buf, tail...)
	}
	return buf
}

// GenerateCoverVarName creates a unique variable name for a file's coverage counters.
func GenerateCoverVarName(importPath string, index int) string {
	sum := sha256.Sum256([]byte(importPath))
	h := fmt.Sprintf("%x", sum[:6])
	return fmt.Sprintf("gococo_%d_%s", index, h)
}

// formatNode prints an AST node back to source. Used for debugging.
func formatNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node)
	return buf.String()
}

// quote returns a Go string literal.
func quote(s string) string {
	return strconv.Quote(s)
}

// addImport inserts an import statement right after the package clause.
func addImport(src []byte, fset *token.FileSet, f *ast.File, importPath string, alias string) []byte {
	insertOffset := fset.Position(f.Name.End()).Offset
	importStmt := fmt.Sprintf("; import %s %q", alias, importPath)

	result := make([]byte, 0, len(src)+len(importStmt))
	result = append(result, src[:insertOffset]...)
	result = append(result, []byte(importStmt)...)
	result = append(result, src[insertOffset:]...)
	return result
}

// lineDirective returns a //line directive to preserve original file positions.
func lineDirective(filename string) string {
	return fmt.Sprintf("//line %s:1\n", filename)
}

// BuildGlobalCoverVarDecl generates the Go source for global coverage variable declarations.
// This produces counter arrays, block metadata, the event channel, emit function,
// and accessor functions that the injected agent code calls.
func BuildGlobalCoverVarDecl(files []*FileInstrumentation, randomID string) string {
	var b strings.Builder

	b.WriteString("package gococodef\n\n")

	// Block event type
	b.WriteString(fmt.Sprintf("type GococoBlock_%s struct {\n", randomID))
	b.WriteString("\tFileIdx  int\n")
	b.WriteString("\tBlockIdx int\n")
	b.WriteString("}\n\n")

	// Channel and enabled flag (unexported internals accessed via exported functions)
	b.WriteString(fmt.Sprintf("var gococoCh_%s = make(chan *GococoBlock_%s, 8192)\n\n", randomID, randomID))
	b.WriteString(fmt.Sprintf("var gococoEnabled_%s bool\n\n", randomID))

	// Emit function: called from instrumented code via dot import
	b.WriteString(fmt.Sprintf("func GococoEmit_%s(fileIdx int, blockIdx int) {\n", randomID))
	b.WriteString(fmt.Sprintf("\tif !gococoEnabled_%s { return }\n", randomID))
	b.WriteString("\tselect {\n")
	b.WriteString(fmt.Sprintf("\tcase gococoCh_%s <- &GococoBlock_%s{FileIdx: fileIdx, BlockIdx: blockIdx}:\n", randomID, randomID))
	b.WriteString("\tdefault:\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	// Exported accessors for the agent package
	b.WriteString(fmt.Sprintf("func SetEnabled_%s(v bool) { gococoEnabled_%s = v }\n\n", randomID, randomID))
	b.WriteString(fmt.Sprintf("func EventChan_%s() <-chan *GococoBlock_%s { return gococoCh_%s }\n\n", randomID, randomID, randomID))

	// Per-file counter arrays and block metadata
	for i, fi := range files {
		nblocks := len(fi.Blocks)
		if nblocks == 0 {
			continue
		}

		// Exported counter array (accessed via dot import)
		b.WriteString(fmt.Sprintf("var GococoCov_%s [%d]uint32 // %s\n", randomID, nblocks, fi.FilePath))

		// Unexported metadata (accessed via exported BlockMeta function)
		b.WriteString(fmt.Sprintf("var gococoMeta_%s_%d = struct {\n", randomID, i))
		b.WriteString("\tFile      string\n")
		b.WriteString(fmt.Sprintf("\tStartLine [%d]int\n", nblocks))
		b.WriteString(fmt.Sprintf("\tStartCol  [%d]int\n", nblocks))
		b.WriteString(fmt.Sprintf("\tEndLine   [%d]int\n", nblocks))
		b.WriteString(fmt.Sprintf("\tEndCol    [%d]int\n", nblocks))
		b.WriteString(fmt.Sprintf("\tNumStmts  [%d]int\n", nblocks))
		b.WriteString("}{\n")
		b.WriteString(fmt.Sprintf("\tFile: %s,\n", strconv.Quote(fi.FilePath)))

		writeIntArray(&b, "StartLine", nblocks, func(j int) int { return fi.Blocks[j].StartLine })
		writeIntArray(&b, "StartCol", nblocks, func(j int) int { return fi.Blocks[j].StartCol })
		writeIntArray(&b, "EndLine", nblocks, func(j int) int { return fi.Blocks[j].EndLine })
		writeIntArray(&b, "EndCol", nblocks, func(j int) int { return fi.Blocks[j].EndCol })
		writeIntArray(&b, "NumStmts", nblocks, func(j int) int { return fi.Blocks[j].NumStmts })

		b.WriteString("}\n\n")
	}

	// Exported accessor: BlockMeta returns metadata for a given file/block index
	b.WriteString(fmt.Sprintf("func BlockMeta_%s(fileIdx int, blockIdx int) (file string, sl, sc, el, ec, stmts int) {\n", randomID))
	b.WriteString("\tswitch fileIdx {\n")
	for i, fi := range files {
		if len(fi.Blocks) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\tcase %d:\n", i))
		b.WriteString(fmt.Sprintf("\t\tm := &gococoMeta_%s_%d\n", randomID, i))
		b.WriteString("\t\treturn m.File, m.StartLine[blockIdx], m.StartCol[blockIdx], m.EndLine[blockIdx], m.EndCol[blockIdx], m.NumStmts[blockIdx]\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn\n")
	b.WriteString("}\n")

	return b.String()
}

func writeIntArray(b *strings.Builder, name string, n int, val func(int) int) {
	b.WriteString(fmt.Sprintf("\t%s: [%d]int{", name, n))
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Itoa(val(i)))
	}
	b.WriteString("},\n")
}
