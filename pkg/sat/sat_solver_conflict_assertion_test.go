package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATConflictMarkAssertionUsesProcessDebugTagAndActions(t *testing.T) {
	s := satConflictFixture(t, 2)
	s.analysis.propagation.variables.mark[0] = true
	s.analysis.propagation.variables.mark[1] = true
	defer func() {
		s.analysis.propagation.variables.mark[0] = false
		s.analysis.propagation.variables.mark[1] = false
	}()
	budget := arena.NewBudget(^uint64(0))
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	satOK(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	satOK(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "sat-mark-input-*")
	satOK(t, err)
	_, err = inputFile.WriteString("CCT")
	satOK(t, err)
	_, err = inputFile.Seek(0, 0)
	satOK(t, err)
	outputFile, err := os.CreateTemp(t.TempDir(), "sat-mark-output-*")
	satOK(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outputFile, Stderr: outputFile})
	satOK(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, inputFile)
	satOK(t, err)
	action, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	satOK(t, err)
	satOK(t, debug.BindAssertionStreams(streams))
	satOK(t, debug.BindAssertionActions(action))
	previous, err := diagnostic.BindProcessDebugContext(debug)
	satOK(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		satOK(t, e)
		satOK(t, action.Close())
		satOK(t, input.Close())
		satOK(t, streams.Close())
		satOK(t, inputFile.Close())
		satOK(t, outputFile.Close())
		satOK(t, debug.Close())
		satOK(t, region.Close())
	})
	satOK(t, s.assertNoMarks())
	satOK(t, debug.EnableDebug("sat_check_marks"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, s.assertNoMarks())
	satOK(t, debug.EnableAssertions(true))
	satOK(t, s.assertNoMarks())
	if satStorageSourceDebug {
		var ex z3exception.DefaultException
		if err = s.assertNoMarks(); !errors.As(err, &ex) {
			t.Fatalf("T action did not transport source exception: %v", err)
		}
		state, e := input.State()
		satOK(t, e)
		if state.Position != 3 {
			t.Fatalf("source debugger consumed %d actions", state.Position)
		}
		contents, e := os.ReadFile(outputFile.Name())
		satOK(t, e)
		want := "ASSERTION VIOLATION\nFile: src/sat/sat_solver.cpp\nLine: 2854\n!is_marked(v)\n" +
			"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
		if got := string(contents); got != strings.Repeat(want, 3) {
			t.Fatalf("source inner SASSERT records: got %q, want %q", got, strings.Repeat(want, 3))
		}
		s.analysis.propagation.variables.mark[0] = false
		s.analysis.propagation.variables.mark[1] = false
		s.analysis.propagation.setConflict(satJustification{}, DefaultLiteral())
		_, e = s.resolveConflict()
		satOK(t, e)
		state, e = input.State()
		satOK(t, e)
		if state.Position != 3 {
			t.Fatalf("valid conflict path consumed extra debugger action: %d", state.Position)
		}
	} else {
		contents, e := os.ReadFile(outputFile.Name())
		satOK(t, e)
		if len(contents) != 0 {
			t.Fatalf("release CASSERT emitted %q", contents)
		}
	}
}
