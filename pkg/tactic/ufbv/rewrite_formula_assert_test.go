package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVFormulaSourceSecondRewriteFixedPointAssertion(t *testing.T) {
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
	f, e := m.MkFuncDecl(symbol.Numerical(690), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(691)}, Body: fx})
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{x, fx, q} {
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
	}
	processor, e := newFormulaProcessor(m, ufbvBooleanFixture(t, m))
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, processor.close()) })
	cacheMust(t, processor.index.insert(q, fx, x))
	before := processor.loop.cache.timestamp
	e = processor.assertRewrittenFixedPoint(fx)
	if diagnostic.DebugBuild {
		if !errors.Is(e, diagnostic.ErrDebugAssertion) {
			t.Fatal("reducible result must fail source fixed-point assertion", e)
		}
		if processor.loop.cache.timestamp == before {
			t.Fatal("checked source assertion skipped the second rewrite")
		}
	} else if e != nil || processor.loop.cache.timestamp != before {
		t.Fatal("erased assertion performed a rewrite", e, processor.loop.cache.timestamp, before)
	}
	cacheMust(t, processor.assertRewrittenFixedPoint(x))
}
