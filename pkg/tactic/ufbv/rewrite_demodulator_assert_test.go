package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVDemodulatorAndMatcherSourceAppAssertions(t *testing.T) {
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
	f, e := m.MkFuncDecl(symbol.Numerical(716), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	g, e := m.MkFuncDecl(symbol.Numerical(838), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	gx, e := m.MkApp(g, []ast.ASTHandle{x})
	cacheMust(t, e)
	fgx, e := m.MkApp(f, []ast.ASTHandle{gx})
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{x, gx, fgx} {
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
	}
	processor, e := newFormulaProcessor(m, ufbvBooleanFixture(t, m))
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, processor.close()) })
	wantGuard := func(e error) {
		t.Helper()
		if diagnostic.DebugBuild {
			if !errors.Is(e, diagnostic.ErrDebugAssertion) {
				t.Fatal("source app assertion", e)
			}
		} else if !errors.Is(e, ast.ErrASTDomain) {
			t.Fatal("erased source app guard", e)
		}
	}
	_, e = processor.demodulatorFunction(x)
	wantGuard(e)
	decl, e := processor.demodulatorFunction(fgx)
	cacheMust(t, e)
	if decl != f {
		t.Fatal("source demodulator declaration", decl, f)
	}
	_, e = processor.index.matcher.matchArgs(fgx, []ast.ASTHandle{b})
	wantGuard(e)
}
