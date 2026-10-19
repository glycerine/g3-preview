// Source: algebraic_numbers.cpp:897-929.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicSavedIntervalFailedConstructorOwnership(t *testing.T) {
	succeeded, retained, partial := false, false, false
	for prefix := int64(0); prefix < 128 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicBigBinaryBoundsRoot(t, c)
			z := c.config.Rationals.Integers()
			var fillers [16]unumeral.Int
			filled := 0
			for z.Cells.Stats().Slots > z.Cells.Stats().Live+z.Cells.Stats().Retired || z.Cells.Stats().Slots < z.Cells.Stats().Pages*2 {
				if filled == len(fillers) {
					t.Fatal("descriptor fill bound")
				}
				serviceMust(t, z.Copy(&fillers[filled], &z.Two64))
				filled++
			}
			var saved algebraicSavedInterval
			z.Budget().FailAfter(prefix)
			body := c.beginSavedInterval(&saved, &a)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
			} else {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				if saved.constructed || !saved.owned {
					t.Fatal("failed constructor became a completed scope")
				}
				retained = retained || z.MatchesCopyAllocationInvocation(&saved.copy)
				partial = partial || saved.value.BorrowLower().BorrowNumerator().Val != 0
				before := saved
				used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
				serviceMust(t, c.finishSavedInterval(&saved, false))
				if !reflect.DeepEqual(before, saved) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
					t.Fatal("source unwind destroyed failed constructor members")
				}
			}
			serviceMust(t, c.finishSavedInterval(&saved, true))
			if saved.owned || saved.constructed || z.MatchesCopyAllocationInvocation(&saved.copy) {
				t.Fatal("owning abort retained constructor storage")
			}
			serviceMust(t, c.Del(&a))
			for i := 0; i < filled; i++ {
				serviceMust(t, z.Del(&fillers[i]))
			}
		}) {
			return
		}
	}
	if !succeeded || !retained || !partial {
		t.Fatal("constructor sweep missed actual allocation stages", succeeded, retained, partial)
	}
}

func TestAlgebraicSavedIntervalActualTransportProtectsOwner(t *testing.T) {
	var c *AlgebraicRootCore
	var a Algebraic
	var saved algebraicSavedInterval
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		if c.savedIntervalExecuting != &saved {
			t.Error("copy transport lost constructing scope")
		}
		before := saved
		original := a
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{
			func() error { return c.Del(&a) }, func() error { return c.Close() }, func() error { return c.CleanupRetained() }, func() error { return c.finishSavedInterval(&saved, true) },
		} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("transport acquired owner", e)
			}
		}
		if !reflect.DeepEqual(before, saved) || original != a || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused callback changed constructor")
		}
		return f
	}
	var rt *umemory.Runtime
	c, _, rt = algebraicNegTLSFixture(t, transport)
	a = algebraicBigBinaryBoundsRoot(t, c)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := c.beginSavedInterval(&saved, &a)
	armed = false
	if body == nil || calls == 0 || saved.constructed {
		t.Fatal("actual copy transport not reached", body, calls)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.finishSavedInterval(&saved, true))
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicSavedIntervalCompletedSourceDestructor(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicBigBinaryBoundsRoot(t, c)
	var saved algebraicSavedInterval
	serviceMust(t, c.beginSavedInterval(&saved, &a))
	if !saved.constructed {
		t.Fatal("successful constructor not published")
	}
	serviceMust(t, c.finishSavedInterval(&saved, false))
	if saved.owned || !saved.restored || saved.target != nil {
		t.Fatal("completed source destructor retained scope")
	}
	serviceMust(t, c.finishSavedInterval(&saved, true))
	serviceMust(t, c.Del(&a))
}
