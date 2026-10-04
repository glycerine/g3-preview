// Licensed under the MIT license.
// Source: src/util/mpq.h bitwise forwards, preserving numerator action before
// output-denominator reset. Arithmetic remains the concrete integer algorithm.
package mpnum

func (m *RationalManager) BitwiseOr(a, b, out *Rational) error {
	return m.rationalBitwise(a, b, out, 0)
}
func (m *RationalManager) BitwiseAnd(a, b, out *Rational) error {
	return m.rationalBitwise(a, b, out, 1)
}
func (m *RationalManager) BitwiseXor(a, b, out *Rational) error {
	return m.rationalBitwise(a, b, out, 2)
}
func (m *RationalManager) rationalBitwise(a, b, out *Rational, op uint8) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if modularSourceDebug && (!isIntegerRational(a) || !isIntegerRational(b)) {
		return ErrRationalNotInteger
	}
	if e := m.check(out); e != nil {
		return e
	}
	var e error
	switch op {
	case 0:
		e = m.z.BitwiseOr(&a.num, &b.num, &out.num)
	case 1:
		e = m.z.BitwiseAnd(&a.num, &b.num, &out.num)
	default:
		e = m.z.BitwiseXor(&a.num, &b.num, &out.num)
	}
	if e != nil {
		return e
	}
	return m.resetDenominator(out)
}
func (m *RationalManager) BitwiseNot(width uint32, a, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if modularSourceDebug && !isIntegerRational(a) {
		return ErrRationalNotInteger
	}
	if e := m.z.BitwiseNot(width, &a.num, &out.num); e != nil {
		return e
	}
	return m.resetDenominator(out)
}
