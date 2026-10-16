// Source: src/sat/sat_simplifier.cpp:692-703.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierActualLearnedBinaryDemotionSourceTrace(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.useList.Init(3))
	satOK(t, s.kernel.InitVisited(3))
	s.kernel.subCounter = 1000
	c, err := l.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, err)
	satOK(t, s.useList.Insert(c))
	satOK(t, l.mkBinClause(0, 2, true))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mark_not_learned_bug")
	l.traceTags = tags
	yes, err := s.subsumeBinaries()
	satOK(t, err)
	if !yes {
		t.Fatal("source binary subsumption completed")
	}
	removed, err := l.propagation.clauses.Removed(c)
	satOK(t, err)
	if !removed {
		t.Fatal("binary subsumed the original clause")
	}
	for _, pair := range []struct {
		watch uint32
		other Literal
	}{{1, 2}, {3, 0}} {
		list, err := l.propagation.variables.watches.Get(pair.watch)
		satOK(t, err)
		it, err := s.watchIterator(list)
		satOK(t, err)
		found := false
		for i := uint32(0); i < it.end; i++ {
			w, err := satPropagationCursorRead(&it, i)
			satOK(t, err)
			if !w.Binary() {
				continue
			}
			other, err := w.Literal()
			satOK(t, err)
			if other != pair.other {
				continue
			}
			learned, err := w.Learned()
			satOK(t, err)
			if learned {
				t.Fatal("both watched halves must be demoted", pair)
			}
			found = true
		}
		if !found {
			t.Fatal("demoted watched half missing", pair)
		}
	}
	want := ""
	if tags.ShouldTrace("mark_not_learned_bug") {
		want = "-------- [mark_not_learned_bug] subsume_with_binaries src/sat/sat_simplifier.cpp:700 ---------\n" +
			"marking as not learned: 1 1\n------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("source binary-demotion trace: got %q, want %q", got, want)
	}
}
