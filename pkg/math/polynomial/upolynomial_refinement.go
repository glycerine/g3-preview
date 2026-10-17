package polynomial

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func (m *UPolynomialManager) isolationBinary(b *unumeral.BinaryRationalManager) error {
	if e := m.check(); e != nil {
		return e
	}
	if b == nil || b.Integers() != m.config.Integer {
		return ErrUPolynomialOwner
	}
	return b.Open()
}
func (m *UPolynomialManager) IsolatingToRefinable(v UPolynomialView, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	sa, e := m.EvalSignBinary(v, a)
	if e != nil {
		return false, e
	}
	sb, e := m.EvalSignBinary(v, b)
	if e != nil {
		return false, e
	}
	if sa != 0 && sb != 0 {
		if polynomialSourceDebug && sa != -sb {
			return false, ErrUPolynomialAssertion
		}
		return true, nil
	}
	if sa == 0 && sb != 0 {
		var next unumeral.BinaryRational
		defer bqm.Del(&next)
		if e = bqm.Add(a, b, &next); e != nil {
			return false, e
		}
		if e = bqm.Div2(&next); e != nil {
			return false, e
		}
		for {
			sn, e := m.EvalSignBinary(v, &next)
			if e != nil {
				return false, e
			}
			if sn != sb {
				if e = bqm.Swap(&next, a); e != nil {
					return false, e
				}
				if polynomialSourceDebug && sn != 0 && sn != -sb {
					return false, ErrUPolynomialAssertion
				}
				return sn != 0, nil
			}
			if polynomialSourceDebug && sn != sb {
				return false, ErrUPolynomialAssertion
			}
			if e = bqm.Swap(b, &next); e != nil {
				return false, e
			}
			if e = bqm.Add(b, a, &next); e != nil {
				return false, e
			}
			if e = bqm.Div2(&next); e != nil {
				return false, e
			}
		}
	}
	if sa != 0 && sb == 0 {
		var next unumeral.BinaryRational
		defer bqm.Del(&next)
		if e = bqm.Add(a, b, &next); e != nil {
			return false, e
		}
		if e = bqm.Div2(&next); e != nil {
			return false, e
		}
		for {
			sn, e := m.EvalSignBinary(v, &next)
			if e != nil {
				return false, e
			}
			if sn != sa {
				if polynomialSourceDebug && sn != 0 && sn != -sa {
					return false, ErrUPolynomialAssertion
				}
				if sn == 0 {
					e = bqm.Swap(&next, a)
				} else {
					e = bqm.Swap(&next, b)
				}
				if e != nil {
					return false, e
				}
				return sn != 0, nil
			}
			if polynomialSourceDebug && sn != sa {
				return false, ErrUPolynomialAssertion
			}
			if e = bqm.Swap(a, &next); e != nil {
				return false, e
			}
			if e = bqm.Add(b, a, &next); e != nil {
				return false, e
			}
			if e = bqm.Div2(&next); e != nil {
				return false, e
			}
		}
	}
	if polynomialSourceDebug && (sa != 0 || sb != 0) {
		return false, ErrUPolynomialAssertion
	}
	var b1, a2, newA, newB unumeral.BinaryRational
	defer bqm.Del(&b1)
	defer bqm.Del(&a2)
	defer bqm.Del(&newA)
	defer bqm.Del(&newB)
	if e = bqm.Add(a, b, &b1); e != nil {
		return false, e
	}
	if e = bqm.Div2(&b1); e != nil {
		return false, e
	}
	if e = bqm.Copy(&a2, &b1); e != nil {
		return false, e
	}
	s1, e := m.EvalSignBinary(v, &b1)
	if e != nil {
		return false, e
	}
	s2 := s1
	if s1 == 0 {
		if e = bqm.Swap(&b1, a); e != nil {
			return false, e
		}
		return false, nil
	}
	if e = bqm.Add(a, &b1, &newA); e != nil {
		return false, e
	}
	if e = bqm.Div2(&newA); e != nil {
		return false, e
	}
	if e = bqm.Add(&a2, b, &newB); e != nil {
		return false, e
	}
	if e = bqm.Div2(&newB); e != nil {
		return false, e
	}
	for {
		snA, e := m.EvalSignBinary(v, &newA)
		if e != nil {
			return false, e
		}
		if snA == 0 {
			if e = bqm.Swap(&newA, a); e != nil {
				return false, e
			}
			return false, nil
		}
		if snA == -s1 {
			if e = bqm.Swap(&newA, a); e != nil {
				return false, e
			}
			if e = bqm.Swap(&b1, b); e != nil {
				return false, e
			}
			return true, nil
		}
		snB, e := m.EvalSignBinary(v, &newB)
		if e != nil {
			return false, e
		}
		if snB == 0 {
			if e = bqm.Swap(&newB, a); e != nil {
				return false, e
			}
			return false, nil
		}
		if snB == -s2 {
			if e = bqm.Swap(&a2, a); e != nil {
				return false, e
			}
			if e = bqm.Swap(&newB, b); e != nil {
				return false, e
			}
			return true, nil
		}
		if polynomialSourceDebug && snA != s1 {
			return false, ErrUPolynomialAssertion
		}
		if e = bqm.Swap(&b1, &newA); e != nil {
			return false, e
		}
		if e = bqm.Add(&b1, a, &newA); e != nil {
			return false, e
		}
		if e = bqm.Div2(&newA); e != nil {
			return false, e
		}
		if polynomialSourceDebug && snB != s2 {
			return false, ErrUPolynomialAssertion
		}
		if e = bqm.Swap(&a2, &newB); e != nil {
			return false, e
		}
		if e = bqm.Add(b, &a2, &newB); e != nil {
			return false, e
		}
		if e = bqm.Div2(&newB); e != nil {
			return false, e
		}
	}
}
func (m *UPolynomialManager) RefineCore(v UPolynomialView, sa int, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	if e := m.refinementSourceSignAssert(v, a, sa, 2758, "sign_a == eval_sign_at(sz, p, a)"); e != nil {
		return false, e
	}
	if sa < math.MinInt32 || sa > math.MaxInt32 || sa == math.MinInt32 {
		return false, arena.ErrBounds
	}
	sb := -sa
	if e := m.refinementSourceSignAssert(v, b, sb, 2761, "sign_b == eval_sign_at(sz, p, b)"); e != nil {
		return false, e
	}
	if e := refinementSourceAssert(sa == -sb, 2762, "sign_a == -sign_b"); e != nil {
		return false, e
	}
	if e := refinementSourceAssert(sa != 0 && sb != 0, 2763, "sign_a != 0 && sign_b != 0"); e != nil {
		return false, e
	}
	var mid unumeral.BinaryRational
	defer bqm.Del(&mid)
	if e := bqm.Add(a, b, &mid); e != nil {
		return false, e
	}
	if e := bqm.Div2(&mid); e != nil {
		return false, e
	}
	sm, e := m.EvalSignBinary(v, &mid)
	if e != nil {
		return false, e
	}
	if sm == 0 {
		if e = bqm.Swap(&mid, a); e != nil {
			return false, e
		}
		return false, nil
	}
	if sm == sa {
		if e = bqm.Swap(&mid, a); e != nil {
			return false, e
		}
		return true, nil
	}
	if e := refinementSourceAssert(sm == sb, 2776, "sign_mid == sign_b"); e != nil {
		return false, e
	}
	if e = bqm.Swap(&mid, b); e != nil {
		return false, e
	}
	return true, nil
}
func (m *UPolynomialManager) Refine(v UPolynomialView, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	sa, e := m.EvalSignBinary(v, a)
	if e != nil {
		return false, e
	}
	if e := refinementSourceAssert(sa != 0, 2784, "sign_a != 0"); e != nil {
		return false, e
	}
	return m.RefineCore(v, sa, bqm, a, b)
}
func (m *UPolynomialManager) RefineCoreToPrecision(v UPolynomialView, sa int, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational, k uint32) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	if e := refinementSourceAssert(sa != 0, 2794, "sign_a != 0"); e != nil {
		return false, e
	}
	if e := m.refinementSourceSignAssert(v, a, sa, 2795, "sign_a  == eval_sign_at(sz, p, a)"); e != nil {
		return false, e
	}
	if e := m.refinementSourceSignAssert(v, b, -sa, 2796, "-sign_a == eval_sign_at(sz, p, b)"); e != nil {
		return false, e
	}
	var width unumeral.BinaryRational
	defer bqm.Del(&width)
	for {
		if e := m.checkpoint(); e != nil {
			return false, e
		}
		if e := bqm.Sub(b, a, &width); e != nil {
			return false, e
		}
		less, e := bqm.LessOneDiv2k(&width, k)
		if e != nil {
			return false, e
		}
		if less {
			return true, nil
		}
		interval, e := m.RefineCore(v, sa, bqm, a, b)
		if e != nil {
			return false, e
		}
		if !interval {
			return false, nil
		}
	}
}
func (m *UPolynomialManager) RefineToPrecision(v UPolynomialView, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational, k uint32) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	sa, e := m.EvalSignBinary(v, a)
	if e != nil {
		return false, e
	}
	if e := m.refinementSourceSignAssert(v, b, -sa, 2812, "eval_sign_at(sz, p, b) == -sign_a"); e != nil {
		return false, e
	}
	if e := refinementSourceAssert(sa != 0, 2813, "sign_a != 0"); e != nil {
		return false, e
	}
	return m.RefineCoreToPrecision(v, sa, bqm, a, b, k)
}
func (m *UPolynomialManager) ConvertRationalToBinaryInterval(v UPolynomialView, qm *unumeral.RationalManager, a, b *unumeral.Rational, bqm *unumeral.BinaryRationalManager, c, d *unumeral.BinaryRational) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	if qm == nil || qm.Integers() != m.config.Integer {
		return false, ErrUPolynomialOwner
	}
	sa, e := m.EvalSignRational(v, a)
	if e != nil {
		return false, e
	}
	sb, e := m.EvalSignRational(v, b)
	if e != nil {
		return false, e
	}
	if polynomialSourceDebug && (sa == 0 || sb == 0 || sa != -sb) {
		return false, ErrUPolynomialAssertion
	}
	foundD := false
	var lower, upper unumeral.BinaryRational
	defer bqm.Del(&lower)
	defer bqm.Del(&upper)
	exact, e := bqm.FromRational(qm, a, &lower)
	if e != nil {
		return false, e
	}
	if exact {
		if e = bqm.Swap(c, &lower); e != nil {
			return false, e
		}
		if polynomialSourceDebug {
			eq, e := bqm.EqualRational(c, qm, a)
			if e != nil {
				return false, e
			}
			if !eq {
				return false, ErrUPolynomialAssertion
			}
		}
	} else {
		if e = bqm.Copy(&upper, &lower); e != nil {
			return false, e
		}
		if e = bqm.Mul2(&upper); e != nil {
			return false, e
		}
		neg, e := m.config.Integer.IsNeg(&(*a.BorrowNumerator()))
		if e != nil {
			return false, e
		}
		if neg {
			lower, upper = upper, lower
		}
		if polynomialSourceDebug {
			lt, e := bqm.LessRational(&lower, qm, a)
			if e != nil {
				return false, e
			}
			if !lt {
				return false, ErrUPolynomialAssertion
			}
			gt, e := bqm.GreaterRational(&upper, qm, a)
			if e != nil {
				return false, e
			}
			if !gt {
				return false, ErrUPolynomialAssertion
			}
		}
		for {
			ge, e := bqm.GreaterEqualRational(&upper, qm, b)
			if e != nil {
				return false, e
			}
			if !ge {
				break
			}
			if e = bqm.RefineUpper(qm, a, &lower, &upper); e != nil {
				return false, e
			}
		}
		if polynomialSourceDebug {
			lt, e := bqm.LessRational(&upper, qm, b)
			if e != nil {
				return false, e
			}
			if !lt {
				return false, ErrUPolynomialAssertion
			}
		}
		for {
			sign, e := m.EvalSignBinary(v, &upper)
			if e != nil {
				return false, e
			}
			if sign == 0 {
				if e = bqm.Swap(c, &upper); e != nil {
					return false, e
				}
				if e = bqm.Del(&lower); e != nil {
					return false, e
				}
				if e = bqm.Del(&upper); e != nil {
					return false, e
				}
				return false, nil
			}
			if sign == sa {
				if e = bqm.Swap(c, &upper); e != nil {
					return false, e
				}
				break
			}
			if polynomialSourceDebug && sign != sb {
				return false, ErrUPolynomialAssertion
			}
			if !foundD {
				foundD = true
				if e = bqm.Copy(d, &upper); e != nil {
					return false, e
				}
			}
			if e = bqm.RefineUpper(qm, a, &lower, &upper); e != nil {
				return false, e
			}
		}
	}
	if !foundD {
		exact, e = bqm.FromRational(qm, b, &lower)
		if e != nil {
			return false, e
		}
		if exact {
			if e = bqm.Swap(d, &lower); e != nil {
				return false, e
			}
			if polynomialSourceDebug {
				eq, e := bqm.EqualRational(d, qm, b)
				if e != nil {
					return false, e
				}
				if !eq {
					return false, ErrUPolynomialAssertion
				}
			}
		} else {
			if e = bqm.Copy(&upper, &lower); e != nil {
				return false, e
			}
			if e = bqm.Mul2(&upper); e != nil {
				return false, e
			}
			neg, e := m.config.Integer.IsNeg(&(*b.BorrowNumerator()))
			if e != nil {
				return false, e
			}
			if neg {
				lower, upper = upper, lower
			}
			if polynomialSourceDebug {
				gt, e := bqm.GreaterRational(&upper, qm, b)
				if e != nil {
					return false, e
				}
				if !gt {
					return false, ErrUPolynomialAssertion
				}
				lt, e := bqm.LessRational(&lower, qm, b)
				if e != nil {
					return false, e
				}
				if !lt {
					return false, ErrUPolynomialAssertion
				}
			}
			for {
				le, e := bqm.LessEqual(&lower, c)
				if e != nil {
					return false, e
				}
				if !le {
					break
				}
				if e = bqm.RefineLower(qm, b, &lower, &upper); e != nil {
					return false, e
				}
			}
			if polynomialSourceDebug {
				lt, e := bqm.Less(c, &lower)
				if e != nil {
					return false, e
				}
				if !lt {
					return false, ErrUPolynomialAssertion
				}
				lt, e = bqm.Less(&lower, &upper)
				if e != nil {
					return false, e
				}
				if !lt {
					return false, ErrUPolynomialAssertion
				}
				lt, e = bqm.LessRational(&lower, qm, b)
				if e != nil {
					return false, e
				}
				if !lt {
					return false, ErrUPolynomialAssertion
				}
			}
			for {
				sign, e := m.EvalSignBinary(v, &lower)
				if e != nil {
					return false, e
				}
				if sign == 0 {
					if e = bqm.Swap(c, &lower); e != nil {
						return false, e
					}
					if e = bqm.Del(&lower); e != nil {
						return false, e
					}
					if e = bqm.Del(&upper); e != nil {
						return false, e
					}
					return false, nil
				}
				if sign == sb {
					if e = bqm.Swap(d, &lower); e != nil {
						return false, e
					}
					break
				}
				if e = bqm.RefineLower(qm, b, &lower, &upper); e != nil {
					return false, e
				}
			}
		}
	}
	if polynomialSourceDebug {
		lt, e := bqm.Less(c, d)
		if e != nil {
			return false, e
		}
		if !lt {
			return false, ErrUPolynomialAssertion
		}
		sc, e := m.EvalSignBinary(v, c)
		if e != nil {
			return false, e
		}
		sd, e := m.EvalSignBinary(v, d)
		if e != nil {
			return false, e
		}
		if sc != -sd {
			return false, ErrUPolynomialAssertion
		}
		ge, e := bqm.GreaterEqualRational(c, qm, a)
		if e != nil {
			return false, e
		}
		if !ge {
			return false, ErrUPolynomialAssertion
		}
		le, e := bqm.LessEqualRational(d, qm, b)
		if e != nil {
			return false, e
		}
		if !le {
			return false, ErrUPolynomialAssertion
		}
	}
	return true, nil
}
func (m *UPolynomialManager) NormalizeIntervalCore(v UPolynomialView, sa int, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational) (bool, error) {
	if e := m.isolationBinary(bqm); e != nil {
		return false, e
	}
	neg, e := bqm.IsNeg(a)
	if e != nil {
		return false, e
	}
	if !neg {
		return true, nil
	}
	pos, e := bqm.IsPos(b)
	if e != nil {
		return false, e
	}
	if !pos {
		return true, nil
	}
	if sa == math.MinInt32 {
		sa, e = m.EvalSignBinary(v, a)
		if e != nil {
			return false, e
		}
	} else if polynomialSourceDebug {
		s, e := m.EvalSignBinary(v, a)
		if e != nil {
			return false, e
		}
		if s != sa {
			return false, ErrUPolynomialAssertion
		}
	}
	if polynomialSourceDebug {
		sb, e := m.EvalSignBinary(v, b)
		if e != nil {
			return false, e
		}
		if -sa != sb || sa == 0 || sb == 0 {
			return false, ErrUPolynomialAssertion
		}
	}
	zero, e := m.HasZeroRoots(v)
	if e != nil {
		return false, e
	}
	if zero {
		return false, nil
	}
	sz, e := m.EvalSignZero(v)
	if e != nil {
		return false, e
	}
	if sa == sz {
		e = bqm.Reset(a)
	} else {
		e = bqm.Reset(b)
	}
	if e != nil {
		return false, e
	}
	if polynomialSourceDebug {
		sa, e = m.EvalSignBinary(v, a)
		if e != nil {
			return false, e
		}
		sb, e := m.EvalSignBinary(v, b)
		if e != nil {
			return false, e
		}
		if sa != -sb {
			return false, ErrUPolynomialAssertion
		}
	}
	return true, nil
}
func (m *UPolynomialManager) NormalizeInterval(v UPolynomialView, bqm *unumeral.BinaryRationalManager, a, b *unumeral.BinaryRational) (bool, error) {
	return m.NormalizeIntervalCore(v, math.MinInt32, bqm, a, b)
}
