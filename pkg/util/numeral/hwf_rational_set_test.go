package numeral

import (
	"errors"
	"math"
	"testing"
)

func TestHardwareFloatSourceRationalInputDirectedBounds(t *testing.T) {
	qm, _ := rationalManager(t, false)
	m := NewHardwareFloatManager()
	q := NewRational()
	defer qm.Del(&q)
	if e := qm.SetString(&q, "1/3"); e != nil {
		t.Fatal(e)
	}
	var out HardwareFloat
	if e := m.SetRational(&out, ArbitraryFloatTowardPositive, qm, &q); e != nil || out.value != math.Nextafter(1.0/3.0, math.Inf(1)) {
		t.Fatal("upper rational bound", out, e)
	}
	if e := m.SetRational(&out, ArbitraryFloatTowardNegative, qm, &q); e != nil || out.value != 1.0/3.0 {
		t.Fatal("lower rational bound", out, e)
	}
	if e := qm.SetString(&q, "-1/3"); e != nil {
		t.Fatal(e)
	}
	if e := m.SetRational(&out, ArbitraryFloatTowardNegative, qm, &q); e != nil || out.value != math.Nextafter(-1.0/3.0, math.Inf(-1)) {
		t.Fatal("negative lower bound", out, e)
	}
	if e := m.SetRatioInt32(&out, ArbitraryFloatTowardPositive, 1, 0); e != nil || !math.IsInf(out.value, 1) {
		t.Fatal("source division by zero", out, e)
	}
	nm, e := NewHardwareFloatNumeralManager(m, 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	if e := nm.SetRatioInt32(&out, 1, 0); !errors.Is(e, ErrHardwareFloatNonRegular) || !math.IsInf(out.value, 1) {
		t.Fatal("wrapper regular-result check", out, e)
	}
	if e := nm.SetRational(&out, qm, &q); e != nil || out.value != -1.0/3.0 {
		t.Fatal("wrapper upward rational bound", out, e)
	}
}

func TestHardwareFloatSourceRationalInputLargeIntegerAndGuards(t *testing.T) {
	qm, _ := rationalManager(t, false)
	m := NewHardwareFloatManager()
	q := NewRational()
	defer qm.Del(&q)
	if e := qm.SetString(&q, "9007199254740993"); e != nil {
		t.Fatal(e)
	}
	var up, down HardwareFloat
	if e := m.SetRational(&up, ArbitraryFloatTowardPositive, qm, &q); e != nil {
		t.Fatal(e)
	}
	if e := m.SetRational(&down, ArbitraryFloatTowardNegative, qm, &q); e != nil {
		t.Fatal(e)
	}
	// UINT_MAX rather than 2^32 is the source's floating radix. This
	// particular projected value is exactly representable in both modes.
	if up.value != 9007199252643841 || down.value != up.value {
		t.Fatal("large integer source radix projection", down, up)
	}
	before := up.RawBits()
	if e := m.SetRational(&up, ArbitraryFloatNearestAway, qm, &q); !errors.Is(e, ErrHardwareFloatRoundingMode) || up.RawBits() != before {
		t.Fatal("rejected rounding changed output", up, e)
	}
	if e := m.SetRational(&up, ArbitraryFloatNearestEven, nil, &q); !errors.Is(e, ErrNumericOwner) || up.RawBits() != before {
		t.Fatal("missing rational manager changed output", up, e)
	}
}
