package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func theoryCaseSplitLiteral(t *testing.T, state *SMTInternalizerState, name string) SMTLiteral {
	t.Helper()
	m := state.manager
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	expression, err := m.MkConst(astTestSymbol(t, m, name), sort)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(expression, true))
	literal, err := state.Literal(expression)
	astTestMust(t, err)
	return literal
}

func TestSMTTheoryCaseSplitBothModesAtMostOneAndScopes(t *testing.T) {
	for _, heuristic := range []bool{false, true} {
		m := astTestManager(t)
		_, _, state := smtInternalizerFixture(t, m, nil)
		astTestMust(t, state.configureBooleanHeuristics(SMTParamsState{TheoryCaseSplit: heuristic, InvDecay: 1}))
		a := theoryCaseSplitLiteral(t, state, "split-a")
		b := theoryCaseSplitLiteral(t, state, "split-b").Negated()
		c := theoryCaseSplitLiteral(t, state, "split-c")
		astTestMust(t, state.MakeTheoryCaseSplit([]SMTLiteral{a, b, c}))
		if heuristic && (len(state.theoryCaseSplitSets) != 1 || len(state.allTheoryCaseSplitLiterals) != 3) {
			t.Fatal("heuristic mode did not register the signed set")
		}
		for _, literal := range []SMTLiteral{a, b, c} {
			if value, err := state.Assignment(literal); err != nil || value != lbool.Undef {
				t.Fatal("case split added an at-least-one assignment", value, err)
			}
		}
		astTestMust(t, state.PushScope())
		astTestMust(t, state.AssignLiteral(a, NewSMTBooleanAxiom()))
		_, err := state.PropagateCore()
		astTestMust(t, err)
		for _, literal := range []SMTLiteral{b, c} {
			if value, err := state.Assignment(literal); err != nil || value != lbool.False {
				t.Fatal("signed case split did not exclude its sibling", heuristic, value, err)
			}
			if heuristic {
				reason, ok := state.booleanData[literal.Var()].Justification().Literal()
				if !ok || reason != a {
					t.Fatal("heuristic assignment lost its binary antecedent")
				}
			}
		}
		astTestMust(t, state.PopScopes(1))
		astTestMust(t, state.PushScope())
		astTestMust(t, state.AssignLiteral(b, NewSMTBooleanAxiom()))
		_, err = state.PropagateCore()
		astTestMust(t, err)
		if value, err := state.Assignment(a); err != nil || value != lbool.False {
			t.Fatal("case split did not survive decision rollback", value, err)
		}
		astTestMust(t, state.PopScopes(1))
		astTestMust(t, state.PushScope())
		astTestMust(t, state.AssignLiteral(a, NewSMTBooleanAxiom()))
		astTestMust(t, state.AssignLiteral(b, NewSMTBooleanAxiom()))
		consistent, err := state.PropagateCore()
		astTestMust(t, err)
		if consistent || !state.Inconsistent() {
			t.Fatal("two active siblings failed to conflict", heuristic)
		}
		astTestMust(t, state.PopScopes(1))
	}
}

func TestSMTTheoryCaseSplitSnapshotAndRegistrationPop(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	state.theoryCaseSplit = true
	a := theoryCaseSplitLiteral(t, state, "snapshot-a")
	b := theoryCaseSplitLiteral(t, state, "snapshot-b")
	c := theoryCaseSplitLiteral(t, state, "snapshot-c")
	astTestMust(t, state.PushScope())
	group := []SMTLiteral{a, b.Negated()}
	astTestMust(t, state.MakeTheoryCaseSplit(group))
	group[0] = c // The registered span must own a copy.
	astTestMust(t, state.MakeTheoryCaseSplit([]SMTLiteral{b, c}))
	head := state.propagationHead
	astTestMust(t, state.AssignLiteral(a, NewSMTBooleanAxiom()))
	_, err := state.Propagate()
	astTestMust(t, err)
	_, err = state.propagateTheoryCaseSplits(head)
	astTestMust(t, err)
	if value, err := state.Assignment(b); err != nil || value != lbool.True {
		t.Fatal("first signed exclusion did not propagate", value, err)
	}
	if value, err := state.Assignment(c); err != nil || value != lbool.Undef {
		t.Fatal("case split consumed its own new assignment in the same snapshot", value, err)
	}
	_, err = state.PropagateCore()
	astTestMust(t, err)
	if value, err := state.Assignment(c); err != nil || value != lbool.False {
		t.Fatal("queued exclusion did not propagate on the next core iteration", value, err)
	}
	astTestMust(t, state.PopScopes(1))
	if len(state.theoryCaseSplitSets) != 0 || len(state.allTheoryCaseSplitLiterals) != 0 {
		t.Fatal("child case split registrations survived pop")
	}
	for _, sets := range state.literalTheoryCaseSplitSets {
		if len(sets) != 0 {
			t.Fatal("per-literal case split links survived pop")
		}
	}
	astTestMust(t, state.AssignLiteral(a, NewSMTBooleanAxiom()))
	astTestMust(t, state.AssignLiteral(b.Negated(), NewSMTBooleanAxiom()))
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent {
		t.Fatal("removed child group still enforced exclusion")
	}
}
