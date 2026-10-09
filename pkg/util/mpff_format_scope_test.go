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
	// Go character scratch adds no Runtime events. The remaining temp96 and
	// numer97 workspaces each expand 16→32→64→128, despite leading zeros.
	if after-before != 6 {
		t.Fatal("source full-precision lexical events", before, after)
	}
	serviceMust(t, m.Del(&a))
}

func TestFastFloatDecimalBuffersAreLexical(t *testing.T) {
	m := fastManager(t, 5)
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 1, 3))
	before := m.words.Stats().Live
	var dst [64]byte
	_, e := m.FormatDecimalInto(&a, 8, 128, dst[:])
	serviceMust(t, e)
	if m.words.Stats().Live != before {
		t.Fatal("source local arrays must not publish retained Words", before, m.words.Stats().Live)
	}
	serviceMust(t, m.Del(&a))
}

// Immediate actual raw-root and cold/backing charge audit; no formatting or
// teardown is used to infer ownership. Runtime.Size validates real tokens.
func fastFormattingPhysicalAudit(t *testing.T, m *FastFloatManager) {
	t.Helper()
	raw := map[Block]bool{}
	var physical, initialization uint64
	add := func(r Block) {
		if r == (Block{}) {
			return
		}
		if raw[r] {
			t.Fatal("duplicate physical formatting root")
		}
		raw[r] = true
		n, e := m.natural.runtime.Size(r)
		if e != nil {
			t.Fatal("stale physical root", e)
		}
		extent := n + uint64(m.natural.runtime.SizeTBits()/8)
		physical += extent
		initialization += ((extent + 63) / 64) * 8
	}
	fastFormattingNamedRoots(m, add)
	if uint32(len(raw)) != m.natural.runtime.Stats().LiveBlocks {
		t.Fatal("unnamed actual runtime root", len(raw), m.natural.runtime.Stats())
	}
	stats := m.natural.runtime.Stats()
	if stats.PhysicalBytes != physical || stats.InitializationBytes != initialization {
		t.Fatal("actual named runtime backing bytes", physical, initialization, stats)
	}

}

// Names only this owner's actual roots. Shared runtime censuses must add every
// participant and explicitly retained caller block before comparing totals.
func fastFormattingNamedRoots(m *FastFloatManager, add func(Block)) {
	for i := range m.natural.Division {
		add(m.natural.Division[i].Raw)
		add(m.natural.Division[i].Pending)
	}
	add(m.natural.Verification.Raw)
	add(m.natural.Verification.Pending)
	for i := range m.natural.Formatting.Locals {
		add(m.natural.Formatting.Locals[i].Raw)
		add(m.natural.Formatting.Locals[i].Pending)
	}
}

func TestFastFloatFormattingEveryConstructorPrefix(t *testing.T) {
	success, foundChar, foundInner := false, false, false
	for fault := int64(0); fault < 80; fault++ {
		m := fastManager(t, 96)
		b := m.config.Storage.Descriptors.Budget
		var a FastFloat
		serviceMust(t, m.SetInt32(&a, -1))
		m.toBuffer(0, &a, true, false)
		serviceMust(t, m.natural.reserveText(2048))
		b.FailAfter(fault)
		var dst [16]byte
		for i := range dst {
			dst[i] = '!'
		}
		n, e := m.FormatInto(&a, dst[:])
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fault, e)
			}
			for _, v := range dst {
				if v != '!' {
					t.Fatal("published caller bytes")
				}
			}
			if m.natural.textRead(0) != '-' {
				t.Fatal("negative source prefix")
			}
			foundChar = foundChar || m.natural.CharFormatting.pending
			foundInner = foundInner || m.natural.Formatting.PendingField
		} else {
			if string(dst[:n]) != "-1" {
				t.Fatal(n, dst)
			}
			success = true
		}
		fastAudit(t, m, &a)
		fastStandaloneBudgetAudit(t, m)
		b.FailAfter(-1)
		serviceMust(t, m.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("retained owner cleanup", fault, b.Used())
		}
		if success {
			break
		}
	}
	if !success || foundChar || !foundInner {
		t.Fatal("complete meaningful sweep", success, foundChar, foundInner)
	}
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

func TestFastFloatDecimalEveryLargeLocalPrefix(t *testing.T) {
	success := false
	charRefusal, innerRefusal := false, false
	for fault := int64(0); fault < 192; fault++ {
		b := arena.NewBudget(1 << 25)
		c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
		c.PrecisionWords = 1025
		c.InitialCapacity = 2
		m, e := NewFastFloatManager(c)
		serviceMust(t, e)
		var a FastFloat
		serviceMust(t, m.SetFraction32(&a, 3, 2))
		serviceMust(t, m.natural.reserveText(4096))
		b.FailAfter(fault)
		var dst [32]byte
		for i := range dst {
			dst[i] = '!'
		}
		n, e := m.FormatDecimalInto(&a, 2, 128, dst[:])
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fault, e)
			}
			for _, v := range dst {
				if v != '!' {
					t.Fatal("early caller publication", fault)
				}
			}
			charRefusal = charRefusal || m.natural.CharFormatting.pending
			innerRefusal = innerRefusal || m.natural.Formatting.PendingField
		} else {
			if string(dst[:n]) != "1.5" {
				t.Fatal(n, dst)
			}
			success = true
		}
		fastAudit(t, m, &a)
		fastStandaloneBudgetAudit(t, m)
		b.FailAfter(-1)
		serviceMust(t, m.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("whole real graph cleanup", fault, b.Used())
		}
		if success {
			break
		}
	}
	if !success || charRefusal || !innerRefusal {
		t.Fatal("complete large local sweep", success, charRefusal, innerRefusal)
	}
}

func TestFastFloatFullPrecisionWarmSourceEvents(t *testing.T) {
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
	if problem != nil || allocations != 12 || after-before != 6*11 || budget.Used() != beforeUsed || fastOwnedBytes(m) != beforeOwned {
		t.Fatal("real fresh scopes/data+init buffers", allocations, after-before, problem)
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
