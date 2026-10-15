package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satMinimizeTraceObserver struct {
	output              bytes.Buffer
	analysis            *satSolverConflictAnalysis
	drops, beforeShrink int
}

func (w *satMinimizeTraceObserver) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "drop: ") {
		w.drops++
		n, err := w.analysis.lemma.Size()
		if err == nil && n == 5 && w.analysis.minimizedLits == 0 {
			w.beforeShrink++
		}
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:2374. Both implied literals emit before the
// lemma is shrunk and the minimized-literal statistic is published.
func TestSATMinimizeLemmaSourceDropTraceBeforeShrink(t *testing.T) {
	s, p, v, _ := satLemmaFixture(t, 6)
	for id := uint32(0); id < 6; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 2 || id == 5 {
			j = newSATBinaryJustification(2)
		}
		if id == 3 {
			j = newSATBinaryJustification(8)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	for _, id := range []uint32{1, 2, 3, 5} {
		v.mark[id] = true
	}
	for _, id := range []uint32{0, 1, 2, 3, 5} {
		satOK(t, s.lemma.Push(2*id+1))
	}
	w := &satMinimizeTraceObserver{analysis: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	p.traceTags = tags
	satOK(t, s.minimizeLemma())
	n, err := s.lemma.Size()
	satOK(t, err)
	if n != 3 || s.minimizedLits != 2 {
		t.Fatal("source lemma minimization")
	}
	if !tags.ShouldTrace("sat") {
		if w.output.Len() != 0 || w.drops != 0 {
			t.Fatal("erased lemma TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat", "minimize_lemma", "2374", "drop: -2\n") +
		satClauseTraceRecord("sat", "minimize_lemma", "2374", "drop: -5\n")
	if got := w.output.String(); got != want || w.drops != 2 || w.beforeShrink != 2 {
		t.Fatalf("lemma trace got %q, want %q; drops=%d before=%d", got, want, w.drops, w.beforeShrink)
	}
}
