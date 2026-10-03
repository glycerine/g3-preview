// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.h::mul, src/util/mpz.cpp::big_mul,
// src/util/mpn.cpp::mul (Copyright (c) 2011 Microsoft Corporation).
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

// Mul translates the source small-int path and big multiplication. Exact
// destination/input aliasing is supported through the manager's result scratch.
func (m *IntegerManager) Mul(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.IsSmall() && b.IsSmall() {
		return m.SetInt64(c, int64(a.val)*int64(b.val))
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	size := uint64(m.cell(ca).size) + uint64(m.cell(cb).size)
	if size > math.MaxUint32 {
		return arena.ErrBounds
	}
	if e := m.ensureTmp(0, uint32(size)); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(0, uint32(size)); e != nil {
		return e
	}
	m.multiplyCells(ca, cb, m.tmp[0])
	sign := int32(1)
	if sa != sb {
		sign = -1
	}
	return m.publishTmp(c, sign, uint32(size))
}

// multiplyCells is indexed Knuth Algorithm M. Destination must not alias an
// input; public Mul stages it in source-equivalent owner scratch.
func (m *IntegerManager) multiplyCells(a, b, c numericCellRef) {
	na, nb := m.cell(a).size, m.cell(b).size
	for i := uint32(0); i < na; i++ {
		m.writeDigit(c, i, 0)
	}
	for j := uint32(0); j < nb; j++ {
		v := m.digit(b, j)
		if v == 0 {
			m.writeDigit(c, j+na, 0)
			continue
		}
		var k uint32
		for i := uint32(0); i < na; i++ {
			t := uint64(m.digit(a, i))*uint64(v) + uint64(m.digit(c, i+j)) + uint64(k)
			m.writeDigit(c, i+j, uint32(t))
			k = uint32(t >> 32)
		}
		m.writeDigit(c, j+na, k)
	}
}
