// Source: src/util/mpf.cpp:748-949; src/util/mpf.h:129,308-324.
package numeral

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
)

func (s *arbitraryFloatFMAScope) assertPower(value *Int, n uint32, ge bool) error {
	if !modularSourceDebug {
		return nil
	}
	p, e := s.power(n)
	if e != nil {
		return e
	}
	cmp, e := s.manager.integer.Compare(value, &p)
	if e != nil {
		return e
	}
	if ge && cmp < 0 || !ge && cmp >= 0 {
		return ErrIntegerAssertion
	}
	return nil
}
func (s *arbitraryFloatFMAScope) shiftInPlace(value *Int, n uint32) error {
	z := s.manager.integer
	if e := z.check(value); e != nil {
		return e
	}
	return z.mul2kCoreWithAllocationContext(value, n, s.context())
}

func (m *ArbitraryFloatManager) FMA(mode ArbitraryFloatRoundingMode, x, y, z, out *ArbitraryFloat) (cause error) {
	for _, v := range [...]*ArbitraryFloat{x, y, z, out} {
		if e := m.valueGuard(v); e != nil {
			return e
		}
	}
	if m.fma.active {
		return ErrNumericPending
	}
	// The source assertion repeats x/y and compares z.ebits with itself.
	if modularSourceDebug && (x.SBits() != y.SBits() || x.EBits() != y.EBits()) {
		return ErrIntegerAssertion
	}
	m.traceFloatDebugValue("fma", "X = ", x, 752)
	m.traceFloatDebugValue("fma", "Y = ", y, 753)
	m.traceFloatDebugValue("fma", "Z = ", z, 754)
	for _, v := range [...]*ArbitraryFloat{x, y, z} {
		nan, e := m.IsNaN(v)
		if e != nil {
			return e
		}
		if nan {
			return m.MkNaN(x.EBits(), x.SBits(), out, nil)
		}
	}
	for _, branch := range [...]struct {
		v, other *ArbitraryFloat
		negative bool
	}{{x, y, false}, {y, x, false}, {x, y, true}, {y, x, true}} {
		var inf bool
		var e error
		if branch.negative {
			inf, e = m.IsNegativeInf(branch.v)
		} else {
			inf, e = m.IsPositiveInf(branch.v)
		}
		if e != nil {
			return e
		}
		if inf {
			zero, e := m.IsZero(branch.other)
			if e != nil {
				return e
			}
			if zero {
				return m.MkNaN(x.EBits(), x.SBits(), out, nil)
			}
			zi, e := m.IsInf(z)
			if e != nil {
				return e
			}
			if zi && (x.Sign() != y.Sign()) != z.Sign() {
				return m.MkNaN(x.EBits(), x.SBits(), out, nil)
			}
			return m.MkInf(x.EBits(), x.SBits(), branch.other.Sign() != branch.negative, out)
		}
	}
	inf, e := m.IsInf(z)
	if e != nil {
		return e
	}
	if inf {
		return m.Copy(out, z)
	}
	zero, e := m.IsZero(x)
	if e != nil {
		return e
	}
	if !zero {
		zero, e = m.IsZero(y)
		if e != nil {
			return e
		}
	}
	if zero {
		sign := x.Sign() != y.Sign()
		zz, e := m.IsZero(z)
		if e != nil {
			return e
		}
		if zz && sign != z.Sign() {
			return m.MkZero(x.EBits(), x.SBits(), mode == ArbitraryFloatTowardNegative, out)
		}
		return m.Copy(out, z)
	}
	out.assignEbits(x.EBits())
	out.assignSbits(x.SBits())
	sb, eb := x.SBits(), x.EBits()
	s := &m.fma
	if s.invocation == ^uint64(0) {
		return ErrNumericPending
	}
	*s = arbitraryFloatFMAScope{manager: m, invocation: s.invocation + 1, active: true, executing: true, mrLive: true, aLive: true, bLive: true, cLive: true}
	for _, v := range [...]*ArbitraryFloat{&s.a, &s.b, &s.c} {
		v.format = arbitraryFloatFormat(eb, sb, false)
	}
	s.hooks.Policy = s
	if e := s.bind(); e != nil {
		s.executing = false
		return e
	}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = JoinNumericErrors(cause, m.finishFMASource())
		cause = JoinNumericErrors(cause, s.unbind())
	}()
	context := s.context()
	im := m.integer
	for _, copy := range [...]struct {
		dst, src *ArbitraryFloat
		receipt  *ArbitraryFloatValueCopyReceipt
	}{{&s.a, x, &s.copyA}, {&s.b, y, &s.copyB}, {&s.c, z, &s.copyC}} {
		if e := m.copyWithValueReceiptAndContext(copy.dst, copy.src, copy.receipt, context); e != nil {
			return e
		}
	}
	for _, v := range [...]*ArbitraryFloat{&s.a, &s.b, &s.c} {
		if e := m.unpackRationalNumeral(v, true, context); e != nil {
			return e
		}
	}
	for _, v := range [...]*Int{&s.a.significand, &s.b.significand, &s.c.significand} {
		if e := s.assertPower(v, sb, false); e != nil {
			return e
		}
	}
	for _, v := range [...]*Int{&s.a.significand, &s.b.significand} {
		if e := s.assertPower(v, sb-1, true); e != nil {
			return e
		}
	}
	ae, be := s.a.exponent, s.b.exponent
	if be > 0 && ae > math.MaxInt64-be || be < 0 && ae < math.MinInt64-be {
		return arena.ErrBounds
	}
	s.mr.format = arbitraryFloatFormat(eb+2, 2*sb-1, s.a.Sign() != s.b.Sign())
	s.mr.exponent = ae + be
	if e := im.SetInt32(&s.mr.significand, 0); e != nil {
		return e
	}
	if e := im.mulWithAllocationContext(&s.a.significand, &s.b.significand, &s.mr.significand, context); e != nil {
		return e
	}
	if e := s.assertPower(&s.mr.significand, 2*sb, false); e != nil {
		return e
	}
	if e := s.assertPower(&s.mr.significand, 2*sb-2, true); e != nil {
		return e
	}
	// Self-significand scoped setters erase numerical SET, preserving ownership.
	s.c.format = arbitraryFloatFormat(eb+2, 2*sb+2, s.c.Sign())
	if e := s.shiftInPlace(&s.c.significand, sb+2); e != nil {
		return e
	}
	s.mr.format = arbitraryFloatFormat(eb+2, 2*sb+2, s.mr.Sign())
	if e := s.shiftInPlace(&s.mr.significand, 3); e != nil {
		return e
	}
	if e := s.assertPower(&s.c.significand, 2*sb+3, false); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := im.IsZero(&s.c.significand)
		if e != nil {
			return e
		}
		if !zero {
			if e := s.assertPower(&s.c.significand, 2*sb+1, true); e != nil {
				return e
			}
		}
	}
	if s.c.exponent > s.mr.exponent {
		s.mr, s.c = s.c, s.mr
	}
	delta, e := roundSignedSub(s.mr.exponent, s.c.exponent)
	if e != nil {
		return e
	}
	if modularSourceDebug && delta < 0 {
		return ErrIntegerAssertion
	}
	if delta > int64(2*sb+3) {
		delta = int64(2*sb + 3)
	}
	s.stickyLive = true
	power, e := s.power(uint32(delta))
	if e != nil {
		return e
	}
	if e := im.machineDivRemWithAllocationContext(&s.c.significand, &power, &s.c.significand, &s.sticky, context); e != nil {
		return e
	}
	empty, e := im.IsZero(&s.sticky)
	if e != nil {
		return e
	}
	alignmentSticky := !empty
	// Construction starts before the allocating copy; its source destructor is
	// reached only after construction succeeds. Owning cleanup joins either case.
	s.resLive = true
	if e := m.copyWithValueReceiptAndContext(&s.res, &s.mr, &s.copyRes, context); e != nil {
		return e
	}
	s.resSourceLive = true
	subtract := s.mr.Sign() != s.c.Sign()
	if e := im.addSubWithAllocationContext(&s.mr.significand, &s.c.significand, &s.res.significand, subtract, context); e != nil {
		return e
	}
	if alignmentSticky {
		even, e := im.IsEven(&s.res.significand)
		if e != nil {
			return e
		}
		if even {
			one := NewInt(1)
			if e := im.addSubWithAllocationContext(&s.res.significand, &one, &s.res.significand, subtract, context); e != nil {
				return e
			}
		}
	}
	if subtract {
		neg, e := im.IsNeg(&s.res.significand)
		if e != nil {
			return e
		}
		if neg {
			if e := im.Abs(&s.res.significand); e != nil {
				return e
			}
			s.res.assignSign(!s.res.Sign())
		}
	}
	renormSticky := false
	if e := s.assertPower(&s.res.significand, 2*sb+4, false); e != nil {
		return e
	}
	power, e = s.power(2*sb + 3)
	if e != nil {
		return e
	}
	cmp, e := im.Compare(&s.res.significand, &power)
	if e != nil {
		return e
	}
	if cmp >= 0 {
		if modularSourceDebug {
			max, e := m.maxExponent(eb)
			if e != nil {
				return e
			}
			if s.res.exponent >= max {
				return ErrIntegerAssertion
			}
		}
		if s.res.exponent == math.MaxInt64 {
			return arena.ErrBounds
		}
		s.res.exponent++
		even, e := im.IsEven(&s.res.significand)
		if e != nil {
			return e
		}
		renormSticky = !even
		if e := im.MachineDiv2kInPlace(&s.res.significand, 1); e != nil {
			return e
		}
	}
	min, e := m.minExponent(eb)
	if e != nil {
		return e
	}
	width, e := im.PrevPowerOfTwo(&s.res.significand)
	if e != nil {
		return e
	}
	width++
	lz := int64(2*sb + 3 - width)
	maxDelta, e := roundSignedSub(s.res.exponent, min)
	if e != nil {
		return e
	}
	if lz > maxDelta {
		lz = maxDelta
	}
	if lz < 0 {
		lz = 0
	}
	renormDelta := uint32(lz)
	exponent, e := roundSignedSub(s.res.exponent, int64(renormDelta))
	if e != nil {
		return e
	}
	s.res.exponent = exponent
	if e := s.shiftInPlace(&s.res.significand, renormDelta); e != nil {
		return e
	}
	zeroNumeral := NewInt(0)
	if e := m.SetComponents(out, eb, sb, s.res.Sign(), s.res.exponent, &zeroNumeral); e != nil {
		return e
	}
	if sb >= 4 {
		power, e := s.power(sb - 1)
		if e != nil {
			return e
		}
		if e := im.machineDivRemWithAllocationContext(&s.res.significand, &power, &out.significand, &s.sticky, context); e != nil {
			return e
		}
		empty, e := im.IsZero(&s.sticky)
		if e != nil {
			return e
		}
		renormSticky = renormSticky || !empty
	} else {
		if e := im.shiftWithAllocationContext(&s.res.significand, 7-sb, &out.significand, true, context); e != nil {
			return e
		}
	}
	if renormSticky {
		even, e := im.IsEven(&out.significand)
		if e != nil {
			return e
		}
		if even {
			one := NewInt(1)
			if e := im.addSubWithAllocationContext(&out.significand, &one, &out.significand, false, context); e != nil {
				return e
			}
		}
	}
	empty, e = im.IsZero(&out.significand)
	if e != nil {
		return e
	}
	if empty {
		return m.MkZero(eb, sb, mode == ArbitraryFloatTowardNegative, out)
	}
	return m.roundWithParentContext(mode, out, &s.roundReceipt, context)
}
