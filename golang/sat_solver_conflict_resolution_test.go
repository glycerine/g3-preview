// Source: src/sat/sat_solver.cpp and .h. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func satConflictFixture(t *testing.T, n uint32) *satSolverConflictResolution {
	t.Helper()
	probing, l := satProbingFixture(t, n)
	p := l.propagation
	a, e := newSATSolverConflictAnalysis(p, l.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, a.Close()) })
	d, e := newSATSolverDynamicSubsumption(a, probing, l.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, d.Close()) })
	s, e := newSATSolverConflictResolution(a, d)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s
}
func satConflictLemma(t *testing.T, s *satSolverConflictResolution) []satLiteral {
	t.Helper()
	n, e := s.analysis.lemma.size()
	satOK(t, e)
	out := make([]satLiteral, n)
	for i := range out {
		v, e := s.analysis.lemma.read(uint32(i))
		satOK(t, e)
		out[i] = satLiteral(v)
	}
	return out
}

// Independent DAG: lower decision a, current decision b, c implied by a and b,
// d implied by c, conflict between b and d. Expected first UIP is b; lower a survives in the learned clause.
func satConflictDAG(t *testing.T, s *satSolverConflictResolution) {
	t.Helper()
	p := s.analysis.propagation
	satConflictPush(t, s)
	satOK(t, p.assign(0, satJustification{}))
	satConflictPush(t, s)
	satOK(t, p.assign(2, satJustification{}))
	satOK(t, p.assign(4, newSATTernaryJustification(1, 3)))
	satOK(t, p.assign(6, newSATBinaryJustification(5)))
	p.setConflict(newSATBinaryJustification(3), 6)
}
func TestSATSolverConflictFirstUIPDefaultPipeline(t *testing.T) {
	s := satConflictFixture(t, 4)
	satConflictDAG(t, s)
	ok, e := s.resolveConflict()
	satOK(t, e)
	if !ok {
		t.Fatal("firstUIP failed")
	}
	// Source lower a is needed to imply c, so lemma is (~b,~a), backtrack to1.
	if got := satConflictLemma(t, s); !reflect.DeepEqual(got, []satLiteral{3, 1}) {
		t.Fatal(got)
	}
	p := s.analysis.propagation
	v := p.variables
	if v.scopeLevel != 1 || p.inconsistent || s.conflicts != 1 || s.conflictStat != 1 || s.conflictsSinceRestart != 1 || s.conflictsSinceGC != 1 || s.phaseCounter != 1 || v.activityInc != 140 {
		t.Fatal("source publication", v.scopeLevel, s.conflicts, v.activityInc)
	}
	value, e := v.valueLiteral(3)
	satOK(t, e)
	if value != lTrue {
		t.Fatal(value)
	}
	for i := uint32(0); i < 4; i++ {
		marked, e := s.analysis.marked(i)
		satOK(t, e)
		if marked {
			t.Fatal(i)
		}
	}
}
func TestSATSolverConflictLevelZeroCountersAndOwners(t *testing.T) {
	s := satConflictFixture(t, 1)
	p := s.analysis.propagation
	p.setConflict(satJustification{}, satDefaultLiteral())
	s.conflicts = ^uint32(0)
	ok, e := s.resolveConflict()
	satOK(t, e)
	if ok || s.conflicts != 0 || s.conflictStat != 1 {
		t.Fatal(ok, s.conflicts)
	}
	copy := *s
	if _, e = copy.resolveConflict(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if _, e = s.resolveConflict(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATSolverConflictAssumptionCoreAndReinit(t *testing.T) {
	s := satConflictFixture(t, 3)
	p := s.analysis.propagation
	l := s.dynamic.probing.lifecycle
	satOK(t, s.assumptions.push(0))
	satOK(t, s.assumptionSet.Insert(0))
	satOK(t, s.userScopeLiterals.push(3))
	satOK(t, s.assumptionSet.Insert(2))
	satOK(t, s.reinitAssumptions())
	if p.variables.scopeLevel != 1 {
		t.Fatal("actual scope")
	}
	n, e := p.variables.trail.Size(p.variables.trailRef)
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	first, e := p.variables.trailRead(0)
	satOK(t, e)
	second, e := p.variables.trailRead(1)
	satOK(t, e)
	if first != 2 || second != 0 {
		t.Fatal(first, second)
	}
	p.setConflict(newSATBinaryJustification(3), 0)
	ok, e := s.resolveConflict()
	satOK(t, e)
	if ok {
		t.Fatal("assumption conflict must yield false")
	}
	n, e = s.core.size()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	satOK(t, l.pop(1))
	satOK(t, s.popReinit(0))
	if p.variables.scopeLevel != 1 {
		t.Fatal("pop0 still reinitializes assumptions")
	}
}
func TestSATSolverConflictPhaseHelpersAndMarkedPushPrefix(t *testing.T) {
	s := satConflictFixture(t, 3)
	satConflictDAGSmall(t, s)
	p := s.analysis.propagation
	v := p.variables
	s.conflictLevel = 2
	s.phaseCacheOn = false
	s.phaseCounter = 99
	s.updatePhaseCounters()
	if !s.phaseCacheOn || s.phaseCounter != 0 {
		t.Fatal("off100")
	}
	s.phaseCounter = 399
	s.updatePhaseCounters()
	if s.phaseCacheOn || s.phaseCounter != 0 {
		t.Fatal("on400")
	}
	satOK(t, s.forgetPhaseOfVars(2))
	x, e := v.phase.read(1)
	satOK(t, e)
	if x != 2 {
		t.Fatal(x)
	}
	// Fill current capacity to force source lemma growth after mark and activity.
	satOK(t, s.analysis.lemma.push(2))
	satOK(t, s.analysis.lemma.push(4))
	before, e := v.activity.Get(0)
	satOK(t, e)
	s.analysis.budget.FailAfter(0)
	marks := uint32(0)
	e = s.processAntecedent(0, &marks)
	if !satAssignmentActualRefusal(p.memory, e) {
		t.Fatal(e)
	}
	marked, xerr := s.analysis.marked(0)
	satOK(t, xerr)
	act, xerr := v.activity.Get(0)
	satOK(t, xerr)
	if !marked || act != before+128 || marks != 0 {
		t.Fatal("actual completed mark/activity prefix", marked, act, marks)
	}
	s.analysis.budget.FailAfter(-1)
}
func satConflictDAGSmall(t *testing.T, s *satSolverConflictResolution) {
	t.Helper()
	p := s.analysis.propagation
	satConflictPush(t, s)
	satOK(t, p.assign(0, satJustification{}))
	satConflictPush(t, s)
	satOK(t, p.assign(2, satJustification{}))
}
func TestSATSolverConflictRuntimeLemmaSpanLifetimeAndGlue(t *testing.T) {
	s := satConflictFixture(t, 3)
	satConflictDAGSmall(t, s)
	a := s.analysis
	satOK(t, a.lemma.push(1))
	satOK(t, a.lemma.push(3))
	span := satConflictRuntimeSpan(&a.lemma)
	n, e := a.propagation.numDiffLevelsSpan(2, span)
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	satOK(t, a.lemma.push(5))
	if _, e = span.literal(0); e == nil {
		t.Fatal("captured raw pointer must die on realloc")
	}
}

func satConflictPush(t *testing.T, s *satSolverConflictResolution) {
	t.Helper()
	ok, e := s.analysis.propagation.propagateCore(false)
	satOK(t, e)
	if !ok {
		t.Fatal("fixture conflict before scope")
	}
	satOK(t, s.dynamic.probing.lifecycle.push())
}
