package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferencePreferredPatternsCompleteAndOwned(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	pin(x)
	y, err := m.MkVar(1, b)
	patternMust(t, err)
	pin(y)
	decl := func(name uint32) ast.ASTHandle {
		t.Helper()
		h, e := m.MkFuncDecl(symbol.Numerical(name), []ast.ASTHandle{b, b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(h)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		patternMust(t, e)
		return pin(h)
	}
	f, g, h := decl(761), decl(762), decl(763)
	first, incomplete, middle, last := app(f, x, y), app(f, x, x), app(h, x, y), app(g, x, y)
	c, err := newPatternCandidates(m, 2, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	for _, item := range []struct {
		h    ast.ASTHandle
		vars []uint32
	}{
		{first, []uint32{0, 1}}, {incomplete, []uint32{0}},
		{middle, []uint32{0, 1}}, {last, []uint32{0, 1}},
	} {
		patternMust(t, c.addCandidate(item.h, item.vars, 1))
	}
	candidates := []ast.ASTHandle{first, incomplete, middle, last}
	if got, found, e := c.selectPreferred(candidates); e != nil || found || len(got) != 0 {
		t.Fatal("empty preferred set must return false", got, found, e)
	}
	base, err := m.GetRefCount(f)
	patternMust(t, err)
	patternMust(t, c.registerPreferred(f))
	patternMust(t, c.registerPreferred(f))
	patternMust(t, c.registerPreferred(g))
	if refs, e := m.GetRefCount(f); e != nil || refs != base {
		t.Fatal("preferred declaration set must borrow identity", refs, base, e)
	}
	patterns, found, err := c.selectPreferred(candidates)
	patternMust(t, err)
	if !found || len(patterns) != 2 {
		t.Fatal("only complete preferred candidates should be selected", found, len(patterns))
	}
	for i, want := range []ast.ASTHandle{first, last} {
		p, e := m.RefGet(patterns[i])
		patternMust(t, e)
		family, e := m.GetFamilyID(p)
		patternMust(t, e)
		kind, e := m.GetDeclKind(p)
		patternMust(t, e)
		child, e := m.GetArg(p, 0)
		patternMust(t, e)
		if family != 2 || kind != 0 || child != want {
			t.Fatal("source preferred singleton pattern and order", family, kind, child, want)
		}
	}
	patternMust(t, c.close())
	for i := range patterns {
		if _, e := m.RefGet(patterns[i]); e != nil {
			t.Fatal("preferred output must survive candidate cleanup", i, e)
		}
		patternMust(t, m.DelRef(&patterns[i]))
	}
}
