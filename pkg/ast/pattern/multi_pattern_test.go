package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceMultiPatternSplitOrderLimitAndResultOwnership(t *testing.T) {
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
	app := func(name uint32, arg ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		d, e := m.MkFuncDecl(symbol.Numerical(name), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		pin(d)
		h, e := m.MkApp(d, []ast.ASTHandle{arg})
		patternMust(t, e)
		return pin(h)
	}
	a, d, e := app(741, x), app(742, y), app(743, y)
	c, err := newPatternCandidates(m, 2, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	for _, item := range []struct {
		h ast.ASTHandle
		v uint32
	}{{a, 0}, {d, 1}, {e, 1}} {
		patternMust(t, c.addCandidate(item.h, []uint32{item.v}, 2))
	}
	check := func(patterns []ast.ASTRef, want [][]ast.ASTHandle) {
		t.Helper()
		if len(patterns) != len(want) {
			t.Fatal("multi-pattern count", len(patterns), len(want))
		}
		for i, owned := range patterns {
			h, er := m.RefGet(owned)
			patternMust(t, er)
			family, er := m.GetFamilyID(h)
			patternMust(t, er)
			kind, er := m.GetDeclKind(h)
			patternMust(t, er)
			count, er := m.GetNumArgs(h)
			patternMust(t, er)
			if family != 2 || kind != 0 || int(count) != len(want[i]) {
				t.Fatal("source multi-pattern declaration", family, kind, count)
			}
			for j, childWant := range want[i] {
				child, er := m.GetArg(h, uint32(j))
				patternMust(t, er)
				if child != childWant {
					t.Fatal("source pre-pattern branch order", i, j, child, childWant)
				}
			}
		}
	}
	patterns, err := c.selectMulti([]ast.ASTHandle{a, d, e}, 2, nil)
	patternMust(t, err)
	check(patterns, [][]ast.ASTHandle{{a, d}, {a, e}})
	limited, err := c.selectMulti([]ast.ASTHandle{a, d, e}, 1, nil)
	patternMust(t, err)
	check(limited, [][]ast.ASTHandle{{a, d}})
	patternMust(t, c.close())
	check(patterns, [][]ast.ASTHandle{{a, d}, {a, e}})
	for i := range limited {
		patternMust(t, m.DelRef(&limited[i]))
	}
	for i := range patterns {
		patternMust(t, m.DelRef(&patterns[i]))
	}
}

func TestPatternInferenceMultiPatternThirtyTwoSplitCap(t *testing.T) {
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
	c, err := newPatternCandidates(m, 2, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	var candidates []ast.ASTHandle
	for i := uint32(0); i < 34; i++ {
		d, e := m.MkFuncDecl(symbol.Numerical(750+i), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		pin(d)
		arg, variable := y, uint32(1)
		if i == 0 {
			arg, variable = x, 0
		}
		h, e := m.MkApp(d, []ast.ASTHandle{arg})
		patternMust(t, e)
		pin(h)
		patternMust(t, c.addCandidate(h, []uint32{variable}, 2))
		candidates = append(candidates, h)
	}
	patterns, err := c.selectMulti(candidates, 100, nil)
	patternMust(t, err)
	// With all skip branches, candidate 0 could pair with each of the 33
	// later candidates. The source's 32-split cap stops after candidate 17.
	if len(patterns) != 17 {
		t.Fatal("source split cap changed completed pattern count", len(patterns))
	}
	for i, owned := range patterns {
		h, e := m.RefGet(owned)
		patternMust(t, e)
		first, e := m.GetArg(h, 0)
		patternMust(t, e)
		second, e := m.GetArg(h, 1)
		patternMust(t, e)
		if first != candidates[0] || second != candidates[i+1] {
			t.Fatal("source capped branch order", i, first, second)
		}
		patternMust(t, m.DelRef(&patterns[i]))
	}
	patternMust(t, c.close())
}
