package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFastNodeBoundPredicates(t *testing.T) {
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
	v, err := owner.Get()
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewFastNode(m, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := n.Close(); e != nil {
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
	check := func(label string, got, want bool, err error) {
		t.Helper()
		must(err)
		if got != want {
			t.Fatalf("%s = %v, want %v", label, got, want)
		}
	}
	got, err := n.IsUnbounded(0)
	check("initial unbounded", got, true, err)
	must(m.SetInt32(v, 0))
	must(n.PushBound(0, v, true, false))
	must(n.PushBound(0, v, false, false))
	got, err = n.IsZero(0)
	check("closed zero singleton", got, true, err)
	got, err = n.IsUpperZero(0)
	check("closed upper zero", got, true, err)
	got, err = n.ConflictingBounds(0)
	check("closed equal bounds conflict", got, false, err)
	must(n.PushBound(0, v, true, true))
	got, err = n.IsZero(0)
	check("open zero singleton", got, false, err)
	got, err = n.ConflictingBounds(0)
	check("open equal bounds conflict", got, true, err)
	must(n.PushBound(0, v, true, false))
	must(m.SetInt32(v, -1))
	must(n.PushBound(1, v, true, false))
	must(m.SetInt32(v, 0))
	must(n.PushBound(1, v, false, true))
	got, err = n.IsUpperZero(1)
	check("open upper zero", got, false, err)
	got, err = n.ConflictingBounds(1)
	check("negative-to-open-zero conflict", got, false, err)
	must(m.SetInt32(v, 1))
	must(n.PushBound(1, v, true, false))
	got, err = n.ConflictingBounds(1)
	check("reversed bounds conflict", got, true, err)
	if _, err := n.IsZero(2); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("invalid variable", err)
	}
	must(n.Close())
	if _, err := n.IsZero(0); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed node", err)
	}
}
