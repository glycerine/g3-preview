// Licensed under the MIT license.
// Source: src/util/mpq.cpp floor/ceil/prev_power_of_two and mpq.h wrappers.
package mpnum

// roundCore retains the source sign-only correction, including unreduced or
// negative-denominator raw values. It captures numerator sign before division
// publishes into a caller that may alias either input component.
func (m *RationalManager) roundCore(a *Rational, out *Int, ceil bool) error {
	return m.roundCoreWithAllocationContext(a, out, ceil, integerAllocationInvocationContext{})
}
func (m *RationalManager) roundCoreWithAllocationContext(a *Rational, out *Int, ceil bool, context integerAllocationInvocationContext) error {
	if isIntegerRational(a) {
		return m.z.copyFromOwnerWithAllocationContext(out, m.z, &a.num, nil, context)
	}
	correct := a.num.val < 0
	if ceil {
		correct = a.num.val > 0
	}
	if e := m.z.machineDivWithAllocationContext(&a.num, &a.den, out, context); e != nil {
		return e
	}
	if correct {
		one := NewInt(1)
		if ceil {
			return m.z.addSubWithAllocationContext(out, &one, out, false, context)
		}
		return m.z.addSubWithAllocationContext(out, &one, out, true, context)
	}
	return nil
}
func (m *RationalManager) roundToInt(a *Rational, out *Int, ceil bool) error {
	return m.roundToIntWithAllocationContext(a, out, ceil, integerAllocationInvocationContext{})
}
func (m *RationalManager) roundToIntWithAllocationContext(a *Rational, out *Int, ceil bool, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(out); e != nil {
		return e
	}
	return m.roundCoreWithAllocationContext(a, out, ceil, context)
}

// FloorToInt and CeilToInt publish the source Copy/quotient into the caller
// before sign correction. Output may alias either input component; errors
// retain completed source actions rather than restoring the old caller.
func (m *RationalManager) FloorToInt(a *Rational, out *Int) error { return m.roundToInt(a, out, false) }
func (m *RationalManager) CeilToInt(a *Rational, out *Int) error  { return m.roundToInt(a, out, true) }
func (m *RationalManager) round(a, out *Rational, ceil bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.roundCore(a, &out.num, ceil); e != nil {
		return e
	}
	return m.resetDenominator(out)
}

// Floor and Ceil are the source rational-output wrappers, with denominator 1.
func (m *RationalManager) Floor(a, out *Rational) error { return m.round(a, out, false) }
func (m *RationalManager) Ceil(a, out *Rational) error  { return m.round(a, out, true) }
func (m *RationalManager) PrevPowerOfTwo(a *Rational) (uint32, error) {
	var tmp Int
	defer m.z.Del(&tmp)
	if e := m.FloorToInt(a, &tmp); e != nil {
		return 0, e
	}
	return m.z.PrevPowerOfTwo(&tmp)
}
