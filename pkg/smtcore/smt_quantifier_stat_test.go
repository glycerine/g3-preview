package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTQuantifierStatSourceShapeAndCaseSplitFactor(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	vars := make([]ast.ASTHandle, 3)
	for i := range vars {
		vars[i], err = m.MkConst(symbol.Numerical(uint32(1900+i)), boolean)
		astTestMust(t, err)
	}
	disjunction, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpOr, nil, vars)
	astTestMust(t, err)
	bound, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	nested, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{symbol.Numerical(1903)}, Body: bound})
	astTestMust(t, err)
	conjunction, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{nested, vars[0]})
	astTestMust(t, err)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpIff, nil, []ast.ASTHandle{disjunction, conjunction})
	astTestMust(t, err)
	outer, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{symbol.Numerical(1904)}, Body: body})
	astTestMust(t, err)
	generator, err := NewSMTQuantifierStatGenerator(m)
	astTestMust(t, err)
	stat, err := generator.Generate(outer, 17)
	astTestMust(t, err)
	if stat.Size() != 7 || stat.Depth() != 2 || stat.Generation() != 17 || stat.CaseSplitFactor() != 48 || stat.NumNestedQuantifiers() != 1 {
		t.Fatal("source structural statistics", stat.Size(), stat.Depth(), stat.Generation(), stat.CaseSplitFactor(), stat.NumNestedQuantifiers())
	}
}

func TestSMTQuantifierStatDepthOnlyRevisitAndCounters(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(symbol.Numerical(1910), boolean)
	astTestMust(t, err)
	b, err := m.MkConst(symbol.Numerical(1911), boolean)
	astTestMust(t, err)
	shared, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpOr, nil, []ast.ASTHandle{a, b})
	astTestMust(t, err)
	negated, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpNot, nil, []ast.ASTHandle{shared})
	astTestMust(t, err)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{shared, negated})
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{symbol.Numerical(1912)}, Body: body})
	astTestMust(t, err)
	generator, err := NewSMTQuantifierStatGenerator(m)
	astTestMust(t, err)
	stat, err := generator.Generate(quantifier, 2)
	astTestMust(t, err)
	if stat.Size() != 5 || stat.Depth() != 3 || stat.CaseSplitFactor() != 3 {
		t.Fatal("depth-only revisit", stat.Size(), stat.Depth(), stat.CaseSplitFactor())
	}
	stat.IncrementInstances()
	stat.IncrementInstancesCurrentBranch()
	stat.UpdateMaxGeneration(9)
	stat.UpdateMaxGeneration(4)
	stat.UpdateMaxCost(2.5)
	stat.UpdateMaxCost(1.5)
	if stat.NumInstances() != 1 || stat.NumInstancesCurrentSearch() != 1 || stat.NumInstancesCurrentBranch() != 1 || stat.MaxGeneration() != 9 || stat.MaxCost() != 2.5 {
		t.Fatal("source mutable counters")
	}
	stat.ResetInstancesCurrentSearch()
	if stat.NumInstancesCurrentSearch() != 0 || stat.NumInstances() != 1 {
		t.Fatal("search reset changed lifetime count")
	}
}
