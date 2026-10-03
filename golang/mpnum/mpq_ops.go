// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpq.h mixed arithmetic and fused operations.
package mpnum

import "errors"

var ErrRationalNotInteger = errors.New("mpnum: operation requires integer-valued rational operands")

// AddInt ports add(mpq,mpz,mpq), including the source integer fast branch
// and rat_add's single local/cached cross-product temporary.
func (m *RationalManager) AddInt(a *Rational, b *Int, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if isIntegerRational(a) {
		if e := m.z.Add(&a.num, b, &out.num); e != nil {
			return e
		}
		return m.resetDenominator(out)
	} else {
		var local Int
		tmp := &m.addTmp1
		if m.synchronized {
			tmp = &local
			defer m.z.Del(&local)
		}
		if e := m.z.Mul(b, &a.den, tmp); e != nil {
			return e
		}
		if e := m.z.Copy(&out.den, &a.den); e != nil {
			return e
		}
		if e := m.z.Add(&a.num, tmp, &out.num); e != nil {
			return e
		}
		if e := m.normalize(out); e != nil {
			return e
		}
	}
	return nil
}

// MulInts implements the inherited integer-product rational-result overload.
func (m *RationalManager) MulInts(a, b *Int, out *Rational) error {
	if e := m.z.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.z.Mul(a, b, &out.num); e != nil {
		return e
	}
	return m.resetDenominator(out)
}
func (m *RationalManager) MulInt(a *Int, b, out *Rational) error {
	if e := m.z.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	integer := isIntegerRational(b)
	if e := m.z.Mul(a, &b.num, &out.num); e != nil {
		return e
	}
	if integer {
		return m.resetDenominator(out)
	}
	if e := m.z.Copy(&out.den, &b.den); e != nil {
		return e
	}
	return m.normalize(out)
}
func (m *RationalManager) DivInt(a *Rational, b *Int, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.z.Copy(&out.num, &a.num); e != nil {
		return e
	}
	if e := m.z.Mul(&a.den, b, &out.den); e != nil {
		return e
	}
	if b.val < 0 {
		if e := m.z.Neg(&out.num); e != nil {
			return e
		}
		if e := m.z.Neg(&out.den); e != nil {
			return e
		}
	}
	return m.normalize(out)
}

// AccDiv publishes denominator multiplication before rereading the divisor.
func (m *RationalManager) AccDiv(a *Rational, b *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.z.Mul(&a.den, b, &a.den); e != nil {
		return e
	}
	if b.val < 0 {
		if e := m.z.Neg(&a.num); e != nil {
			return e
		}
		if e := m.z.Neg(&a.den); e != nil {
			return e
		}
	}
	return m.normalize(a)
}

// RatDiv deliberately retains raw denominator sign and actual Copy aliases.
func (m *RationalManager) RatDiv(a, b *Int, out *Rational) error {
	if e := m.z.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.z.Copy(&out.num, a); e != nil {
		return e
	}
	if e := m.z.Copy(&out.den, b); e != nil {
		return e
	}
	return m.normalize(out)
}
func (m *RationalManager) AddMul(a, b, c, out *Rational) error { return m.fused(a, b, c, out, false) }
func (m *RationalManager) SubMul(a, b, c, out *Rational) error { return m.fused(a, b, c, out, true) }
func (m *RationalManager) fused(a, b, c, out *Rational, subtract bool) error {
	for _, q := range []*Rational{a, b, c, out} {
		if e := m.check(q); e != nil {
			return e
		}
	}
	if isIntegerRational(b) && b.num.IsSmall() && (b.num.val == 1 || b.num.val == -1) {
		if subtract == (b.num.val == 1) {
			return m.Sub(a, c, out)
		}
		return m.Add(a, c, out)
	}
	local := NewRational()
	tmp := &m.addMulTmp
	if m.synchronized {
		tmp = &local
		defer m.drop(&local)
	}
	if e := m.Mul(b, c, tmp); e != nil {
		return e
	}
	if subtract {
		return m.Sub(a, tmp, out)
	}
	return m.Add(a, tmp, out)
}
func (m *RationalManager) AddMulInt(a *Rational, b *Int, c, out *Rational) error {
	return m.fusedInt(a, b, c, out, false)
}
func (m *RationalManager) SubMulInt(a *Rational, b *Int, c, out *Rational) error {
	return m.fusedInt(a, b, c, out, true)
}
func (m *RationalManager) fusedInt(a *Rational, b *Int, c, out *Rational, subtract bool) error {
	for _, q := range []*Rational{a, c, out} {
		if e := m.check(q); e != nil {
			return e
		}
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if b.IsSmall() && (b.val == 1 || b.val == -1) {
		if subtract == (b.val == 1) {
			return m.Sub(a, c, out)
		}
		return m.Add(a, c, out)
	}
	local := NewRational()
	tmp := &m.addMulTmp
	if m.synchronized {
		tmp = &local
		defer m.drop(&local)
	}
	if e := m.MulInt(b, c, tmp); e != nil {
		return e
	}
	if subtract {
		return m.Sub(a, tmp, out)
	}
	return m.Add(a, tmp, out)
}

const (
	qIntegerDiv uint8 = iota
	qIntegerMachineDiv
	qIntegerRem
	qIntegerMod
)

func (m *RationalManager) integerOperands(a, b *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if modularSourceDebug && (!isIntegerRational(a) || !isIntegerRational(b)) {
		return ErrRationalNotInteger
	}
	return nil
}
func (m *RationalManager) integerOp(a, b *Rational, out *Int, operation uint8) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if modularSourceDebug && (!isIntegerRational(a) || !isIntegerRational(b)) {
		return ErrRationalNotInteger
	}
	// Direct output identity preserves the source base-manager's divisor/output
	// alias quirks for Euclidean division and modulus.
	switch operation {
	case qIntegerDiv:
		return m.z.Div(&a.num, &b.num, out)
	case qIntegerMachineDiv:
		return m.z.MachineDiv(&a.num, &b.num, out)
	case qIntegerRem:
		return m.z.Rem(&a.num, &b.num, out)
	default:
		return m.z.Mod(&a.num, &b.num, out)
	}
}
func (m *RationalManager) integerOpRational(a, b, out *Rational, operation uint8) error {
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.integerOp(a, b, &out.num, operation); e != nil {
		return e
	}
	return m.resetDenominator(out)
}
func (m *RationalManager) IDiv(a, b, out *Rational) error {
	return m.integerOpRational(a, b, out, qIntegerDiv)
}
func (m *RationalManager) MachineIDiv(a, b, out *Rational) error {
	return m.integerOpRational(a, b, out, qIntegerMachineDiv)
}
func (m *RationalManager) Rem(a, b, out *Rational) error {
	return m.integerOpRational(a, b, out, qIntegerRem)
}
func (m *RationalManager) Mod(a, b, out *Rational) error {
	return m.integerOpRational(a, b, out, qIntegerMod)
}
func (m *RationalManager) IDivToInt(a, b *Rational, out *Int) error {
	return m.integerOp(a, b, out, qIntegerDiv)
}
func (m *RationalManager) MachineIDivToInt(a, b *Rational, out *Int) error {
	return m.integerOp(a, b, out, qIntegerMachineDiv)
}
func (m *RationalManager) RemToInt(a, b *Rational, out *Int) error {
	return m.integerOp(a, b, out, qIntegerRem)
}
func (m *RationalManager) ModToInt(a, b *Rational, out *Int) error {
	return m.integerOp(a, b, out, qIntegerMod)
}
