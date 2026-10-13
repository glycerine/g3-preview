package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func patternMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func patternManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	patternMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	patternMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	patternMust(t, err)
	t.Cleanup(func() {
		patternMust(t, m.Close())
		patternMust(t, q.Close())
		patternMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("pattern manager retained bytes", b.Used())
		}
	})
	return m
}

func TestSmallerPatternConcretePairCacheAndExternalVariables(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
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
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	pin(x)
	y, err := m.MkVar(1, b)
	patternMust(t, err)
	pin(y)
	z, err := m.MkVar(2, b)
	patternMust(t, err)
	pin(z)
	f, g, h := decl(701, 2), decl(702, 2), decl(703, 1)
	a, c := app(decl(704, 0)), app(decl(705, 0))
	s, err := newSmallerPattern(m)
	patternMust(t, err)
	check := func(n uint32, left, right ast.ASTHandle, want bool) {
		t.Helper()
		got, e := s.smaller(n, left, right)
		patternMust(t, e)
		if got != want {
			t.Fatalf("smaller(%d, %v, %v) = %t, want %t", n, left, right, got, want)
		}
	}
	check(1, app(f, x, x), app(f, a, a), true)
	check(1, app(f, x, x), app(f, a, c), false)
	if s.bindings[0] != c {
		t.Fatal("last argument must bind first before the mismatch", s.bindings)
	}
	check(2, app(f, x, y), app(f, a, c), true)
	check(1, app(f, x, z), app(f, a, z), true)
	check(1, app(f, x, z), app(f, a, y), false)
	check(0, app(f, a, c), app(g, a, c), false)
	check(1, x, app(h, a), true) // A bound pattern var accepts any AST kind.
	check(0, x, a, false)        // The same variable is external with no bindings.

	left := app(f, app(h, x), app(h, x))
	right := app(f, app(h, a), app(h, a))
	check(1, left, right, true)
	if len(s.cache) != 3 || len(s.todo) != 0 || s.bindings[0] != a {
		t.Fatal("shared child pairs must be visited once and retain one binding", len(s.cache), s.todo, s.bindings)
	}
	check(1, x, c, true)
	if len(s.cache) != 1 || s.bindings[0] != c {
		t.Fatal("each comparison resets the borrowed pair cache and bindings")
	}
}
