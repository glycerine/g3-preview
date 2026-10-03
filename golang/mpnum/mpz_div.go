// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.h and mpz.cpp internal signed division wrappers/core.
// Knuth division translated from src/util/mpn.cpp (Copyright (c) 2011
// Microsoft Corporation; Christoph Wintersteiger).
package mpnum

import (
	"errors"
	"math"
)

var ErrDivideByZero = errors.New("mpnum: integer division by zero")

const (
	quotientOnly         = 0
	remainderOnly        = 1
	quotientAndRemainder = 2
)

// MachineDivRem truncates toward zero; the remainder has the numerator's sign.
// Either output may alias either input. If q==r the source publication order
// is retained: the general/small path ends in remainder, whereas the
// shorter-numerator limb guard copies remainder then resets quotient to zero.
func (m *IntegerManager) MachineDivRem(a, b, q, r *Int) error {
	for _, n := range []*Int{a, b, q, r} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		av, bv := int64(a.val), int64(b.val)
		qv, rv := av/bv, av%bv
		// Only INT_MIN/-1 promotes a small quotient. Allocate before publishing.
		var prepared numericCellRef
		if qv > math.MaxInt32 && q.cell == 0 {
			var e error
			prepared, e = m.allocate(6, 0)
			if e != nil {
				return e
			}
		}
		if prepared != 0 {
			q.cell = prepared
			m.publishCell(prepared)
		}
		if e := m.SetInt64(q, qv); e != nil {
			return e
		}
		return m.SetInt64(r, rv)
	}
	return m.quotRemCore(a, b, q, r, quotientAndRemainder)
}
func (m *IntegerManager) MachineDiv(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		return m.SetInt64(c, int64(a.val)/int64(b.val))
	}
	var dummy Int
	return m.quotRemCore(a, b, c, &dummy, quotientOnly)
}
func (m *IntegerManager) Rem(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		return m.SetInt64(c, int64(a.val)%int64(b.val))
	}
	var dummy Int
	return m.quotRemCore(a, b, &dummy, c, remainderOnly)
}

// Div implements the source Euclidean quotient adjustment for a negative
// numerator. The source rereads b after output publication; when c==b this
// preserves a known source alias defect (documented in PORT_STATUS.md).
func (m *IntegerManager) Div(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.val < 0 {
		var tmp Int
		defer m.Del(&tmp) // explicit Go cleanup on failed source actions
		if e := m.MachineDivRem(a, b, c, &tmp); e != nil {
			return e
		}
		if tmp.val != 0 {
			one := NewInt(1)
			if b.val < 0 {
				return m.Add(c, &one, c)
			}
			return m.Sub(c, &one, c)
		}
		return nil
	}
	return m.MachineDiv(a, b, c)
}

// Mod implements source nonnegative modulus, including its reread of an
// overwritten divisor when c==b (the source alias defect).
func (m *IntegerManager) Mod(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if e := m.Rem(a, b, c); e != nil {
		return e
	}
	if c.val < 0 {
		if b.val > 0 {
			return m.Add(c, b, c)
		}
		return m.Sub(c, b, c)
	}
	return nil
}

func (m *IntegerManager) normalizedTmpSize(idx int, size uint32) uint32 {
	for size > 0 && m.digit(m.tmp[idx], size-1) == 0 {
		size--
	}
	return size
}
func (m *IntegerManager) quotRemCore(a, b, q, r *Int, mode int) error {
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	na, nb := m.cell(ca).size, m.cell(cb).size
	if nb > na {
		if mode != quotientOnly {
			if e := m.Copy(r, a); e != nil {
				return e
			}
		}
		if mode != remainderOnly {
			return m.Reset(q)
		}
		return nil
	}
	qsize, rsize := na-nb+1, nb
	if e := m.ensureTmp(0, qsize); e != nil {
		return e
	}
	if e := m.ensureTmp(1, rsize); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(0, qsize); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(1, rsize); e != nil {
		return e
	}
	if e := m.divideCells(ca, cb); e != nil {
		return e
	}
	sign := int32(1)
	if sa != sb {
		sign = -1
	}
	if mode != remainderOnly {
		if e := m.publishTmpAt(0, q, sign, qsize); e != nil {
			return e
		}
	}
	if mode != quotientOnly {
		return m.publishTmpAt(1, r, sa, rsize)
	}
	return nil
}

// divideCells borrows ephemeral indexed views; no arena backing slice escapes.
func (m *IntegerManager) divideCells(a, b numericCellRef) error {
	ca, cb := m.cell(a), m.cell(b)
	cq, cr := m.cell(m.tmp[0]), m.cell(m.tmp[1])
	_, e := m.natural.divide(limbView{allocator: m.allocator, backing: ca.backing, size: ca.size}, limbView{allocator: m.allocator, backing: cb.backing, size: cb.size}, limbView{allocator: m.allocator, backing: cq.backing, size: cq.span}, limbView{allocator: m.allocator, backing: cr.backing, size: cr.span})
	return e
}
