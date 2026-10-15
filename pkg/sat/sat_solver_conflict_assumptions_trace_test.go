package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type satReinitAssumptionsTraceObserver struct {
	output                 bytes.Buffer
	solver                 *Solver
	beforeSeen, beforePush bool
	afterSeen, afterAssign bool
}

func (w *satReinitAssumptionsTraceObserver) Write(p []byte) (int, error) {
	text := string(p)
	v := w.solver.Variables
	if text == "0\n" {
		w.beforeSeen = true
		w.beforePush = v.scopeLevel == 0
	}
	if strings.HasPrefix(text, "0: ") {
		w.afterSeen = true
		value, err := v.valueLiteral(0)
		w.afterAssign = err == nil && value == lbool.True && v.scopeLevel == 1
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:1143,1151-1157. The assumption list precedes
// scope push; antecedent rows follow the actual reasserted assignments.
func TestSATReinitAssumptionsSourceTraceAroundScopeAndAssignments(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	satOK(t, s.conflict.assumptions.Push(0))
	satOK(t, s.conflict.assumptionSet.Insert(0))
	var antecedents satSolverIndexSet
	defer func() { satOK(t, antecedents.close()) }()
	satOK(t, antecedents.insert(2))
	satOK(t, antecedents.insert(5))
	satOK(t, s.antecedents.insert(0, &antecedents))
	w := &satReinitAssumptionsTraceObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	s.conflict.traceTags = tags
	satOK(t, s.conflict.reinitAssumptions())
	if s.Variables.scopeLevel != 1 {
		t.Fatal("source scope push")
	}
	if !tags.ShouldTrace("sat") {
		if w.output.Len() != 0 || w.beforeSeen || w.afterSeen {
			t.Fatal("erased reinit TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat", "reinit_assumptions", "1143", "0\n") +
		satClauseTraceRecord("sat", "reinit_assumptions", "1151", "0: 1 -2 \n")
	if got := w.output.String(); got != want || !w.beforeSeen || !w.beforePush || !w.afterSeen || !w.afterAssign {
		t.Fatalf("reinit trace got %q, want %q; before=%v after=%v", got, want, w.beforePush, w.afterAssign)
	}
}
