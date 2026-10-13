package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceRepeatedBodiesResetCandidatesKeepPreferences(t *testing.T) {
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
	decl := func(name uint32) ast.ASTHandle {
		t.Helper()
		d, e := m.MkFuncDecl(symbol.Numerical(name), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(d)
	}
	app := func(d ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, []ast.ASTHandle{x})
		patternMust(t, e)
		return pin(h)
	}
	f, g := decl(821), decl(822)
	fx, gx := app(f), app(g)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{fx, gx})
	patternMust(t, err)
	pin(body)
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	check := func(patterns []ast.ASTRef, want ast.ASTHandle) {
		t.Helper()
		if len(patterns) != 1 {
			t.Fatal("expected one pattern", len(patterns))
		}
		p, e := m.RefGet(patterns[0])
		patternMust(t, e)
		child, e := m.GetArg(p, 0)
		patternMust(t, e)
		if child != want {
			t.Fatal("source selected candidate", child, want)
		}
		if len(c.candidates) != 0 || len(c.info) != 0 || len(c.collectRefs) != 0 || c.noPatterns != nil {
			t.Fatal("each inference call must reset candidate state")
		}
	}
	first, err := c.inferPatternBody(fx, 1, nil, 0)
	patternMust(t, err)
	check(first, fx)
	if got, e := c.inferPatternBody(fx, 1, []ast.ASTHandle{fx}, 0); e != nil || len(got) != 0 {
		t.Fatal("per-call excluded candidate", got, e)
	}
	patternMust(t, c.registerPreferred(g))
	preferred, err := c.inferPatternBody(body, 1, nil, 0)
	patternMust(t, err)
	check(preferred, gx)
	if len(c.preferred) != 1 {
		t.Fatal("preferred declaration should survive per-call reset")
	}
	fallback, err := c.inferPatternBody(body, 1, []ast.ASTHandle{gx}, 0)
	patternMust(t, err)
	check(fallback, fx)
	if got, e := c.inferPatternBody(fx, 2, nil, 0); e != nil || len(got) != 0 {
		t.Fatal("per-call binding count should govern variable coverage", got, e)
	}
	patternMust(t, c.close())
	for _, results := range [][]ast.ASTRef{first, preferred, fallback} {
		for i := range results {
			if _, e := m.RefGet(results[i]); e != nil {
				t.Fatal("returned patterns own their ASTs after owner close", e)
			}
			patternMust(t, m.DelRef(&results[i]))
		}
	}
}
