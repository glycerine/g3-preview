package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatNumeralSourceDirectedSetAndArithmetic(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, e := NewArbitraryFloatNumeralManager(core, 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	}()
	if m.RoundingMode() != ArbitraryFloatTowardPositive || m.ExponentBits() != 11 || m.SignificandBits() != 53 {
		t.Fatal("source constructor fields")
	}
	var x, y, out ArbitraryFloat
	defer core.Del(&x)
	defer core.Del(&y)
	defer core.Del(&out)
	if e := m.SetFractionSigned(&out, 1, 3); e != nil {
		t.Fatal(e)
	}
	upper, e := core.ToDouble(&out)
	if e != nil || upper != math.Nextafter(1.0/3.0, math.Inf(1)) {
		t.Fatal("upper fraction", upper, e)
	}
	m.RoundToMinusInf()
	if e := m.SetFractionSigned(&out, 1, 3); e != nil {
		t.Fatal(e)
	}
	lower, e := core.ToDouble(&out)
	if e != nil || lower != 1.0/3.0 {
		t.Fatal("lower fraction", lower, e)
	}
	if e := m.SetInt32(&x, 1); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt32(&y, 3); e != nil {
		t.Fatal(e)
	}
	if e := m.Div(&x, &y, &out); e != nil {
		t.Fatal(e)
	}
	got, e := core.ToDouble(&out)
	if e != nil || got != lower {
		t.Fatal("lower division", got, e)
	}
	if e := m.Add(&x, &y, &out); e != nil {
		t.Fatal(e)
	}
	got, e = core.ToDouble(&out)
	if e != nil || got != 4 {
		t.Fatal("add", got, e)
	}
	if e := m.Sub(&y, &x, &out); e != nil {
		t.Fatal(e)
	}
	got, e = core.ToDouble(&out)
	if e != nil || got != 2 {
		t.Fatal("sub", got, e)
	}
	if e := m.Mul(&x, &y, &out); e != nil {
		t.Fatal(e)
	}
	got, e = core.ToDouble(&out)
	if e != nil || got != 3 {
		t.Fatal("mul", got, e)
	}
}

func TestArbitraryFloatNumeralSourcePostCheckAndClose(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, e := NewArbitraryFloatNumeralManager(core, 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	var out ArbitraryFloat
	if e := m.SetDouble(&out, math.Inf(1)); !errors.Is(e, ErrArbitraryFloatNonRegular) {
		t.Fatal("post-set regular check", e)
	}
	inf, e := core.IsPositiveInf(&out)
	if e != nil || !inf {
		t.Fatal("source output remains published", inf, e)
	}
	if e := core.Del(&out); e != nil {
		t.Fatal(e)
	}
	if e := m.Close(); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt32(&out, 1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed wrapper", e)
	}
	if e := m.Close(); e != nil {
		t.Fatal("idempotent close", e)
	}
}

