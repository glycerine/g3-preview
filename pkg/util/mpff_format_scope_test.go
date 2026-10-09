// Source mpff.cpp:1173–1378.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatFormattingRetainsSourcePrecision(t *testing.T) {
	m := fastManager(t, 96)
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, 1))
	// Prime only persistent source vector and explicit Go output assembly.
	m.toBuffer(0, &a, true, false)
	serviceMust(t, m.natural.reserveText(2048))
	before, e := m.natural.runtime.AllocationCount()
	serviceMust(t, e)
	var dst [8]byte
	n, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "1" {
		t.Fatal(n, dst)
	}
	after, e := m.natural.runtime.AllocationCount()
	serviceMust(t, e)
	// Full-precision scratch is native and adds no Runtime events.
	if after != before {
		t.Fatal("source full-precision lexical events", before, after)
	}
	serviceMust(t, m.Del(&a))
}

func TestFastFloatDecimalBuffersAreLexical(t *testing.T) {
	m := fastManager(t, 5)
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 1, 3))
	var dst [64]byte
	_, e := m.FormatDecimalInto(&a, 8, 128, dst[:])
	serviceMust(t, e)
	serviceMust(t, m.Del(&a))
}

// Native formatting/division scratch contributes no Runtime payload blocks.
func fastFormattingPhysicalAudit(t *testing.T, m *FastFloatManager) {
	t.Helper()
	stats := m.natural.runtime.Stats()
	if stats.LiveBlocks != 0 || stats.PhysicalBytes != 0 || stats.InitializationBytes != 0 {
		t.Fatal("native formatting allocated Runtime payload", stats)
	}
}

func TestFastFloatNativeFullPrecisionFormattingWithoutReservations(t *testing.T) {
	m := fastManager(t, 96)
	b := m.config.Storage.Descriptors.Budget
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, -1))
	m.toBuffer(0, &a, true, false)
	serviceMust(t, m.natural.reserveText(2048))
	b.FailAfter(0)
	var dst [16]byte
	n, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "-1" {
		t.Fatal(n, dst)
	}
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	b.FailAfter(-1)
	serviceMust(t, m.Del(&a))
}

func TestFastFloatDecimalScratchResetsAfterFormattingFailure(t *testing.T) {
	m := fastManager(t, 2)
	budget := m.config.Storage.Descriptors.Budget
	serviceMust(t, m.beginFastFormatting())
	for i := range m.formatting.buffers {
		m.formatting.buffers[i] = make([]uint32, 1025)
	}
	budget.FailAfter(0)
	cause := arena.ErrBounds
	if e := m.endFastFormatting(cause); e != cause {
		t.Fatal("original formatting cause", e, cause)
	}
	for _, limbs := range m.formatting.buffers {
		if len(limbs) != 0 {
			t.Fatal("decimal scratch not reset after failure")
		}
	}
	fastAudit(t, m)
	fastStandaloneBudgetAudit(t, m)
	budget.FailAfter(-1)
	serviceMust(t, m.CleanupFormatting())
	var value FastFloat
	serviceMust(t, m.SetFraction32(&value, 3, 2))
	var dst [32]byte
	n, e := m.FormatDecimalInto(&value, 8, 128, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "1.5" {
		t.Fatal("formatting after cleanup", n, dst)
	}
	serviceMust(t, m.Del(&value))
}

func TestFastFloatFormattingScopeGuardsAndInlineWarm(t *testing.T) {
	m := fastManager(t, 2)
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, 7))
	var dst [64]byte
	_, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	b := m.config.Storage.Descriptors.Budget
	b.FailAfter(0)
	var problem error
	allocations := testing.AllocsPerRun(20, func() { _, problem = m.FormatInto(&a, dst[:]) })
	if allocations != 0 || problem != nil {
		t.Fatal("source inline scope must stay host-allocation free", allocations, problem)
	}
	b.FailAfter(-1)
	serviceMust(t, m.beginFastFormatting())
	if e = m.beginFastFormatting(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("actual holder reuse", e)
	}
	serviceMust(t, m.endFastFormatting(nil))
	serviceMust(t, m.Del(&a))
}

// Cross the former inline-storage boundary with finite and recurring fractions.
// Expected decimal text is independent of the scratch-storage representation.
func TestFastFloatDecimalLargePrecision(t *testing.T) {
	for _, precision := range []uint32{1023, 1024, 1025, 2049} {
		m := fastManager(t, precision)
		for _, tc := range []struct {
			numerator   int32
			denominator uint32
			want        string
		}{{3, 2, "1.5"}, {-5, 8, "-0.625"}, {1, 3, "0.33333?"}} {
			var value FastFloat
			serviceMust(t, m.SetFraction32(&value, tc.numerator, tc.denominator))
			var dst [32]byte
			n, e := m.FormatDecimalInto(&value, 5, 128, dst[:])
			serviceMust(t, e)
			if got := string(dst[:n]); got != tc.want {
				t.Fatal(precision, tc.numerator, tc.denominator, got, tc.want)
			}
			for _, scratch := range m.formatting.buffers {
				if len(scratch) != 0 {
					t.Fatal("decimal scratch not reset after output", precision)
				}
			}
			serviceMust(t, m.Del(&value))
		}
		serviceMust(t, m.CleanupFormatting())
		for _, scratch := range m.formatting.buffers {
			if scratch != nil {
				t.Fatal("explicit cleanup retained scratch capacity", precision)
			}
		}
	}
}

func TestFastFloatFullPrecisionWarmNativeScratch(t *testing.T) {
	m := fastManager(t, 96)
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, 1))
	var dst [64]byte
	_, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	budget := m.config.Storage.Descriptors.Budget
	beforeUsed, beforeOwned := budget.Used(), fastOwnedBytes(m)
	before, e := m.natural.runtime.AllocationCount()
	serviceMust(t, e)
	var problem error
	allocations := testing.AllocsPerRun(10, func() { _, problem = m.FormatInto(&a, dst[:]) })
	after, e := m.natural.runtime.AllocationCount()
	serviceMust(t, e)
	if problem != nil || allocations != 0 || after != before || budget.Used() != beforeUsed || fastOwnedBytes(m) != beforeOwned {
		t.Fatal("reusable native scratch", allocations, after-before, problem)
	}
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	serviceMust(t, m.Del(&a))
}

func fastStandaloneBudgetAudit(t *testing.T, m *FastFloatManager) {
	t.Helper()
	if got, want := m.config.Storage.Descriptors.Budget.Used(), fastOwnedBytes(m); got != want {
		t.Fatal("standalone actual owned graph", got, want)
	}
}

func TestFastFloatDecimalNativeDivisionWithoutReservations(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
	c.PrecisionWords, c.InitialCapacity = 1025, 2
	m, e := NewFastFloatManager(c)
	serviceMust(t, e)
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 3, 2))
	state, stats, used := m.natural.runtime.State(), m.natural.runtime.Stats(), b.Used()
	b.FailAfter(0)
	var dst [32]byte
	n, e := m.FormatDecimalInto(&a, 2, 128, dst[:])
	if e != nil || string(dst[:n]) != "1.5" {
		t.Fatal("native cold decimal division", n, e, dst)
	}
	allocations := testing.AllocsPerRun(20, func() { n, e = m.FormatDecimalInto(&a, 2, 128, dst[:]) })
	if e != nil || allocations != 0 || string(dst[:n]) != "1.5" {
		t.Fatal("native warm decimal division", allocations, e)
	}
	if m.natural.runtime.State() != state || m.natural.runtime.Stats() != stats || b.Used() != used {
		t.Fatal("decimal workspace entered Runtime")
	}
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	b.FailAfter(-1)
	serviceMust(t, m.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("owner cleanup", b.Used())
	}
}
