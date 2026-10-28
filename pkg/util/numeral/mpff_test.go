package numeral

// Source src/test/mpff.cpp.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func fastManager(t *testing.T, p uint32) *FastFloatManager {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
	c.PrecisionWords = p
	m, e := NewFastFloatManager(c)
	serviceMust(t, e)
	t.Cleanup(func() {
		m.Close()
		if b.Used() != 0 {
			t.Errorf("retained fast float %d", b.Used())
		}
	})
	return m
}
func fastText(t *testing.T, m *FastFloatManager, a *FastFloat, w string) {
	t.Helper()
	s, e := m.String(a)
	if e != nil || s != w {
		t.Fatalf("%q %v want %q", s, e, w)
	}
}
func fastEnsure(t *testing.T, ok bool, e error) {
	t.Helper()
	serviceMust(t, e)
	if !ok {
		t.Fatal("source assertion")
	}
}
func TestFastFloatSourceBasic(t *testing.T) {
	m := fastManager(t, 2)
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	b, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	defer a.Close()
	defer b.Close()
	av, e := a.Get()
	serviceMust(t, e)
	bv, e := b.Get()
	serviceMust(t, e)
	serviceMust(t, m.SetUint64(av, 1<<63))
	ok, e := m.IsZero(bv)
	fastEnsure(t, ok, e)
	ok, e = m.Less(bv, av)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetInt32(bv, -1))
	ok, e = m.Less(bv, av)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetFraction32(av, 1, 3))
	serviceMust(t, b.Assign(a))
	serviceMust(t, m.Next(bv))
	ok, e = m.Less(av, bv)
	fastEnsure(t, ok, e)
	serviceMust(t, m.Prev(bv))
	ok, e = m.Equal(av, bv)
	fastEnsure(t, ok, e)
	serviceMust(t, m.CeilInPlace(bv))
	fastText(t, m, bv, "1")
	serviceMust(t, m.SetFraction32(bv, 4, 3))
	serviceMust(t, m.CeilInPlace(bv))
	fastText(t, m, bv, "2")
}
func TestFastFloatSourceDecimal(t *testing.T) {
	for _, tc := range []struct {
		n         int64
		d         uint64
		up        bool
		p, places uint32
		w         string
	}{
		{1, 3, false, 2, ^uint32(0), "0.3333333333333333333152632971252415927665424533188343048095703125"},
		{1, 3, false, 4, ^uint32(0), "0.33333333333333333333333333333333333333235375470764809374335938621898146193515111203602326039874270691143465228378772735595703125"},
		{-1, 3, true, 2, ^uint32(0), "-0.3333333333333333333152632971252415927665424533188343048095703125"},
		{-1, 3, false, 2, ^uint32(0), "-0.33333333333333333334236835143737920361672877334058284759521484375"},
		{0, 1, false, 2, ^uint32(0), "0"}, {2, 1, false, 2, ^uint32(0), "2"}, {-3, 1, false, 2, ^uint32(0), "-3"},
		{9223372036854775807, 1, false, 2, ^uint32(0), "9223372036854775807"},
		{4, 5, false, 2, ^uint32(0), "0.79999999999999999995663191310057982263970188796520233154296875"},
		{4, 5, false, 2, 10, "0.7999999999?"},
		{32, 5, true, 2, ^uint32(0), "6.4000000000000000000867361737988403547205962240695953369140625"},
		{32, 5, false, 2, ^uint32(0), "6.39999999999999999965305530480463858111761510372161865234375"},
		{-32, 5, false, 2, ^uint32(0), "-6.4000000000000000000867361737988403547205962240695953369140625"},
		{-32, 5, true, 2, ^uint32(0), "-6.39999999999999999965305530480463858111761510372161865234375"},
	} {
		m := fastManager(t, tc.p)
		var a FastFloat
		serviceMust(t, m.SetRounding(tc.up))
		serviceMust(t, m.SetFraction64(&a, tc.n, tc.d))
		ordinary, e := m.String(&a)
		serviceMust(t, e)
		sig, exp := fastFractionModel(tc.n, tc.d, tc.p, tc.up)
		wantOrdinary := fastModelText(sig, exp)
		if tc.n < 0 {
			wantOrdinary = "-" + wantOrdinary
		}
		if ordinary != wantOrdinary {
			t.Fatal("source printed ordinary", ordinary, wantOrdinary)
		}
		printed, e := m.DecimalString(&a, tc.places, 128)
		serviceMust(t, e)
		if printed != tc.w {
			t.Fatal("source printed decimal", printed, tc.w)
		}
		s, e := m.DecimalString(&a, tc.places, 128)
		if e != nil || s != tc.w {
			t.Fatalf("decimal %v %q %v", tc, s, e)
		}
		serviceMust(t, m.Del(&a))
	}
}
