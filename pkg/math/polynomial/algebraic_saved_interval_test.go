// Source: algebraic_numbers.cpp:897-929.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
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

// A completed save_intervals holds a reference to the caller's numeral. The
// numeral may change before restoration; an allocated object prefix below has
// no completed algebraic-cell constructor and proves accessor read order only.
func TestAlgebraicSavedIntervalRestoreAccessorBeforeMemberRead(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("accessor assertion erased")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicBigBinaryBoundsRoot(t, c)
	var saved algebraicSavedInterval
	serviceMust(t, c.beginSavedInterval(&saved, &a))
	original := a
	raw := algebraicQueryRawCell(t, c, 1)
	a = raw
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	body := c.restoreBinaryInterval(&saved)
	a = original
	restored := saved.restored
	serviceMust(t, c.finishSavedInterval(&saved, true))
	serviceMust(t, c.CleanupValue(&a))
	if !errors.Is(body, arena.ErrClosed) || !restored {
		t.Fatal("restore read members before accessor switch or missed flag publication", body)
	}
}

func TestAlgebraicSavedIntervalRestoreSkipsAccessorAfterBasicPromotion(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicBigBinaryBoundsRoot(t, c)
	var saved algebraicSavedInterval
	serviceMust(t, c.beginSavedInterval(&saved, &a))
	original := a
	a = Algebraic{}
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	}
	body := c.restoreBinaryInterval(&saved)
	a = original
	serviceMust(t, body)
	if !saved.restored {
		t.Fatal("basic early return missed restore flag")
	}
	serviceMust(t, c.finishSavedInterval(&saved, true))
	serviceMust(t, c.CleanupValue(&a))
}

func TestAlgebraicSavedIntervalSelectionOwnsPartialConstructorAbort(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 1024 && !reached; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicBigBinaryBoundsRoot(t, c)
			b := algebraicBigBinaryBoundsRoot(t, c)
			z := c.config.Rationals.Integers()
			var out Algebraic
			z.Budget().FailAfter(prefix)
			body := c.Add(&a, &b, &out)
			z.Budget().FailAfter(-1)
			for i := range c.binary.saved {
				saved := &c.binary.saved[i]
				if saved.owned && !saved.constructed && saved.value.BorrowLower().BorrowNumerator().Val != 0 {
					reached = true
					if body == nil || saved.constructed || !saved.owned {
						t.Fatal("selector lost failed constructor frontier", body)
					}
				}
			}
			if !reached {
				// Earlier project operations have independent allocation ownership. This
				// vector makes no new lifetime claim for their already retained suffixes.
				serviceMust(t, z.CleanupPendingAllocation())
			}
			serviceMust(t, c.CleanupRetained())
			if c.binary.active {
				t.Fatal("selector owning abort retained source scope")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !reached {
		t.Fatal("selection sweep did not retain a partially copied saved interval")
	}
}
