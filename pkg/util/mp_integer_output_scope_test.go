// Source mpz.cpp:1385-1395; mpn.cpp:360-399; buffer.h:84-95.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerCharacterConstructorBeforeLimbLocals(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
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
	if e = m.Natural.reserveText(2); e != nil {
		t.Fatal(e)
	}
	before, e := m.Memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e = m.prepareInteger(&a, 0)
	if !numericPhysicalReservationRefusal(e) || m.Natural.textRead(0) != '-' {
		t.Fatal("source sign/constructor prefix", e)
	}
	s := &m.Natural.CharFormatting
	if !s.pending || s.Completed || s.Buffer.pos != 1024 || m.Natural.Formatting.Active || m.Natural.Formatting.PendingField {
		t.Fatal("char ctor must precede limb constructors", s.Buffer.pos)
	}
	after, e := m.Memory.AllocationCount()
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
	if SourceExtendedGCDDebug {
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
	s := &m.CharFormatting
	if !s.Completed || s.Buffer.pos != 2049 || s.Buffer.capacity() != 4096 || m.runtime.Stats().LiveBlocks != 1 {
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
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
		if e != nil {
			t.Fatal(e)
		}
		digits := make([]uint32, 130)
		digits[0], digits[129] = 7, 1
		var a Int
		if e = m.SetLimbs(&a, digits); e != nil {
			t.Fatal(e)
		}
		if e = m.Natural.reserveText(4096); e != nil {
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
			if m.Natural.Formatting.PendingField {
				i := m.Natural.Formatting.failedLocal
				if i < 2 {
					found[i] = true
				}
				// Completed char is destroyed after completed inner locals;
				// the failed inner constructor's raw prefix remains owned.
				if m.Natural.CharFormatting.active || m.Natural.CharFormatting.Buffer.Raw != (Block{}) || m.Natural.Formatting.CompletedField != 0 {
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
	s := &m.CharFormatting
	if s.Buffer.pos != 0 || s.Buffer.capacity() != 1024 {
		t.Fatal("wrapped size must retain true inline backing")
	}
	e := m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{character: &s.Buffer, assembly: m}, 0)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		for i, v := range []byte("4294967296\x00") {
			got, e := s.Buffer.readPhysical(m.runtime, uint64(i))
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
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		e = m.Natural.beginIntegerCharacters(2049)
		s := &m.Natural.CharFormatting
		if e != nil && s.Buffer.Raw != (Block{}) {
			found = true
			if !numericPhysicalReservationRefusal(e) || s.Completed || !s.pending || s.Buffer.pos != 2048 {
				t.Fatal("failed ctor physical prefix", fault, e, s.Buffer.pos)
			}
			raw := s.Buffer.Raw
			integerPhysicalAudit(t, m)
			b.FailAfter(-1)
			if value, e := m.String(&Int{Val: 19}); e != nil || value != "19" || s.Buffer.Raw != raw {
				t.Fatal("unused pending char must not poison small output", value, e)
			}
			if e = m.SourceFinalize(); e != nil {
				t.Fatal(e)
			}
			if !m.SourceFinalized || s.Buffer.Raw != raw || m.Memory.Stats().LiveBlocks != 1 || m.Natural == nil || m.Natural.HeaderBytes == 0 {
				t.Fatal("source member destruction reclaimed failed char local", m.Memory.Stats())
			}
			if v, e := m.Memory.Read(raw, 0); e != nil || v != 0 {
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
	ref, e := m.Allocate(8, 0)
	if e != nil {
		t.Fatal(e)
	}
	a := Int{Cell: ref, Val: 1}
	m.PublishCell(ref)
	if _, e = m.prepareInteger(&a, 0); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal("actual size access must precede char ctor", e)
	}
	if m.Natural.CharFormatting.active || m.Natural.CharFormatting.pending || m.Natural.Formatting.Active {
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
	if e := m.Z.SetLimbs(&a.Num, []uint32{0, 1}); e != nil {
		t.Fatal(e)
	}
	digits := make([]uint32, 130)
	digits[0], digits[129] = 7, 1
	if e := m.Z.SetLimbs(&a.Den, digits); e != nil {
		t.Fatal(e)
	}
	if e := m.Z.Natural.reserveText(4096); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e := m.prepareRational(&a)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	want := []byte("4294967296/")
	for i, v := range want {
		if m.Z.Natural.textRead(uint32(i)) != v {
			t.Fatal("source NUM insertion and slash before DEN ctor", i)
		}
	}
	if !m.Z.Natural.CharFormatting.pending || m.Z.Natural.CharFormatting.Completed || m.Z.Natural.Formatting.PendingField {
		t.Fatal("source DEN character ctor refusal")
	}
	integerPhysicalAudit(t, m.Z, &a.Num, &a.Den)
	b.FailAfter(-1)
}

func TestIntegerCharacterLatePhysicalFailureKeepsBothCtorRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Memory: rt, Profile: SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	if e = rt.SetMaxCount(1); e != nil {
		t.Fatal(e)
	}
	e = m.Natural.beginIntegerCharacters(70000)
	var refusal Failure
	s := &m.Natural.CharFormatting
	if !errors.As(e, &refusal) || refusal.Kind != FailureAllocExceeded || s.Completed || !s.pending || s.Buffer.pos != 32768 || s.Buffer.Raw == (Block{}) || s.Buffer.PendingField == (Block{}) {
		t.Fatal("real threshold sync after new physical allocation", e, s.Buffer.pos)
	}
	raw, pending := s.Buffer.Raw, s.Buffer.PendingField
	integerPhysicalAudit(t, m)
	if e = m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if s.Buffer.Raw != raw || s.Buffer.PendingField != pending || rt.Stats().LiveBlocks != 2 {
		t.Fatal("source dtor invented failed constructor frees", rt.Stats())
	}
	if e = m.CloseChecked(); e != nil || rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes {
		t.Fatal("explicit owning roots cleanup", e, b.Used(), rt.Stats())
	}
	if e = rt.Close(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}
