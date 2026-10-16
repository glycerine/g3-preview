package sat

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerNestedBoolVarAssertionsUseSourceOrderAndActions(t *testing.T) {
	g, _ := satGCFixture(t, 3)
	c := g.cleaner
	v := g.lifecycle().propagation.variables
	decision, assigned := v.decision, v.assignedSinceGC
	defer func() { v.decision, v.assignedSinceGC = decision, assigned }()
	v.decision, v.assignedSinceGC = nil, nil
	budget := arena.NewBudget(^uint64(0))
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	satOK(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	satOK(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "sat-checker-actions-*")
	satOK(t, err)
	_, err = inputFile.WriteString("CCT")
	satOK(t, err)
	_, err = inputFile.Seek(0, 0)
	satOK(t, err)
	outputFile, err := os.CreateTemp(t.TempDir(), "sat-checker-output-*")
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
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	if err = c.assertInvariant(); err != nil {
		contents, _ := os.ReadFile(outputFile.Name())
		t.Fatalf("first nested continuation: %v; records %q", err, contents)
	}
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, c.assertInvariant())
		return
	}
	if err = c.assertInvariant(); err != nil {
		contents, _ := os.ReadFile(outputFile.Name())
		t.Fatalf("first nested continuation: %v; records %q", err, contents)
	}
	var ex z3exception.DefaultException
	if err = c.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("nested T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 3 {
		t.Fatalf("nested actions consumed %d bytes", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	assertion := func(line, condition string) string {
		return "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: " + line + "\n" + condition + "\n" + prompt
	}
	want := assertion("137", "s.m_decision.size() == s.num_vars()") +
		assertion("145", "s.m_assigned_since_gc.size() == s.num_vars()") +
		assertion("137", "s.m_decision.size() == s.num_vars()")
	if string(contents) != want {
		t.Fatalf("nested source records: got %q, want %q", contents, want)
	}
	v.decision, v.assignedSinceGC = decision, assigned
	satOK(t, c.assertInvariant())
}
