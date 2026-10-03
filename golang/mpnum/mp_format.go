// Copyright (c) 2011 Microsoft Corporation
// Source: mpn.cpp to_string and mpz.cpp/mpq.cpp display (2006 copyright).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"strings"
)

var ErrShortBuffer = errors.New("mpnum: decimal output buffer too short")

// reserveText retains budgeted scalar-indexed bytes. Growth preserves the
// numerator prefix while preparing a rational denominator. DirectOnly is a
// cold-storage adaptation, not the source allocation profile.
func (m *NaturalManager) reserveText(needed uint32) error {
	if needed <= m.textCapacity {
		return nil
	}
	capacity := m.textCapacity
	if capacity == 0 {
		capacity = 1024
	}
	for capacity < needed {
		if capacity > math.MaxUint32/2 {
			return arena.ErrBounds
		}
		capacity *= 2
	}
	if m.text == nil {
		owner, e := arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: m.config.Descriptors})
		if e != nil {
			return e
		}
		m.text = owner
	}
	ref, e := m.text.Allocate(capacity)
	if e != nil {
		return e
	}
	if m.textRef != (arena.SmallBytesRef{}) {
		for i := uint32(0); i < m.textCapacity; i++ {
			v, e := m.text.Read(m.textRef, i)
			if e != nil {
				panic(e)
			}
			if e = m.text.Write(ref, i, v); e != nil {
				panic(e)
			}
		}
		if e = m.text.Deallocate(m.textCapacity, m.textRef); e != nil {
			panic(e)
		}
	}
	m.textRef = ref
	m.textCapacity = capacity
	return nil
}
func (m *NaturalManager) textRead(i uint32) byte {
	v, e := m.text.Read(m.textRef, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *NaturalManager) textWrite(i uint32, v byte) {
	if e := m.text.Write(m.textRef, i, v); e != nil {
		panic(e)
	}
}
func (m *NaturalManager) reverseText(start, end uint32) {
	for start < end {
		end--
		if start >= end {
			break
		}
		a, b := m.textRead(start), m.textRead(end)
		m.textWrite(start, b)
		m.textWrite(end, a)
		start++
	}
}
func checkedTextEnd(offset uint32, n uint64) (uint32, error) {
	if n > math.MaxUint32-uint64(offset) {
		return 0, arena.ErrBounds
	}
	return offset + uint32(n), nil
}

// prepareNatural follows source normalize / div_1 / unnormalize by ten. The
// three source local limb buffers have fresh source lifetimes in the embedded
// charged owner. Text remains an explicit output assembly adapter.
func (m *NaturalManager) prepareNatural(a limbView, offset uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	end, e := checkedTextEnd(offset, 11*uint64(a.size))
	if e != nil {
		return 0, e
	}
	if end == offset {
		end, e = checkedTextEnd(offset, 1)
		if e != nil {
			return 0, e
		}
	}
	if e = m.reserveText(end); e != nil {
		return 0, e
	}
	j := offset
	if a.size == 1 {
		value := a.read(0)
		for {
			m.textWrite(j, '0'+byte(value%10))
			j++
			value /= 10
			if value == 0 {
				break
			}
		}
		m.reverseText(offset, j)
		return j, nil
	}
	if e = m.beginFormattingLocals(a.size); e != nil {
		return 0, e
	}
	temp := limbView{runtime: m.runtime, source: &m.formatting.locals[0], size: a.size}
	numer := limbView{runtime: m.runtime, source: &m.formatting.locals[1], size: a.size + 1}
	denom := limbView{runtime: m.runtime, source: &m.formatting.locals[2], size: 1}
	for i := uint32(0); i < a.size; i++ {
		temp.write(i, a.read(i))
	}
	// Ten normalizes by28 bits in the source 32-bit limb path.
	const d = 28
	denom.write(0, uint32(10)<<d)
	for temp.size > 0 && (temp.size > 1 || temp.read(0) != 0) {
		numer.size = temp.size + 1
		numer.source.pos = numer.size
		numer.write(temp.size, temp.read(temp.size-1)>>(32-d))
		for i := temp.size - 1; i > 0; i-- {
			numer.write(i, temp.read(i)<<d|temp.read(i-1)>>(32-d))
		}
		numer.write(0, temp.read(0)<<d)
		divideViewOne(numer, denom.read(0), temp)
		remainder := numer.read(0) >> d
		m.textWrite(j, '0'+byte(remainder))
		j++
		for temp.size > 0 && temp.read(temp.size-1) == 0 {
			temp.size--
			temp.source.pos = temp.size
		}
	}
	// Source writes the terminator before its size_t j--. Empty input
	// endangers the following reversal; retain earlier constructor/actions.
	m.textWrite(j, 0)
	if guard := formattingReversalGuard(j - offset); guard != nil {
		if e = m.finishFormattingLocals(); e != nil {
			return 0, e
		}
		return 0, guard
	}
	m.reverseText(offset, j)
	if e = m.finishFormattingLocals(); e != nil {
		return 0, e
	}
	return j, nil
}
func (m *NaturalManager) publishText(dst []byte, n uint32) (int, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	if uint64(n) > uint64(len(dst)) {
		return int(n), ErrShortBuffer
	}
	for i := uint32(0); i < n; i++ {
		dst[i] = m.textRead(i)
	}
	return int(n), nil
}

// finalString is the host-output ownership boundary: only the escaped Go string
// backing is caller-owned. Numerical/digit scratch is budgeted; no transient
// budget charge misrepresents the escaped string lifetime.
func (m *NaturalManager) finalString(n uint32) (string, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	var output strings.Builder
	output.Grow(int(n))
	for i := uint32(0); i < n; i++ {
		output.WriteByte(m.textRead(i))
	}
	return output.String(), nil
}

// FormatInto excludes the C terminator. It writes exactly n bytes, leaving dst
// unchanged on a short buffer or predictable storage failure.
func (m *NaturalManager) FormatInto(a []uint32, dst []byte) (int, error) {
	if uint64(len(a)) > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	n, e := m.prepareNatural(limbView{external: a, size: uint32(len(a))}, 0)
	if e != nil {
		return 0, e
	}
	return m.publishText(dst, n)
}
func (m *NaturalManager) String(a []uint32) (string, error) {
	if uint64(len(a)) > math.MaxUint32 {
		return "", arena.ErrBounds
	}
	n, e := m.prepareNatural(limbView{external: a, size: uint32(len(a))}, 0)
	if e != nil {
		return "", e
	}
	return m.finalString(n)
}
func (m *IntegerManager) prepareInteger(a *Int, offset uint32) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		value := int64(a.val)
		end, e := checkedTextEnd(offset, 12)
		if e != nil {
			return 0, e
		}
		if e = m.natural.reserveText(end); e != nil {
			return 0, e
		}
		if value < 0 {
			m.natural.textWrite(offset, '-')
			offset++
			value = -value
		}
		j := offset
		for {
			m.natural.textWrite(j, '0'+byte(value%10))
			j++
			value /= 10
			if value == 0 {
				break
			}
		}
		m.natural.reverseText(offset, j)
		return j, nil
	}
	if a.val < 0 {
		end, e := checkedTextEnd(offset, 1)
		if e != nil {
			return 0, e
		}
		if e = m.natural.reserveText(end); e != nil {
			return 0, e
		}
		m.natural.textWrite(offset, '-')
		offset = end
	}
	c := m.cell(a.cell)
	return m.natural.prepareNatural(limbView{allocator: m.allocator, backing: c.backing, size: c.size}, offset)
}
func (m *IntegerManager) FormatInto(a *Int, dst []byte) (int, error) {
	n, e := m.prepareInteger(a, 0)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}
func (m *IntegerManager) String(a *Int) (string, error) {
	n, e := m.prepareInteger(a, 0)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *RationalManager) prepareRational(a *Rational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n, e := m.z.prepareInteger(&a.num, 0)
	if e != nil {
		return 0, e
	}
	if isIntegerRational(a) {
		return n, nil
	}
	end, e := checkedTextEnd(n, 1)
	if e != nil {
		return 0, e
	}
	if e = m.z.natural.reserveText(end); e != nil {
		return 0, e
	}
	m.z.natural.textWrite(n, '/')
	return m.z.prepareInteger(&a.den, end)
}
func (m *RationalManager) FormatInto(a *Rational, dst []byte) (int, error) {
	n, e := m.prepareRational(a)
	if e != nil {
		return 0, e
	}
	return m.z.natural.publishText(dst, n)
}
func (m *RationalManager) String(a *Rational) (string, error) {
	n, e := m.prepareRational(a)
	if e != nil {
		return "", e
	}
	return m.z.natural.finalString(n)
}
