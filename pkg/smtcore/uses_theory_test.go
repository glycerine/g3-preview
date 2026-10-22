package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTUsesTheoryTraversesQuantifierAnnotations(t *testing.T) {
	m := astTestManager(t)
	arithmetic, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	integer, err := m.GetSort(zero)
	astTestMust(t, err)
	variable, err := m.MkVar(0, integer)
	astTestMust(t, err)
	annotation, err := m.MkFamilyApp(arithmetic, smtArithOpAdd, nil, []ast.ASTHandle{variable, zero})
	astTestMust(t, err)
	truth, err := m.MkBoolVal(true)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{symbol.Numerical(1801)},
		Body: truth, NoPatterns: []ast.ASTHandle{annotation},
	})
	astTestMust(t, err)
	found, err := SMTUsesTheory(m, quantifier, arithmetic)
	astTestMust(t, err)
	if !found {
		t.Fatal("theory use in quantifier annotation was missed")
	}
	found, err = SMTUsesTheory(m, quantifier, m.BasicFamilyID())
	astTestMust(t, err)
	if !found {
		t.Fatal("theory use in quantifier body was missed")
	}
}

func TestSMTUsesTheorySharedVisitedSkipsPreviouslyMarkedSubgraph(t *testing.T) {
	m := astTestManager(t)
	arithmetic, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	two, err := m.ArithmeticIntegerNumeralText("2")
	astTestMust(t, err)
	sum, err := m.MkArithmeticAdd(one, two)
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(symbol.Numerical(1802), boolean)
	astTestMust(t, err)
	root, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{p, p})
	astTestMust(t, err)
	visited, err := ast.NewASTExprMark(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, visited.Close()) })
	found, err := SMTUsesTheoryWithMark(m, root, arithmetic, visited)
	astTestMust(t, err)
	if found {
		t.Fatal("unrelated expression reported arithmetic")
	}
	found, err = SMTUsesTheoryWithMark(m, sum, arithmetic, visited)
	astTestMust(t, err)
	if !found {
		t.Fatal("new arithmetic subgraph was not visited")
	}
	found, err = SMTUsesTheoryWithMark(m, sum, arithmetic, visited)
	astTestMust(t, err)
	if found {
		t.Fatal("previously marked root was revisited")
	}
}
