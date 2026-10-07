// Source: src/sat/sat_solver.cpp:723–802,993–1118,1191–1346.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverDefaultSearchIndependentThreeVariableCNF(t *testing.T) {
	var random util.Random
	random.SetSeed(91)
	for seed := 0; seed < 64; seed++ {
		s, _, _ := satSolverCheckFixture(t)
		for i := 0; i < 3; i++ {
			_, e := s.MkVar(false, true)
			satOK(t, e)
		}
		clauses := [][]Literal{}
		for j := 0; j < 8; j++ {
			c := []Literal{MakeLiteral(0, random.Bounded(2) != 0), MakeLiteral(1, random.Bounded(2) != 0), MakeLiteral(2, random.Bounded(2) != 0)}
			clauses = append(clauses, c)
			copy := append([]Literal(nil), c...)
			satOK(t, s.MkClause(copy))
		}
		got, e := s.Check(nil)
		satOK(t, e)
		if want := satSolverFiniteCNFOracle(clauses, 3); got != want {
			t.Fatal(seed, got, want)
		}
		satFullSolverRuntimeAudit(t, s)
		satFullSolverUserClauseAudit(t, s)
		satOK(t, s.Close())
	}
}
func TestSATSolverActualAssumptionCoreAndLaterPlainCheck(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	got, e := s.Check([]Literal{1, 3})
	satOK(t, e)
	if got != util.LFalse {
		t.Fatal(got)
	}
	n, e := s.conflict.core.Size()
	satOK(t, e)
	if n != 2 {
		t.Fatal("actual assumption core", n)
	}
	seen := map[uint32]bool{}
	for i := uint32(0); i < n; i++ {
		l, e := s.conflict.core.Read(i)
		satOK(t, e)
		seen[l] = true
	}
	if !seen[1] || !seen[3] {
		t.Fatal(seen)
	}
	got, e = s.Check(nil)
	satOK(t, e)
	if got != util.LTrue {
		t.Fatal(got)
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverCheckRealCancellationDoesNotCatchSolverException(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	s.Limit.Cancel()
	_, e := s.Check(nil)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	var ex SATSolverException
	if !errors.As(e, &ex) {
		t.Fatal("source solver_exception category", e)
	}
	if s.Variables.modelIsCurrent || s.propagation.numCheckpoints != 0 {
		t.Fatal("real checkpoint cancellation prefix")
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverOptionalDispatchGuardFollowsRealPopToBase(t *testing.T) {
	s, ctx, p := satSolverCheckFixture(t)
	_, e := s.MkVar(true, true)
	satOK(t, e)
	satOK(t, s.Lifecycle.push())
	satOK(t, s.propagation.assign(0, satJustification{}))
	key := "dimacs.display"
	satOK(t, ctx.SetBoolName(&p, &key, true))
	satOK(t, s.UpdateParams(&p))
	_, e = s.Check(nil)
	if !errors.Is(e, ErrSATMissingDimacsBinding) {
		t.Fatal("printer dispatch", e)
	}
	n, e := s.Variables.trail.Size(s.Variables.trailRef)
	satOK(t, e)
	if s.Variables.scopeLevel != 0 || n != 0 {
		t.Fatal("dispatch preceded source pop_to_base", s.Variables.scopeLevel, n)
	}
}
