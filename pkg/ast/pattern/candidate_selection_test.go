package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceStableWeightAndUnaryPatternOwnership(t *testing.T) {
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
	decl := func(name uint32, arity int) ast.ASTHandle {
		t.Helper()
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = b
		}
		h, e := m.MkFuncDecl(symbol.Numerical(name), domain, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(h)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		patternMust(t, e)
		return pin(h)
	}
	f, g, h := decl(731, 1), decl(732, 1), decl(733, 2)
	a, d := app(f, x), app(g, x)
	u, v := app(h, x, y), app(h, a, y)
	c, err := newPatternCandidates(m, 2, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	for _, item := range []struct {
		h    ast.ASTHandle
		vars []uint32
		size uint32
	}{
		{d, []uint32{0}, 2}, {v, []uint32{0, 1}, 4},
		{a, []uint32{0}, 2}, {u, []uint32{0, 1}, 3},
	} {
		patternMust(t, c.addCandidate(item.h, item.vars, item.size))
	}
	ordered := []ast.ASTHandle{d, v, a, u}
	patternMust(t, c.sortByWeight(ordered))
	if ordered[0] != u || ordered[1] != v || ordered[2] != d || ordered[3] != a {
		t.Fatal("source weight order or stable equal-weight tie", ordered)
	}
	// The source creates unary patterns before sorting only the remaining
	// candidates for multi-pattern construction.
	patterns, remaining, err := c.selectUnary([]ast.ASTHandle{d, v, a, u})
	patternMust(t, err)
	if len(patterns) != 2 || len(remaining) != 2 || remaining[0] != d || remaining[1] != a {
		t.Fatal("unary selection and remaining candidate order", patterns, remaining)
	}
	patternMust(t, c.sortByWeight(remaining))
	if remaining[0] != d || remaining[1] != a {
		t.Fatal("equal-weight remaining candidates must remain stable", remaining)
	}
	for i, want := range []ast.ASTHandle{v, u} {
		p, e := m.RefGet(patterns[i])
		patternMust(t, e)
		family, e := m.GetFamilyID(p)
		patternMust(t, e)
		kind, e := m.GetDeclKind(p)
		patternMust(t, e)
		child, e := m.GetArg(p, 0)
		patternMust(t, e)
		if family != 2 || kind != 0 || child != want {
			t.Fatal("source singleton pattern factory", family, kind, child, want)
		}
	}
	patternMust(t, c.close())
	for i := range patterns {
		if _, e := m.RefGet(patterns[i]); e != nil {
			t.Fatal("result buffer must own pattern after candidates close", i, e)
		}
		patternMust(t, m.DelRef(&patterns[i]))
	}
}
