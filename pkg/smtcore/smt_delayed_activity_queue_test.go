package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func delayedActivityFixture(t *testing.T, frequency float64) (*ast.ASTManager, *SMTInternalizerState, *SMTActivityCaseSplitQueue) {
	t.Helper()
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{CaseSplitStrategy: 1, InvDecay: 1.052, RandomVarFreq: frequency}
	astTestMust(t, s.configureBooleanHeuristics(params))
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	astTestMust(t, q.ConfigurePhases(params))
	return m, s, q
}

func TestSMTDelayedActivityActualPromotionAndSingleRandomDraw(t *testing.T) {
	m, s, q := delayedActivityFixture(t, 0)
	base := smtRelevancyBool(t, m, "delayed-heap-base")
	astTestMust(t, s.Internalize(base, true))
	bv, _ := s.BoolVar(base)
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	astTestMust(t, s.PushScope())
	var vars []int32
	for i, name := range []string{"delayed-heap-x", "delayed-heap-y"} {
		x := smtRelevancyBool(t, m, name)
		astTestMust(t, s.Internalize(x, true))
		v, _ := s.BoolVar(x)
		vars = append(vars, v)
		for n := 0; n <= i; n++ {
			astTestMust(t, s.incrementBooleanActivity(v))
		}
		if s.basicActivity.Contains(v) || !s.delayedActivity.Contains(v) {
			t.Fatal("search creation did not enter delayed heap")
		}
	}
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != bv {
		t.Fatal("delayed priority bypassed base heap", got, bv)
	}
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(bv, false), NewSMTBooleanAxiom()))
	expected := s.random
	expected.Next()
	got, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if got != vars[1] || s.random != expected || !s.delayedActivity.Empty() || !s.basicActivity.Contains(vars[0]) {
		t.Fatal("source swap/priority/single RNG draw", got, vars)
	}
	z := smtRelevancyBool(t, m, "delayed-heap-after-swap")
	astTestMust(t, s.Internalize(z, true))
	zv, _ := s.BoolVar(z)
	for i := 0; i < 10; i++ {
		astTestMust(t, s.incrementBooleanActivity(zv))
	}
	got, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if got != vars[0] {
		t.Fatal("new delayed generation bypassed remaining primary", got, vars[0])
	}
	got, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if got != zv {
		t.Fatal("second promotion lost new search variable", got, zv)
	}
	astTestMust(t, s.PopScopes(1))
	if !s.basicActivity.Contains(bv) {
		t.Fatal("base unassignment did not reinsert primary")
	}
	for _, v := range append(vars, zv) {
		if s.basicActivity.Contains(v) || s.delayedActivity.Contains(v) {
			t.Fatal("deleted variable retained after actual promotion", v)
		}
	}
}

func TestSMTDelayedActivityDualMembershipAndDeletion(t *testing.T) {
	m, s, q := delayedActivityFixture(t, 1)
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	astTestMust(t, s.PushScope())
	x := smtRelevancyBool(t, m, "delayed-random-unassign")
	astTestMust(t, s.Internalize(x, true))
	v, _ := s.BoolVar(x)
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != v || !s.delayedActivity.Contains(v) {
		t.Fatal("random choice must leave delayed heap present", got, v)
	}
	astTestMust(t, s.PushScope())
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(v, true), NewSMTBooleanAxiom()))
	astTestMust(t, s.PopScopes(1))
	if !s.basicActivity.Contains(v) || !s.delayedActivity.Contains(v) {
		t.Fatal("inherited unassign must insert primary despite delayed membership")
	}
	astTestMust(t, s.incrementBooleanActivity(v))
	astTestMust(t, s.PopScopes(1))
	if s.basicActivity.Contains(v) || s.delayedActivity.Contains(v) {
		t.Fatal("deletion did not erase both heap memberships")
	}
}

func TestSMTDelayedActivityAvailabilityDoesNotPromoteOrConsume(t *testing.T) {
	m, s, q := delayedActivityFixture(t, 0)
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	x := smtRelevancyBool(t, m, "delayed-probe")
	astTestMust(t, s.Internalize(x, true))
	v, _ := s.BoolVar(x)
	rng := s.random
	available, err := q.hasCaseSplit()
	astTestMust(t, err)
	if !available || s.random != rng || s.basicActivity.Contains(v) || !s.delayedActivity.Contains(v) {
		t.Fatal("availability mutated delayed membership or RNG")
	}
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != v {
		t.Fatal(got, v)
	}
	got, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if got != SMTNullBoolVar {
		t.Fatal("consumed delayed variable regenerated", got)
	}
}

func TestSMTDelayedActivityConstructorUnwindsBothOwnerPrefixes(t *testing.T) {
	for _, stop := range []int64{0, 1} {
		m := astTestManager(t)
		_, _, s := smtInternalizerFixture(t, m, nil)
		q, err := NewSMTActivityCaseSplitQueue(s)
		astTestMust(t, err)
		before := m.Budget().Used()
		m.Budget().FailAfter(stop)
		err = q.ConfigurePhases(SMTParamsState{CaseSplitStrategy: 1})
		m.Budget().FailAfter(-1)
		if err == nil || s.basicActivity != nil || s.delayedActivity != nil || m.Budget().Used() != before {
			t.Fatal("derived constructor did not unwind actual owned prefix", stop, err, before, m.Budget().Used())
		}
		astTestMust(t, q.ConfigurePhases(SMTParamsState{CaseSplitStrategy: 1}))
		if s.basicActivity == nil || s.delayedActivity == nil {
			t.Fatal("retry did not construct source pair")
		}
	}
}

func TestSMTDelayedActivityNongateNotDecisionBeforeChildAssignment(t *testing.T) {
	m, s, q := delayedActivityFixture(t, 0)
	x := smtRelevancyBool(t, m, "delayed-nongate-not-child")
	not, err := m.MkNot(x)
	astTestMust(t, err)
	astTestMust(t, s.Internalize(not, false))
	xv, _ := s.BoolVar(x)
	nv, found := s.BoolVar(not)
	if !found {
		t.Fatal("source nongate NOT needs its own Boolean variable")
	}
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != nv {
		t.Fatal("fixture did not reach native NOT-before-child decision order", got, nv)
	}
	before, err := s.Assignment(NewSMTLiteral(xv, false))
	astTestMust(t, err)
	if before != lbool.Undef {
		t.Fatal("child was assigned before parent decision", before)
	}
	astTestMust(t, s.assignDecisionLiteral(NewSMTLiteral(nv, true)))
	after, err := s.Assignment(NewSMTLiteral(xv, false))
	astTestMust(t, err)
	if after != lbool.Undef {
		t.Fatal("assignment callback unexpectedly propagated child", after)
	}
	consistent, err := s.PropagateCore()
	astTestMust(t, err)
	after, err = s.Assignment(NewSMTLiteral(xv, false))
	astTestMust(t, err)
	if !consistent || after != lbool.True {
		t.Fatal("actual NOT gate clauses did not propagate opposite child", consistent, after)
	}
}
