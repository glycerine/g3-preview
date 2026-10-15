package ast

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestExprAbstractorNativePatternAndNestedBinder(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	constantDecl, e := m.MkFuncDecl(astTestSymbol(t, m, "abstract-c"), nil, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	constant, e := m.MkApp(constantDecl, nil)
	astTestMust(t, e)
	fn, e := m.MkFuncDecl(astTestSymbol(t, m, "abstract-f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	body, e := m.MkApp(fn, []ASTHandle{constant})
	astTestMust(t, e)
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{body})
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "inner")}, Body: body, Patterns: []ASTHandle{pattern}})
	astTestMust(t, e)
	astTestPin(t, m, q)
	out, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	a, e := NewExprAbstractor(m)
	astTestMust(t, e)
	if e = a.Abstract(0, []ASTHandle{constant}, q, out); e != nil {
		t.Fatal(e)
	}
	result, e := m.RefGet(out)
	astTestMust(t, e)
	if result == q {
		t.Fatal("quantifier was not rebuilt")
	}
	rewrittenBody, e := m.GetQuantifierBody(result)
	astTestMust(t, e)
	arg, e := m.GetArg(rewrittenBody, 0)
	astTestMust(t, e)
	index, e := m.GetVarIndex(arg)
	astTestMust(t, e)
	if index != 1 {
		t.Fatal("outer binder index under inner quantifier", index)
	}
	rewrittenPattern, e := m.GetPattern(result, 0)
	astTestMust(t, e)
	patternBody, e := m.GetArg(rewrittenPattern, 0)
	astTestMust(t, e)
	if patternBody != rewrittenBody {
		t.Fatal("pattern and body did not share the rewritten application")
	}
	if len(a.pinned) != 0 {
		t.Fatal("temporary expression references were not released")
	}
	if e = a.Abstract(0, nil, constant, out); e != nil {
		t.Fatal("reused abstractor", e)
	}
	if got, e := m.RefGet(out); e != nil || got != constant {
		t.Fatal("zero-bound source shortcut", got, e)
	}
	if e = a.Abstract(0, []ASTHandle{constant}, q, out); e != nil {
		t.Fatal("reused abstractor with binder", e)
	}
	result, e = m.RefGet(out)
	astTestMust(t, e)
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if got, e := m.RefGet(out); e != nil || got != result {
		t.Fatal("result lost after temporary pins released", got, e)
	}
}

func TestExprAbstractorRetainsInterruptedTemporaryUntilClose(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "abstract-fault-c"), nil, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	c, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, c)
	out, e := m.NewRef(c)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	a, e := NewExprAbstractor(m)
	astTestMust(t, e)
	if e = a.Abstract(0, []ASTHandle{c, {}}, c, out); !errors.Is(e, ErrASTHandle) {
		t.Fatal("invalid second bound constant", e)
	}
	if len(a.pinned) != 1 {
		t.Fatal("first generated variable was not retained for cleanup")
	}
	if got, e := m.RefGet(out); e != nil || got != c {
		t.Fatal("failed abstraction changed output", got, e)
	}
	if e = a.Close(); e != nil {
		t.Fatal("temporary reference cleanup", e)
	}
	if len(a.pinned) != 0 || !a.closed {
		t.Fatal("cleanup did not finish")
	}
}
