package ast

import (
	"errors"
	"testing"
)

func TestASTDatalogOriginalFilterTraversalAndGuards(t *testing.T) {
	m := astTestManager(t)
	a, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "filter-a"), 3)
	astTestMust(t, e)
	b, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "filter-b"), 5)
	astTestMust(t, e)
	sort, e := m.MkDatalogRelationSort([]ASTHandle{a, b})
	astTestMust(t, e)
	relation, e := m.MkDatalogEmpty(sort)
	astTestMust(t, e)
	x, e := m.MkVar(0, a)
	astTestMust(t, e)
	predicate, e := m.MkEq(x, x)
	astTestMust(t, e)
	filter, e := m.MkDatalogFilter(relation, predicate)
	astTestMust(t, e)
	astTestPin(t, m, filter)
	if kind, e := m.GetDeclKind(filter); e != nil || kind != DatalogOpFilter {
		t.Fatal("filter declaration", kind, e)
	}
	got, e := m.GetSort(filter)
	astTestMust(t, e)
	if got != sort {
		t.Fatal("filter range", got, sort)
	}
	outOfRange, e := m.MkVar(2, a)
	astTestMust(t, e)
	badIndex, e := m.MkEq(outOfRange, outOfRange)
	astTestMust(t, e)
	wrongSort, e := m.MkVar(0, b)
	astTestMust(t, e)
	badSort, e := m.MkEq(wrongSort, wrongSort)
	astTestMust(t, e)
	for name, value := range map[string]ASTHandle{
		"non-Boolean": x,
		"index":       badIndex,
		"sort":        badSort,
	} {
		if _, e := m.MkDatalogFilter(relation, value); !errors.Is(e, ErrASTDomain) {
			t.Fatal(name, e)
		}
	}
}

func TestASTDatalogOriginalRuleRepAndAbs(t *testing.T) {
	m := astTestManager(t)
	a, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "conversion-a"), 3)
	astTestMust(t, e)
	b, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "conversion-b"), 5)
	astTestMust(t, e)
	for _, tc := range []struct {
		kind int32
		name string
		mk   func() (ASTHandle, error)
	}{
		{DatalogOpRep, "rep", func() (ASTHandle, error) { return m.MkDatalogRepDecl(a, b) }},
		{DatalogOpAbs, "abs", func() (ASTHandle, error) { return m.MkDatalogAbsDecl(a, b) }},
	} {
		decl, e := tc.mk()
		astTestMust(t, e)
		astTestPin(t, m, decl)
		kind, e := m.GetDeclKind(decl)
		astTestMust(t, e)
		rangeSort, e := m.GetRange(decl)
		astTestMust(t, e)
		name, e := m.GetASTName(decl)
		astTestMust(t, e)
		text, e := m.Symbols.Display(name)
		astTestMust(t, e)
		if kind != tc.kind || rangeSort != b || text != tc.name {
			t.Fatal("conversion declaration", kind, rangeSort, text)
		}
	}
	one, e := m.MkDatalogNumeral(1, a)
	astTestMust(t, e)
	rule, e := m.MkDatalogRule(astTestSymbol(t, m, "rule-name"), []ASTHandle{one})
	astTestMust(t, e)
	astTestPin(t, m, rule)
	ruleSort, e := m.GetSort(rule)
	astTestMust(t, e)
	if yes, e := m.IsDatalogRuleSort(ruleSort); e != nil || !yes {
		t.Fatal("rule application range", yes, e)
	}
}
