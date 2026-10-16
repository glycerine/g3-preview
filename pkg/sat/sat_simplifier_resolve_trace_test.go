// Source: src/sat/sat_simplifier.cpp:1224-1246.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierMissingReciprocalBinarySourceTrace(t *testing.T) {
	s, l := satSimplifierFixture(t, 2)
	first, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	satOK(t, first.Push(satSimplifierBinaryRecord(2)))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("resolve_bug")
	l.traceTags = tags
	err = s.addNonLearnedBinary(0, 2)
	if satStorageSourceDebug {
		if err != ErrSATAssertion {
			t.Fatal("source missing-reciprocal assertion", err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	want := ""
	if tags.ShouldTrace("resolve_bug") {
		want = "-------- [resolve_bug] add_non_learned_binary_clause src/sat/sat_simplifier.cpp:1241 ---------\n" +
			"-0 -> 1\n-1 -> \n------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("source reciprocal-watch diagnostic: got %q, want %q", got, want)
	}
	got, err := first.Read(0)
	satOK(t, err)
	learned, err := got.Learned()
	satOK(t, err)
	if learned {
		t.Fatal("first watched half was converted before reciprocal assertion")
	}
	other, err := l.propagation.variables.watches.Get(3)
	satOK(t, err)
	n, err := other.Size()
	satOK(t, err)
	if n != 0 {
		t.Fatal("missing reciprocal remains absent", n)
	}
}

func TestSATSimplifierInvalidResolvePivotSourceTrace(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.kernel.InitVisited(3))
	s.kernel.elimCounter = 10
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("resolve_bug")
	l.traceTags = tags
	_, err := s.kernel.resolve(newSATBinaryWrapper(2, 4), newSATBinaryWrapper(1, 4), 0, s.literalStore, s.newCls)
	if satStorageSourceDebug {
		if err != ErrSATAssertion || s.kernel.elimCounter != 10 {
			t.Fatal("source pivot assertion before cost mutation", err, s.kernel.elimCounter)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	want := ""
	if tags.ShouldTrace("resolve_bug") {
		want = "-------- [resolve_bug] resolve src/sat/sat_simplifier.cpp:1178 ---------\n" +
			"(1 2)\n(-0 2)\nl: 0\n------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("source invalid resolver input: got %q, want %q", got, want)
	}
}
