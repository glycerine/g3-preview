package numeral

import (
	"errors"
	"math"
	"testing"
)

func TestHardwareFloatRoundToIntegralSourceModes(t *testing.T) {
	m := NewHardwareFloatManager()
	rows := []struct {
		mode        ArbitraryFloatRoundingMode
		input, want float64
	}{
		{ArbitraryFloatNearestEven, 2.5, 2},
		{ArbitraryFloatNearestEven, 3.5, 4},
		{ArbitraryFloatNearestEven, -0.5, math.Copysign(0, -1)},
		{ArbitraryFloatTowardPositive, -0.25, math.Copysign(0, -1)},
		{ArbitraryFloatTowardPositive, 1.25, 2},
		{ArbitraryFloatTowardNegative, -1.25, -2},
		{ArbitraryFloatTowardZero, -1.75, -1},
		{ArbitraryFloatTowardZero, -0.25, math.Copysign(0, -1)},
		{ArbitraryFloatTowardNegative, math.Inf(1), math.Inf(1)},
	}
	for _, row := range rows {
		a := HardwareFloat{row.input}
		if err := m.RoundToIntegral(row.mode, &a, &a); err != nil || a.RawBits() != math.Float64bits(row.want) {
			t.Fatalf("mode %d, input %v: bits %#x want %#x: %v", row.mode, row.input, a.RawBits(), math.Float64bits(row.want), err)
		}
	}
	a := HardwareFloat{1.25}
	if err := m.RoundToIntegral(ArbitraryFloatNearestAway, &a, &a); !errors.Is(err, ErrHardwareFloatRoundingMode) || a.value != 1.25 {
		t.Fatal("unsupported rounding changed value", a, err)
	}
}

func TestHardwareFloatSourceRemainderAndSSESelection(t *testing.T) {
	m := NewHardwareFloatManager()
	a, b, out := HardwareFloat{7}, HardwareFloat{4}, HardwareFloat{}
	if err := m.Rem(&a, &b, &out); err != nil || out.value != -1 {
		t.Fatal("IEEE nearest-even remainder", out, err)
	}
	a.value, b.value = 6, 4
	if err := m.Rem(&a, &b, &out); err != nil || out.value != -2 {
		t.Fatal("tie-even quotient remainder", out, err)
	}
	a.value, b.value = math.Copysign(0, -1), 0
	if err := m.Maximum(&a, &b, &out); err != nil || out.RawBits() != 0 {
		t.Fatal("equal max selects second signed zero", out, err)
	}
	if err := m.Minimum(&b, &a, &out); err != nil || out.RawBits() != a.RawBits() {
		t.Fatal("equal min selects second signed zero", out, err)
	}
	a.value, b.value = math.Float64frombits(0x7ff8000000001234), 2
	if err := m.Maximum(&a, &b, &out); err != nil || out.value != 2 {
		t.Fatal("unordered max selects second finite", out, err)
	}
	if err := m.Minimum(&b, &a, &out); err != nil || out.RawBits() != a.RawBits() {
		t.Fatal("unordered min preserves second NaN bits", out, err)
	}
	if err := m.Rem(&a, nil, &out); !errors.Is(err, ErrNumericOwner) {
		t.Fatal(err)
	}
}
