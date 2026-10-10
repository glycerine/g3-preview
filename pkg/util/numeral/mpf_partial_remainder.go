// Source: src/util/mpf.cpp:1272-1436; src/util/mpf.h:232.
package numeral

import (
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math"
)

// PartialRemainder operates on normalized, unpacked inputs, as the source
// helper does. It updates x; y remains borrowed through the final publication.
func (m *ArbitraryFloatManager) PartialRemainder(x, y *ArbitraryFloat, expDiff int64, partial bool) (cause error) {
	if m == nil || m.self != m {
		return umemory.ErrRuntimeOwner
	}
	if m.partialRemainder.active {
		return ErrNumericPending
	}
	for _, v := range [...]*ArbitraryFloat{x, y} {
		if e := m.valueGuard(v); e != nil {
			return e
		}
	}
	s := &m.partialRemainder
	if s.invocation == ^uint64(0) {
		return ErrNumericPending
	}
	*s = arbitraryFloatPartialRemainderScope{manager: m, invocation: s.invocation + 1, active: true, executing: true}
	s.hooks.Policy = s
	if e := s.bind(); e != nil {
		s.executing = false
		return e
	}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = JoinNumericErrors(cause, m.finishPartialRemainderSource())
		cause = JoinNumericErrors(cause, s.unbind())
	}()
	return s.run(x, y, expDiff, partial)
}
func (s *arbitraryFloatPartialRemainderScope) power(n uint32) (Int, error) {
	w, e := s.manager.powerTwoWrapper(n, false)
	if e != nil {
		return Int{}, e
	}
	return w.Capture()
}
func (s *arbitraryFloatPartialRemainderScope) assertPower(v *Int, n uint32, ge bool) error {
	if !modularSourceDebug {
		return nil
	}
	p, e := s.power(n)
	if e != nil {
		return e
	}
	cmp, e := s.manager.integer.Compare(v, &p)
	if e != nil {
		return e
	}
	if ge && cmp < 0 || !ge && cmp >= 0 {
		return ErrIntegerAssertion
	}
	return nil
}
func (s *arbitraryFloatPartialRemainderScope) shift(v *Int, n uint32) error {
	z := s.manager.integer
	if e := z.check(v); e != nil {
		return e
	}
	return z.mul2kCoreWithAllocationContext(v, n, s.context())
}
func (s *arbitraryFloatPartialRemainderScope) copyValue(dst, src *Int) error {
	z := s.manager.integer
	return z.copyFromOwnerWithAllocationContext(dst, z, src, &s.copy, s.context())
}
func (s *arbitraryFloatPartialRemainderScope) inc(v *Int) error {
	one := NewInt(1)
	return s.manager.integer.addSubWithAllocationContext(v, &one, v, false, s.context())
}
func (s *arbitraryFloatPartialRemainderScope) endNumeral(index int) error {
	if !s.live[index] {
		return nil
	}

	if _, e := SourceNumeralDel(s.manager.integer, &s.values[index]); e != nil {
		s.sourceCause = e
		return e
	}
	s.live[index] = false
	return nil
}
func (s *arbitraryFloatPartialRemainderScope) run(x, y *ArbitraryFloat, expDiff int64, partial bool) error {
	if !s.NumericBound(true) {
		return ErrNumericOwner
	}
	m, z, c := s.manager, s.manager.integer, s.context()
	eb, sb := x.EBits(), x.SBits()
	if modularSourceDebug && (expDiff < -1 || expDiff == 1<<63-1 || expDiff >= int64(eb+sb) && !partial) {
		return ErrIntegerAssertion
	}
	d := int64(int32(expDiff))
	n := int64(sb - 1)
	for _, v := range [...]*Int{&x.significand, &y.significand} {
		if e := s.assertPower(v, sb, false); e != nil {
			return e
		}
		if e := s.assertPower(v, sb-1, true); e != nil {
			return e
		}
	}
	// Actual source declarations: shifted numerator, large quotient, its
	// remainder, and the unused x_rem_y_sig are all constructed here.
	for i := 0; i < 4; i++ {
		s.live[i] = true
	}
	shifted, large, divRem := &s.values[0], &s.values[1], &s.values[2]
	if e := z.shiftWithAllocationContext(&x.significand, 2*sb+2, shifted, true, c); e != nil {
		return e
	}
	if e := z.machineDivRemWithAllocationContext(shifted, &y.significand, large, divRem, c); e != nil {
		return e
	}
	s.live[4], s.live[5] = true, true
	q, qr := &s.values[4], &s.values[5]
	qe := d
	chosen := qe
	if partial {
		chosen = n
	}
	qShift := (sb - 1) + (sb + 3) - uint32(chosen)
	if partial {
		if modularSourceDebug && !(0 < n && n < qe && qe < 2147483647) {
			return ErrIntegerAssertion
		}
		if e := z.shiftWithAllocationContext(large, qShift, q, false, c); e != nil {
			return e
		}
	} else {
		p, e := s.power(qShift)
		if e != nil {
			return e
		}
		if e := z.machineDivRemWithAllocationContext(large, &p, q, qr, c); e != nil {
			return e
		}
		half, e := s.power(qShift - 1)
		if e != nil {
			return e
		}
		tie, e := z.Equal(qr, &half)
		if e != nil {
			return e
		}
		more, e := z.Greater(qr, &half)
		if e != nil {
			return e
		}
		increment := more
		if tie {
			odd, e := z.IsOdd(q)
			if e != nil {
				return e
			}
			increment = increment || odd
		}
		if increment {
			if e := s.inc(q); e != nil {
				return e
			}
		}
	}
	if e := s.shift(q, qShift); e != nil {
		return e
	}
	if e := z.MachineDiv2kInPlace(q, sb+3); e != nil {
		return e
	}
	if e := m.renormalizeWithContext(eb, sb, &qe, q, c); e != nil {
		return e
	}
	if d == -1 || partial {
		empty, e := z.IsZero(q)
		if e != nil {
			return e
		}
		if empty {
			return nil
		}
	}
	if modularSourceDebug {
		empty, e := z.IsZero(q)
		if e != nil {
			return e
		}
		if empty {
			return ErrIntegerAssertion
		}
	}
	if e := s.assertPower(q, sb, false); e != nil {
		return e
	}
	if e := s.assertPower(q, sb-1, true); e != nil {
		return e
	}
	s.live[6] = true
	product := &s.values[6]
	if y.exponent > 0 && qe > math.MaxInt64-y.exponent || y.exponent < 0 && qe < math.MinInt64-y.exponent {
		return arena.ErrBounds
	}
	pe := qe + y.exponent
	if e := z.mulWithAllocationContext(&y.significand, q, product, c); e != nil {
		return e
	}
	if e := m.renormalizeWithContext(eb, 2*sb-1, &pe, product, c); e != nil {
		return e
	}
	if e := s.assertPower(product, 2*sb-1, false); e != nil {
		return e
	}
	if modularSourceDebug {
		p, e := s.power(2*sb - 2)
		if e != nil {
			return e
		}
		ge, e := z.GreaterEqual(product, &p)
		if e != nil {
			return e
		}
		if !ge {
			bottom, e := m.bottomExponent(eb)
			if e != nil {
				return e
			}
			if pe > bottom {
				return ErrIntegerAssertion
			}
		}
	}
	re := x.exponent
	s.live[7] = true
	result := &s.values[7]
	delta, e := roundSignedSub(x.exponent, pe)
	if e != nil {
		return e
	}
	if modularSourceDebug && !(delta > -2147483648 && delta <= 2147483647) {
		return ErrIntegerAssertion
	}
	s.live[8], s.live[9], s.live[10] = true, true, true
	minuend, subtrahend, xLarge := &s.values[8], &s.values[9], &s.values[10]
	if e := z.shiftWithAllocationContext(&x.significand, sb-1, xLarge, true, c); e != nil {
		return e
	}
	if e := s.copyValue(minuend, xLarge); e != nil {
		return e
	}
	if e := s.copyValue(subtrahend, product); e != nil {
		return e
	}
	for _, v := range [...]*Int{minuend, subtrahend} {
		if e := s.assertPower(v, 2*sb-1, false); e != nil {
			return e
		}
		if e := s.assertPower(v, 2*sb-2, true); e != nil {
			return e
		}
	}
	if delta != 0 {
		s.live[11] = true
		sticky := &s.values[11]
		if e := z.SetInt32(sticky, 0); e != nil {
			return e
		}
		if delta > int64(sb+5) {
			if e := z.Swap(sticky, subtrahend); e != nil {
				return e
			}
		} else if delta > 0 {
			p, e := s.power(uint32(delta))
			if e != nil {
				return e
			}
			if e := z.machineDivRemWithAllocationContext(subtrahend, &p, subtrahend, sticky, c); e != nil {
				return e
			}
		} else {
			if delta == -1<<63 {
				return arena.ErrBounds
			}
			delta = -delta
			if e := s.shift(subtrahend, uint32(int32(delta))); e != nil {
				return e
			}
		}
		empty, e := z.IsZero(sticky)
		if e != nil {
			return e
		}
		if !empty {
			even, e := z.IsEven(subtrahend)
			if e != nil {
				return e
			}
			if even {
				if e := s.inc(subtrahend); e != nil {
					return e
				}
			}
		}
		if e := s.endNumeral(11); e != nil {
			return e
		}
	}
	if e := z.addSubWithAllocationContext(minuend, subtrahend, result, true, c); e != nil {
		return e
	}
	neg, e := z.IsNeg(result)
	if e != nil {
		return e
	}
	if neg {
		if e := z.Neg(result); e != nil {
			return e
		}
	}
	sign := x.Sign() != neg
	empty, e := z.IsZero(result)
	if e != nil {
		return e
	}
	if empty {
		return m.MkZero(eb, sb, x.Sign(), x)
	}
	if e := m.renormalizeWithContext(eb, 2*sb-1, &re, result, c); e != nil {
		return e
	}
	if e := s.assertPower(result, 2*sb-1, false); e != nil {
		return e
	}
	s.live[12] = true
	rnd := &s.values[12]
	p, e := s.power(sb - 1)
	if e != nil {
		return e
	}
	if e := z.machineDivRemWithAllocationContext(result, &p, result, rnd, c); e != nil {
		return e
	}
	// Preserve the source constant, independently of sbits.
	half := NewInt(32)
	tie, e := z.Equal(rnd, &half)
	if e != nil {
		return e
	}
	if tie {
		odd, e := z.IsOdd(result)
		if e != nil {
			return e
		}
		if odd {
			if e := s.inc(result); e != nil {
				return e
			}
		}
	} else {
		more, e := z.Greater(rnd, &half)
		if e != nil {
			return e
		}
		if more {
			if e := s.inc(result); e != nil {
				return e
			}
		}
	}
	// mpz-significand SET publishes the exponent only after successful COPY.
	x.format = arbitraryFloatFormat(eb, sb, sign)
	if e := s.copyValue(&x.significand, result); e != nil {
		return e
	}
	x.exponent = re
	return s.endNumeral(12)
}
