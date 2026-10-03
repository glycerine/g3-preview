// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
// Source mpz.cpp:1385-1395; mpn.cpp:360-399; buffer.h:84-95.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerCharacterConstructorBeforeLimbLocals(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	digits := make([]uint32, 130)
	digits[0], digits[129] = 7, 1
	var a Int
	if e = m.SetLimbs(&a, digits); e != nil {
		t.Fatal(e)
	}
	if e = m.Neg(&a); e != nil {
		t.Fatal(e)
	}
	if e = m.natural.reserveText(2); e != nil {
		t.Fatal(e)
	}
	before, e := m.memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e = m.prepareInteger(&a, 0)
	if !numericPhysicalReservationRefusal(e) || m.natural.textRead(0) != '-' {
		t.Fatal("source sign/constructor prefix", e)
	}
	s := &m.natural.charFormatting
	if !s.pending || s.completed || s.buffer.pos != 1024 || m.natural.formatting.active || m.natural.formatting.pending {
		t.Fatal("char ctor must precede limb constructors", s.buffer.pos)
	}
	after, e := m.memory.AllocationCount()
	if e != nil || after != before+1 {
		t.Fatal("actual attempted char malloc", before, after, e)
	}
	integerPhysicalAudit(t, m, &a)
	b.FailAfter(-1)
	if e = m.CloseChecked(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}

func TestSourceCStringPhysicalBytesAndLbuf(t *testing.T) {
	m := naturalManager(t)
	var c sourceCharBuffer
	if e := c.writePhysical(m.runtime, 9, 'x'); e != nil {
		t.Fatal(e)
	}
	if c.pos != 0 {
		t.Fatal("Cptr write changed logical buffer size")
	}
	if v, e := c.readPhysical(m.runtime, 9); e != nil || v != 'x' {
		t.Fatal(v, e)
	}
	for _, test := range []struct {
		limit uint64
		want  []byte
	}{{2, []byte{'1', 0, '?', '?'}}, {99, []byte{'1', '2', '3', 0}}} {
		dst := []byte{'?', '?', '?', '?'}
		if e := m.sourceToCString(limbView{external: []uint32{123}, size: 1}, sourceDecimalDestination{external: dst}, test.limit); e != nil {
			t.Fatal(e)
		}
		for i, v := range test.want {
			if dst[i] != v {
				t.Fatal(test.limit, dst)
			}
		}
	}
	dst := []byte{'o', 'l', 'd', 0}
	e := m.sourceToCString(limbView{external: []uint32{123}, size: 1}, sourceDecimalDestination{external: dst}, 0)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("source lbuf assertion", e)
		}
	} else if e != nil || string(dst) != "old\x00" {
		t.Fatal("snprintf zero writes nothing", dst, e)
	}
	dst = []byte{'?', 0}
	if e = m.sourceToCString(limbView{external: []uint32{7}, size: 1}, sourceDecimalDestination{external: dst}, ^uint64(0)); e != nil || dst[0] != '7' || dst[1] != 0 {
		t.Fatal("large virtual lbuf tiny actual result", dst, e)
	}
}

func TestIntegerCharacterScopeActualCapacityAndCleanup(t *testing.T) {
	m := naturalManager(t)
	if e := m.beginIntegerCharacters(2049); e != nil {
		t.Fatal(e)
	}
	s := &m.charFormatting
	if !s.completed || s.buffer.pos != 2049 || s.buffer.capacity() != 4096 || m.runtime.Stats().LiveBlocks != 1 {
		t.Fatal("actual doubled source character storage")
	}
	if e := m.destroyIntegerCharacters(); e != nil {
		t.Fatal(e)
	}
	if m.runtime.Stats().LiveBlocks != 0 || s.active || s.pending {
		t.Fatal("completed char destructor")
	}
}

func TestIntegerCharacterCompletedBeforeFailedInnerConstructor(t *testing.T) {
	found := [2]bool{}
	success := false
	for fault := int64(0); fault < 100; fault++ {
		b := arena.NewBudget(1 << 24)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		if e != nil {
			t.Fatal(e)
		}
		digits := make([]uint32, 130)
		digits[0], digits[129] = 7, 1
		var a Int
		if e = m.SetLimbs(&a, digits); e != nil {
			t.Fatal(e)
		}
		if e = m.natural.reserveText(4096); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		_, e = m.prepareInteger(&a, 0)
		if e == nil {
			success = true
		} else {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fault, e)
			}
			if m.natural.formatting.pending {
				i := m.natural.formatting.failedLocal
				if i < 2 {
					found[i] = true
				}
				// Completed char is destroyed after completed inner locals;
				// the failed inner constructor's raw prefix remains owned.
				if m.natural.charFormatting.active || m.natural.charFormatting.buffer.raw != (memory.Block{}) || m.natural.formatting.completed != 0 {
					t.Fatal("wrong unwind prefix", fault, i)
				}
			}
		}
		integerPhysicalAudit(t, m, &a)
		b.FailAfter(-1)
		if e = m.CloseChecked(); e != nil || b.Used() != 0 {
			t.Fatal(fault, e, b.Used())
		}
		if success {
			break
		}
	}
	if !success || !found[0] || !found[1] {
		t.Fatal("bounded complete prefix sweep", found, success)
	}
}

func TestIntegerCharacterWrappedSizePhysicalCapacity(t *testing.T) {
	m := naturalManager(t)
	if e := m.beginIntegerCharacters(0); e != nil {
		t.Fatal(e)
	}
	s := &m.charFormatting
	if s.buffer.pos != 0 || s.buffer.capacity() != 1024 {
		t.Fatal("wrapped size must retain true inline backing")
	}
	e := m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{character: &s.buffer, assembly: m}, 0)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		for i, v := range []byte("4294967296\x00") {
			got, e := s.buffer.readPhysical(m.runtime, uint64(i))
			if e != nil || got != v {
				t.Fatal(i, got, e)
			}
		}
	}
	if e = m.destroyIntegerCharacters(); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerCharacterFailedConstructorSurvivesSourceFinalize(t *testing.T) {
	found := false
	for fault := int64(0); fault < 12; fault++ {
		b := arena.NewBudget(1 << 24)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		e = m.natural.beginIntegerCharacters(2049)
		s := &m.natural.charFormatting
		if e != nil && s.buffer.raw != (memory.Block{}) {
			found = true
			if !numericPhysicalReservationRefusal(e) || s.completed || !s.pending || s.buffer.pos != 2048 {
				t.Fatal("failed ctor physical prefix", fault, e, s.buffer.pos)
			}
			raw := s.buffer.raw
			integerPhysicalAudit(t, m)
			b.FailAfter(-1)
			if value, e := m.String(&Int{val: 19}); e != nil || value != "19" || s.buffer.raw != raw {
				t.Fatal("unused pending char must not poison small output", value, e)
			}
			if e = m.SourceFinalize(); e != nil {
				t.Fatal(e)
			}
			if !m.sourceFinalized || s.buffer.raw != raw || m.memory.Stats().LiveBlocks != 1 || m.natural == nil || m.natural.headerBytes == 0 {
				t.Fatal("source member destruction reclaimed failed char local", m.memory.Stats())
			}
			if v, e := m.memory.Read(raw, 0); e != nil || v != 0 {
				t.Fatal(v, e)
			}
			if _, e = m.String(&Int{}); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("ordinary destroyed owner read", e)
			}
		}
		b.FailAfter(-1)
		if e = m.CloseChecked(); e != nil || b.Used() != 0 {
			t.Fatal(e, b.Used())
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("missing actual failed ctor raw root")
	}
}

func TestIntegerCharacterUsesActualInitializedSizeHeader(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	ref, e := m.allocate(8, 0)
	if e != nil {
		t.Fatal(e)
	}
	a := Int{cell: ref, val: 1}
	m.publishCell(ref)
	if _, e = m.prepareInteger(&a, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("actual size access must precede char ctor", e)
	}
	if m.natural.charFormatting.active || m.natural.charFormatting.pending || m.natural.formatting.active {
		t.Fatal("invented local construction")
	}
	integerPhysicalAudit(t, m, &a)
	if e = m.CloseChecked(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}

func TestIntegerCharacterRationalNumeratorBeforeDenominator(t *testing.T) {
	m, b := rationalManager(t, false)
	a := NewRational()
	if e := m.z.SetLimbs(&a.num, []uint32{0, 1}); e != nil {
		t.Fatal(e)
	}
	digits := make([]uint32, 130)
	digits[0], digits[129] = 7, 1
	if e := m.z.SetLimbs(&a.den, digits); e != nil {
		t.Fatal(e)
	}
	if e := m.z.natural.reserveText(4096); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e := m.prepareRational(&a)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	want := []byte("4294967296/")
	for i, v := range want {
		if m.z.natural.textRead(uint32(i)) != v {
			t.Fatal("source NUM insertion and slash before DEN ctor", i)
		}
	}
	if !m.z.natural.charFormatting.pending || m.z.natural.charFormatting.completed || m.z.natural.formatting.pending {
		t.Fatal("source DEN character ctor refusal")
	}
	integerPhysicalAudit(t, m.z, &a.num, &a.den)
	b.FailAfter(-1)
}

func TestIntegerCharacterLatePhysicalFailureKeepsBothCtorRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	if e = rt.SetMaxCount(1); e != nil {
		t.Fatal(e)
	}
	e = m.natural.beginIntegerCharacters(70000)
	var refusal memory.Failure
	s := &m.natural.charFormatting
	if !errors.As(e, &refusal) || refusal.Kind != memory.FailureAllocExceeded || s.completed || !s.pending || s.buffer.pos != 32768 || s.buffer.raw == (memory.Block{}) || s.buffer.pending == (memory.Block{}) {
		t.Fatal("real threshold sync after new physical allocation", e, s.buffer.pos)
	}
	raw, pending := s.buffer.raw, s.buffer.pending
	integerPhysicalAudit(t, m)
	if e = m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if s.buffer.raw != raw || s.buffer.pending != pending || rt.Stats().LiveBlocks != 2 {
		t.Fatal("source dtor invented failed constructor frees", rt.Stats())
	}
	if e = m.CloseChecked(); e != nil || rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes {
		t.Fatal("explicit owning roots cleanup", e, b.Used(), rt.Stats())
	}
	if e = rt.Close(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}
