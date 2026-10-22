package smtcore

import "testing"

func TestSMTSequenceNonContainsQueueAndAssignmentScopes(t *testing.T) {
	m := astTestManager(t)
	left, err := m.MkString("left")
	astTestMust(t, err)
	right, err := m.MkString("right")
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(left, right)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.PushScope())
	astTestMust(t, state.Internalize(contains, true))
	variable, found := state.BoolVar(contains)
	if !found {
		t.Fatal("contains Boolean variable missing")
	}
	astTestMust(t, theory.Assign(variable, false))
	if theory.nonContains.Len() != 1 {
		t.Fatal("non-contains assignment queue", theory.nonContains.Len())
	}
	if theory.relations.Len() != 1 {
		t.Fatal("relation assignment queue", theory.relations.Len())
	}
	expression, dependency, err := theory.nonContains.At(0)
	astTestMust(t, err)
	if expression != contains || dependency == nil {
		t.Fatal("non-contains entry", expression, dependency)
	}
	assumptions := theory.dependencies.Linearize(dependency)
	if len(assumptions) != 1 || assumptions[0].literal != NewSMTLiteral(variable, true) {
		t.Fatal("non-contains explanation", assumptions)
	}
	astTestMust(t, state.PopScopes(1))
	if theory.nonContains.Len() != 0 {
		t.Fatal("non-contains scope rollback", theory.nonContains.Len())
	}
	if theory.relations.Len() != 0 {
		t.Fatal("relation scope rollback", theory.relations.Len())
	}
}
