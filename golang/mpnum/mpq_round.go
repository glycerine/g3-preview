// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpq.cpp floor/ceil/prev_power_of_two and mpq.h wrappers.
package mpnum

// roundCore retains the source normalized-fraction branch. A noninteger
// normalized fraction has a nonzero remainder, so the source sign correction
// suffices without an extra divisibility computation.
func (m *RationalManager) roundCore(a *Rational, out *Int, ceil bool) error {
	if isIntegerRational(a) {
		return m.z.Copy(out, &a.num)
	}
	correct := a.num.val < 0
	if ceil {
		correct = a.num.val > 0
	}
	if e := m.z.MachineDiv(&a.num, &a.den, out); e != nil {
		return e
	}
	if correct {
		one := NewInt(1)
		if ceil {
			return m.z.Add(out, &one, out)
		}
		return m.z.Sub(out, &one, out)
	}
	return nil
}
func (m *RationalManager) roundToInt(a *Rational, out *Int, ceil bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(out); e != nil {
		return e
	}
	var work Int
	defer m.z.Del(&work)
	if e := m.roundCore(a, &work, ceil); e != nil {
		return e
	}
	return m.z.Move(out, &work)
}

// FloorToInt and CeilToInt publish an owner-local integer after the complete
// source truncation/sign correction succeeds. Output may alias the numerator.
func (m *RationalManager) FloorToInt(a *Rational, out *Int) error { return m.roundToInt(a, out, false) }
func (m *RationalManager) CeilToInt(a *Rational, out *Int) error  { return m.roundToInt(a, out, true) }
func (m *RationalManager) round(a, out *Rational, ceil bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.roundCore(a, &work.num, ceil); e != nil {
		return e
	}
	if e := m.resetDenominator(&work); e != nil {
		return e
	}
	return m.Move(out, &work)
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
