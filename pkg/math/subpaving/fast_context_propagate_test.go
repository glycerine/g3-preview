package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFastContextQueuedPolynomialWatchers(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	owners := make([]*numeral.ScopedFastFloat, 4)
	for i := range owners {
		owners[i], err = numeral.NewScopedFastFloat(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	values := make([]*numeral.FastFloat, len(owners))
	for i, owner := range owners {
		values[i], err = owner.Get()
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx, err := NewFastContext(m, &control.Reslimit{Limit: 10000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := ctx.Close(); e != nil {
			t.Error(e)
		}
		for i := len(owners) - 1; i >= 0; i-- {
			if e := owners[i].Close(); e != nil {
				t.Error(e)
			}
		}
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("retained fast-float roots", budget.Used())
		}
	}()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	zero, coefficient, value, expected := values[0], values[1], values[2], values[3]
	must(m.SetInt32(coefficient, 2))
	x, err := ctx.DefinePolynomial(zero, []FastPolynomialTerm{{Variable: 0, Coefficient: coefficient}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt32(value, 1))
	must(ctx.PropagateBound(root, 0, value, true, false))
	must(m.SetInt32(value, 2))
	must(ctx.PropagateBound(root, 0, value, false, false))
	must(ctx.PropagateQueued(root))
	lower, err := root.bound(x, true)
	must(err)
	upper, err := root.bound(x, false)
	must(err)
	if lower == nil || upper == nil {
		t.Fatal("watched polynomial did not publish both bounds")
	}
	must(m.SetInt32(expected, 2))
	le, err := m.Equal(lower.value, expected)
	must(err)
	must(m.SetInt32(expected, 4))
	ue, err := m.Equal(upper.value, expected)
	must(err)
	if !le || !ue || len(ctx.queue) != 0 || ctx.queueHead != 0 || ctx.numVisited != 2 || ctx.polynomials[x].visited != 4 {
		t.Fatal("watched polynomial bounds or queue state")
	}
}

func TestSubpavingFastContextQueueCancelAndRecentChild(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		t.Fatal(err)
	}
	value, err := owner.Get()
	if err != nil {
		t.Fatal(err)
	}
	limit := &control.Reslimit{Limit: 1}
	ctx, err := NewFastContext(m, limit, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	var child *FastNode
	defer func() {
		if child != nil {
			if e := child.Close(); e != nil {
				t.Error(e)
			}
		}
		if e := ctx.Close(); e != nil {
			t.Error(e)
		}
		if e := owner.Close(); e != nil {
			t.Error(e)
		}
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("retained fast-float roots", budget.Used())
		}
	}()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	root, err := ctx.Root()
	must(err)
	must(m.SetInt32(value, 1))
	must(ctx.PropagateBound(root, 0, value, true, false))
	must(m.SetInt32(value, 2))
	must(ctx.PropagateBound(root, 0, value, false, false))
	if err := ctx.PropagateQueued(root); !errors.Is(err, ErrFastContextCanceled) {
		t.Fatal("first queue drain", err)
	}
	if ctx.queueHead != 1 || len(ctx.queue) != 2 || limit.Steps != 2 {
		t.Fatal("canceled queue lost suffix")
	}
	limit.Limit = 100
	must(ctx.PropagateQueued(root))
	if ctx.queueHead != 0 || len(ctx.queue) != 0 || limit.Steps != 3 {
		t.Fatal("resumed queue state")
	}
	child, err = ctx.NewChild(root)
	must(err)
	must(m.SetInt32(value, 3))
	must(ctx.DecideBound(child, 0, value, false, false))
	must(m.SetInt32(value, 4))
	must(ctx.DecideBound(child, 0, value, false, false))
	must(m.SetInt32(value, 0))
	must(ctx.DecideBound(child, 0, value, true, false))
	must(ctx.AddRecentBounds(child))
	if len(ctx.queue) != 2 || ctx.queue[0] != child.lower[0] || ctx.queue[1] != child.upper[0] ||
		ctx.queue[0].timestamp != ctx.timestamp || ctx.queue[1].timestamp != ctx.timestamp {
		t.Fatal("recent child bound order or timestamp")
	}
	if err := ctx.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("replay into nonempty queue", err)
	}
	must(ctx.PropagateQueued(child))
	must(m.SetInt32(value, 1))
	must(ctx.PropagateBound(root, 0, value, true, false))
	must(ctx.PropagateQueued(root))
	stamp := child.trail.timestamp
	if err := ctx.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("changed parent trail replay", err)
	}
	if len(ctx.queue) != 0 || child.trail.timestamp != stamp {
		t.Fatal("failed replay changed queue or timestamp")
	}
}

func TestSubpavingFastContextInitialPolynomialSweep(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	owners := make([]*numeral.ScopedFastFloat, 3)
	for i := range owners {
		owners[i], err = numeral.NewScopedFastFloat(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	values := make([]*numeral.FastFloat, len(owners))
	for i, owner := range owners {
		values[i], err = owner.Get()
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx, err := NewFastContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := ctx.Close(); e != nil {
			t.Error(e)
		}
		for i := len(owners) - 1; i >= 0; i-- {
			if e := owners[i].Close(); e != nil {
				t.Error(e)
			}
		}
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("retained fast-float roots", budget.Used())
		}
	}()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(m.SetInt32(values[1], 2))
	x, err := ctx.DefinePolynomial(values[0], []FastPolynomialTerm{{Variable: 0, Coefficient: values[1]}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt32(values[2], 1))
	must(ctx.PropagateBound(root, 0, values[2], true, false))
	must(m.SetInt32(values[2], 2))
	must(ctx.PropagateBound(root, 0, values[2], false, false))
	must(ctx.PropagateAllDefinitions(root))
	if ctx.numVisited != 1 || root.lower[x] == nil || root.upper[x] == nil || len(ctx.queue) != 4 {
		t.Fatal("initial definition sweep failed to publish both bounds")
	}
	must(ctx.PropagateQueued(root))
	if len(ctx.queue) != 0 || root.inconsistent {
		t.Fatal("post-sweep queue or conflict")
	}
}
