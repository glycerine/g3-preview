// Licensed under the MIT license.
// Source: src/util/mpq.cpp power, including its final unused square.
package mpnum

// Power preserves source rational repeated squaring, including zero^zero=1.
// Unlike integer Power it has no 0/1/2 fast paths. Nonterminating source mask
// inputs fail only at their actual mask wrap, retaining completed publications.
func (m *RationalManager) Power(a *Rational, p uint32, out *Rational) (_rawErr error) {
	if e := m.checkScalarValue(a); e != nil {
		return e
	}
	if e := m.checkScalarValue(out); e != nil {
		return e
	}
	r := &m.rawArithmetic
	if r.powerActive {
		return ErrRationalRawArithmeticPending
	}
	generation, e := m.beginRawGeneration()
	if e != nil {
		return e
	}
	r.powerGeneration = generation
	r.power = NewRational()
	r.powerActive = true
	r.powerEnd = false
	r.powerGuard = false
	r.powerCursor = 0
	r.powerCause = nil
	power := &r.power
	defer m.finishRawPower(&_rawErr)
	if e := m.Copy(power, a); e != nil {
		return e
	}
	if e := m.SetInt64(out, 1); e != nil {
		return e
	}
	for mask := uint32(1); mask <= p; mask <<= 1 {
		if mask == 0 {
			return &PowerDomainError{Exponent: p, Reason: PowerMaskOverflow}
		}
		if mask&p != 0 {
			if e := m.Mul(out, power, out); e != nil {
				return e
			}
		}
		if e := m.Mul(power, power, power); e != nil {
			return e
		}
	}
	return m.rawPowerEnd()
}
