package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTSequenceUnitConcreteCharacterRetainsEqualityEvidence(t *testing.T) {
	m := astTestManager(t)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	value, err := m.BitVectorNumeralUnsigned(97, 8)
	astTestMust(t, err)
	sort, err := m.GetSort(value)
	astTestMust(t, err)
	character, err := m.MkConst(astTestSymbol(t, m, "unit-evidence-character"), sort)
	astTestMust(t, err)
	unit, err := m.MkSequenceUnit(character)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(unit, false))
	if _, fixed, _, err := theory.concreteString(unit, nil, make(map[ast.ASTHandle]struct{})); err != nil || fixed {
		t.Fatal("unassigned character was treated as concrete", fixed, err)
	}
	equality, err := m.MkEq(character, value)
	astTestMust(t, err)
	astTestMust(t, state.Assert(equality, ast.ASTHandle{}, 0))
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent {
		t.Fatal("character equality was inconsistent")
	}
	text, fixed, dependency, err := theory.concreteString(unit, nil, make(map[ast.ASTHandle]struct{}))
	astTestMust(t, err)
	if !fixed || text != "a" {
		t.Fatal("established character equality was not used", text, fixed)
	}
	literals, equalities := theory.dependencyEvidence(dependency)
	if len(literals) != 0 || len(equalities) != 1 || equalities[0].First.Root() != equalities[0].Second.Root() {
		t.Fatal("concrete character lost equality evidence", literals, equalities)
	}
}
