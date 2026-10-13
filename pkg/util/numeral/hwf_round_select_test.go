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
