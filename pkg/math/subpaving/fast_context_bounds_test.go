package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFastBoundTimestampSourceExceptionAfterPublication(t *testing.T) {
	m, c, value, root := fastMonomialFixture(t, []bool{false})
	if err := m.SetInt32(value, 1); err != nil {
		t.Fatal(err)
	}
	c.timestamp = math.MaxUint64 - 1
	err := c.PropagateBound(root, 0, value, true, false)
	if !errors.Is(err, ErrFastContextTimestamp) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("source bound timestamp exception", err)
	}
	if root.trail == nil || root.trail.timestamp != math.MaxUint64-1 || c.timestamp != math.MaxUint64 || len(c.queue) != 1 {
		t.Fatal("bound publication must precede timestamp exception")
	}
}

func TestSubpavingFastContextBoundFlowAndRelevance(t *testing.T) {
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
	ctx, err := NewFastContext(m, &control.Reslimit{Limit: 1000}, []bool{false, true}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	var child *FastNode
	var view *FastInterval
	defer func() {
		if view != nil {
			if e := view.Close(); e != nil {
				t.Error(e)
			}
		}
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
	set := func(n int32) { t.Helper(); must(m.SetInt32(value, n)) }
	relevant := func(k int32, lower, open, want bool) {
		t.Helper()
		set(k)
		got, e := ctx.RelevantNewBound(root, 0, value, lower, open)
		must(e)
		if got != want {
			t.Fatalf("relevance %d lower=%t open=%t = %t, want %t", k, lower, open, got, want)
		}
	}
	relevant(11, true, false, false)
	relevant(-11, false, false, false)
	set(0)
	must(ctx.PropagateBound(root, 0, value, true, false))
	set(10)
	must(ctx.PropagateBound(root, 0, value, false, false))
	if len(ctx.queue) != 2 || ctx.queue[0].timestamp != 0 || ctx.queue[1].timestamp != 1 {
		t.Fatal("source queue timestamps")
	}
	relevant(0, true, false, false)
	relevant(1, true, false, true)
	relevant(10, true, true, true) // equal to the upper bound; openness conflicts
	relevant(10, false, false, false)
	relevant(9, false, false, true)
	relevant(0, false, true, true) // equal to the lower bound; openness conflicts
	relevant(-1, false, false, true)
	child, err = ctx.NewChild(root)
	must(err)
	view, err = NewFastInterval(m)
	must(err)
	must(view.SetConstant(child, 0))
	set(5)
	must(ctx.PropagateBound(root, 0, value, true, false))
	lo, hi, err := view.endpoints()
	must(err)
	must(m.SetInt32(value, 0))
	lowerEqual, err := m.Equal(lo, value)
	must(err)
	must(m.SetInt32(value, 10))
	upperEqual, err := m.Equal(hi, value)
	must(err)
	if !lowerEqual || !upperEqual {
		t.Fatal("child lost inherited bound snapshot")
	}
	set(-1)
	must(ctx.PropagateBound(root, 0, value, false, false))
	if !root.inconsistent || root.conflictVar != 0 || len(ctx.queue) != 4 || ctx.timestamp != 4 {
		t.Fatal("conflict/queue state")
	}
	if err := ctx.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("context closed with child", err)
	}
	must(child.Close())
	must(ctx.Close())
	if _, err := ctx.Root(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context", err)
	}
}

func TestSubpavingFastContextIntegerNormalizationAndCandidate(t *testing.T) {
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
	ctx, err := NewFastContext(m, &control.Reslimit{Limit: 1000}, []bool{true}, 0, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
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
	for _, tc := range []struct {
		num, den int32
		lower    bool
		want     int32
	}{
		{3, 2, true, 2}, {3, 2, false, 1},
		{2, 1, true, 3}, {2, 1, false, 1},
	} {
		must(m.SetFraction32(value, tc.num, uint32(tc.den)))
		open := true
		must(ctx.NormalizeBound(0, value, tc.lower, &open))
		if open {
			t.Fatal("normalized integer bound remained open")
		}
		wantOwner, e := numeral.NewScopedFastFloat(m)
		must(e)
		want, e := wantOwner.Get()
		must(e)
		must(m.SetInt32(want, tc.want))
		equal, e := m.Equal(value, want)
		must(e)
		must(wantOwner.Close())
		if !equal {
			t.Fatal("integer normalization", tc)
		}
	}
	root, err := ctx.Root()
	must(err)
	must(m.SetFraction32(value, 3, 2))
	must(ctx.PropagateBound(root, 0, value, true, true))
	bound, err := root.bound(0, true)
	must(err)
	if bound == nil || bound.open {
		t.Fatal("published fractional integer bound remained open")
	}
	must(m.SetInt32(value, 2))
	equal, err := m.Equal(bound.value, value)
	must(err)
	if !equal {
		t.Fatal("published integer lower bound")
	}
	must(m.SetInt32(value, 3))
	published, err := ctx.PropagateCandidate(root, 0, value, true, false)
	must(err)
	if !published || len(ctx.queue) != 2 {
		t.Fatal("stronger candidate was skipped")
	}
	published, err = ctx.PropagateCandidate(root, 0, value, true, false)
	must(err)
	if published || len(ctx.queue) != 2 {
		t.Fatal("equal candidate was published")
	}
}
