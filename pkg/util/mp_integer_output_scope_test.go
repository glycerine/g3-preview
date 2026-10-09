// Source mpz.cpp:1385-1395; mpn.cpp:360-399; buffer.h:84-95.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSourceCStringBoundedBytesAndLbuf(t *testing.T) {
	m := naturalManager(t)
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

func TestIntegerCharacterNativeScratchAndCleanup(t *testing.T) {
	m := naturalManager(t)
	if e := m.beginIntegerCharacters(2049); e != nil {
		t.Fatal(e)
	}
	s := &m.CharFormatting
	if !s.Completed || len(s.Buffer) != 2049 || m.runtime.Stats().LiveBlocks != 0 {
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
				if m.Natural.CharFormatting.active || len(m.Natural.CharFormatting.Buffer) != 0 || m.Natural.Formatting.CompletedField != 0 {
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
	if m.Z.Natural.CharFormatting.pending || m.Z.Natural.CharFormatting.Completed || !m.Z.Natural.Formatting.PendingField {
		t.Fatal("DEN numerical workspace refusal must retire native character scope")
	}
	integerPhysicalAudit(t, m.Z, &a.Num, &a.Den)
	b.FailAfter(-1)
}
