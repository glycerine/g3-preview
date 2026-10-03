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
		// Owner-local staged values preserve outputs on predictable allocation
		// failures. Source branch operands are unchanged except that c==b reads
		// the quotient work value, emulating the source's overwritten divisor.
		var work, tmp Int
		defer m.Del(&work)
		defer m.Del(&tmp)
		if e := m.MachineDivRem(a, b, &work, &tmp); e != nil {
			return e
		}
		divisor := b
		if c == b {
			divisor = &work
		}
		if tmp.val != 0 {
			one := NewInt(1)
			var e error
			if divisor.val < 0 {
				e = m.Add(&work, &one, &work)
			} else {
				e = m.Sub(&work, &one, &work)
			}
			if e != nil {
				return e
			}
		}
		return m.Move(c, &work)
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
	var work Int
	defer m.Del(&work)
	if e := m.Rem(a, b, &work); e != nil {
		return e
	}
	divisor := b
	if c == b {
		divisor = &work
	}
	if work.val < 0 {
		var e error
		if divisor.val > 0 {
			e = m.Add(&work, divisor, &work)
		} else {
			e = m.Sub(&work, divisor, &work)
		}
		if e != nil {
			return e
		}
	}
	return m.Move(c, &work)
}

func (m *IntegerManager) normalizedTmpSize(idx int, size uint32) uint32 {
	for size > 0 && m.digit(m.tmp[idx], size-1) == 0 {
		size--
	}
	return size
}
func (m *IntegerManager) replacementForTmp(idx int, dst *Int, size uint32) (numericCellRef, error) {
	size = m.normalizedTmpSize(idx, size)
	if dst.cell != 0 || size == 0 || size == 1 && m.digit(m.tmp[idx], 0) <= math.MaxInt32 {
		return 0, nil
	}
	return m.allocate(6, 0)
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
	m.resizeCell(m.tmp[0], qsize)
	m.resizeCell(m.tmp[1], rsize)
	if e := m.divideCells(ca, cb); e != nil {
		return e
	}
	var qreplacement, rreplacement numericCellRef
	var e error
	if mode != remainderOnly {
		qreplacement, e = m.replacementForTmp(0, q, qsize)
		if e != nil {
			return e
		}
	}
	if mode != quotientOnly {
		rreplacement, e = m.replacementForTmp(1, r, rsize)
		if e != nil {
			if qreplacement != 0 {
				m.free(qreplacement)
			}
			return e
		}
	}
	sign := int32(1)
	if sa != sb {
		sign = -1
	}
	if mode != remainderOnly {
		if e = m.publishTmpAt(0, q, sign, qsize, qreplacement); e != nil {
			if rreplacement != 0 {
				m.free(rreplacement)
			}
			return e
		}
	}
	if mode != quotientOnly {
		return m.publishTmpAt(1, r, sa, rsize, rreplacement)
	}
	return nil
}

// divideCells borrows ephemeral indexed views; no arena backing slice escapes.
func (m *IntegerManager) divideCells(a, b numericCellRef) error {
	ca, cb := m.cell(a), m.cell(b)
	cq, cr := m.cell(m.tmp[0]), m.cell(m.tmp[1])
	_, e := m.natural.divide(limbView{words: m.words, ref: ca.payload, size: ca.size}, limbView{words: m.words, ref: cb.payload, size: cb.size}, limbView{words: m.words, ref: cq.payload, size: cq.size}, limbView{words: m.words, ref: cr.payload, size: cr.size})
	return e
}
