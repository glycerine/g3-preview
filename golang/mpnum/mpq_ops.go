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
	work := NewRational()
	defer m.drop(&work)
	if isIntegerRational(a) {
		if e := m.z.Add(&a.num, b, &work.num); e != nil {
			return e
		}
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
		if e := m.z.Copy(&work.den, &a.den); e != nil {
			return e
		}
		if e := m.z.Add(&a.num, tmp, &work.num); e != nil {
			return e
		}
		if e := m.normalize(&work); e != nil {
			return e
		}
	}
	return m.Move(out, &work)
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
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Mul(a, b, &work.num); e != nil {
		return e
	}
	return m.Move(out, &work)
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
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Mul(a, &b.num, &work.num); e != nil {
		return e
	}
	if !isIntegerRational(b) {
		if e := m.z.Copy(&work.den, &b.den); e != nil {
			return e
		}
		if e := m.normalize(&work); e != nil {
			return e
		}
	}
	return m.Move(out, &work)
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
	if b.val == 0 {
		return ErrDivideByZero
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Copy(&work.num, &a.num); e != nil {
		return e
	}
	// Source set(out.num,a.num) happens before the divisor is read.
	divisor := b
	if b == &out.num {
		divisor = &work.num
	}
	if e := m.z.Mul(&a.den, divisor, &work.den); e != nil {
		return e
	}
	// Source tests divisor sign after denominator publication.
	signValue := divisor
	if b == &out.den {
		signValue = &work.den
	}
	if signValue.val < 0 {
		if e := m.z.Neg(&work.num); e != nil {
			return e
		}
		if e := m.z.Neg(&work.den); e != nil {
			return e
		}
	}
	if e := m.normalize(&work); e != nil {
		return e
	}
	return m.Move(out, &work)
}

// AccDiv follows acc_div's denominator multiplication, sign repair, then gcd
// reduction. A staged pair protects the old value on predictable failures.
func (m *RationalManager) AccDiv(a *Rational, b *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Mul(&a.den, b, &work.den); e != nil {
		return e
	}
	if e := m.z.Copy(&work.num, &a.num); e != nil {
		return e
	}
	signValue := b
	if b == &a.den {
		signValue = &work.den
	}
	if signValue.val < 0 {
		if e := m.z.Neg(&work.num); e != nil {
			return e
		}
		if e := m.z.Neg(&work.den); e != nil {
			return e
		}
	}
	if e := m.normalize(&work); e != nil {
		return e
	}
	return m.Move(a, &work)
}

// RatDiv preserves rat_div's deliberate lack of denominator sign repair;
// this differs from SetParts and DivInt for a negative divisor.
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
	if b.val == 0 {
		return ErrDivideByZero
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Copy(&work.num, a); e != nil {
		return e
	}
	// Source copies numerator first and then rereads a component-aliased b.
	divisor := b
	if b == &out.num {
		divisor = &work.num
	}
	if e := m.z.Copy(&work.den, divisor); e != nil {
		return e
	}
	if e := m.normalize(&work); e != nil {
		return e
	}
	return m.Move(out, &work)
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
	if !isIntegerRational(a) || !isIntegerRational(b) {
		return ErrRationalNotInteger
	}
	return nil
}
func (m *RationalManager) integerOp(a, b *Rational, out *Int, operation uint8) error {
	if e := m.integerOperands(a, b); e != nil {
		return e
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
