package ast

import (
	"strings"
	"testing"
)

func TestASTPPUtilCollectRemoveDeclarationsAndAssertions(t *testing.T) {
	m := astTestManager(t)
	util, err := NewASTPPUtil(m)
	astTestMust(t, err)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "pp-A"), nil)
	astTestMust(t, err)
	astTestPin(t, m, sort)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "pp-a"), sort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "pp-f"), []ASTHandle{sort}, sort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, f)
	fa, err := m.MkApp(f, []ASTHandle{a})
	astTestMust(t, err)
	astTestPin(t, m, fa)
	predicate, err := m.MkFuncDecl(astTestSymbol(t, m, "pp-p"), []ASTHandle{sort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, predicate)
	root, err := m.MkApp(predicate, []ASTHandle{fa})
	astTestMust(t, err)
	astTestPin(t, m, root)
	astTestMust(t, util.CollectAll([]ASTHandle{root}))
	astTestMust(t, util.RemoveDeclaration(f))
	var out strings.Builder
	astTestMust(t, util.DisplayDeclarations(&out))
	want := "(declare-sort pp-A)\n(declare-fun pp-p (pp-A) Bool)\n(declare-fun pp-a () pp-A)\n"
	if out.String() != want {
		t.Fatalf("AST pp declarations %q, want %q", out.String(), want)
	}
	for _, neat := range []bool{true, false} {
		out.Reset()
		astTestMust(t, util.DisplayAssertions(&out, []ASTHandle{root}, neat))
		if out.String() != "(assert (pp-p (pp-f pp-a)))\n" {
			t.Fatalf("AST pp assertions neat=%t: %q", neat, out.String())
		}
	}
}
