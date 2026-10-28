//go:build g3debug

package api

// Source: src/api/api_context.cpp:39,216; src/util/debug.h:63.

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAPIContextSourceObjectAndTrailAssertions(t *testing.T) {
	m := astTestManager(t)
	c, err := newAPIContextResults(m, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	goal, err := newAPIGoalObject(c)
	astTestMust(t, err)
	foreign := astTestManager(t)
	foreignAST := apiResultsAtom(t, foreign, "api-foreign-trail")

	budget := arena.NewBudget(^uint64(0))
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	astTestMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	astTestMust(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "api-assert-input-*")
	astTestMust(t, err)
	_, err = inputFile.WriteString("CCT")
	astTestMust(t, err)
	_, err = inputFile.Seek(0, 0)
	astTestMust(t, err)
	outputFile, err := os.CreateTemp(t.TempDir(), "api-assert-output-*")
	astTestMust(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outputFile, Stderr: outputFile})
	astTestMust(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, inputFile)
	astTestMust(t, err)
	action, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	astTestMust(t, err)
	astTestMust(t, debug.BindAssertionStreams(streams))
	astTestMust(t, debug.BindAssertionActions(action))
	previous, err := diagnostic.BindProcessDebugContext(debug)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		astTestMust(t, e)
		astTestMust(t, action.Close())
		astTestMust(t, input.Close())
		astTestMust(t, streams.Close())
		astTestMust(t, inputFile.Close())
		astTestMust(t, outputFile.Close())
		astTestMust(t, debug.Close())
		astTestMust(t, region.Close())
	})

	if err = goal.object.decRef(); !errors.Is(err, errAPIObjectOwner) || goal.object.references != 0 {
		t.Fatal("C action and reference-count guard", err, goal.object.references)
	}
	if err = c.saveASTTrail(foreignAST); !errors.Is(err, ast.ErrASTHandle) || len(c.astTrail) != 0 {
		t.Fatal("C action and AST membership guard", err, len(c.astTrail))
	}
	var thrown z3exception.DefaultException
	if err = goal.object.decRef(); !errors.As(err, &thrown) || goal.object.references != 0 {
		t.Fatal("T action must precede reference mutation", err, goal.object.references)
	}
	state, err := input.State()
	astTestMust(t, err)
	if state.Position != 3 {
		t.Fatal("debugger action order", state.Position)
	}
	got, err := os.ReadFile(outputFile.Name())
	astTestMust(t, err)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	want := "ASSERTION VIOLATION\nFile: src/api/api_context.cpp\nLine: 39\nm_ref_count > 0\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/api/api_context.cpp\nLine: 216\nm().contains(n)\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/api/api_context.cpp\nLine: 39\nm_ref_count > 0\n" + prompt
	if string(got) != want {
		t.Fatalf("source assertion records: got %q, want %q", strings.TrimSpace(string(got)), strings.TrimSpace(want))
	}
	astTestMust(t, c.Close())
}
