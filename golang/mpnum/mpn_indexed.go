// Copyright (c) 2011 Microsoft Corporation
// Source: src/util/mpn.cpp, Knuth Algorithm D.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

// limbView is ephemeral: external buffers are borrowed; Words stays indexed.
// No view is persisted in a manager or exposes an arena backing slice.
type limbView struct {
	external  []uint32
	allocator *memory.SmallAllocator
	backing   sourceCellBacking
	words     *arena.Words
	ref       arena.VectorRef
	size      uint32
}

func (v limbView) read(i uint32) uint32 {
	if v.allocator != nil {
		n, e := v.backing.readWord(v.allocator, 8+4*uint64(i))
		if e != nil {
			panic(e)
		}
		return n
	}
	if v.words == nil {
		return v.external[i]
	}
	n, e := v.words.Read(v.ref, i)
	if e != nil {
		panic(e)
	}
	return n
}
func (v limbView) write(i, n uint32) {
	if v.allocator != nil {
		if e := v.backing.writeWord(v.allocator, 8+4*uint64(i), n); e != nil {
			panic(e)
		}
		return
	}
	if v.words == nil {
		v.external[i] = n
		return
	}
	if e := v.words.Write(v.ref, i, n); e != nil {
		panic(e)
	}
}

// Div retains the source branch boolean. Invalid source buffer preconditions
// panic. len(a)>=len(b), the denominator is normalized/nonzero, q has
// len(a)-len(b)+1 limbs, and r has len(b). Neither output overlaps an input
// or the other output. The integer manager stages its own permitted aliases.
// Predictable scratch failures leave both borrowed outputs unchanged.
func (m *NaturalManager) Div(a, b, q, r []uint32) (bool, error) {
	if len(b) == 0 || len(a) < len(b) || b[len(b)-1] == 0 || len(q) != len(a)-len(b)+1 || len(r) != len(b) {
		panic("mpnum: invalid division operands or buffers")
	}
	if uint64(len(a)) >= math.MaxUint32 {
		return false, arena.ErrBounds
	}
	return m.divide(limbView{external: a, size: uint32(len(a))}, limbView{external: b, size: uint32(len(b))}, limbView{external: q, size: uint32(len(q))}, limbView{external: r, size: uint32(len(r))})
}

// divide is shared source Algorithm D over concrete ephemeral limb views.
func (m *NaturalManager) divide(a, b, q, r limbView) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	na, nb := a.size, b.size
	if na == 1 && nb == 1 {
		q.write(0, a.read(0)/b.read(0))
		r.write(0, a.read(0)%b.read(0))
		return false, nil
	}
	if na == nb && a.read(na-1) < b.read(nb-1) {
		q.write(0, 0)
		for i := uint32(0); i < nb; i++ {
			r.write(i, a.read(i))
		}
		return false, nil
	}
	if na == math.MaxUint32 {
		return false, arena.ErrBounds
	}
	var d uint
	for b.read(nb-1)<<d&0x80000000 == 0 {
		d++
	}
	if e := m.resizeDiv(0, na+1); e != nil {
		return false, e
	}
	if e := m.resizeDiv(1, nb); e != nil {
		return false, e
	}
	if nb > 1 {
		if nb > math.MaxUint32-2 {
			return false, arena.ErrBounds
		}
		if e := m.resizeDiv(2, nb+1); e != nil {
			return false, e
		}
		// Reserve possible add-back before borrowed outputs publish. This may
		// grow ab on a path that never uses it; arithmetic remains one source
		// Algorithm D pass. The storage-profile adaptation is documented.
		if e := m.resizeDiv(3, nb+2); e != nil {
			return false, e
		}
	}
	u, v := m.divScratch[0], m.divScratch[1]
	if d == 0 {
		m.divWrite(u, na, 0)
		for i := uint32(0); i < na; i++ {
			m.divWrite(u, i, a.read(i))
		}
		for i := uint32(0); i < nb; i++ {
			m.divWrite(v, i, b.read(i))
		}
	} else {
		m.divWrite(u, na, a.read(na-1)>>(32-d))
		for i := na - 1; i > 0; i-- {
			m.divWrite(u, i, a.read(i)<<d|a.read(i-1)>>(32-d))
		}
		m.divWrite(u, 0, a.read(0)<<d)
		for i := nb - 1; i > 0; i-- {
			m.divWrite(v, i, b.read(i)<<d|b.read(i-1)>>(32-d))
		}
		m.divWrite(v, 0, b.read(0)<<d)
	}
	if nb == 1 {
		divideViewOne(limbView{words: m.words, ref: u, size: na + 1}, m.divRead(v, 0), q)
	} else {
		if e := m.divideMany(na+1, nb, q); e != nil {
			return false, e
		}
	}
	if d == 0 {
		for i := uint32(0); i < nb; i++ {
			r.write(i, m.divRead(u, i))
		}
	} else {
		mask := uint32(1)<<d - 1
		for i := uint32(0); i+1 < nb; i++ {
			r.write(i, m.divRead(u, i)>>d|(m.divRead(u, i+1)&mask)<<(32-d))
		}
		r.write(nb-1, m.divRead(u, nb-1)>>d)
	}
	return true, nil
}
func divideViewOne(a limbView, b uint32, q limbView) {
	denom := uint64(b)
	for j := a.size - 1; j > 0; j-- {
		temp := uint64(a.read(j))<<32 | uint64(a.read(j-1))
		qh := temp / denom
		if qh >= uint64(1)<<32 {
			panic("mpnum: unnormalized single-limb quotient")
		}
		ms := temp - qh*denom
		borrow := ms > temp
		a.write(j-1, uint32(ms))
		a.write(j, uint32(ms>>32))
		q.write(j-1, uint32(qh))
		if borrow {
			q.write(j-1, uint32(qh)-1)
			a.write(j, a.read(j-1)+b)
		}
	}
}

// divideMany is source Knuth Algorithm D, including estimate correction,
// single-limb multiplication, modular subtraction, and conditional add-back.
func (m *NaturalManager) divideMany(size, n uint32, q limbView) error {
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
		q.write(j, small)
		if borrow != 0 {
			q.write(j, small-1)
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
