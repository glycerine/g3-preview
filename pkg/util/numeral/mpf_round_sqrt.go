// Source: src/util/mpf.cpp:2100-2131; src/util/mpf.h:229.
package numeral

import (
	"errors"
)

// RoundSqrt follows the source's specialized guard-bit switch. The caller owns
// the significand and exponent; this helper declares no lexical NUM or RAW.
func (m *ArbitraryFloatManager) RoundSqrt(mode ArbitraryFloatRoundingMode, x *ArbitraryFloat) error {
	return m.RoundSqrtWithReceipt(mode, x, nil)
}
func (m *ArbitraryFloatManager) RoundSqrtWithReceipt(mode ArbitraryFloatRoundingMode, x *ArbitraryFloat, r *ArbitraryFloatRoundReceipt) (err error) {
	if e := m.valueGuard(x); e != nil {
		return e
	}
	s := &m.RoundSqrtScope
	if s.active || s.Invocation == ^uint64(0) {
		return ErrNumericPending
	}
	invocation := s.Invocation + 1
	*s = ArbitraryFloatRoundScope{Manager: m, Invocation: invocation, active: true, executing: true, receipt: r}
	m.valueExecuting = true
	defer func() {
		m.valueExecuting = false
		s.executing = false
		s.bodyCause = err
		err = errors.Join(err, m.finishRoundSqrtSource())
		s.recordRoundHistory()
	}()
	z := m.integer
	c := integerAllocationInvocationContext{mpfRoundSqrt: s, mpfRoundSqrtInvocation: invocation}
	even, e := z.IsEven(&x.significand)
	if e != nil {
		return e
	}
	sticky := !even
	if e = z.MachineDiv2kInPlace(&x.significand, 1); e != nil {
		return e
	}
	if !sticky {
		even, e = z.IsEven(&x.significand)
		if e != nil {
			return e
		}
		sticky = !even
	}
	if e = z.MachineDiv2kInPlace(&x.significand, 1); e != nil {
		return e
	}
	even, e = z.IsEven(&x.significand)
	if e != nil {
		return e
	}
	round := !even
	if e = z.MachineDiv2kInPlace(&x.significand, 1); e != nil {
		return e
	}
	// Source computes last even though the specialized switch does not use it.
	if _, e = z.IsEven(&x.significand); e != nil {
		return e
	}
	inc := false
	switch mode {
	case ArbitraryFloatNearestEven, ArbitraryFloatNearestAway:
		inc = round && sticky
	case ArbitraryFloatTowardNegative, ArbitraryFloatTowardZero:
	case ArbitraryFloatTowardPositive:
		inc = round || sticky
	default:
		return ErrIntegerAssertion
	}
	if inc {
		one := NewInt(1)
		if e = z.addSubWithAllocationContext(&x.significand, &one, &x.significand, false, c); e != nil {
			return e
		}
	}
	p, e := m.roundPower(x.SBits()-1, false)
	if e != nil {
		return e
	}
	return z.addSubWithAllocationContext(&x.significand, &p, &x.significand, true, c)
}
