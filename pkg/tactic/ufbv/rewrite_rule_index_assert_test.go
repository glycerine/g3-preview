package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestRewriteForwardIndexSourcePreconditionsAndRuleGuards(t *testing.T) {
	m := cacheFixture(t)
	if diagnostic.DebugBuild {
		budget := arena.NewBudget(1 << 20)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
		cacheMust(t, e)
		debug, e := diagnostic.NewDebugContext(region, budget)
		cacheMust(t, e)
		previous, e := diagnostic.BindProcessDebugContext(debug)
		cacheMust(t, e)
		t.Cleanup(func() {
			_, e := diagnostic.BindProcessDebugContext(previous)
			cacheMust(t, e)
			cacheMust(t, debug.Close())
			cacheMust(t, region.Close())
		})
	}
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(997), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	g, e := m.MkFuncDecl(symbol.Numerical(998), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	gx, e := m.MkApp(g, []ast.ASTHandle{x})
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(999)}, Body: x})
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{fx, gx, q, x} {
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
	}
	newIndex := func(t *testing.T) *rewriteRuleIndex {
		t.Helper()
		r, e := newRewriteRuleIndex(m)
		cacheMust(t, e)
		t.Cleanup(func() { cacheMust(t, r.close()) })
		return r
	}
	want := func(t *testing.T, e error, release error) {
		t.Helper()
		if diagnostic.DebugBuild {
			if !errors.Is(e, diagnostic.ErrDebugAssertion) {
				t.Fatal("checked source assertion", e)
			}
		} else if !errors.Is(e, release) {
			t.Fatal("erased assertion guard", e, release)
		}
	}
	t.Run("non-app-large-217", func(t *testing.T) {
		r := newIndex(t)
		want(t, r.insert(q, x, x), ast.ErrASTDomain)
		if len(r.byFunc) != 0 || len(r.rules) != 0 {
			t.Fatal("invalid large expression published a rule")
		}
	})
	t.Run("null-demodulator-218", func(t *testing.T) {
		r := newIndex(t)
		want(t, r.insert(ast.ASTHandle{}, fx, x), ast.ErrASTHandle)
	})
	t.Run("null-small-219", func(t *testing.T) {
		r := newIndex(t)
		want(t, r.insert(q, fx, ast.ASTHandle{}), ast.ErrASTDomain)
		if len(r.byFunc) != 0 || len(r.owned) != 0 {
			t.Fatal("null RHS published an index or reference")
		}
	})
	t.Run("null-existing-set-231", func(t *testing.T) {
		r := newIndex(t)
		r.byFunc[f] = nil
		want(t, r.insert(q, fx, x), ast.ErrASTDomain)
		delete(r.byFunc, f)
	})
	t.Run("missing-rule-297", func(t *testing.T) {
		r := newIndex(t)
		cacheMust(t, r.insert(q, fx, x))
		rule := r.rules[q]
		delete(r.rules, q)
		_, _, e := r.tryRewrite(f, []ast.ASTHandle{x})
		want(t, e, ast.ErrASTDomain)
		r.rules[q] = rule
	})
	t.Run("wrong-declaration-307", func(t *testing.T) {
		r := newIndex(t)
		cacheMust(t, r.insert(q, fx, x))
		rule := r.rules[q]
		wrong := rule
		wrong.lhs = gx
		r.rules[q] = wrong
		_, _, e := r.tryRewrite(f, []ast.ASTHandle{x})
		want(t, e, ast.ErrASTDomain)
		r.rules[q] = rule
	})
}
