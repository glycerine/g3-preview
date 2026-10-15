package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satSimplifyTraceObserver struct {
	output      bytes.Buffer
	solver      *Solver
	beforeScope uint32
}

func (w *satSimplifyTraceObserver) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("simplify\n")) {
		w.beforeScope = w.solver.Variables.scopeLevel
	}
	return w.output.Write(p)
}

func TestSATSolverSimplifyTraceAfterThresholdBeforeCleanup(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	w := &satSimplifyTraceObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	s.traceTags = tags
	s.nextSimplify = s.conflict.conflicts + 1
	satOK(t, s.simplifyProblem())
	if w.output.Len() != 0 {
		t.Fatal("threshold-return emitted simplify TRACE", w.output.String())
	}
	satOK(t, s.Lifecycle.push())
	s.nextSimplify = s.conflict.conflicts
	satOK(t, s.simplifyProblem())
	if s.Variables.scopeLevel != 0 {
		t.Fatal("source simplification did not pop the scope")
	}
	if tags.ShouldTrace("sat") {
		var display bytes.Buffer
		satOK(t, s.traceUserDisplayInto(&display))
		want := satClauseTraceRecord("sat", "simplify_problem", "1199", "simplify\n") +
			satClauseTraceRecord("sat", "simplify_problem", "1237", "consistent: 1\n"+display.String())
		if got := w.output.String(); got != want || w.beforeScope != 1 {
			t.Fatalf("source simplify record: got %q, want %q", got, want)
		}
	} else if w.output.Len() != 0 {
		t.Fatal("erased simplify TRACE emitted", w.output.String())
	}
}
