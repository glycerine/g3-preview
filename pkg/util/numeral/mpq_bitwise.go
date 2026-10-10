// Source: src/util/mpq.h bitwise forwards, preserving numerator action before
// output-denominator reset. Arithmetic remains the concrete integer algorithm.
package numeral

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
	return m.rationalBitwiseWithAllocationContext(a, b, out, op, integerAllocationInvocationContext{})
}
func (m *RationalManager) rationalBitwiseWithAllocationContext(a, b, out *Rational, op uint8, context integerAllocationInvocationContext) error {
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
	e := m.Z.bitwiseBinaryWithAllocationContext(&a.Num, &b.Num, &out.Num, op, context)
	if e != nil {
		return e
	}
	return m.resetDenominator(out)
}
func (m *RationalManager) BitwiseNot(width uint32, a, out *Rational) error {
	return m.bitwiseNotWithAllocationContext(width, a, out, integerAllocationInvocationContext{})
}
func (m *RationalManager) bitwiseNotWithAllocationContext(width uint32, a, out *Rational, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if modularSourceDebug && !isIntegerRational(a) {
		return ErrRationalNotInteger
	}
	if e := m.Z.bitwiseNotWithAllocationContext(width, &a.Num, &out.Num, context); e != nil {
		return e
	}
	return m.resetDenominator(out)
}
