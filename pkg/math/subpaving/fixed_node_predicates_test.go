package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedNodeBoundPredicates(t *testing.T) {
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
	n, err := NewFixedNode(m, 2, nil)
	must(err)
	check := func(label string, got, want bool, err error) {
		t.Helper()
		must(err)
		if got != want {
			t.Fatalf("%s = %v, want %v", label, got, want)
		}
	}
	got, err := n.IsUnbounded(0)
	check("initial unbounded", got, true, err)
	must(m.SetInt64(v, 0))
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
	must(m.SetInt64(v, -1))
	must(n.PushBound(1, v, true, false))
	must(m.SetInt64(v, 0))
	must(n.PushBound(1, v, false, true))
	got, err = n.IsUpperZero(1)
	check("open upper zero", got, false, err)
	got, err = n.ConflictingBounds(1)
	check("negative-to-open-zero conflict", got, false, err)
	must(m.SetInt64(v, 1))
	must(n.PushBound(1, v, true, false))
	got, err = n.ConflictingBounds(1)
	check("reversed bounds conflict", got, true, err)
	if _, err := n.IsZero(2); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("invalid variable error = %v", err)
	}
	must(n.Close())
	if _, err := n.IsZero(0); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed node error = %v", err)
	}
	must(v.Close())
	must(m.Close())
}
