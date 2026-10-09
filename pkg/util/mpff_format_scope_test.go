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
	// Source char(11*96) expands once. temp96 and numer97 each expand
	// 16→32→64→128: six further actual allocation calls, despite leading0s.
	if after-before != 7 {
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
	add(m.natural.CharFormatting.Buffer.Raw)
	add(m.natural.CharFormatting.Buffer.PendingField)
	add(m.formatting.small.Raw)
	add(m.formatting.small.Pending)
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
	if !success || !foundChar || !foundInner {
		t.Fatal("complete meaningful sweep", success, foundChar, foundInner)
	}
}

func TestFastFloatFormattingFailedCharCtorRetainedAndExplicitCleanup(t *testing.T) {
	m := fastManager(t, 200)
	b := m.config.Storage.Descriptors.Budget
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, 1))
	m.toBuffer(0, &a, true, false)
	serviceMust(t, m.natural.reserveText(4096))
	// Prime only the real runtime's retained cold locator directory. This
	// actual zero-byte allocation/free is outside the observed source invocation.
	prime, e := m.natural.runtime.Allocate(0)
	serviceMust(t, e)
	serviceMust(t, m.natural.runtime.Deallocate(&prime))
	b.FailAfter(1)
	var dst [32]byte
	_, e = m.FormatInto(&a, dst[:])
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	c := &m.natural.CharFormatting
	if c.Completed || !c.pending || c.Buffer.pos != 2048 || c.Buffer.Raw == (Block{}) || !m.formatting.pending {
		t.Fatal("failed parameterized constructor root", c.Buffer.pos)
	}
	raw := c.Buffer.Raw
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	b.FailAfter(-1)
	// Unrelated raw display is source-defined and need not reuse the holder.
	bigDst := make([]byte, 2000)
	if _, e = m.FormatRawInto(&a, bigDst); e != nil || c.Buffer.Raw != raw {
		t.Fatal("unused holder poisoned raw output", e)
	}
	serviceMust(t, m.CleanupFormatting())
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	if _, e = m.natural.runtime.Size(raw); e == nil {
		t.Fatal("owning cleanup omitted failed ctor raw")
	}
	n, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "1" {
		t.Fatal(n, dst)
	}
	serviceMust(t, m.Del(&a))
}

func TestFastFloatCompletedDefaultBuffersUnwindOnGrowthRefusal(t *testing.T) {
	m := fastManager(t, 2)
	b := m.config.Storage.Descriptors.Budget
	serviceMust(t, m.beginFastFormatting())
	m.formatting.completed = 7
	r := m.natural.runtime
	// The three default constructors completed before their body pushes.
	m.formatting.large[0] = make([]uint32, 1025)
	m.formatting.large[1] = make([]uint32, 1025)
	serviceMust(t, m.formatting.small.resize(r, 32))
	small := m.formatting.small.Raw
	b.FailAfter(0)
	e := m.formatting.small.push(r, 7)
	if !numericPhysicalReservationRefusal(e) || m.formatting.completed != 7 {
		t.Fatal("body refusal must not unconstruct default objects", e)
	}
	if e2 := m.endFastFormatting(e); e2 != e {
		t.Fatal("original action cause", e2, e)
	}
	if _, e = r.Size(small); e == nil {
		t.Fatal("completed source destructor omitted raw")
	}
	if len(m.formatting.large[0]) != 0 || len(m.formatting.large[1]) != 0 {
		t.Fatal("decimal scratch not reset after formatting")
	}
	fastAudit(t, m)
	fastStandaloneBudgetAudit(t, m)
	b.FailAfter(-1)
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
			for _, scratch := range m.formatting.large {
				if len(scratch) != 0 {
					t.Fatal("decimal scratch not reset after output", precision)
				}
			}
			serviceMust(t, m.Del(&value))
		}
		serviceMust(t, m.CleanupFormatting())
		for _, scratch := range m.formatting.large {
			if scratch != nil {
				t.Fatal("explicit cleanup retained scratch capacity", precision)
			}
		}
	}
}

func TestFastFloatDecimalEveryLargeLocalPrefix(t *testing.T) {
	success := false
	defaultRefusal, charRefusal, innerRefusal := false, false, false
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
			defaultRefusal = defaultRefusal || (!m.natural.CharFormatting.pending && !m.natural.Formatting.PendingField)
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
	if !success || !defaultRefusal || !charRefusal || !innerRefusal {
		t.Fatal("complete large local sweep", success, defaultRefusal, charRefusal, innerRefusal)
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
	if problem != nil || allocations != 14 || after-before != 7*11 || budget.Used() != beforeUsed || fastOwnedBytes(m) != beforeOwned {
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
