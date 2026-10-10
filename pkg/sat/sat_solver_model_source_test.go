// Source: src/sat/sat_solver.cpp:1257–1346; src/sat/sat_clause.cpp satisfied_by.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverSourceModelUsesRealAssignmentAndConverter(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.Variables.mkVar(false, true)
		satOK(t, e)
	}
	_, e := s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	satOK(t, s.propagation.assign(0, satJustification{}))
	satOK(t, s.propagation.assign(3, satJustification{}))
	satOK(t, s.mkModel())
	m, e := s.AppendModel(nil)
	satOK(t, e)
	if len(m) != 2 || m[0] != lbool.True || m[1] != lbool.False {
		t.Fatal(m)
	}
	good, e := s.checkModel()
	satOK(t, e)
	if !good {
		t.Fatal("valid model rejected")
	}
	satOK(t, (satSolverModelSpan{column: &s.model}).write(0, lbool.False))
	good, e = s.checkModel()
	satOK(t, e)
	if good {
		t.Fatal("unsatisfied binary model accepted")
	}
}

func TestSATSolverSourceModelNativeGrowthPublishesCurrent(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, e := s.Variables.mkVar(false, true)
	satOK(t, e)
	s.Variables.modelIsCurrent = false
	s.budget.FailAfter(0)
	e = s.mkModel()
	if e != nil || !s.Variables.modelIsCurrent {
		t.Fatal("native model growth/current publication", e)
	}
	if n, e := s.model.Size(); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	satFullSolverRuntimeAudit(t, s)
	s.budget.FailAfter(-1)
	satOK(t, s.Close())
	if e = s.mkModel(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
