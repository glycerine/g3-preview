// Licensed under the MIT license.
// Source: src/util/mpq.cpp root and mpq.h square predicates/output wrappers.
package util

// Root retains the source short circuit: an inexact numerator root publishes
// that numerator and leaves the previous output denominator unchanged. A false
// denominator result also publishes; no rational normalization is performed.
// Staged work preserves source aliases while delaying publication on errors.
func (m *RationalManager) Root(a *Rational, n uint32, out *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(out); e != nil {
		return false, e
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.Copy(&work, out); e != nil {
		return false, e
	}
	input := a
	if a == out {
		input = &work
	}
	exact, e := m.Z.Root(&input.Num, n, &work.Num)
	if e != nil {
		return false, e
	}
	if exact {
		exact, e = m.Z.Root(&input.Den, n, &work.Den)
		if e != nil {
			return false, e
		}
	}
	if e = m.Move(out, &work); e != nil {
		return false, e
	}
	return exact, nil
}

// IsPerfectSquare retains denominator-reset, short-circuit and alias-reset
// behavior from source. Only a fully precise fractional result is normalized.
func (m *RationalManager) IsPerfectSquare(a, out *Rational) (bool, error) {
	return m.perfectSquare(a, out, false)
}
func (m *RationalManager) IsIntPerfectSquare(a, out *Rational) (bool, error) {
	return m.perfectSquare(a, out, true)
}
func (m *RationalManager) perfectSquare(a, out *Rational, integerOnly bool) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(out); e != nil {
		return false, e
	}
	integer := isIntegerRational(a)
	if integerOnly && !integer {
		return false, &RootDomainError{Reason: RootRequiresInteger}
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.Copy(&work, out); e != nil {
		return false, e
	}
	input := a
	if a == out {
		input = &work
	}
	var exact bool
	var e error
	if integer {
		if e = m.resetDenominator(&work); e != nil {
			return false, e
		}
		exact, e = m.Z.IsPerfectSquare(&input.Num, &work.Num)
	} else {
		exact, e = m.Z.IsPerfectSquare(&input.Num, &work.Num)
		if e == nil && exact {
			exact, e = m.Z.IsPerfectSquare(&input.Den, &work.Den)
		}
		if e == nil && exact {
			e = m.normalize(&work)
		}
	}
	if e != nil {
		return false, e
	}
	if e = m.Move(out, &work); e != nil {
		return false, e
	}
	return exact, nil
}
