//go:build g3debug

package heap

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestHeapInvalidOperationsUseSourceAssertions(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	heapMust(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	heapMust(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "heap-input-*")
	heapMust(t, err)
	_, err = inputFile.WriteString("CCCCCCCT")
	heapMust(t, err)
	_, err = inputFile.Seek(0, io.SeekStart)
	heapMust(t, err)
	outputFile, err := os.CreateTemp(t.TempDir(), "heap-output-*")
	heapMust(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: outputFile, Stderr: outputFile})
	heapMust(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, inputFile)
	heapMust(t, err)
	actions, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	heapMust(t, err)
	heapMust(t, d.BindAssertionStreams(streams))
	heapMust(t, d.BindAssertionActions(actions))
	h, err := New(2, Config{Budget: b, Debug: d})
	heapMust(t, err)
	t.Cleanup(func() {
		heapMust(t, h.Close())
		heapMust(t, actions.Close())
		heapMust(t, input.Close())
		heapMust(t, streams.Close())
		heapMust(t, inputFile.Close())
		heapMust(t, outputFile.Close())
		heapMust(t, d.Close())
		heapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("heap assertion owner leak", b.Used())
		}
	})
	heapMust(t, d.EnableAssertions(false))
	if e := h.Insert(-1); !errors.Is(e, ErrValue) {
		t.Fatal("erased source assertion must reach the Go bounds guard", e)
	}
	if _, e := h.Min(); !errors.Is(e, ErrValue) {
		t.Fatal("erased minimum assertion must reach the Go guard", e)
	}
	if _, e := h.EraseMin(); !errors.Is(e, ErrValue) {
		t.Fatal("erased erase-min assertions must reach the Go guard", e)
	}
	for _, action := range []func(int32) error{h.Erase, h.Decreased, h.Increased} {
		if e := action(0); !errors.Is(e, ErrValue) {
			t.Fatal("erased membership assertion must reach the Go guard", e)
		}
	}
	heapMust(t, d.EnableAssertions(true))
	// SASSERT is independent of the conditional "heap" debug tag.
	if e := h.Insert(-1); !errors.Is(e, ErrValue) {
		t.Fatal("continuing nested assertion must reach the bounds guard", e)
	}
	if _, e := h.Min(); !errors.Is(e, ErrValue) {
		t.Fatal("continuing minimum assertion must reach the Go guard", e)
	}
	if _, e := h.EraseMin(); !errors.Is(e, ErrValue) {
		t.Fatal("continuing erase-min assertions must reach the Go guard", e)
	}
	for _, action := range []func(int32) error{h.Erase, h.Decreased, h.Increased} {
		if e := action(0); !errors.Is(e, ErrValue) {
			t.Fatal("continuing membership assertion must reach the Go guard", e)
		}
	}
	var thrown z3exception.DefaultException
	if e := h.Insert(2); !errors.As(e, &thrown) {
		t.Fatal("nested source assertion must run its T action", e)
	}
	if h.Size() != 0 {
		t.Fatal("invalid values changed heap membership")
	}
	got, err := os.ReadFile(outputFile.Name())
	heapMust(t, err)
	record := func(line, condition string) string {
		return "ASSERTION VIOLATION\nFile: src/util/heap.h\nLine: " + line + "\n" + condition + "\n(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	}
	want := record("57", "v >= 0 && v < static_cast<int>(m_value2indices.size())") +
		record("169", "!empty()") + record("175", "!empty()") + record("176", "m_values.size() >= 2") +
		record("197", "contains(val)") + record("221", "contains(val)") + record("226", "contains(val)") +
		record("57", "v >= 0 && v < static_cast<int>(m_value2indices.size())")
	if string(got) != want {
		t.Fatalf("heap source diagnostics:\ngot %q\nwant %q", got, want)
	}
}

func TestHeapSourceConditionalAssertionsGateAndPreMutationFailure(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	heapMust(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	heapMust(t, err)
	priorities := map[int32]int32{1: 1, 2: 2}
	comparisons := 0
	h, err := New(4, Config{Budget: b, Debug: d, Less: func(a, b int32) bool {
		comparisons++
		return priorities[a] < priorities[b]
	}})
	heapMust(t, err)
	t.Cleanup(func() {
		heapMust(t, h.Close())
		heapMust(t, d.Close())
		heapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("heap/debug owner leak", b.Used())
		}
	})
	heapMust(t, h.Insert(2))
	heapMust(t, h.Insert(1))
	comparisons = 0
	heapMust(t, h.Reserve(4))
	if comparisons != 0 {
		t.Fatal("disabled heap tag evaluated invariant", comparisons)
	}
	heapMust(t, d.EnableDebug("heap"))
	heapMust(t, h.Reserve(4))
	if comparisons != 2 {
		t.Fatal("source reserve pre/post CASSERT order", comparisons)
	}
	heapMust(t, d.EnableAssertions(false))
	comparisons = 0
	heapMust(t, h.Reserve(4))
	if comparisons != 0 {
		t.Fatal("disabled assertion switch evaluated invariant", comparisons)
	}
	heapMust(t, d.EnableAssertions(true))
	priorities[1], priorities[2] = 3, 0
	if err := h.Reset(); !errors.Is(err, diagnostic.ErrDebugNotificationBinding) || h.Empty() {
		t.Fatal("source pre-reset CASSERT must stop before mutation", err, h.Empty())
	}
	priorities[1], priorities[2] = 1, 2
	if err := h.Insert(1); !errors.Is(err, diagnostic.ErrDebugNotificationBinding) || h.Size() != 2 {
		t.Fatal("duplicate insert CASSERT must precede publication", err, h.Size())
	}
	priorities[1], priorities[2] = 3, 0
	file, err := os.CreateTemp(t.TempDir(), "heap-assert-")
	heapMust(t, err)
	defer file.Close()
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: file, Stderr: file})
	heapMust(t, err)
	defer func() { heapMust(t, streams.Close()) }()
	heapMust(t, d.BindAssertionStreams(streams))
	if err := h.Reset(); !errors.Is(err, diagnostic.ErrDebugAssertion) || h.Empty() {
		t.Fatal("nested invariant SASSERT must stop before reset", err, h.Empty())
	}
	_, err = file.Seek(0, io.SeekStart)
	heapMust(t, err)
	text, err := io.ReadAll(file)
	heapMust(t, err)
	if !strings.Contains(string(text), "Line: 64\n") || !strings.Contains(string(text), "parent(idx) == 0 || !less_than") {
		t.Fatal("expected nested SASSERT line and condition", string(text))
	}
	priorities[1], priorities[2] = 1, 2
	heapMust(t, d.DisableDebug("heap"))
	heapMust(t, h.Reset())
}

func TestHeapBorrowsProcessDebugUnlessExplicit(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	heapMust(t, err)
	process, err := diagnostic.NewDebugContext(region, b)
	heapMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(process)
	heapMust(t, err)
	defer func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		heapMust(t, e)
		heapMust(t, process.Close())
		heapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("process heap owner leak", b.Used())
		}
	}()
	comparisons := 0
	less := func(a, b int32) bool { comparisons++; return a < b }
	h, err := New(4, Config{Budget: b, Less: less})
	heapMust(t, err)
	defer func() { heapMust(t, h.Close()) }()
	heapMust(t, h.Insert(2))
	heapMust(t, h.Insert(1))
	comparisons = 0
	heapMust(t, h.Reserve(4))
	if comparisons != 0 {
		t.Fatal("disabled process tag evaluated invariant", comparisons)
	}
	heapMust(t, process.EnableDebug("heap"))
	heapMust(t, h.Reserve(4))
	if comparisons == 0 {
		t.Fatal("heap did not borrow the process tag")
	}
	explicit, err := diagnostic.NewDebugContext(region, b)
	heapMust(t, err)
	defer func() { heapMust(t, explicit.Close()) }()
	other, err := New(4, Config{Budget: b, Less: less, Debug: explicit})
	heapMust(t, err)
	defer func() { heapMust(t, other.Close()) }()
	heapMust(t, other.Insert(2))
	heapMust(t, other.Insert(1))
	comparisons = 0
	heapMust(t, other.Reserve(4))
	if comparisons != 0 {
		t.Fatal("explicit debug owner must override process tag", comparisons)
	}
	heapMust(t, process.EnableAssertions(false))
	comparisons = 0
	heapMust(t, h.Reserve(4))
	if comparisons != 0 {
		t.Fatal("disabled process assertions evaluated invariant", comparisons)
	}
	heapMust(t, process.EnableAssertions(true))
	heapMust(t, process.DisableDebug("heap"))
	replacement, err := diagnostic.NewDebugContext(region, b)
	heapMust(t, err)
	defer func() { heapMust(t, replacement.Close()) }()
	if _, err := diagnostic.BindProcessDebugContext(replacement); err != nil {
		t.Fatal(err)
	}
	heapMust(t, replacement.EnableDebug("heap"))
	comparisons = 0
	heapMust(t, h.Reserve(4))
	if comparisons == 0 {
		t.Fatal("existing heap did not read replacement process owner")
	}
	if _, err := diagnostic.BindProcessDebugContext(process); err != nil {
		t.Fatal(err)
	}
}
