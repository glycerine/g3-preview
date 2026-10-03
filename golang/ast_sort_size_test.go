// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Standalone source sort_size boundary checks; no active source ENSURE exists.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
	"strconv"
	"testing"
)

func TestSortSizeSourceDefaultFactoriesAndGuards(t *testing.T) {
	var zero SortSize
	for _, s := range []SortSize{zero, InfiniteSortSize()} {
		if !s.IsInfinite() || s.IsFinite() || s.IsVeryBig() || s.String() != "infinite" {
			t.Fatal(s)
		}
		if _, e := s.Size(); !errors.Is(e, ErrSortSizeNotFinite) {
			t.Fatal(e)
		}
	}
	big := VeryBigSortSize()
	if !big.IsVeryBig() || big.IsInfinite() || big.IsFinite() || big.String() != "very-big" {
		t.Fatal(big)
	}
	if _, e := big.Size(); !errors.Is(e, ErrSortSizeNotFinite) {
		t.Fatal(e)
	}
	for _, n := range []uint64{0, 1, 2, 63, 64, math.MaxUint32, 1 << 63, math.MaxUint64} {
		for _, s := range []SortSize{NewSortSize(n), FiniteSortSize(n)} {
			copy := s
			if !copy.IsFinite() || copy.IsVeryBig() || copy.IsInfinite() {
				t.Fatal(copy)
			}
			got, e := copy.Size()
			if e != nil || got != n || copy.String() != strconv.FormatUint(n, 10) {
				t.Fatal(n, got, e, copy.String())
			}
		}
	}
	for _, n := range []uint32{0, 1, 63, 64, 65, math.MaxUint32} {
		if SortSizeIsVeryBigBase2(n) != (n >= 64) {
			t.Fatal(n)
		}
	}
}
func TestSortSizeRationalActualUint64Classification(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	config := mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}}
	m, e := mpnum.NewRationalService(config)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		m.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	v, _ := m.NewValue()
	defer m.Del(&v)
	for _, n := range []uint64{0, 1, math.MaxInt64, math.MaxUint64} {
		if e = m.SetUint64(&v, n); e != nil {
			t.Fatal(e)
		}
		before := b.Used()
		stats, _ := m.Stats()
		b.FailAfter(0)
		s, e := SortSizeFromRational(m, &v)
		b.FailAfter(-1)
		after, _ := m.Stats()
		got, _ := s.Size()
		if e != nil || !s.IsFinite() || got != n || before != b.Used() || stats != after {
			t.Fatal(n, s, e)
		}
	}
	for _, text := range []string{"-1", "-18446744073709551615", "1/2", "-1/2", "18446744073709551616"} {
		if e = m.SetString(&v, text); e != nil {
			t.Fatal(e)
		}
		before := b.Used()
		s, e := SortSizeFromRational(m, &v)
		if e != nil || !s.IsVeryBig() || b.Used() != before {
			t.Fatal(text, s, e)
		}
	}
	// Exact rational integer result follows IsUint64, not lexical denominator shape.
	if e = m.SetFraction(&v, 4, 2); e != nil {
		t.Fatal(e)
	}
	s, e := SortSizeFromRational(m, &v)
	n, _ := s.Size()
	if e != nil || n != 2 {
		t.Fatal(s, e)
	}
}
func TestSortSizeRationalBorrowedOwnerLifetime(t *testing.T) {
	m, _ := mpnum.NewRationalService(mpnum.RationalServiceConfig{})
	other, _ := mpnum.NewRationalService(mpnum.RationalServiceConfig{})
	defer other.Close()
	v, _ := m.NewValue()
	m.SetUint64(&v, math.MaxUint64)
	s, e := SortSizeFromRational(m, &v)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = SortSizeFromRational(other, &v); !errors.Is(e, mpnum.ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if _, e = SortSizeFromRational(m, nil); !errors.Is(e, mpnum.ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if _, e = SortSizeFromRational(nil, &v); e == nil {
		t.Fatal("nil")
	}
	m.Del(&v)
	if _, e = SortSizeFromRational(m, &v); !errors.Is(e, mpnum.ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	m.Close()
	if _, e = SortSizeFromRational(m, &v); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	n, e := s.Size()
	if e != nil || n != math.MaxUint64 {
		t.Fatal(s, e)
	}
}
func TestSortSizeIndependentModelPointerFreedomAndWarmAllocations(t *testing.T) {
	if e := arena.ValidateRecordType(reflect.TypeOf(SortSize{})); e != nil {
		t.Fatal(e)
	}
	state := uint64(1)
	for i := 0; i < 2000; i++ {
		state = state*6364136223846793005 + 1
		s := FiniteSortSize(state)
		if s.String() != strconv.FormatUint(state, 10) {
			t.Fatal(state)
		}
	}
	s := FiniteSortSize(123)
	if n := testing.AllocsPerRun(100, func() {
		if !s.IsFinite() || s.IsInfinite() || s.IsVeryBig() {
			panic("predicate")
		}
		s.Size()
		SortSizeIsVeryBigBase2(64)
	}); n != 0 {
		t.Fatal(n)
	}
}
