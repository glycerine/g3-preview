// Source: src/ast/array_decl_plugin.cpp:566-569;
// src/ast/ast.cpp:1568-1638.
package ast

import "testing"

func TestASTArrayUtilFamilyAndBorrowedApplications(t *testing.T) {
	m := astTestManager(t)
	family, e := m.ArrayFamilyID()
	astTestMust(t, e)
	registered, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	if family != registered {
		t.Fatal("family constructor registered a different name")
	}
	_, sort, _ := arrayDeclFixture(t, m, 1)
	symbol := astTestSymbol(t, m, "borrowed-array-base")
	decl, e := m.MkFuncDecl(symbol, nil, sort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	base, e := m.MkApp(decl, nil)
	astTestMust(t, e)
	astTestPin(t, m, base)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	stored, e := m.MkFamilyApp(family, ASTArrayOpStore, nil, []ASTHandle{base, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, stored)
	borrowed, e := m.AppArgumentView(stored)
	astTestMust(t, e)
	same, e := m.MkArrayAppArguments(family, ASTArrayOpStore, borrowed)
	astTestMust(t, e)
	if same != stored {
		t.Fatal("borrowed store application changed identity")
	}
	args := []ASTHandle{stored, yes}
	selected, e := m.MkFamilyApp(family, ASTArrayOpSelect, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, selected)
	borrowed, e = m.AppArgumentView(selected)
	astTestMust(t, e)
	same, e = m.MkArrayAppArguments(family, ASTArrayOpSelect, borrowed)
	astTestMust(t, e)
	if same != selected {
		t.Fatal("borrowed select application changed identity")
	}
}
