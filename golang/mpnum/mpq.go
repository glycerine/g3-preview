// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpq.h and src/util/mpq.cpp, Leonardo de Moura (2010-06-21).
package mpnum

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

var ErrUninitializedRational = errors.New("mpnum: rational has zero denominator; use NewRational")

// Rational is the source pair of pointer-free, owner-local integer numerals.
// NewRational is required: a raw Go zero value has an invalid denominator.
// Big values must never be shallow-copied or used with another manager.
type Rational struct{ num, den Int }

func NewRational() Rational       { return Rational{den: NewInt(1)} }
func (a *Rational) IsSmall() bool { return a.num.IsSmall() && a.den.IsSmall() }

type RationalConfig struct {
	Integers     IntegerConfig
	Synchronized bool
}

// RationalManager owns an IntegerManager and the source scalar scratch fields.
// Synchronized selects source local-temporary algorithms; false selects the
// source reusable scratch algorithms. Both are owner-confined, not concurrent.
// Manager header bytes include all retained scalar rational scratch members.
type RationalManager struct {
	z                         *IntegerManager
	nTmp, addTmp1, addTmp2    Int
	addMulTmp, ltTmp1, ltTmp2 Rational
	synchronized, closed      bool
	budget                    *arena.Budget
	headerBytes               uint64
}

func NewRationalManager(config RationalConfig) (*RationalManager, error) {
	if config.Integers.Cells.Budget != config.Integers.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	budget := config.Integers.Cells.Budget
	header := uint64(reflect.TypeOf(RationalManager{}).Size())
	if e := budget.Reserve(header); e != nil {
		return nil, e
	}
	z, e := NewIntegerManager(config.Integers)
	if e != nil {
		budget.Release(header)
		return nil, e
	}
	return &RationalManager{z: z, synchronized: config.Synchronized, budget: budget, headerBytes: header, addMulTmp: NewRational(), ltTmp1: NewRational(), ltTmp2: NewRational()}, nil
}
func (m *RationalManager) Integers() *IntegerManager { return m.z }
func (m *RationalManager) check(a *Rational) error {
	if m.closed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if e := m.z.check(&a.num); e != nil {
		return e
	}
	if e := m.z.check(&a.den); e != nil {
		return e
	}
	if a.den.val == 0 {
		return ErrUninitializedRational
	}
	return nil
}
func (m *RationalManager) checkCleanup(a *Rational) error {
	if m.closed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if e := m.z.checkCleanup(&a.num); e != nil {
		return e
	}
	if e := m.z.checkCleanup(&a.den); e != nil {
		return e
	}
	// Cleanup/reset acts on owned components, including finite source results
	// with denominator zero. This does not authorize ordinary numeral reads.
	return nil
}
func (m *RationalManager) drop(a *Rational) { m.z.Del(&a.num); m.z.Del(&a.den) }
func (m *RationalManager) Close() {
	if m.closed {
		return
	}
	// Source destructor order, then base-manager cleanup. Caller numerals are
	// invalidated with the source-owned numerical service at owner close.
	m.z.Del(&m.nTmp)
	m.z.Del(&m.addTmp1)
	m.z.Del(&m.addTmp2)
	m.drop(&m.ltTmp1)
	m.drop(&m.ltTmp2)
	m.drop(&m.addMulTmp)
	m.z.Close()
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
}
func (m *RationalManager) resetDenominator(a *Rational) error {
	if e := m.z.Del(&a.den); e != nil {
		return e
	}
	a.den.val = 1
	return nil
}
func isIntegerRational(a *Rational) bool { return a.den.IsSmall() && a.den.val == 1 }
func isOneInteger(a *Int) bool           { return a.IsSmall() && a.val == 1 }

func (m *RationalManager) Del(a *Rational) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if e := m.z.Del(&a.num); e != nil {
		return e
	}
	return m.z.Del(&a.den)
}
func (m *RationalManager) Reset(a *Rational) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if e := m.z.Reset(&a.num); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) Neg(a *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Neg(&a.num)
}
func (m *RationalManager) Abs(a *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Abs(&a.num)
}
func (m *RationalManager) Move(dst, src *Rational) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if e := m.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.z.Move(&dst.num, &src.num); e != nil {
		return e
	}
	return m.z.Move(&dst.den, &src.den)
}
func (m *RationalManager) Swap(a, b *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	if e := m.z.Swap(&a.num, &b.num); e != nil {
		return e
	}
	return m.z.Swap(&a.den, &b.den)
}
func (m *RationalManager) Copy(dst, src *Rational) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if e := m.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.z.Copy(&dst.num, &src.num); e != nil {
		return e
	}
	return m.z.Copy(&dst.den, &src.den)
}
func (m *RationalManager) SetInt64(a *Rational, n int64) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.SetInt64(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetFraction(a *Rational, n int64, d uint64) error {
	if e := m.check(a); e != nil {
		return e
	}
	if d == 0 {
		return ErrDivideByZero
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.SetInt64(&work.num, n); e != nil {
		return e
	}
	if e := m.z.SetUint64(&work.den, d); e != nil {
		return e
	}
	if e := m.normalize(&work); e != nil {
		return e
	}
	return m.Move(a, &work)
}
func (m *RationalManager) SetParts(a *Rational, n, d *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.check(n); e != nil {
		return e
	}
	if e := m.z.check(d); e != nil {
		return e
	}
	if d.val == 0 {
		return ErrDivideByZero
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.Copy(&work.num, n); e != nil {
		return e
	}
	if e := m.z.Copy(&work.den, d); e != nil {
		return e
	}
	if d.val < 0 {
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
func (m *RationalManager) normalize(a *Rational) error {
	var local Int
	tmp := &m.nTmp
	if m.synchronized {
		tmp = &local
		defer m.z.Del(&local)
	}
	if e := m.z.GCD(&a.num, &a.den, tmp); e != nil {
		return e
	}
	if isOneInteger(tmp) {
		return nil
	}
	if e := m.z.Div(&a.num, tmp, &a.num); e != nil {
		return e
	}
	return m.z.Div(&a.den, tmp, &a.den)
}

// Inv copies into the actual destination before the source in-place action.
func (m *RationalManager) Inv(a, out *Rational) error {
	if e := m.Copy(out, a); e != nil {
		return e
	}
	return m.InvInPlace(out)
}

// InvInPlace retains the source debug assertion and release finite zero Swap.
// A resulting denominator0 is invalid for ordinary reads but remains owned and
// can be destroyed or explicitly Reset using cleanup-only validation.
func (m *RationalManager) InvInPlace(a *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if modularSourceDebug && a.num.val == 0 {
		return ErrIntegerAssertion
	}
	if a.num.val < 0 {
		if e := m.z.Neg(&a.num); e != nil {
			return e
		}
		if e := m.z.Neg(&a.den); e != nil {
			return e
		}
	}
	return m.z.Swap(&a.num, &a.den)
}

func (m *RationalManager) Equal(a, b *Rational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(b); e != nil {
		return false, e
	}
	n, e := m.z.Compare(&a.num, &b.num)
	if e != nil || n != 0 {
		return false, e
	}
	d, e := m.z.Compare(&a.den, &b.den)
	return d == 0, e
}

// Compare preserves source rat_lt's sign shortcuts and exact cross-products.
func (m *RationalManager) Compare(a, b *Rational) (int, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if e := m.check(b); e != nil {
		return 0, e
	}
	if isIntegerRational(a) && isIntegerRational(b) {
		return m.z.Compare(&a.num, &b.num)
	}
	sa, sb := a.num.val, b.num.val
	if sa < 0 && sb >= 0 {
		return -1, nil
	}
	if sa == 0 {
		if sb > 0 {
			return -1, nil
		}
		if sb < 0 {
			return 1, nil
		}
		return 0, nil
	}
	if sa > 0 && sb <= 0 {
		return 1, nil
	}
	if sa < 0 && sb == 0 {
		return -1, nil
	}
	one, two := &m.ltTmp1, &m.ltTmp2
	if m.synchronized {
		localOne, localTwo := NewRational(), NewRational()
		one, two = &localOne, &localTwo
		defer m.drop(one)
		defer m.drop(two)
	}
	if e := m.z.Mul(&a.num, &b.den, &one.num); e != nil {
		return 0, e
	}
	if e := m.resetDenominator(one); e != nil {
		return 0, e
	}
	if e := m.z.Mul(&b.num, &a.den, &two.num); e != nil {
		return 0, e
	}
	if e := m.resetDenominator(two); e != nil {
		return 0, e
	}
	return m.z.Compare(&one.num, &two.num)
}

func (m *RationalManager) Add(a, b, out *Rational) error { return m.addSub(a, b, out, false) }
func (m *RationalManager) Sub(a, b, out *Rational) error { return m.addSub(a, b, out, true) }
func (m *RationalManager) addSub(a, b, out *Rational, subtract bool) error {
	for _, n := range []*Rational{a, b, out} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	work := NewRational()
	defer m.drop(&work)
	if isIntegerRational(a) && isIntegerRational(b) {
		var e error
		if subtract {
			e = m.z.Sub(&a.num, &b.num, &work.num)
		} else {
			e = m.z.Add(&a.num, &b.num, &work.num)
		}
		if e != nil {
			return e
		}
		if e = m.resetDenominator(&work); e != nil {
			return e
		}
		return m.Move(out, &work)
	}
	var localOne, localTwo Int
	one, two := &m.addTmp1, &m.addTmp2
	if m.synchronized {
		one, two = &localOne, &localTwo
		defer m.z.Del(one)
		defer m.z.Del(two)
	}
	if e := m.z.Mul(&a.num, &b.den, one); e != nil {
		return e
	}
	if e := m.z.Mul(&b.num, &a.den, two); e != nil {
		return e
	}
	if e := m.z.Mul(&a.den, &b.den, &work.den); e != nil {
		return e
	}
	var e error
	if subtract {
		e = m.z.Sub(one, two, &work.num)
	} else {
		e = m.z.Add(one, two, &work.num)
	}
	if e != nil {
		return e
	}
	if e = m.normalize(&work); e != nil {
		return e
	}
	return m.Move(out, &work)
}
func (m *RationalManager) Mul(a, b, out *Rational) error {
	for _, n := range []*Rational{a, b, out} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	// Source chooses this branch before output aliases can mutate operands.
	integers := isIntegerRational(a) && isIntegerRational(b)
	if e := m.z.Mul(&a.num, &b.num, &out.num); e != nil {
		return e
	}
	if integers {
		return m.resetDenominator(out)
	}
	if e := m.z.Mul(&a.den, &b.den, &out.den); e != nil {
		return e
	}
	return m.normalize(out)
}

// Div preserves the actual divisor/destination branch and component order.
// Errors retain completed caller fields; no staged pair masks normalization.
func (m *RationalManager) Div(a, b, out *Rational) error {
	for _, n := range []*Rational{a, b, out} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b == out {
		var tmp Int
		cleanup := true
		defer func() {
			if cleanup {
				m.z.Del(&tmp)
			}
		}()
		if e := m.z.Mul(&a.num, &b.den, &tmp); e != nil {
			return e
		}
		if e := m.z.Mul(&a.den, &b.num, &out.den); e != nil {
			return e
		}
		if e := m.z.Copy(&out.num, &tmp); e != nil {
			return e
		}
		if e := m.z.Del(&tmp); e != nil {
			return e
		}
		cleanup = false // source Del statement precedes sign repair and normalization
	} else {
		if e := m.z.Mul(&a.num, &b.den, &out.num); e != nil {
			return e
		}
		if e := m.z.Mul(&a.den, &b.num, &out.den); e != nil {
			return e
		}
	}
	if out.den.val < 0 {
		if e := m.z.Neg(&out.num); e != nil {
			return e
		}
		if e := m.z.Neg(&out.den); e != nil {
			return e
		}
	}
	return m.normalize(out)
}
