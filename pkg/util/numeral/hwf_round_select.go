// Source: src/util/hwf.cpp:297-383 (Linux x86-64 intrinsic path).
package numeral

import "math"

// RoundToIntegral follows the active source rounding mode without changing
// the process-wide FPU environment. The result remains binary64.
func (m *HardwareFloatManager) RoundToIntegral(mode ArbitraryFloatRoundingMode, in, out *HardwareFloat) error {
	if m == nil || in == nil || out == nil {
		return ErrNumericOwner
	}
	x := in.value
	switch mode {
	case ArbitraryFloatNearestEven:
		out.value = math.RoundToEven(x)
	case ArbitraryFloatTowardPositive:
		out.value = math.Ceil(x)
	case ArbitraryFloatTowardNegative:
		out.value = math.Floor(x)
	case ArbitraryFloatTowardZero:
		out.value = math.Trunc(x)
	default:
		return ErrHardwareFloatRoundingMode
	}
	return nil
}

func (m *HardwareFloatManager) Rem(x, y, out *HardwareFloat) error {
	if m == nil || x == nil || y == nil || out == nil {
		return ErrNumericOwner
	}
	out.value = math.Remainder(x.value, y.value)
	return nil
}

// SSE min/max select the second operand for unordered or equal comparisons.
// Copying that operand also preserves its signed zero and NaN payload.
func (m *HardwareFloatManager) Maximum(x, y, out *HardwareFloat) error {
	if m == nil || x == nil || y == nil || out == nil {
		return ErrNumericOwner
	}
	a, b := x.value, y.value
	if a > b {
		out.value = a
	} else {
		out.value = b
	}
	return nil
}

func (m *HardwareFloatManager) Minimum(x, y, out *HardwareFloat) error {
	if m == nil || x == nil || y == nil || out == nil {
		return ErrNumericOwner
	}
	a, b := x.value, y.value
	if a < b {
		out.value = a
	} else {
		out.value = b
	}
	return nil
}
