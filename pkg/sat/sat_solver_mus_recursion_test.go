package sat

// Source: src/sat/sat_mus.cpp:52–174; src/sat/sat_solver.cpp:2072–2083.

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSATSolverMUSActualQXSixtyFiveNecessaryAssumptions(t *testing.T) {
	s, ctx, params := satSolverCheckFixture(t)
	var clause, assumptions []Literal
	for i := 0; i < 65; i++ {
		v, e := s.MkVar(true, true)
		satOK(t, e)
		assumptions = append(assumptions, MakeLiteral(v, false))
		clause = append(clause, MakeLiteral(v, true))
	}
	satOK(t, s.MkClause(clause))
	key := "core.minimize"
	satOK(t, ctx.SetBoolName(&params, &key, true))
	satOK(t, s.UpdateParams(&params))
	result, e := s.Check(assumptions)
	satOK(t, e)
	if result != lbool.False {
		t.Fatal(result)
	}
	n, e := s.conflict.core.Size()
	satOK(t, e)
	if n != 65 {
		t.Fatal("all65 assumptions are independently necessary", n)
	}
	var seen [65]bool
	for i := uint32(0); i < n; i++ {
		v, e := s.conflict.core.Read(i)
		satOK(t, e)
		if v&1 != 0 || v/2 >= 65 || seen[v/2] {
			t.Fatal("core literal", v)
		}
		seen[v/2] = true
	}
	if s.mus.frames != nil || s.mus.isActive {
		t.Fatal("recursive source locals remain after success")
	}
	satFullSolverRuntimeAudit(t, s)
	// Independent finite symbolic oracle: removing assumption i admits exactly
	// the witness with variable i false and all others true, satisfying the only
	// clause. Actual solver checks additionally validate each returned model.
	satOK(t, ctx.SetBoolName(&params, &key, false))
	satOK(t, s.UpdateParams(&params))
	for missing := 0; missing < 65; missing++ {
		subset := make([]Literal, 0, 64)
		subset = append(subset, assumptions[:missing]...)
		subset = append(subset, assumptions[missing+1:]...)
		result, e = s.Check(subset)
		satOK(t, e)
		if result != lbool.True {
			t.Fatal(missing, result)
		}
		v, e := s.model.Read(uint32(missing))
		satOK(t, e)
		if lbool.LBool(int32(v)) != lbool.False {
			t.Fatal("required negative witness", missing, v)
		}
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverMUSActualCancellationPublishesRemainingCoreAndRestoresFlets(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{1, 3, 5}))
	r, e := s.Check([]Literal{0, 2, 4})
	satOK(t, e)
	if r != lbool.False {
		t.Fatal(r)
	}
	oldRestart := s.config.state.RestartMax
	s.config.state.CoreMinimize = true
	s.Limit.Cancel()
	r, e = s.mus.run()
	satOK(t, e)
	if r != lbool.Undef {
		t.Fatal(r)
	}
	n, e := s.conflict.core.Size()
	satOK(t, e)
	if n != 3 || s.mus.isActive || !s.config.state.CoreMinimize || s.config.state.RestartMax != oldRestart {
		t.Fatal("canceled source prefix/flets", n)
	}
	satFullSolverRuntimeAudit(t, s)
}
