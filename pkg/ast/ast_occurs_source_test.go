// Source: src/ast/occurs.cpp:26-76; src/ast/occurs.h:22-33.
package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTOccursExpressionDeclarationAndQuantifierAnnotations(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "occurs-x"), b)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkConst(astTestSymbol(t, m, "occurs-y"), b)
	astTestMust(t, e)
	astTestPin(t, m, y)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "occurs-f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fx, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, fx)
	root, e := m.MkAnd(fx, y)
	astTestMust(t, e)
	astTestPin(t, m, root)
	for _, row := range []struct {
		needle ASTHandle
		want   bool
	}{{root, true}, {fx, true}, {x, true}} {
		got, err := m.OccursExpr(row.needle, root)
		astTestMust(t, err)
		if got != row.want {
			t.Fatal("expression occurrence", row.needle, got)
		}
	}
	missing, e := m.MkConst(astTestSymbol(t, m, "occurs-missing"), b)
	astTestMust(t, e)
	got, e := m.OccursExpr(missing, root)
	astTestMust(t, e)
	if got {
		t.Fatal("absent expression reported present")
	}
	got, e = m.OccursDecl(f, root)
	astTestMust(t, e)
	if !got {
		t.Fatal("application declaration not found")
	}
	missingDecl, e := m.MkFuncDecl(astTestSymbol(t, m, "occurs-g"), nil, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	got, e = m.OccursDecl(missingDecl, root)
	astTestMust(t, e)
	if got {
		t.Fatal("absent declaration reported present")
	}
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{fx})
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "q")}, Body: y, Patterns: []ASTHandle{pattern}})
	astTestMust(t, e)
	astTestPin(t, m, q)
	got, e = m.OccursExpr(fx, q)
	astTestMust(t, e)
	if !got {
		t.Fatal("quick_for_each_expr must visit quantifier patterns")
	}
	got, e = m.OccursDecl(f, q)
	astTestMust(t, e)
	if !got {
		t.Fatal("declaration in quantifier pattern not found")
	}
}

func TestASTOccursDeepPostorderAndGuards(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, x)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "occurs-chain"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	root := x
	for i := 0; i < 4096; i++ {
		root, e = m.MkApp(f, []ASTHandle{root})
		astTestMust(t, e)
		astTestPin(t, m, root)
	}
	got, e := m.OccursExpr(x, root)
	astTestMust(t, e)
	if !got {
		t.Fatal("deep expression occurrence lost")
	}
	got, e = m.OccursDecl(f, root)
	astTestMust(t, e)
	if !got {
		t.Fatal("deep declaration occurrence lost")
	}
	if _, e = m.OccursExpr(b, root); e == nil {
		t.Fatal("non-expression needle accepted")
	}
	if _, e = m.OccursDecl(b, root); e == nil {
		t.Fatal("non-declaration needle accepted")
	}
}
