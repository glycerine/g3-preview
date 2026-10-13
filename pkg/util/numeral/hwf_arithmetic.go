// Source: src/util/hwf.cpp:222-255,586-650.
package numeral

import (
	"errors"
	"math"
	"math/big"
)

var ErrHardwareFloatRoundingMode = errors.New("numeral: unsupported hardware-float rounding mode")

// roundHardwareFloat rounds an exact finite result to binary64. Rat.Float64
// supplies nearest-even; one adjacent value gives each directed result.
func roundHardwareFloat(exact *big.Rat, mode ArbitraryFloatRoundingMode, negativeZero bool) float64 {
	if exact.Sign() == 0 {
		if negativeZero {
			return math.Copysign(0, -1)
		}
		return 0
	}
	nearest, precise := exact.Float64()
	if precise || mode == ArbitraryFloatNearestEven {
		return nearest
	}
	var relation int
	if math.IsInf(nearest, 1) {
		relation = 1
	} else if math.IsInf(nearest, -1) {
		relation = -1
	} else {
		relation = new(big.Rat).SetFloat64(nearest).Cmp(exact)
	}
	if (mode == ArbitraryFloatTowardPositive || mode == ArbitraryFloatTowardZero && exact.Sign() < 0) && relation < 0 {
		return math.Nextafter(nearest, math.Inf(1))
	}
	if (mode == ArbitraryFloatTowardNegative || mode == ArbitraryFloatTowardZero && exact.Sign() > 0) && relation > 0 {
		return math.Nextafter(nearest, math.Inf(-1))
	}
	return nearest
}

func (m *HardwareFloatManager) arithmetic(mode ArbitraryFloatRoundingMode, x, y, out *HardwareFloat, op byte) error {
	if m == nil || x == nil || y == nil || out == nil {
		return ErrNumericOwner
	}
	// The source's set_rounding_mode rejects nearest-away on this platform.
	if mode != ArbitraryFloatNearestEven && mode != ArbitraryFloatTowardPositive && mode != ArbitraryFloatTowardNegative && mode != ArbitraryFloatTowardZero {
		return ErrHardwareFloatRoundingMode
	}
	a, b := x.value, y.value
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) || op == '/' && b == 0 {
		switch op {
		case '+':
			out.value = a + b
		case '-':
			out.value = a - b
		case '*':
			out.value = a * b
		case '/':
			out.value = a / b
		}
		return nil
	}
	ra, rb := new(big.Rat).SetFloat64(a), new(big.Rat).SetFloat64(b)
	exact := new(big.Rat)
	negativeZero := false
	switch op {
	case '+':
		exact.Add(ra, rb)
		negativeZero = (a == 0 && b == 0 && math.Signbit(a) && math.Signbit(b)) || (mode == ArbitraryFloatTowardNegative && (a != 0 || b != 0 || math.Signbit(a) != math.Signbit(b)))
	case '-':
		exact.Sub(ra, rb)
		negativeZero = (a == 0 && b == 0 && math.Signbit(a) && !math.Signbit(b)) || (mode == ArbitraryFloatTowardNegative && (a != 0 || b != 0 || math.Signbit(a) == math.Signbit(b)))
	case '*':
		exact.Mul(ra, rb)
		negativeZero = math.Signbit(a) != math.Signbit(b)
	case '/':
		exact.Quo(ra, rb)
		negativeZero = math.Signbit(a) != math.Signbit(b)
	}
	out.value = roundHardwareFloat(exact, mode, negativeZero)
	return nil
}

func (m *HardwareFloatManager) Add(mode ArbitraryFloatRoundingMode, x, y, out *HardwareFloat) error {
	return m.arithmetic(mode, x, y, out, '+')
}
func (m *HardwareFloatManager) Sub(mode ArbitraryFloatRoundingMode, x, y, out *HardwareFloat) error {
	return m.arithmetic(mode, x, y, out, '-')
}
func (m *HardwareFloatManager) Mul(mode ArbitraryFloatRoundingMode, x, y, out *HardwareFloat) error {
	return m.arithmetic(mode, x, y, out, '*')
}
func (m *HardwareFloatManager) Div(mode ArbitraryFloatRoundingMode, x, y, out *HardwareFloat) error {
	return m.arithmetic(mode, x, y, out, '/')
}
