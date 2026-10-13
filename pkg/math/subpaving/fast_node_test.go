package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFastNodeConstantIntervalViews(t *testing.T) {
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
	root, err := NewFastNode(m, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	intervals := make([]*FastInterval, 4)
	for i := range intervals {
		intervals[i], err = NewFastInterval(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	var child *FastNode
	defer func() {
		if child != nil {
			if e := child.Close(); e != nil {
				t.Error(e)
			}
		}
		if e := root.Close(); e != nil {
			t.Error(e)
		}
		for i := len(intervals) - 1; i >= 0; i-- {
			if e := intervals[i].Close(); e != nil {
				t.Error(e)
			}
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
	view, childView, other, result := intervals[0], intervals[1], intervals[2], intervals[3]
	check := func(v *FastInterval, lo, hi int32, lowerInf, upperInf, lowerOpen, upperOpen bool) {
		t.Helper()
		actualLo, actualHi, e := v.endpoints()
		must(e)
		if v.lowerInf != lowerInf || v.upperInf != upperInf || v.lowerOpen != lowerOpen || v.upperOpen != upperOpen {
			t.Fatalf("view shape: inf=(%t,%t) open=(%t,%t)", v.lowerInf, v.upperInf, v.lowerOpen, v.upperOpen)
		}
		if !lowerInf {
			must(m.SetInt32(values[1], lo))
			equal, e := m.Equal(actualLo, values[1])
			must(e)
			if !equal {
				t.Fatal("view lower endpoint")
			}
		}
		if !upperInf {
			must(m.SetInt32(values[2], hi))
			equal, e := m.Equal(actualHi, values[2])
			must(e)
			if !equal {
				t.Fatal("view upper endpoint")
			}
		}
		must(v.CheckInvariant())
	}
	push := func(n *FastNode, x uint32, k int32, lower, open bool) {
		t.Helper()
		must(m.SetInt32(values[0], k))
		must(n.PushBound(x, values[0], lower, open))
	}
	must(view.SetConstant(root, 0))
	check(view, 0, 0, true, true, true, true)
	push(root, 0, -2, true, true)
	check(view, -2, 0, false, true, true, true)
	push(root, 0, 3, false, false)
	check(view, -2, 3, false, false, true, false)
	must(result.Power(view, 2))
	check(result, 0, 9, false, false, false, false)
	must(m.SetInt32(values[0], 2))
	must(other.SetFinite(values[0], values[0], false, false))
	must(result.Mul(view, other))
	check(result, -4, 6, false, false, true, false)
	child, err = NewFastNode(m, 2, root)
	must(err)
	must(childView.SetConstant(child, 0))
	push(root, 0, 1, true, false)
	push(root, 0, 4, false, false)
	check(view, 1, 4, false, false, false, false)
	check(childView, -2, 3, false, false, true, false)
	push(child, 0, 2, false, true)
	check(childView, -2, 2, false, false, true, true)
	must(result.Div(view, other))
	resultLo, resultHi, err := result.endpoints()
	must(err)
	must(m.SetFraction32(values[1], 1, 2))
	must(m.SetInt32(values[2], 2))
	lowerEqual, err := m.Equal(resultLo, values[1])
	must(err)
	upperEqual, err := m.Equal(resultHi, values[2])
	must(err)
	if !lowerEqual || !upperEqual || result.lowerOpen || result.upperOpen {
		t.Fatal("root view quotient")
	}
	if err := view.SetFinite(values[0], values[0], false, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("constant view mutation", err)
	}
	if err := view.Reset(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("constant view reset", err)
	}
	if err := view.Mul(other, other); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("constant view output", err)
	}
	if err := root.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("parent closed with child", err)
	}
	must(child.Close())
	if _, _, err := childView.endpoints(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed-node view", err)
	}
	must(root.Close())
}
