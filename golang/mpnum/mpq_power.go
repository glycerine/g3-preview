// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpq.cpp power, including its final unused square.
package mpnum

// Power preserves source rational repeated squaring, including zero^zero=1.
// Unlike integer Power it has no 0/1/2 fast paths. Nonterminating source mask
// inputs fail before allocations/publication; all other work is owner-staged.
func (m *RationalManager) Power(a *Rational, p uint32, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if p >= 1<<31 {
		return &PowerDomainError{Exponent: p, Reason: PowerMaskOverflow}
	}
	power, work := NewRational(), NewRational()
	defer m.drop(&power)
	defer m.drop(&work)
	if e := m.Copy(&power, a); e != nil {
		return e
	}
	if e := m.SetInt64(&work, 1); e != nil {
		return e
	}
	for mask := uint32(1); mask <= p; mask <<= 1 {
		if mask&p != 0 {
			if e := m.Mul(&work, &power, &work); e != nil {
				return e
			}
		}
		if e := m.Mul(&power, &power, &power); e != nil {
			return e
		}
	}
	return m.Move(out, &work)
}
