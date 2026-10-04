// Licensed under the MIT license.
// Source mpff.cpp1173–1378.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
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
	raw := map[memory.Block]bool{}
	var physical, initialization uint64
	add := func(r memory.Block) {
		if r == (memory.Block{}) {
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
func fastFormattingNamedRoots(m *FastFloatManager, add func(memory.Block)) {
	for i := range m.natural.division {
		add(m.natural.division[i].raw)
		add(m.natural.division[i].pending)
	}
	add(m.natural.verification.raw)
	add(m.natural.verification.pending)
	for i := range m.natural.formatting.locals {
		add(m.natural.formatting.locals[i].raw)
		add(m.natural.formatting.locals[i].pending)
	}
	add(m.natural.charFormatting.buffer.raw)
	add(m.natural.charFormatting.buffer.pending)
	for i := range m.formatting.large {
		add(m.formatting.large[i].raw)
		add(m.formatting.large[i].pending)
	}
	add(m.formatting.small.raw)
	add(m.formatting.small.pending)
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
			foundChar = foundChar || m.natural.charFormatting.pending
			foundInner = foundInner || m.natural.formatting.pending
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
	c := &m.natural.charFormatting
	if c.completed || !c.pending || c.buffer.pos != 2048 || c.buffer.raw == (memory.Block{}) || !m.formatting.pending {
		t.Fatal("failed parameterized constructor root", c.buffer.pos)
	}
	raw := c.buffer.raw
	fastAudit(t, m, &a)
	fastStandaloneBudgetAudit(t, m)
	b.FailAfter(-1)
	// Unrelated raw display is source-defined and need not reuse the holder.
	bigDst := make([]byte, 2000)
	if _, e = m.FormatRawInto(&a, bigDst); e != nil || c.buffer.raw != raw {
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
	serviceMust(t, m.formatting.large[0].resize(r, 1025))
	serviceMust(t, m.formatting.large[1].resize(r, 1025))
	serviceMust(t, m.formatting.small.resize(r, 32))
	first, second, small := m.formatting.large[0].raw, m.formatting.large[1].raw, m.formatting.small.raw
	b.FailAfter(0)
	e := m.formatting.small.push(r, 7)
	if !numericPhysicalReservationRefusal(e) || m.formatting.completed != 7 {
		t.Fatal("body refusal must not unconstruct default objects", e)
	}
	if e2 := m.endFastFormatting(e); e2 != e {
		t.Fatal("original action cause", e2, e)
	}
	for _, raw := range []memory.Block{first, second, small} {
		if _, e = r.Size(raw); e == nil {
			t.Fatal("completed source destructor omitted raw")
		}
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

func TestFastFloatUnsigned1024SourceBoundaries(t *testing.T) {
	m := fastManager(t, 2)
	r := m.natural.runtime
	for _, n := range []uint32{1023, 1024, 1025, 2049} {
		serviceMust(t, m.beginFastFormatting())
		m.formatting.completed = 1
		s := &m.formatting.large[0]
		before, e := r.AllocationCount()
		serviceMust(t, e)
		serviceMust(t, s.resize(r, n))
		serviceMust(t, s.write(r, n-1, 0xdeadbeef))
		for i := uint32(0); i < n; i++ {
			v, e := s.read(r, i)
			serviceMust(t, e)
			want := uint32(0)
			if i == n-1 {
				want = 0xdeadbeef
			}
			if v != want {
				t.Fatal(n, i, v)
			}
		}
		capacity, events := uint32(1024), uint64(0)
		for capacity < n {
			capacity *= 2
			events++
		}
		after, e := r.AllocationCount()
		serviceMust(t, e)
		if s.capacity() != capacity || after-before != events {
			t.Fatal(n, s.capacity(), after-before)
		}
		serviceMust(t, m.endFastFormatting(nil))
		fastAudit(t, m)
		fastStandaloneBudgetAudit(t, m)
		if r.Stats().LiveBlocks != 0 {
			t.Fatal("completed local destructor", n)
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
			charRefusal = charRefusal || m.natural.charFormatting.pending
			innerRefusal = innerRefusal || m.natural.formatting.pending
			defaultRefusal = defaultRefusal || (!m.natural.charFormatting.pending && !m.natural.formatting.pending)
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
