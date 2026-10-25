package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

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

func TestSMTSequenceNonContainsForcesEmptyBeforeTailExposure(t *testing.T) {
	m := astTestManager(t)
	needle, err := m.MkString("z")
	astTestMust(t, err)
	sort, err := m.GetSort(needle)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "lazy-noncontains-x"), sort)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	contains, err := m.MkSequenceContains(x, needle)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(contains, true))
	literal, err := state.Literal(contains)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(literal.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	empty, err := m.MkSequenceEmpty(sort)
	astTestMust(t, err)
	emptyEquality, err := m.MkEq(x, empty)
	astTestMust(t, err)
	emptyLiteral, err := state.Literal(emptyEquality)
	astTestMust(t, err)
	data, err := state.BooleanData(emptyLiteral.Var())
	astTestMust(t, err)
	if value, err := state.Assignment(emptyLiteral); err != nil || value != lbool.Undef || !data.PhaseAvailable || !data.Phase || !state.isRelevant(emptyEquality) {
		t.Fatal("source empty phase was not forced before recursive containment", value, err)
	}
	zero, err := theory.integer(0)
	astTestMust(t, err)
	tail, err := theory.relationSkolem("seq.tail", []ast.ASTHandle{x, zero}, sort)
	astTestMust(t, err)
	tailContains, err := m.MkSequenceContains(tail, needle)
	astTestMust(t, err)
	astTestPin(t, m, tailContains)
	if _, found := state.BoolVar(tailContains); found {
		t.Fatal("recursive tail atom exposed before the empty decision")
	}
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(emptyLiteral.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	astTestMust(t, theory.addNegativeContainsStep(contains))
	astTestMust(t, theory.Propagate())
	_, err = state.PropagateCore()
	astTestMust(t, err)
	if _, found := state.BoolVar(tailContains); !found {
		t.Fatal("nonempty branch did not expose the deferred tail")
	}
	astTestMust(t, state.PopScopes(1))
	if _, found := state.BoolVar(tailContains); found {
		t.Fatal("child tail exposure survived scope pop")
	}
}
