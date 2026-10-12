package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedRoundRobinSourceDecisionAndDefinitionSkip(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	v, err := m.New()
	must(err)
	one, err := m.New()
	must(err)
	zero, err := m.New()
	must(err)
	must(m.SetInt64(one, 1))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false, false, false}, 0, 2, 8192)
	must(err)
	def, err := ctx.DefinePolynomial(zero, []FixedPolynomialTerm{{Variable: 1, Coefficient: one}})
	must(err)
	root, err := ctx.Root()
	must(err)
	for _, item := range []struct {
		variable uint32
		value    int64
	}{{0, 1}, {2, 3}} {
		must(m.SetInt64(v, item.value))
		must(ctx.PropagateBound(root, item.variable, v, true, false))
		must(ctx.PropagateBound(root, item.variable, v, false, false))
	}
	must(ctx.PropagateQueued(root))
	x, found, err := ctx.SelectBranchVariable(root, true)
	must(err)
	if !found || x != 1 {
		t.Fatalf("root selection = %d, %v; want variable 1", x, found)
	}
	left, right, err := ctx.SplitMidpoint(root, x, true, 1)
	must(err)
	x, found, err = ctx.SelectBranchVariable(left, true)
	must(err)
	if !found || x != 1 {
		t.Fatalf("child default selection = %d, %v; want variable 1", x, found)
	}
	x, found, err = ctx.SelectBranchVariable(left, false)
	must(err)
	if !found || x != def {
		t.Fatalf("child including definitions = %d, %v; want %d", x, found, def)
	}
	must(m.SetInt64(v, 1))
	must(ctx.PropagateBound(left, 0, v, true, false))
	must(ctx.PropagateQueued(left))
	x, found, err = ctx.SelectBranchVariable(left, true)
	must(err)
	if !found || x != 1 {
		t.Fatalf("selection after newer propagated bound = %d, %v; want decision variable 1", x, found)
	}
	must(left.Close())
	must(right.Close())
	must(ctx.Close())
	must(zero.Close())
	must(one.Close())
	must(v.Close())
	must(m.Close())
}

func TestSubpavingFixedRoundRobinSingletonAndUndecidedChild(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	v, err := m.New()
	must(err)
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 7))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(ctx.PropagateBound(root, 0, v, false, false))
	must(ctx.PropagateQueued(root))
	_, found, err := ctx.SelectBranchVariable(root, true)
	must(err)
	if found {
		t.Fatal("fixed singleton selected for split")
	}
	child, err := ctx.NewChild(root)
	must(err)
	if _, _, err := ctx.SelectBranchVariable(child, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("undecided child selection = %v", err)
	}
	must(child.Close())
	must(ctx.Close())
	must(v.Close())
	must(m.Close())
}
