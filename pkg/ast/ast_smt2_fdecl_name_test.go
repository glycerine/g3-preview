package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTSMT2FunctionNameSourceSymbolBranches(t *testing.T) {
	m := astTestManager(t)
	simple, err := m.Symbols.Intern("plain-name")
	astTestMust(t, err)
	quoted, err := m.Symbols.Intern("a b|\\c")
	astTestMust(t, err)
	alreadyQuoted, err := m.Symbols.Intern("|spaced name|")
	astTestMust(t, err)
	for _, tc := range []struct {
		name usymbol.Symbol
		want string
	}{
		{simple, "plain-name"},
		{quoted, "|a b\\|\\\\c|"},
		{alreadyQuoted, "|spaced name|"},
		{usymbol.Numerical(17), "k!17"},
		{usymbol.Symbol{}, "null"},
	} {
		got, length, e := m.SMT2FdeclName(tc.name)
		astTestMust(t, e)
		want, e := m.FormatString(tc.want)
		astTestMust(t, e)
		if got != want || length != uint32(len(tc.want)) {
			t.Fatalf("name %#v: format=%v want=%v length=%d want=%d", tc.name, got, want, length, len(tc.want))
		}
	}
}

func TestASTSMT2FunctionNameSourceBuiltinAndUserDeclarations(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkFreshConst("p", boolSort)
	astTestMust(t, err)
	q, err := m.MkFreshConst("q", boolSort)
	astTestMust(t, err)
	implies, err := m.MkImplies(p, q)
	astTestMust(t, err)
	ite, err := m.MkIte(p, q, p)
	astTestMust(t, err)
	iff, err := m.MkIff(p, q)
	astTestMust(t, err)
	name, err := m.Symbols.Intern("iff")
	astTestMust(t, err)
	user, err := m.MkFuncDecl(name, nil, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	for _, tc := range []struct {
		decl ASTHandle
		want string
	}{
		{implies, "=>"},
		{ite, "ite"},
		{iff, "="},
		{user, "iff"},
	} {
		decl := tc.decl
		if decl != user {
			decl, err = m.GetDecl(decl)
			astTestMust(t, err)
		}
		got, length, e := m.SMT2FdeclNameDecl(decl)
		astTestMust(t, e)
		want, e := m.FormatString(tc.want)
		astTestMust(t, e)
		if got != want || length != uint32(len(tc.want)) {
			t.Fatalf("decl %v: format=%v want=%v length=%d", decl, got, want, length)
		}
	}
}
