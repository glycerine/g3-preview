package numeral

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHardwareFloatSourceBinary64Classification(t *testing.T) {
	m := NewHardwareFloatManager()
	rows := []struct {
		bits                                     uint64
		zero, neg, pos, nan, inf, normal, denorm bool
		integer                                  bool
	}{
		{0, true, false, true, false, false, false, false, false},
		{0x8000000000000000, true, true, false, false, false, false, false, false},
		{0x0000000000000001, false, false, true, false, false, false, true, false},
		{0x0010000000000000, false, false, true, false, false, true, false, false},
		{0x3ff0000000000000, false, false, true, false, false, true, false, true},
		{0x3ff8000000000000, false, false, true, false, false, true, false, false},
		{0x4000000000000000, false, false, true, false, false, true, false, true},
		{0x7ff0000000000000, false, false, true, false, true, false, false, false},
		{0xfff0000000000000, false, true, false, false, true, false, false, false},
		{0x7ff0000000000001, false, false, false, true, false, false, false, false},
		{0xfff8000000000001, false, false, false, true, false, false, false, false},
	}
	for _, row := range rows {
		a := HardwareFloat{value: math.Float64frombits(row.bits)}
		got := [...]bool{m.IsZero(&a), m.IsNeg(&a), m.IsPos(&a), m.IsNaN(&a), m.IsInf(&a), m.IsNormal(&a), m.IsDenormal(&a), m.IsInteger(&a)}
		want := [...]bool{row.zero, row.neg, row.pos, row.nan, row.inf, row.normal, row.denorm, row.integer}
		if got != want {
			t.Fatalf("bits %#x classification %v, want %v", row.bits, got, want)
		}
		if m.IsRegular(&a) != !(row.nan || row.inf) {
			t.Fatalf("bits %#x regular mismatch", row.bits)
		}
		if m.Sign(&a) != (row.bits>>63 != 0) || m.Significand(&a) != row.bits&0x000fffffffffffff || m.Exponent(&a) != int32((row.bits>>52)&0x7ff)-1023 {
			t.Fatalf("bits %#x source fields mismatch", row.bits)
		}
	}
}

func TestHardwareFloatSourceValueConstructionAndCopy(t *testing.T) {
	m := NewHardwareFloatManager()
	var a, b HardwareFloat
	if e := m.MkZero(&a, true); e != nil || !m.IsNegativeZero(&a) {
		t.Fatal(a, e)
	}
	if e := m.Copy(&b, &a); e != nil || b.RawBits() != a.RawBits() {
		t.Fatal(b, e)
	}
	if e := m.SetInt32(&a, 17); e != nil || a.RawBits() != math.Float64bits(17) {
		t.Fatal(a, e)
	}
	if e := m.SetFloat32(&a, float32(1.25)); e != nil || a.RawBits() != math.Float64bits(1.25) {
		t.Fatal(a, e)
	}
	if e := m.SetComponents(&a, true, 1, -1022); e != nil || a.RawBits() != 0x8010000000000001 {
		t.Fatal(a, e)
	}
	before := a.RawBits()
	if e := m.SetComponents(&a, false, uint64(1)<<52, 0); e != arena.ErrBounds || a.RawBits() != before {
		t.Fatal("invalid significand changed output", e)
	}
	if e := m.MkNaN(&a); e != nil || a.RawBits() != 0x7ff0000000000001 {
		t.Fatal(a, e)
	}
	if e := m.MkInf(&b, true); e != nil || !m.IsNegativeInf(&b) {
		t.Fatal(b, e)
	}
	if e := m.Swap(&a, &b); e != nil || !m.IsNegativeInf(&a) || !m.IsNaN(&b) {
		t.Fatal(a, b, e)
	}
	if e := m.Reset(&a); e != nil || !m.IsPositiveZero(&a) {
		t.Fatal(a, e)
	}
}

func TestHardwareFloatSourceSignAndComparison(t *testing.T) {
	m := NewHardwareFloatManager()
	a, b, out := HardwareFloat{}, HardwareFloat{}, HardwareFloat{}
	if e := m.SetFloat64(&a, -1.5); e != nil {
		t.Fatal(e)
	}
	if e := m.SetFloat64(&b, 2); e != nil {
		t.Fatal(e)
	}
	if !m.Less(&a, &b) || !m.LessEqual(&a, &b) || m.Equal(&a, &b) || m.Greater(&a, &b) || m.GreaterEqual(&a, &b) {
		t.Fatal("finite source comparison order")
	}
	if e := m.Abs(&out, &a); e != nil || m.ToFloat64(&out) != 1.5 {
		t.Fatal(out, e)
	}
	if e := m.Neg(&out, &out); e != nil || m.ToFloat64(&out) != -1.5 {
		t.Fatal(out, e)
	}
	if e := m.MkZero(&a, true); e != nil {
		t.Fatal(e)
	}
	if e := m.Abs(&out, &a); e != nil || !m.IsPositiveZero(&out) {
		t.Fatal("absolute value of negative zero", out, e)
	}
	if e := m.Neg(&out, &out); e != nil || !m.IsNegativeZero(&out) || !m.Equal(&a, &out) {
		t.Fatal("negation of positive zero", out, e)
	}
	if e := m.MkNaN(&a); e != nil {
		t.Fatal(e)
	}
	if m.Equal(&a, &a) || m.Less(&a, &b) || m.LessEqual(&a, &b) || m.Greater(&a, &b) || m.GreaterEqual(&a, &b) {
		t.Fatal("NaN comparison must be unordered")
	}
	if e := m.Neg(&out, &a); e != nil || out.RawBits() != a.RawBits()^(uint64(1)<<63) {
		t.Fatal("NaN sign-bit negation", out, e)
	}
}
