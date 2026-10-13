package numeral

import (
	"errors"
	"math"
	"testing"
)

func TestHardwareFloatDirectedArithmetic(t *testing.T) {
	m := NewHardwareFloatManager()
	third := 1.0 / 3.0
	rows := []struct {
		name string
		op   byte
		a, b float64
		mode ArbitraryFloatRoundingMode
		want float64
	}{
		{"div-nearest", '/', 1, 3, ArbitraryFloatNearestEven, third},
		{"div-up", '/', 1, 3, ArbitraryFloatTowardPositive, math.Nextafter(third, math.Inf(1))},
		{"div-down", '/', 1, 3, ArbitraryFloatTowardNegative, third},
		{"div-zero-negative", '/', -1, 3, ArbitraryFloatTowardZero, -third},
		{"div-down-negative", '/', -1, 3, ArbitraryFloatTowardNegative, math.Nextafter(-third, math.Inf(-1))},
		{"add-up", '+', 1, math.Ldexp(1, -53), ArbitraryFloatTowardPositive, math.Nextafter(1, math.Inf(1))},
		{"add-down", '+', 1, math.Ldexp(1, -53), ArbitraryFloatTowardNegative, 1},
		{"sub-down-zero", '-', 1, 1, ArbitraryFloatTowardNegative, math.Copysign(0, -1)},
		{"mul-overflow-zero", '*', math.MaxFloat64, 2, ArbitraryFloatTowardZero, math.MaxFloat64},
		{"mul-overflow-up", '*', math.MaxFloat64, 2, ArbitraryFloatTowardPositive, math.Inf(1)},
		{"mul-overflow-down-negative", '*', -math.MaxFloat64, 2, ArbitraryFloatTowardNegative, math.Inf(-1)},
		{"mul-underflow-up", '*', math.SmallestNonzeroFloat64, .5, ArbitraryFloatTowardPositive, math.SmallestNonzeroFloat64},
		{"mul-underflow-down", '*', math.SmallestNonzeroFloat64, .5, ArbitraryFloatTowardNegative, 0},
		{"mul-negative-zero", '*', math.Copysign(0, -1), 2, ArbitraryFloatNearestEven, math.Copysign(0, -1)},
		{"add-negative-zero", '+', math.Copysign(0, -1), math.Copysign(0, -1), ArbitraryFloatNearestEven, math.Copysign(0, -1)},
		{"add-mixed-zero", '+', math.Copysign(0, -1), 0, ArbitraryFloatTowardNegative, math.Copysign(0, -1)},
		{"div-zero", '/', 0, -3, ArbitraryFloatTowardPositive, math.Copysign(0, -1)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			x, y, out := HardwareFloat{row.a}, HardwareFloat{row.b}, HardwareFloat{}
			var err error
			switch row.op {
			case '+':
				err = m.Add(row.mode, &x, &y, &out)
			case '-':
				err = m.Sub(row.mode, &x, &y, &out)
			case '*':
				err = m.Mul(row.mode, &x, &y, &out)
			case '/':
				err = m.Div(row.mode, &x, &y, &out)
			}
			if err != nil || out.RawBits() != math.Float64bits(row.want) {
				t.Fatalf("result %#x, want %#x: %v", out.RawBits(), math.Float64bits(row.want), err)
			}
		})
	}
}

func TestHardwareFloatArithmeticGuardsAndAliasing(t *testing.T) {
	m := NewHardwareFloatManager()
	x, y := HardwareFloat{1}, HardwareFloat{3}
	if err := m.Div(ArbitraryFloatTowardPositive, &x, &y, &x); err != nil || x.value != math.Nextafter(1.0/3.0, math.Inf(1)) {
		t.Fatal(x, err)
	}
	before := x.RawBits()
	if err := m.Add(ArbitraryFloatNearestAway, &x, &y, &x); !errors.Is(err, ErrHardwareFloatRoundingMode) || x.RawBits() != before {
		t.Fatal("unsupported rounding changed output", err)
	}
	if err := m.Add(ArbitraryFloatNearestEven, &x, nil, &x); !errors.Is(err, ErrNumericOwner) || x.RawBits() != before {
		t.Fatal("nil input changed output", err)
	}
	zero := HardwareFloat{}
	if err := m.Div(ArbitraryFloatTowardZero, &zero, &zero, &x); err != nil || !math.IsNaN(x.value) {
		t.Fatal("zero/zero", x, err)
	}
}
