package sat

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

type satDecisionTraceObserver struct {
	output                             bytes.Buffer
	decision                           *satSolverDecision
	randomSeen, randomBeforeValueCheck bool
	decideSeen, decideAfterAssignment  bool
}

func (w *satDecisionTraceObserver) Write(p []byte) (int, error) {
	text := string(p)
	v := w.decision.conflict.analysis.propagation.variables
	if strings.HasPrefix(text, "next: ") {
		w.randomSeen = true
		w.randomBeforeValueCheck = v.scopeLevel == 0 && w.decision.decisions == 0
	}
	if strings.HasPrefix(text, "1: next-case-split: ") {
		w.decideSeen = true
		value, err := v.valueVar(0)
		w.decideAfterAssignment = err == nil && value == lbool.False && v.scopeLevel == 1 && w.decision.decisions == 1
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:940. The selected random variable and its
// value are printed before eligibility and queue fallback are evaluated.
func TestSATRandomSplitSourceTraceCandidate(t *testing.T) {
	d, g := satDecisionFixture(t, 4)
	p := g.lifecycle().propagation
	p.config.state.RandomFreq = 1
	d.rng.SetSeed(17)
	oracle := z3rand.New(17)
	oracle.Next()
	wantNext := BoolVar(oracle.Next() % 4)
	w := &satDecisionTraceObserver{decision: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("random_split")
	d.traceTags = tags
	next, err := d.nextVar()
	satOK(t, err)
	if next != wantNext {
		t.Fatal("random candidate", next, wantNext)
	}
	if !tags.ShouldTrace("random_split") {
		if w.output.Len() != 0 || w.randomSeen {
			t.Fatal("erased random TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("random_split", "next_var", "940", "next: "+strconv.FormatUint(uint64(next), 10)+" value(next): l_undef\n")
	if got := w.output.String(); got != want || !w.randomSeen || !w.randomBeforeValueCheck {
		t.Fatalf("random trace got %q, want %q; before=%v", got, want, w.randomBeforeValueCheck)
	}
}

// Source: src/sat/sat_solver.cpp:989. The decision trace follows the actual
// assignment and records the new scope level and selected literal.
func TestSATDecisionSourceTraceAfterAssignment(t *testing.T) {
	d, g := satDecisionFixture(t, 1)
	p := g.lifecycle().propagation
	p.config.state.RandomFreq = 0
	p.config.state.Phase = satPhaseAlwaysFalse
	w := &satDecisionTraceObserver{decision: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_decide")
	d.traceTags = tags
	yes, err := d.decide()
	satOK(t, err)
	if !yes {
		t.Fatal("missing source decision")
	}
	if !tags.ShouldTrace("sat_decide") {
		if w.output.Len() != 0 || w.decideSeen {
			t.Fatal("erased decision TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat_decide", "decide", "989", "1: next-case-split: -0\n")
	if got := w.output.String(); got != want || !w.decideSeen || !w.decideAfterAssignment {
		t.Fatalf("decision trace got %q, want %q; assigned=%v", got, want, w.decideAfterAssignment)
	}
}
