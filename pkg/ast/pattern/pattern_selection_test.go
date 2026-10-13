package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceSelectionStagesAndMultiBudget(t *testing.T) {
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
	decl := func(name uint32, args int) ast.ASTHandle {
		t.Helper()
		domain := make([]ast.ASTHandle, args)
		for i := range domain {
			domain[i] = b
		}
		d, e := m.MkFuncDecl(symbol.Numerical(name), domain, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(d)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		patternMust(t, e)
		return pin(h)
	}
	f, g, h := decl(781, 2), decl(782, 1), decl(783, 1)
	full, left, right := app(f, x, y), app(g, x), app(h, y)
	newSet := func(withFull bool) *patternCandidates {
		t.Helper()
		c, e := newPatternCandidates(m, 2, nil, false)
		patternMust(t, e)
		patternMust(t, c.addCandidate(left, []uint32{0}, 2))
		if withFull {
			patternMust(t, c.addCandidate(full, []uint32{0, 1}, 3))
		}
		patternMust(t, c.addCandidate(right, []uint32{1}, 2))
		return c
	}
	check := func(patterns []ast.ASTRef, want ...[]ast.ASTHandle) {
		t.Helper()
		if len(patterns) != len(want) {
			t.Fatalf("pattern count %d, want %d", len(patterns), len(want))
		}
		for i, children := range want {
			p, e := m.RefGet(patterns[i])
			patternMust(t, e)
			count, e := m.GetNumArgs(p)
			patternMust(t, e)
			if int(count) != len(children) {
				t.Fatal("pattern arity", i, count, len(children))
			}
			for j, child := range children {
				got, e := m.GetArg(p, uint32(j))
				patternMust(t, e)
				if got != child {
					t.Fatal("pattern child/order", i, j, got, child)
				}
			}
		}
	}
	release := func(patterns []ast.ASTRef) {
		t.Helper()
		for i := range patterns {
			patternMust(t, m.DelRef(&patterns[i]))
		}
	}
	c := newSet(true)
	patterns, err := c.selectPatterns(0)
	patternMust(t, err)
	check(patterns, []ast.ASTHandle{full})
	release(patterns)
	patterns, err = c.selectPatterns(1)
	patternMust(t, err)
	check(patterns, []ast.ASTHandle{full}, []ast.ASTHandle{left, right})
	release(patterns)
	patternMust(t, c.registerPreferred(g))
	patterns, err = c.selectPatterns(1)
	patternMust(t, err)
	check(patterns, []ast.ASTHandle{full}, []ast.ASTHandle{left, right}) // Incomplete preferred candidate falls through.
	release(patterns)
	patternMust(t, c.registerPreferred(f))
	patterns, err = c.selectPatterns(1)
	patternMust(t, err)
	check(patterns, []ast.ASTHandle{full}) // Complete preferred candidate short-circuits.
	patternMust(t, c.close())
	check(patterns, []ast.ASTHandle{full}) // Result owns its AST after candidate teardown.
	release(patterns)

	partials := newSet(false)
	patterns, err = partials.selectPatterns(0)
	patternMust(t, err)
	check(patterns, []ast.ASTHandle{left, right}) // No unary grants one multi-pattern.
	patternMust(t, partials.close())
	release(patterns)
}
