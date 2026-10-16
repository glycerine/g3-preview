//go:build g3debug

package ufbv

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestRewriteBackIndexNilSetSourceContinueAndThrow(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(996), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	r, e := newRewriteBackIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.close()) })
	r.byDecl[f] = nil
	defer delete(r.byDecl, f)
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	cacheMust(t, e)
	debug, e := diagnostic.NewDebugContext(region, budget)
	cacheMust(t, e)
	inputFile, e := os.CreateTemp(t.TempDir(), "back-actions-*")
	cacheMust(t, e)
	_, e = inputFile.WriteString("CT")
	cacheMust(t, e)
	_, e = inputFile.Seek(0, 0)
	cacheMust(t, e)
	outputFile, e := os.CreateTemp(t.TempDir(), "back-output-*")
	cacheMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outputFile, Stderr: outputFile})
	cacheMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, inputFile)
	cacheMust(t, e)
	action, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	cacheMust(t, e)
	cacheMust(t, debug.BindAssertionStreams(streams))
	cacheMust(t, debug.BindAssertionActions(action))
	previous, e := diagnostic.BindProcessDebugContext(debug)
	cacheMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		cacheMust(t, e)
		cacheMust(t, action.Close())
		cacheMust(t, input.Close())
		cacheMust(t, streams.Close())
		cacheMust(t, inputFile.Close())
		cacheMust(t, outputFile.Close())
		cacheMust(t, debug.Close())
		cacheMust(t, region.Close())
	})
	if e := r.addProcessed(fx); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("C action must continue to the Go nil-set guard", e)
	}
	var thrown z3exception.DefaultException
	if e := r.rescheduleProcessed(f); !errors.As(e, &thrown) {
		t.Fatal("T action must stop at source assertion", e)
	}
	contents, e := os.ReadFile(outputFile.Name())
	cacheMust(t, e)
	frame := func(line string) string {
		return "ASSERTION VIOLATION\nFile: src/tactic/ufbv/ufbv_rewriter.cpp\nLine: " + line + "\nit->m_value\n(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	}
	if got, want := string(contents), frame("482")+frame("518"); got != want {
		t.Fatalf("back-index source actions:\ngot %q\nwant %q", got, want)
	}
	state, e := input.State()
	cacheMust(t, e)
	if state.Position != 2 || strings.Count(string(contents), "ASSERTION VIOLATION") != 2 {
		t.Fatal("source action count", state.Position)
	}
}
