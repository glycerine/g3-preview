package tactic_test

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
)

func goalTestNew(t *testing.T, m *ast.ASTManager, proofs, cores bool) *tactic.Goal {
	t.Helper()
	g, e := tactic.NewGoal(m, proofs, true, cores)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return g
}
func goalTestAtom(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	h := proofTestBool(t, m, name)
	astTestPin(t, m, h)
	return h
}
func goalTestApp(t *testing.T, m *ast.ASTManager, op int32, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(0, op, nil, args)
	astTestMust(t, e)
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
	limit, e := control.NewReslimit(ctx.BudgetField)
	satOK(t, e)
	t.Cleanup(limit.Close)
	s, e := sat.NewSATSolver(sat.SolverConfig{ParamsContext: ctx, Params: p, RLimit: limit, Budget: ctx.BudgetField, Runtime: runtime})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, ctx, p
}

func satOK(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

type modelConverterTestLeaf struct {
	name   string
	calls  *[]string
	refs   uint32
	closed bool
	label  util.Symbol
	run    func(*tactic.ModelRef, uint32) error
	fail   error
}

func (c *modelConverterTestLeaf) IncRef() error {
	if c.closed {
		return tactic.ErrModelConverterOwner
	}
	c.refs++
	return nil
}
func (c *modelConverterTestLeaf) DecRef() error {
	if c.closed || c.refs == 0 {
		return tactic.ErrModelConverterOwner
	}
	c.refs--
	if c.refs == 0 {
		return c.Close()
	}
	return nil
}
func (c *modelConverterTestLeaf) Close() error {
	if !c.closed {
		c.closed = true
		*c.calls = append(*c.calls, "destroy:"+c.name)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyModel(m *tactic.ModelRef, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("model:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, i)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyUnindexed(m *tactic.ModelRef) error {
	*c.calls = append(*c.calls, "unindexed:"+c.name)
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, 0)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyLabels(l *tactic.ModelLabels, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("labels:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	return l.Append(c.label)
}
func (c *modelConverterTestLeaf) Cancel() error {
	*c.calls = append(*c.calls, "cancel:"+c.name)
	return nil
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
