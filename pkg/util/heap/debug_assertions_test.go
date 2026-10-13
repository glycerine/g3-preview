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
)

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
