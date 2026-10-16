package ufbv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func ufbvBooleanFixture(t *testing.T, m *ast.ASTManager) *rewriter.BoolRewriter {
	t.Helper()
	// The UFBV constructor configures elim_and=true on its Boolean reducer.
	budget := arena.NewBudget(1 << 30)
	numbers, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	cacheMust(t, e)
	ctx, e := uparams.NewParamsContext(m.Symbols, numbers, uparams.ParamsConfig{Budget: budget, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	cacheMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: budget, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	cacheMust(t, e)
	globals, e := uparams.NewGlobalParamsService(m.Symbols, ctx, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	cacheMust(t, e)
	cacheMust(t, globals.InitializeSource())
	params, e := ctx.NewValue()
	cacheMust(t, e)
	key := "elim_and"
	cacheMust(t, ctx.SetBoolName(&params, &key, true))
	boolean, e := rewriter.NewBoolRewriter(m, ctx, globals, &params)
	cacheMust(t, e)
	t.Cleanup(func() {
		cacheMust(t, boolean.Close())
		cacheMust(t, ctx.Del(&params))
		cacheMust(t, globals.Close())
		cacheMust(t, catalog.Close())
		cacheMust(t, ctx.Close())
		cacheMust(t, numbers.Close())
		if budget.Used() != 0 {
			t.Fatal("boolean service roots", budget.Used())
		}
	})
	return boolean
}

func TestUFBVRewriteLoopNestedRulesBooleanAndQuantifier(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	decl := func(id uint32, arity int) ast.ASTHandle {
		t.Helper()
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = b
		}
		h, e := m.MkFuncDecl(symbol.Numerical(id), domain, b, ast.ASTFuncDeclInfo{})
		cacheMust(t, e)
		return pin(h)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		cacheMust(t, e)
		return pin(h)
	}
	quant := func(id uint32, sorts []ast.ASTHandle, body ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		names := make([]symbol.Symbol, len(sorts))
		for i := range names {
			names[i] = symbol.Numerical(id + uint32(i))
		}
		h, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: sorts, Names: names, Body: body})
		cacheMust(t, e)
		return pin(h)
	}
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	pin(x)
	f, g, h, aDecl, cDecl := decl(101, 1), decl(102, 1), decl(103, 2), decl(104, 0), decl(105, 0)
	a, c := app(aDecl), app(cDecl)
	fx, gx := app(f, x), app(g, x)
	qF, qG := quant(106, []ast.ASTHandle{b}, fx), quant(107, []ast.ASTHandle{b}, gx)
	index, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, index.close()) })
	cacheMust(t, index.insert(qF, fx, gx))
	cacheMust(t, index.insert(qG, gx, x))
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("demodulator_bug")
	index.trace = tags
	loop, e := newRewriteTraversal(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, loop.close()) })

	boolean := ufbvBooleanFixture(t, m)

	fa, fc := app(f, a), app(f, c)
	root := app(h, fa, fc)
	want := app(h, a, c)
	for _, tc := range []struct {
		name       string
		root, want ast.ASTHandle
	}{
		{"nested-rules-and-rebuilt-parent", root, want},
		{"second-call-lazy-cache-reset", root, want},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := loop.rewrite(index, boolean, tc.root)
			cacheMust(t, e)
			got, e := m.RefGet(out)
			cacheMust(t, e)
			if got != tc.want {
				t.Fatal("source rewrite result", got, tc.want)
			}
			cacheMust(t, m.DelRef(&out))
		})
	}
	if tags.ShouldTrace("demodulator_bug") && !strings.Contains(trace.String(), "rewrite src/tactic/ufbv/ufbv_rewriter.cpp:423 ---------\ne:\n") {
		t.Fatal("nested source rewrite did not reach rebuilt application trace", trace.String())
	}
	truth, e := m.MkTrue()
	cacheMust(t, e)
	and, e := m.MkAnd(truth, fa)
	cacheMust(t, e)
	pin(and)
	out, e := loop.rewrite(index, boolean, and)
	cacheMust(t, e)
	got, e := m.RefGet(out)
	cacheMust(t, e)
	if got != a {
		t.Fatal("Boolean simplification after rule rewriting", got, a)
	}
	cacheMust(t, m.DelRef(&out))

	// The first declaration is unused; rewriting the body retains only the
	// final binder, and elim_unused_vars performs the source remapping.
	qTwo := quant(108, []ast.ASTHandle{b, b}, fx)
	out, e = loop.rewrite(index, boolean, qTwo)
	cacheMust(t, e)
	got, e = m.RefGet(out)
	cacheMust(t, e)
	n, e := m.GetNumDeclarations(got)
	cacheMust(t, e)
	body, e := m.GetQuantifierBody(got)
	cacheMust(t, e)
	if n != 1 || body != x {
		t.Fatal("quantifier body rewrite and unused declaration removal", n, body, x)
	}
	cacheMust(t, m.DelRef(&out))
	if len(loop.newExprs) != 2 {
		t.Fatal("source quantifier intermediates", len(loop.newExprs))
	}
	retained := append([]ast.ASTRef(nil), loop.newExprs...)
	out, e = loop.rewrite(index, boolean, root)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	for _, ref := range retained {
		if _, e := m.RefGet(ref); e != nil {
			t.Fatal("source member buffer lost a quantifier across calls", e)
		}
	}
	cacheMust(t, loop.close())
	for _, ref := range retained {
		if _, e := m.RefGet(ref); e == nil {
			t.Fatal("source member buffer retained a quantifier after close")
		}
	}
}
