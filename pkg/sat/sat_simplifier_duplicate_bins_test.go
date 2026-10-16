package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierBinaryWatchNativeStableSort(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	w, e := l.propagation.variables.watches.Get(0)
	satOK(t, e)
	binary := func(l Literal, learned bool) satWatched {
		v, e := newSATBinaryWatch(l, learned)
		satOK(t, e)
		return v
	}
	ternary, e := newSATTernaryWatch(2, 4)
	satOK(t, e)
	external, e := newSATExternalWatch(3)
	satOK(t, e)
	input := []satWatched{
		binary(4, true), ternary, binary(2, true), binary(4, false), external, binary(2, false),
	}
	for _, v := range input {
		satOK(t, w.Push(v))
	}
	before := l.budget.Used()
	l.budget.FailAfter(0)
	satOK(t, s.sortBinaryWatches(w))
	l.budget.FailAfter(-1)
	if l.budget.Used() != before {
		t.Fatal("native watch sort charged arena workspace", before, l.budget.Used())
	}
	want := []satWatched{input[5], input[2], input[3], input[0], ternary, external}
	for i, expected := range want {
		got, e := w.Read(uint32(i))
		satOK(t, e)
		if !got.Equal(expected) {
			t.Fatal("source stable binary ordering", i, got, expected)
		}
	}
}

func TestSATSimplifierDuplicateBinarySourceTrace(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, lifecycle := satSimplifierFixture(t, 2)
		ws := lifecycle.propagation.variables.watches
		for _, pair := range []struct {
			index uint32
			other Literal
		}{{0, 2}, {3, 1}} {
			list, err := ws.Get(pair.index)
			satOK(t, err)
			for range 2 {
				watch, err := newSATBinaryWatch(pair.other, false)
				satOK(t, err)
				satOK(t, list.Push(watch))
			}
		}
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("subsumption")
		}
		lifecycle.traceTags = tags
		satOK(t, s.eliminateDuplicateBinaries())
		want := ""
		if tags.ShouldTrace("subsumption") {
			want = "-------- [subsumption] elim_dup_bins src/sat/sat_simplifier.cpp:769 ---------\n" +
				"eliminating: -0 1\n" +
				"------------------------------------------------\n" +
				"-------- [subsumption] elim_dup_bins src/sat/sat_simplifier.cpp:769 ---------\n" +
				"eliminating: 1 -0\n" +
				"------------------------------------------------\n"
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
		if s.numSubsumed != 1 {
			t.Fatalf("one binary clause removed: %d", s.numSubsumed)
		}
		for _, index := range []uint32{0, 3} {
			list, err := ws.Get(index)
			satOK(t, err)
			size, err := list.Size()
			satOK(t, err)
			if size != 1 {
				t.Fatalf("watch %d has size %d", index, size)
			}
		}
	}
}
