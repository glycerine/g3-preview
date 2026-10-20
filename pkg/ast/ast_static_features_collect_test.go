package ast

import "testing"

func TestStaticFeaturesBooleanTraversalClausesSharingAndDepth(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "sf-a"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	b, err := m.MkConst(astTestSymbol(t, m, "sf-b"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, b)
	shared, err := m.MkAnd(a, b)
	astTestMust(t, err)
	astTestPin(t, m, shared)
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	astTestPin(t, m, notA)
	clause, err := m.MkOr(shared, notA)
	astTestMust(t, err)
	astTestPin(t, m, clause)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	astTestMust(t, f.Collect(clause, shared))
	if f.NumRoots != 1 || f.NumClauses != 1 || f.NumBinClauses != 1 || f.SumClauseSize != 2 {
		t.Fatal("root clause accounting", f.NumRoots, f.NumClauses, f.NumBinClauses, f.SumClauseSize)
	}
	if f.NumSharing < 2 || f.MaxDepth != 3 || f.NumFormulaTrees != 1 || f.NumOrAndTrees != 1 {
		t.Fatal("tree accounting", f.NumSharing, f.MaxDepth, f.NumFormulaTrees, f.NumOrAndTrees)
	}
	if f.CNF || f.NumANDs != 1 || f.NumBoolExprs < 3 {
		t.Fatal("gate accounting", f.CNF, f.NumANDs, f.NumBoolExprs)
	}
}

func TestStaticFeaturesTermITEIsNotAFormulaTree(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	condition, err := m.MkConst(astTestSymbol(t, m, "sf-ite-condition"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, condition)
	termSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "sf-ite-sort"), nil)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "sf-ite-left"), termSort)
	astTestMust(t, err)
	astTestPin(t, m, left)
	right, err := m.MkConst(astTestSymbol(t, m, "sf-ite-right"), termSort)
	astTestMust(t, err)
	astTestPin(t, m, right)
	ite, err := m.MkIte(condition, left, right)
	astTestMust(t, err)
	astTestPin(t, m, ite)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	astTestMust(t, f.Collect(ite))
	if f.NumITETerms != 1 || f.NumITEFormulas != 0 || f.NumITETrees != 1 {
		t.Fatal("term ITE accounting", f.NumITETerms, f.NumITEFormulas, f.NumITETrees)
	}
	if f.NumFormulaTrees != 0 || f.MaxFormulaDepth != 0 {
		t.Fatal("term ITE counted as formula tree", f.NumFormulaTrees, f.MaxFormulaDepth)
	}
}
