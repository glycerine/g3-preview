package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTNamedNodeFieldsAndSortNullInfoPredicates(t *testing.T) {
	m := astTestManager(t)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	name := astTestSymbol(t, m, "NullInfoSort")
	plain, e := m.MkSort(name, ASTSortInfo{})
	astTestMust(t, e)
	astTestPin(t, m, plain)
	infinite, e := m.SortIsInfinite(plain)
	astTestMust(t, e)
	veryBig, e := m.SortIsVeryBig(plain)
	astTestMust(t, e)
	if !infinite || !veryBig {
		t.Fatal("source null info returns both predicates true")
	}
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "predicate"), []ASTHandle{plain}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, decl)
	arity, e := m.GetArity(decl)
	astTestMust(t, e)
	domain, e := m.GetDomain(decl, 0)
	astTestMust(t, e)
	rangeSort, e := m.GetRange(decl)
	astTestMust(t, e)
	if arity != 1 || domain != plain || rangeSort != boolSort {
		t.Fatal("declaration fields")
	}
	variable, e := m.MkVar(0, plain)
	astTestMust(t, e)
	astTestPin(t, m, variable)
	app, e := m.MkApp(decl, []ASTHandle{variable})
	astTestMust(t, e)
	astTestPin(t, m, app)
	d, e := m.GetDecl(app)
	astTestMust(t, e)
	count, e := m.GetNumArgs(app)
	astTestMust(t, e)
	ground, e := m.IsGround(app)
	astTestMust(t, e)
	if d != decl || count != 1 || ground {
		t.Fatal("actual app fields/flags")
	}
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{plain}, Names: []usymbol.Symbol{name}, Body: app, Weight: 13, QID: name})
	astTestMust(t, e)
	astTestPin(t, m, q)
	body, e := m.GetQuantifierBody(q)
	astTestMust(t, e)
	n, e := m.GetNumDeclarations(q)
	astTestMust(t, e)
	sort, e := m.GetDeclarationSort(q, 0)
	astTestMust(t, e)
	symbol, e := m.GetDeclarationName(q, 0)
	astTestMust(t, e)
	weight, e := m.GetQuantifierWeight(q)
	astTestMust(t, e)
	if body != app || n != 1 || sort != plain || symbol != name || weight != 13 {
		t.Fatal("actual quantifier fields")
	}
}
