// Source: src/ast/used_symbols.h.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTUsedSymbolsApplicationsQuantifiersAndCoreOperations(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	nameF := astTestSymbol(t, m, "used-f")
	nameG := astTestSymbol(t, m, "used-g")
	binder := astTestSymbol(t, m, "used-x")
	qid := astTestSymbol(t, m, "used-qid")
	f, e := m.MkFuncDecl(nameF, []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	g, e := m.MkFuncDecl(nameG, []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	fx, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	gx, e := m.MkApp(g, []ASTHandle{x})
	astTestMust(t, e)
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{gx})
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []symbol.Symbol{binder}, Body: fx, QID: qid, Patterns: []ASTHandle{pattern}})
	astTestMust(t, e)
	astTestPin(t, m, q)

	u, e := NewUsedSymbols(m)
	astTestMust(t, e)
	astTestMust(t, u.Collect(q, false))
	for _, s := range []symbol.Symbol{qid, binder, nameF, nameG} {
		if !u.Contains(s) {
			t.Fatal("missing symbol", s)
		}
	}
	u.Erase(nameG)
	if u.Contains(nameG) {
		t.Fatal("erase core")
	}
	u.Insert(nameG)
	if !u.Contains(nameG) {
		t.Fatal("insert core")
	}
	astTestMust(t, u.Collect(q, true))
	for _, s := range []symbol.Symbol{qid, binder, nameF, nameG} {
		if u.Contains(s) {
			t.Fatal("ignored quantifier leaked symbol", s)
		}
	}
}
