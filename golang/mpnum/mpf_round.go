// Source: src/util/mpf.cpp:1948-2098; src/util/mpf.h:31-37.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
)

type ArbitraryFloatRoundingMode uint8

const (
	ArbitraryFloatNearestEven ArbitraryFloatRoundingMode = iota
	ArbitraryFloatNearestAway
	ArbitraryFloatTowardPositive
	ArbitraryFloatTowardNegative
	ArbitraryFloatTowardZero
)

func roundSignedSub(a, b int64) (int64, error) {
	if b > 0 && a < math.MinInt64+b || b < 0 && a > math.MaxInt64+b {
		return 0, arena.ErrBounds
	}
	return a - b, nil
}
func (m *ArbitraryFloatManager) roundPower(n uint32, minusOne bool) (Int, error) {
	saved := m.valueExecuting
	m.valueExecuting = false
	defer func() { m.valueExecuting = saved }()
	var w *arbitraryFloatPowerWrapper
	var e error
	if minusOne {
		w, e = m.powerTwoMinusOneWrapper(n, false)
	} else {
		w, e = m.powerTwoWrapper(n, false)
	}
	if e != nil {
		return Int{}, e
	}
	return w.capture()
}
func (m *ArbitraryFloatManager) roundBound(ebits uint32, min bool) (int64, error) {
	saved := m.valueExecuting
	m.valueExecuting = false
	defer func() { m.valueExecuting = saved }()
	if min {
		return m.minExponent(ebits)
	}
	return m.maxExponent(ebits)
}
func (m *ArbitraryFloatManager) roundOverflow(mode ArbitraryFloatRoundingMode, x *ArbitraryFloat) error {
	sign := x.Sign()
	maximum := mode == ArbitraryFloatTowardZero || !sign && mode == ArbitraryFloatTowardNegative || sign && mode == ArbitraryFloatTowardPositive
	x.assignSbits(x.SBits())
	x.assignEbits(x.EBits())
	x.assignSign(sign)
	saved := m.valueExecuting
	m.valueExecuting = false
	top, e := m.topExponent(x.EBits())
	m.valueExecuting = saved
	if e != nil {
		return e
	}
	if !maximum {
		x.exponent = top
		return m.integer.SetInt32(&x.significand, 0)
	}
	x.exponent = top - 1
	p, e := m.roundPower(x.SBits()-1, true)
	if e != nil {
		return e
	}
	return m.integer.copyFromOwnerWithAllocationContext(&x.significand, m.integer, &p, nil, integerAllocationInvocationContext{mpfRound: &m.roundScope, mpfRoundInvocation: m.roundScope.invocation})
}

func (m *ArbitraryFloatManager) Round(mode ArbitraryFloatRoundingMode, x *ArbitraryFloat) (err error) {
	return m.RoundWithReceipt(mode, x, nil)
}
func (m *ArbitraryFloatManager) RoundWithReceipt(mode ArbitraryFloatRoundingMode, x *ArbitraryFloat, receipt *ArbitraryFloatRoundReceipt) (err error) {
	if e := m.valueGuard(x); e != nil {
		return e
	}
	s := &m.roundScope
	if s.active {
		return ErrPolynomialPending
	}
	if s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	invocation := s.invocation + 1
	*s = arbitraryFloatRoundScope{manager: m, invocation: invocation, active: true, executing: true, receipt: receipt}
	m.valueExecuting = true
	defer func() {
		m.valueExecuting = false
		s.executing = false
		s.bodyCause = err
		err = errors.Join(err, m.finishRoundSource())
		s.recordRoundHistory()
	}()
	z := m.integer
	context := integerAllocationInvocationContext{mpfRound: s, mpfRoundInvocation: s.invocation}
	one := NewInt(1)
	sb := x.SBits()
	eb := x.EBits()
	if modularSourceDebug {
		p, e := m.roundPower(sb+4, false)
		if e != nil {
			return e
		}
		c, e := z.Compare(&x.significand, &p)
		if e != nil {
			return e
		}
		if c >= 0 {
			return ErrIntegerAssertion
		}
	}
	emax, e := m.roundBound(eb, false)
	if e != nil {
		return e
	}
	emin, e := m.roundBound(eb, true)
	if e != nil {
		return e
	}
	width, e := z.PrevPowerOfTwo(&x.significand)
	if e != nil {
		return e
	}
	width++
	lz := int64(sb + 4 - width) // source unsigned32 subtraction before signed assignment
	beta, e := roundSignedSub(x.exponent, lz)
	if e != nil {
		return e
	}
	if beta == math.MaxInt64 {
		return arena.ErrBounds
	}
	beta++
	s.sigmaConstructed = true
	if beta < emin {
		delta, e := roundSignedSub(x.exponent, emin)
		if e != nil {
			return e
		}
		if e = z.setInt64WithAllocationContext(&s.sigma, delta, context); e != nil {
			return e
		}
		x.exponent = emin
	} else {
		if e = z.setInt64WithAllocationContext(&s.sigma, lz-1, context); e != nil {
			return e
		}
		x.exponent = beta
	}
	s.capConstructed = true
	if e = z.setUint32WithAllocationContext(&s.cap, sb+2, context); e != nil {
		return e
	}
	if e = z.Neg(&s.cap); e != nil {
		return e
	}
	c, e := z.Compare(&s.sigma, &s.cap)
	if e != nil {
		return e
	}
	if c < 0 {
		if e = z.copyFromOwnerWithAllocationContext(&s.sigma, z, &s.cap, nil, context); e != nil {
			return e
		}
	}
	minusOne := NewInt(-1)
	c, e = z.Compare(&s.sigma, &minusOne)
	if e != nil {
		return e
	}
	sigma, e := z.Int64(&s.sigma)
	if e != nil {
		return e
	}
	if c <= 0 {
		s.stickyConstructed = true
		p, e := m.roundPower(uint32(-sigma), false)
		if e != nil {
			return e
		}
		if e = z.machineDivRemWithAllocationContext(&x.significand, &p, &x.significand, &s.sticky, context); e != nil {
			return e
		}
		zero, e := z.IsZero(&s.sticky)
		if e != nil {
			return e
		}
		if !zero {
			even, e := z.IsEven(&x.significand)
			if e != nil {
				return e
			}
			if even {
				if e = z.addSubWithAllocationContext(&x.significand, &one, &x.significand, false, context); e != nil {
					return e
				}
			}
		}
		if _, e = upolynomialSourceNumeralDel(z, &s.sticky); e != nil {
			s.sourceCause = e
			return e
		}
		s.stickyConstructed = false
	} else {
		if e = z.mul2kCoreWithAllocationContext(&x.significand, uint32(sigma), context); e != nil {
			return e
		}
	}
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
	even, e = z.IsEven(&x.significand)
	if e != nil {
		return e
	}
	last := !even
	inc := false
	switch mode {
	case ArbitraryFloatNearestEven:
		inc = round && (last || sticky)
	case ArbitraryFloatNearestAway:
		inc = round
	case ArbitraryFloatTowardPositive:
		inc = !x.Sign() && (round || sticky)
	case ArbitraryFloatTowardNegative:
		inc = x.Sign() && (round || sticky)
	case ArbitraryFloatTowardZero:
	default:
		return ErrIntegerAssertion
	}
	if inc {
		if e = z.addSubWithAllocationContext(&x.significand, &one, &x.significand, false, context); e != nil {
			return e
		}
	}
	p, e := m.roundPower(sb, false)
	if e != nil {
		return e
	}
	c, e = z.Compare(&x.significand, &p)
	if e != nil {
		return e
	}
	if c >= 0 {
		if e = z.MachineDiv2kInPlace(&x.significand, 1); e != nil {
			return e
		}
		if x.exponent == math.MaxInt64 {
			return arena.ErrBounds
		}
		x.exponent++
	}
	if x.exponent > emax {
		return m.roundOverflow(mode, x)
	}
	p, e = m.roundPower(sb-1, false)
	if e != nil {
		return e
	}
	c, e = z.Compare(&x.significand, &p)
	if e != nil {
		return e
	}
	if c >= 0 {
		return z.addSubWithAllocationContext(&x.significand, &p, &x.significand, true, context)
	}
	saved := m.valueExecuting
	m.valueExecuting = false
	bottom, e := m.bottomExponent(eb)
	m.valueExecuting = saved
	if e != nil {
		return e
	}
	x.exponent = bottom
	return nil
}
