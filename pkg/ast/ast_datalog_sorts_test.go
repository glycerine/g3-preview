package ast

import (
	"errors"
	"math"
	"testing"
)

func TestASTDatalogOriginalRelationAndRuleSorts(t *testing.T) {
	m := astTestManager(t)
	a := astArrayFiniteTestSort(t, m, 3, "dl-column-three")
	b := astArrayFiniteTestSort(t, m, 5, "dl-column-five")
	relation, e := m.MkDatalogRelationSort([]ASTHandle{a, b})
	astTestMust(t, e)
	astTestPin(t, m, relation)
	if yes, e := m.IsDatalogRelationSort(relation); e != nil || !yes {
		t.Fatal("relation recognizer", yes, e)
	}
	columns, e := m.DatalogRelationColumns(relation)
	astTestMust(t, e)
	if len(columns) != 2 || columns[0] != a || columns[1] != b {
		t.Fatal("relation columns", columns, a, b)
	}
	size, e := m.GetSortSize(relation)
	astTestMust(t, e)
	value, e := size.Size()
	astTestMust(t, e)
	if value != 15 {
		t.Fatal("relation cardinality", value)
	}
	empty, e := m.MkDatalogRelationSort(nil)
	astTestMust(t, e)
	astTestPin(t, m, empty)
	size, e = m.GetSortSize(empty)
	astTestMust(t, e)
	value, e = size.Size()
	astTestMust(t, e)
	if value != 1 {
		t.Fatal("zero-column relation cardinality", value)
	}
	rule, e := m.MkDatalogRuleSort()
	astTestMust(t, e)
	astTestPin(t, m, rule)
	if yes, e := m.IsDatalogRuleSort(rule); e != nil || !yes {
		t.Fatal("rule recognizer", yes, e)
	}
	size, e = m.GetSortSize(rule)
	astTestMust(t, e)
	if !size.IsInfinite() {
		t.Fatal("rule cardinality", size)
	}
}

func TestASTDatalogOriginalRelationCardinalityAndFiniteSize(t *testing.T) {
	m := astTestManager(t)
	name := astTestSymbol(t, m, "DL-zero")
	if _, e := m.MkDatalogFiniteSort(name, 0); !errors.Is(e, ErrASTDomain) {
		t.Fatal("utility empty finite sort guard", e)
	}
	large := astArrayFiniteTestSort(t, m, math.MaxUint64, "dl-max")
	two := astArrayFiniteTestSort(t, m, 2, "dl-two")
	relation, e := m.MkDatalogRelationSort([]ASTHandle{large, two})
	astTestMust(t, e)
	astTestPin(t, m, relation)
	size, e := m.GetSortSize(relation)
	astTestMust(t, e)
	if !size.IsVeryBig() {
		t.Fatal("overflowed relation cardinality", size)
	}
	truth, e := m.MkTrue()
	astTestMust(t, e)
	if _, e = m.MkDatalogRelationSort([]ASTHandle{truth}); !errors.Is(e, ErrASTDomain) {
		t.Fatal("nonsort relation column", e)
	}
}
