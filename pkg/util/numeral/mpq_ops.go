// Source: src/util/mpq.h mixed arithmetic and fused operations.
package numeral

import (
	"errors"
)

var ErrRationalNotInteger = errors.New("mpnum: operation requires integer-valued rational operands")

// AddInt ports add(mpq,mpz,mpq), including the source integer fast branch
// and rat_add's single local/cached cross-product temporary.
func (m *RationalManager) AddInt(a *Rational, b *Int, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if isIntegerRational(a) {
		if e := m.Z.Add(&a.Num, b, &out.Num); e != nil {
			return e
		}
		return m.resetDenominator(out)
	} else {
		var local Int
		tmp := &m.AddTmp1
		if m.Synchronized {
			tmp = &local
			defer m.Z.Del(&local)
		}
		if e := m.Z.Mul(b, &a.Den, tmp); e != nil {
			return e
		}
		if e := m.Z.Copy(&out.Den, &a.Den); e != nil {
			return e
		}
		if e := m.Z.Add(&a.Num, tmp, &out.Num); e != nil {
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
	if e := m.Z.check(a); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.Z.Mul(a, b, &out.Num); e != nil {
		return e
	}
	return m.resetDenominator(out)
}
func (m *RationalManager) MulInt(a *Int, b, out *Rational) error {
	return m.mulIntWithAllocationContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *RationalManager) mulIntWithAllocationContext(a *Int, b, out *Rational, context integerAllocationInvocationContext) error {
	if e := m.Z.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	integer := isIntegerRational(b)
	if e := m.Z.mulWithAllocationContext(a, &b.Num, &out.Num, context); e != nil {
		return e
	}
	if integer {
		return m.resetDenominator(out)
	}
	if e := m.Z.copyFromOwnerWithAllocationContext(&out.Den, m.Z, &b.Den, nil, context); e != nil {
		return e
	}
	return m.normalizeWithAllocationContext(out, context)
}
func (m *RationalManager) DivInt(a *Rational, b *Int, out *Rational) error {
	return m.divIntWithAllocationContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *RationalManager) divIntWithAllocationContext(a *Rational, b *Int, out *Rational, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.Z.copyFromOwnerWithAllocationContext(&out.Num, m.Z, &a.Num, nil, context); e != nil {
		return e
	}
	if e := m.Z.mulWithAllocationContext(&a.Den, b, &out.Den, context); e != nil {
		return e
	}
	if b.Val < 0 {
		if e := m.Z.negWithAllocationContext(&out.Num, context); e != nil {
			return e
		}
		if e := m.Z.negWithAllocationContext(&out.Den, context); e != nil {
			return e
		}
	}
	return m.normalizeWithAllocationContext(out, context)
}

// AccDiv publishes denominator multiplication before rereading the divisor.
func (m *RationalManager) AccDiv(a *Rational, b *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.Z.Mul(&a.Den, b, &a.Den); e != nil {
		return e
	}
	if b.Val < 0 {
		if e := m.Z.Neg(&a.Num); e != nil {
			return e
		}
		if e := m.Z.Neg(&a.Den); e != nil {
			return e
		}
	}
	return m.normalize(a)
}

// RatDiv deliberately retains raw denominator sign and actual Copy aliases.
func (m *RationalManager) RatDiv(a, b *Int, out *Rational) error {
	if e := m.Z.check(a); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.Z.Copy(&out.Num, a); e != nil {
		return e
	}
	if e := m.Z.Copy(&out.Den, b); e != nil {
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
	if isIntegerRational(b) && b.Num.IsSmall() && (b.Num.Val == 1 || b.Num.Val == -1) {
		if subtract == (b.Num.Val == 1) {
			return m.Sub(a, c, out)
		}
		return m.Add(a, c, out)
	}
	local := NewRational()
	tmp := &m.AddMulTmp
	if m.Synchronized {
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
	if e := m.Z.check(b); e != nil {
		return e
	}
	if b.IsSmall() && (b.Val == 1 || b.Val == -1) {
		if subtract == (b.Val == 1) {
			return m.Sub(a, c, out)
		}
		return m.Add(a, c, out)
	}
	local := NewRational()
	tmp := &m.AddMulTmp
	if m.Synchronized {
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
	return m.integerOpWithAllocationContext(a, b, out, operation, integerAllocationInvocationContext{})
}
func (m *RationalManager) integerOpWithAllocationContext(a, b *Rational, out *Int, operation uint8, context integerAllocationInvocationContext) error {
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
		return m.Z.divWithAllocationContext(&a.Num, &b.Num, out, context)
	case qIntegerMachineDiv:
		return m.Z.machineDivWithAllocationContext(&a.Num, &b.Num, out, context)
	case qIntegerRem:
		return m.Z.remWithAllocationContext(&a.Num, &b.Num, out, context)
	default:
		return m.Z.modWithAllocationContext(&a.Num, &b.Num, out, context)
	}
}
func (m *RationalManager) integerOpRational(a, b, out *Rational, operation uint8) error {
	return m.integerOpRationalWithAllocationContext(a, b, out, operation, integerAllocationInvocationContext{})
}
func (m *RationalManager) integerOpRationalWithAllocationContext(a, b, out *Rational, operation uint8, context integerAllocationInvocationContext) error {
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.integerOpWithAllocationContext(a, b, &out.Num, operation, context); e != nil {
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
	return m.modWithAllocationContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *RationalManager) modWithAllocationContext(a, b, out *Rational, context integerAllocationInvocationContext) error {
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.integerOperands(a, b); e != nil {
		return e
	}
	if e := m.Z.modWithAllocationContext(&a.Num, &b.Num, &out.Num, context); e != nil {
		return e
	}
	return m.resetDenominator(out)
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
