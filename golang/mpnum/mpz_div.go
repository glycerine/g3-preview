// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.h and mpz.cpp internal signed division wrappers/core.
// Knuth division translated from src/util/mpn.cpp (Copyright (c) 2011
// Microsoft Corporation; Christoph Wintersteiger).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
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

func (m *IntegerManager) divRead(ref arena.VectorRef, i uint32) uint32 {
	v, e := m.words.Read(ref, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *IntegerManager) divWrite(ref arena.VectorRef, i, v uint32) {
	if e := m.words.Write(ref, i, v); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) resizeDiv(idx int, size uint32) error {
	ref := m.divScratch[idx]
	var e error
	if ref == (arena.VectorRef{}) {
		ref, e = m.words.Allocate(size, max(size, 16))
	} else {
		ref, e = m.words.Resize(ref, size)
	}
	if e != nil {
		return e
	}
	m.divScratch[idx] = ref
	return nil
}

// divideCells follows mpn::div with the mpz smaller-length guard already
// checked. All four division buffers are reusable owner-budgeted Words refs.
// The source's bool return is irrelevant here: quot_rem_core ignored it.
func (m *IntegerManager) divideCells(a, b numericCellRef) error {
	na, nb := m.cell(a).size, m.cell(b).size
	if na == 1 && nb == 1 {
		m.writeDigit(m.tmp[0], 0, m.digit(a, 0)/m.digit(b, 0))
		m.writeDigit(m.tmp[1], 0, m.digit(a, 0)%m.digit(b, 0))
		return nil
	}
	if na == nb && m.digit(a, na-1) < m.digit(b, nb-1) {
		m.writeDigit(m.tmp[0], 0, 0)
		for i := uint32(0); i < nb; i++ {
			m.writeDigit(m.tmp[1], i, m.digit(a, i))
		}
		return nil
	}
	if na == math.MaxUint32 {
		return arena.ErrBounds
	}
	var d uint
	for m.digit(b, nb-1)<<d&0x80000000 == 0 {
		d++
	}
	if e := m.resizeDiv(0, na+1); e != nil {
		return e
	}
	if e := m.resizeDiv(1, nb); e != nil {
		return e
	}
	u, v := m.divScratch[0], m.divScratch[1]
	if d == 0 {
		m.divWrite(u, na, 0)
		for i := uint32(0); i < na; i++ {
			m.divWrite(u, i, m.digit(a, i))
		}
		for i := uint32(0); i < nb; i++ {
			m.divWrite(v, i, m.digit(b, i))
		}
	} else {
		m.divWrite(u, na, m.digit(a, na-1)>>(32-d))
		for i := na - 1; i > 0; i-- {
			m.divWrite(u, i, m.digit(a, i)<<d|m.digit(a, i-1)>>(32-d))
		}
		m.divWrite(u, 0, m.digit(a, 0)<<d)
		for i := nb - 1; i > 0; i-- {
			m.divWrite(v, i, m.digit(b, i)<<d|m.digit(b, i-1)>>(32-d))
		}
		m.divWrite(v, 0, m.digit(b, 0)<<d)
	}
	if nb == 1 {
		m.divideCellOne(na + 1)
	} else {
		if e := m.divideCellMany(na+1, nb); e != nil {
			return e
		}
	}
	if d == 0 {
		for i := uint32(0); i < nb; i++ {
			m.writeDigit(m.tmp[1], i, m.divRead(u, i))
		}
	} else {
		mask := uint32(1)<<d - 1
		for i := uint32(0); i+1 < nb; i++ {
			m.writeDigit(m.tmp[1], i, m.divRead(u, i)>>d|(m.divRead(u, i+1)&mask)<<(32-d))
		}
		m.writeDigit(m.tmp[1], nb-1, m.divRead(u, nb-1)>>d)
	}
	return nil
}
func (m *IntegerManager) divideCellOne(size uint32) {
	u, v := m.divScratch[0], m.divScratch[1]
	denom := uint64(m.divRead(v, 0))
	for j := size - 1; j > 0; j-- {
		temp := uint64(m.divRead(u, j))<<32 | uint64(m.divRead(u, j-1))
		qh := temp / denom
		if qh >= uint64(1)<<32 {
			panic("mpnum: unnormalized single-limb quotient")
		}
		ms := temp - qh*denom
		borrow := ms > temp
		m.divWrite(u, j-1, uint32(ms))
		m.divWrite(u, j, uint32(ms>>32))
		m.writeDigit(m.tmp[0], j-1, uint32(qh))
		if borrow {
			m.writeDigit(m.tmp[0], j-1, uint32(qh)-1)
			m.divWrite(u, j, m.divRead(u, j-1)+uint32(denom))
		}
	}
}

// divideCellMany is source Knuth Algorithm D, including estimate correction,
// single-limb multiplication, modular subtraction, and conditional add-back.
func (m *IntegerManager) divideCellMany(size, n uint32) error {
	if n > math.MaxUint32-2 {
		return arena.ErrBounds
	}
	if e := m.resizeDiv(2, n+1); e != nil {
		return e
	}
	u, v, ms := m.divScratch[0], m.divScratch[1], m.divScratch[2]
	count := size - n
	const base = uint64(1) << 32
	for remaining := count; remaining > 0; remaining-- {
		j := remaining - 1
		temp := uint64(m.divRead(u, j+n))<<32 | uint64(m.divRead(u, j+n-1))
		qh := temp / uint64(m.divRead(v, n-1))
		rh := temp % uint64(m.divRead(v, n-1))
		for {
			if qh < base && qh*uint64(m.divRead(v, n-2)) <= (rh<<32)+uint64(m.divRead(u, j+n-2)) {
				break
			}
			qh--
			rh += uint64(m.divRead(v, n-1))
			if rh >= base {
				break
			}
		}
		if qh >= base {
			panic("mpnum: invalid quotient estimate")
		}
		small := uint32(qh)
		// Source mul(&small,1,v,n,ms), Algorithm M with the zero-digit branch.
		m.divWrite(ms, 0, 0)
		for i := uint32(0); i < n; i++ {
			digit := m.divRead(v, i)
			if digit == 0 {
				m.divWrite(ms, i+1, 0)
				continue
			}
			t := uint64(small)*uint64(digit) + uint64(m.divRead(ms, i))
			m.divWrite(ms, i, uint32(t))
			m.divWrite(ms, i+1, uint32(t>>32))
		}
		var borrow uint32
		for i := uint32(0); i < n+1; i++ {
			a, b := m.divRead(u, j+i), m.divRead(ms, i)
			r := a - b
			c1 := r > a
			out := r - borrow
			c2 := out > r
			m.divWrite(u, j+i, out)
			borrow = 0
			if c1 || c2 {
				borrow = 1
			}
		}
		m.writeDigit(m.tmp[0], j, small)
		if borrow != 0 {
			m.writeDigit(m.tmp[0], j, small-1)
			if e := m.resizeDiv(3, n+2); e != nil {
				return e
			}
			ab := m.divScratch[3]
			var carry uint32
			for i := uint32(0); i < n+1; i++ {
				a := uint32(0)
				if i < n {
					a = m.divRead(v, i)
				}
				b := m.divRead(u, j+i)
				r := a + b
				c1 := r < a
				out := r + carry
				c2 := out < r
				m.divWrite(ab, i, out)
				carry = 0
				if c1 || c2 {
					carry = 1
				}
			}
			m.divWrite(ab, n+1, carry)
			for i := uint32(0); i < n+1; i++ {
				m.divWrite(u, j+i, m.divRead(ab, i))
			}
		}
	}
	return nil
}
