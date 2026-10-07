package ast_test

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func goalTestAtom(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	h := proofTestBool(t, m, name)
	astTestPin(t, m, h)
	return h
}
func satSolverCheckFixture(t *testing.T) (*sat.Solver, *util.ParamsContext, util.ParamsRef) {
	t.Helper()
	ctx, _, _ := paramsTestOwner(t)
	p, e := ctx.NewValue()
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ctx.Del(&p)) })
	runtime, e := sat.NewRuntime(sat.RuntimeConfig{Budget: ctx.BudgetField, ParamsContext: ctx})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, runtime.Close()) })
	limit, e := util.NewReslimit(ctx.BudgetField)
	satOK(t, e)
	t.Cleanup(limit.Close)
	s, e := sat.NewSATSolver(sat.SolverConfig{ParamsContext: ctx, Params: p, RLimit: limit, Budget: ctx.BudgetField, Runtime: runtime})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, ctx, p
}

func proofTestBool(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkConst(astTestSymbol(t, m, name), s)
	astTestMust(t, e)
	return h
}
func paramsTestOwner(t *testing.T) (*util.ParamsContext, *util.SymbolManager, *util.RationalService) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	if e != nil {
		t.Fatal(e)
	}
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := c.Close(); e != nil {
			t.Error(e)
		}
		if e := q.Close(); e != nil {
			t.Error(e)
		}
		if e := sy.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("retained %d", b.Used())
		}
	})
	return c, sy, q
}
func satOK(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func astTestSymbol(t *testing.T, m *ast.ASTManager, text string) util.Symbol {
	t.Helper()
	s, e := m.Symbols.Intern(text)
	astTestMust(t, e)
	return s
}
