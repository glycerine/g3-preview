package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Use ordinary interned values: a closed process owner must be observed only
// when the selected source path actually evaluates an assertion switch.
func TestMonomialOrderAssertionReadPaths(t *testing.T) {
	m := orderFixture(t)
	unit := orderValue(t, m, [6]uint32{})
	x := orderValue(t, m, [6]uint32{1})
	xx := orderValue(t, m, [6]uint32{2})
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	defer func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}()
	serviceMust(t, d.Close())
	cases := []struct {
		name    string
		a, b    Monomial
		minimum bool
		reads   bool
		want    int
	}{
		{"ordinary-tail", unit, x, false, true, -1},
		{"minimum-unequal-tail", unit, x, true, false, -1},
		{"ordinary-power", x, xx, false, true, -1},
		{"minimum-power", x, xx, true, true, -1},
		{"ordinary-equal", x, x, false, false, 0},
		{"minimum-equal", x, x, true, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got int
			var err error
			if c.minimum {
				got, err = m.LexWithMinimum(c.a, c.b, 5)
			} else {
				got, err = m.Lex(c.a, c.b)
			}
			if polynomialSourceDebug && c.reads {
				if !errors.Is(err, arena.ErrClosed) {
					t.Fatalf("got %d, %v; want closed assertion owner", got, err)
				}
			} else if err != nil || got != c.want {
				t.Fatalf("got %d, %v; want %d", got, err, c.want)
			}
		})
	}
}

// These are source-tail scalar states, not claims that canonical monomials
// naturally violate uniqueness. Exercise each retained assertion action.
func TestMonomialOrderTailAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := "TTTT"
			if action == "C" {
				choices = "CCCC"
			}
			polynomialAssertionActionsFixture(t, choices)
			if action == "disabled" && polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			m := orderFixture(t)
			x := orderValue(t, m, [6]uint32{1})
			// get_power returns a reference after Continue/disabled; no
			// variable or degree field at this out-of-range index is read.
			_, powerError := m.orderPowerLocation(x, 1)
			for i, err := range []error{powerError, m.lexDistinctEndAssert(-1, -1), m.lexDistinctEndAssert(0, 0), m.lexMinimumEndAssert(0, 0)} {
				want := polynomialSourceDebug && action == "T"
				if errors.Is(err, ErrMonomialAssertion) != want {
					t.Fatalf("site %d: %v", i, err)
				}
				if !want && err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
