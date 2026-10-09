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

func TestIntegerCharacterNativeLargeFormattingWithReservationsDisabled(t *testing.T) {
	m, _ := rationalManager(t, false)
	digits := make([]uint32, 130)
	digits[0], digits[129] = 7, 1
	var a Int
	serviceMust(t, m.Z.SetLimbs(&a, digits))
	serviceMust(t, m.Z.Natural.reserveText(4096))
	m.Z.BudgetField.FailAfter(0)
	_, e := m.Z.prepareInteger(&a, 0)
	serviceMust(t, e)
	if m.Z.Natural.CharFormatting.active || m.Z.Natural.Formatting.Active {
		t.Fatal("formatting scope retained")
	}
	integerPhysicalAudit(t, m.Z, &a)
	m.Z.BudgetField.FailAfter(-1)
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
	if e != nil {
		t.Fatal(e)
	}
	want := []byte("4294967296/")
	for i, v := range want {
		if m.Z.Natural.textRead(uint32(i)) != v {
			t.Fatal("source NUM insertion and slash before DEN ctor", i)
		}
	}
	if m.Z.Natural.CharFormatting.pending || m.Z.Natural.CharFormatting.Completed || m.Z.Natural.Formatting.PendingField {
		t.Fatal("native denominator formatting must retire character scope")
	}
	integerPhysicalAudit(t, m.Z, &a.Num, &a.Den)
	b.FailAfter(-1)
}
