package protomodel

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestProtoModelCleanupOwnsRewrittenFallbackUntilInstalled(t *testing.T) {
	m, p := protoFixture(t)
	b, err := m.MkBoolSort()
	protoMust(t, err)
	x, err := m.MkVar(0, b)
	protoMust(t, err)
	c, err := m.MkConst(protoSymbol(t, m, "cleanup-late"), b)
	protoMust(t, err)
	yes, err := m.MkTrue()
	protoMust(t, err)
	no, err := m.MkFalse()
	protoMust(t, err)
	body, err := m.MkIte(x, c, yes)
	protoMust(t, err)
	d, err := m.MkFuncDecl(protoSymbol(t, m, "cleanup-rewritten"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	protoMust(t, err)
	protoPin(t, m, d)
	fi, err := model.NewFunctionInterpretation(m, 1)
	protoMust(t, err)
	protoMust(t, fi.SetElse(body))
	protoMust(t, p.Core().RegisterFunction(d, fi))
	protoMust(t, p.Cleanup())
	cleaned, err := fi.Else()
	protoMust(t, err)
	if cleaned == body {
		t.Fatal("cleanup did not replace the uninterpreted constant")
	}
	out := protoPin(t, m, ast.ASTHandle{})
	for _, argument := range []ast.ASTHandle{yes, no} {
		application, err := m.MkApp(d, []ast.ASTHandle{argument})
		protoMust(t, err)
		_, err = p.Eval(application, out, true)
		protoMust(t, err)
		value, err := m.RefGet(out)
		protoMust(t, err)
		if (argument == yes && !m.IsFalse(value)) || (argument != yes && !m.IsTrue(value)) {
			t.Fatal("rewritten fallback lost its Boolean meaning", argument, value)
		}
	}
}
