// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpq.h/.cpp inspection and thin extraction/setter branches;
// explicit inclusive 32-bit projections come from src/util/rational.h.
package mpnum

import "errors"

var ErrRationalFloatDomain = errors.New("mpnum: rational converted floating denominator must be positive")

func (m *RationalManager) Precise() bool { return true }
func (m *RationalManager) Field() bool   { return true }

// Sign retains source m_val: small numerators return their entire inline value,
// while big numerators return -1/+1. It is not a normalized mathematical sign.
func (m *RationalManager) Sign(a *Rational) (int32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	return a.num.val, nil
}
func (m *RationalManager) IsPos(a *Rational) (bool, error) { v, e := m.Sign(a); return v > 0, e }
func (m *RationalManager) IsNeg(a *Rational) (bool, error) { v, e := m.Sign(a); return v < 0, e }
func (m *RationalManager) IsZero(a *Rational) (bool, error) {
	v, e := m.Sign(a)
	if e != nil {
		return false, e
	}
	return v == 0, nil
}
func (m *RationalManager) IsNonPos(a *Rational) (bool, error) {
	v, e := m.Sign(a)
	if e != nil {
		return false, e
	}
	return v <= 0, nil
}
func (m *RationalManager) IsNonNeg(a *Rational) (bool, error) {
	v, e := m.Sign(a)
	if e != nil {
		return false, e
	}
	return v >= 0, nil
}
func (m *RationalManager) IsInt(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return isIntegerRational(a), nil
}
func (m *RationalManager) IsOne(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return isOneInteger(&a.num) && isOneInteger(&a.den), nil
}
func (m *RationalManager) IsMinusOne(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return a.num.IsSmall() && a.num.val == -1 && isOneInteger(&a.den), nil
}

// Hash is deliberately numerator-only; fraction collisions and sign-free big
// magnitude hashes are source behavior, resolved by source structural equality.
func (m *RationalManager) Hash(a *Rational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	return m.z.Hash(&a.num)
}
func (m *RationalManager) IsUint64(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if !isIntegerRational(a) {
		return false, nil
	}
	return m.z.IsUint64(&a.num)
}
func (m *RationalManager) IsInt64(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if !isIntegerRational(a) {
		return false, nil
	}
	return m.z.IsInt64(&a.num)
}
func (m *RationalManager) Uint64(a *Rational) (uint64, error) {
	ok, e := m.IsUint64(a)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrConversion
	}
	return m.z.Uint64(&a.num)
}
func (m *RationalManager) Int64(a *Rational) (int64, error) {
	ok, e := m.IsInt64(a)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrConversion
	}
	return m.z.Int64(&a.num)
}

// IsUnsigned/Unsigned project rational.h's inclusive UINT_MAX boundary onto
// the explicit owner service. MPZ IsUint32/Uint32 instead exclude UINT_MAX.
func (m *RationalManager) IsUnsigned(a *Rational) (bool, error) {
	ok, e := m.IsUint64(a)
	if e != nil || !ok {
		return false, e
	}
	v, e := m.z.Uint64(&a.num)
	return v < uint64(1)<<32, e
}
func (m *RationalManager) Unsigned(a *Rational) (uint32, error) {
	ok, e := m.IsUnsigned(a)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrConversion
	}
	v, e := m.z.Uint64(&a.num)
	return uint32(v), e
}

// IsInt32/Int32 preserve rational.h's inclusive signed boundaries, unlike the
// source MPZ strict-bound variants. No global-manager wrapper is introduced.
func (m *RationalManager) IsInt32(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.IsSmall() && isIntegerRational(a) {
		return true, nil
	}
	ok, e := m.IsInt64(a)
	if e != nil || !ok {
		return false, e
	}
	v, e := m.z.Int64(&a.num)
	return v >= -2147483648 && v <= 2147483647, e
}
func (m *RationalManager) Int32(a *Rational) (int32, error) {
	ok, e := m.IsInt32(a)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrConversion
	}
	v, e := m.z.Int64(&a.num)
	return int32(v), e
}
func (m *RationalManager) IsEven(a *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if !isIntegerRational(a) {
		return false, nil
	}
	return m.z.IsEven(&a.num)
}
func (m *RationalManager) IsPowerOfTwo(a *Rational) (bool, uint32, error) {
	if e := m.check(a); e != nil {
		return false, 0, e
	}
	if !isIntegerRational(a) {
		return false, 0, nil
	}
	return m.z.IsPowerOfTwo(&a.num)
}
func (m *RationalManager) Bitsize(a *Rational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n, e := m.z.Bitsize(&a.num)
	if e != nil || isIntegerRational(a) {
		return n, e
	}
	d, e := m.z.Bitsize(&a.den)
	return n + d, e // source unsigned sum
}
func (m *RationalManager) StorageSize(a *Rational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n, e := m.z.SizeInfo(&a.num)
	if e != nil {
		return 0, e
	}
	d, e := m.z.SizeInfo(&a.den)
	return n + d, e // sorting weight, not owner bytes
}

// Float64 retains each underlying source UINT_MAX-radix conversion and n/d.
// The source SASSERT(d>0) becomes an explicit development-domain error, including
// negative or NaN denominators; numerator NaN and valid-denominator Inf remain.
func (m *RationalManager) Float64(a *Rational) (float64, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n, e := m.z.Float64(&a.num)
	if e != nil {
		return 0, e
	}
	d, e := m.z.Float64(&a.den)
	if e != nil {
		return 0, e
	}
	if !(d > 0) {
		return 0, ErrRationalFloatDomain
	}
	return n / d, nil
}

// Extraction copies owned payloads explicitly and does not normalize either
// source or result. Component output aliases follow the source Copy action.
func (m *RationalManager) Numerator(a *Rational, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Copy(out, &a.num)
}
func (m *RationalManager) Denominator(a *Rational, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Copy(out, &a.den)
}
func (m *RationalManager) NumeratorRational(a, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.SetInteger(out, &a.num)
}
func (m *RationalManager) DenominatorRational(a, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.SetInteger(out, &a.den)
}

// SetInteger and the thin setters publish source numerator actions followed by
// reset_denominator. Errors retain actual Integer publication prefixes and leave
// DEN untouched if NUM fails. No borrowed limb slice remains.
func (m *RationalManager) SetInteger(a *Rational, n *Int) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.Copy(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetUint64(a *Rational, n uint64) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.SetUint64(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetLimbs(a *Rational, digits []uint32) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.SetLimbs(&a.num, digits); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SwapNumerator(n *Int, a *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Swap(n, &a.num)
}
func (m *RationalManager) Inc(a *Rational) error { one := NewInt(1); return m.AddInt(a, &one, a) }
func (m *RationalManager) Dec(a *Rational) error {
	minusOne := NewInt(-1)
	return m.AddInt(a, &minusOne, a)
}
