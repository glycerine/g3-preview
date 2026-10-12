package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func ufbvTacticServicesFixture(t *testing.T, m *ast.ASTManager) (*uparams.ParamsContext, *uparams.GlobalParamsService) {
	t.Helper()
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
	t.Cleanup(func() {
		cacheMust(t, globals.Close())
		cacheMust(t, catalog.Close())
		cacheMust(t, ctx.Close())
		cacheMust(t, numbers.Close())
		if budget.Used() != 0 {
			t.Fatal("tactic service roots", budget.Used())
		}
	})
	return ctx, globals
}

func TestUFBVRewriterTacticGoalTransitionAndCoreRefusal(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	pin(x)
	f, e := m.MkFuncDecl(symbol.Numerical(401), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	pin(f)
	aDecl, e := m.MkFuncDecl(symbol.Numerical(402), nil, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	pin(aDecl)
	a, e := m.MkApp(aDecl, nil)
	cacheMust(t, e)
	pin(a)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	pin(fx)
	fa, e := m.MkApp(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	pin(fa)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(403)}, Body: fx})
	cacheMust(t, e)
	pin(q)
	ctx, globals := ufbvTacticServicesFixture(t, m)
	rewriter, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, rewriter.Close()) })
	goal, e := tactic.NewGoal(m, false, true, false)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(q))
	cacheMust(t, goal.Assert(fa))
	cacheMust(t, goal.SetDepth(7))
	got, e := rewriter.Apply(goal)
	cacheMust(t, e)
	if got != goal {
		t.Fatal("source tactic must publish the input goal")
	}
	n, e := goal.Size()
	cacheMust(t, e)
	form, e := goal.Form(0)
	cacheMust(t, e)
	depth, e := goal.Depth()
	cacheMust(t, e)
	if n != 1 || form != q || depth != 8 {
		t.Fatal("rule reschedule, true elimination and depth", n, form, depth)
	}
	coreGoal, e := tactic.NewGoal(m, false, true, true)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, coreGoal.Close()) })
	cacheMust(t, coreGoal.Assert(fa))
	if _, e := rewriter.Apply(coreGoal); !errors.Is(e, ErrRewriterUnsatCores) {
		t.Fatal("source unsat-core guard", e)
	}
	n, e = coreGoal.Size()
	cacheMust(t, e)
	if n != 1 {
		t.Fatal("core refusal changed input goal", n)
	}
	cacheMust(t, rewriter.Close())
	if _, e := rewriter.Apply(goal); e == nil {
		t.Fatal("closed tactic accepted goal")
	}
}

func TestUFBVRewriterTacticProofModePreservesFormulaAndProof(t *testing.T) {
	m := cacheFixtureProofMode(t, ast.ASTProofEnabled)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	p, e := m.MkConst(symbol.Numerical(501), b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(p))
	t.Cleanup(func() { cacheMust(t, m.DecRef(p)) })
	ctx, globals := ufbvTacticServicesFixture(t, m)
	rewriter, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, rewriter.Close()) })
	goal, e := tactic.NewGoal(m, true, true, false)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(p))
	beforeForm, e := goal.Form(0)
	cacheMust(t, e)
	beforeProof, e := goal.Proof(0)
	cacheMust(t, e)
	if beforeProof.IsNull() {
		t.Fatal("proof-enabled source goal lacks an asserted proof")
	}
	got, e := rewriter.Apply(goal)
	cacheMust(t, e)
	if got != goal {
		t.Fatal("proof branch must publish the same goal")
	}
	n, e := goal.Size()
	cacheMust(t, e)
	afterForm, e := goal.Form(0)
	cacheMust(t, e)
	afterProof, e := goal.Proof(0)
	cacheMust(t, e)
	depth, e := goal.Depth()
	cacheMust(t, e)
	if n != 1 || afterForm != beforeForm || afterProof != beforeProof || depth != 1 {
		t.Fatal("source proof bypass and goal depth", n, afterForm, afterProof, depth)
	}
}
