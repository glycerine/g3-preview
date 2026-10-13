// Source: src/util/mpn.cpp, Knuth Algorithm D.
package numeral

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// limbView is ephemeral: external buffers are borrowed; numeric blocks stay indexed.
// No view is persisted in a manager or exposes an arena backing slice.
type limbView struct {
	external  []uint32
	allocator *umemory.SmallAllocator
	backing   SourceCellBacking
	size      uint32
}

func (v limbView) read(i uint32) uint32 {
	if i >= v.size {
		panic(arena.ErrBounds)
	}
	if v.allocator != nil {
		n, e := v.backing.ReadWord(v.allocator, 8+4*uint64(i))
		if e != nil {
			panic(e)
		}
		return n
	}
	return v.external[i]
}
func (v limbView) write(i, n uint32) {
	if i >= v.size {
		panic(arena.ErrBounds)
	}
	if v.allocator != nil {
		if e := v.backing.writeWord(v.allocator, 8+4*uint64(i), n); e != nil {
			panic(e)
		}
		return
	}
	v.external[i] = n
}

// Div retains the source branch boolean. Invalid source buffer preconditions
// panic. len(a)>=len(b), the denominator is normalized/nonzero, q has
// len(a)-len(b)+1 limbs, and r has len(b). Neither output overlaps an input
// or the other output. The integer manager stages its own permitted aliases.
// Division and checked verification use reusable native slices. The source
// arithmetic and quotient/remainder publication order are preserved.
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
	// src/util/mpn.cpp:158-168 precedes denominator checks and returns
	// before the common debug verification. Unsigned underflow is retained;
	// only each endangered output access is guarded.
	if na < nb {
		count := uint64(na) - uint64(nb) + 1
		if m.runtime.SizeTBits() == 32 {
			count = uint64(uint32(count))
		}
		for i := uint64(0); i < count; i++ {
			if i >= uint64(q.size) {
				return false, arena.ErrBounds
			}
			q.write(uint32(i), 0)
		}
		for i := uint32(0); i < nb; i++ {
			if i >= r.size {
				return false, arena.ErrBounds
			}
			var v uint32
			if i < na {
				v = a.read(i)
			}
			r.write(i, v)
		}
		return false, nil
	}
	if na == 1 && nb == 1 {
		q.write(0, a.read(0)/b.read(0))
		r.write(0, a.read(0)%b.read(0))
		return false, m.sourceVerifyDivision(a, b, q, r)
	}
	if na == nb && a.read(na-1) < b.read(nb-1) {
		q.write(0, 0)
		for i := uint32(0); i < nb; i++ {
			r.write(i, a.read(i))
		}
		return false, m.sourceVerifyDivision(a, b, q, r)
	}
	if na == math.MaxUint32 {
		return false, arena.ErrBounds
	}
	var d uint
	for b.read(nb-1)<<d&0x80000000 == 0 {
		d++
	}
	if e := m.ResizeDiv(0, na+1); e != nil {
		return false, e
	}
	if e := m.ResizeDiv(1, nb); e != nil {
		return false, e
	}
	u, v := 0, 1
	if d == 0 {
		m.sourceDivWrite(u, na, 0)
		for i := uint32(0); i < na; i++ {
			m.sourceDivWrite(u, i, a.read(i))
		}
		for i := uint32(0); i < nb; i++ {
			m.sourceDivWrite(v, i, b.read(i))
		}
	} else {
		m.sourceDivWrite(u, na, a.read(na-1)>>(32-d))
		for i := na - 1; i > 0; i-- {
			m.sourceDivWrite(u, i, a.read(i)<<d|a.read(i-1)>>(32-d))
		}
		m.sourceDivWrite(u, 0, a.read(0)<<d)
		for i := nb - 1; i > 0; i-- {
			m.sourceDivWrite(v, i, b.read(i)<<d|b.read(i-1)>>(32-d))
		}
		m.sourceDivWrite(v, 0, b.read(0)<<d)
	}
	m.traceDivisionNormalize(m.Division[u][:na+1], m.Division[v][:nb])
	if nb == 1 {
		m.divideViewOne(limbView{external: m.Division[u], size: na + 1}, m.sourceDivRead(v, 0), q)
	} else {
		if e := m.divideMany(na+1, nb, q); e != nil {
			return false, e
		}
	}
	if d == 0 {
		for i := uint32(0); i < nb; i++ {
			r.write(i, m.sourceDivRead(u, i))
		}
	} else {
		mask := uint32(1)<<d - 1
		for i := uint32(0); i+1 < nb; i++ {
			r.write(i, m.sourceDivRead(u, i)>>d|(m.sourceDivRead(u, i+1)&mask)<<(32-d))
		}
		r.write(nb-1, m.sourceDivRead(u, nb-1)>>d)
	}
	return true, m.sourceVerifyDivision(a, b, q, r)
}
func (m *NaturalManager) divideViewOne(a limbView, b uint32, q limbView) {
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
		m.traceDivisionOneStep(j, qh, temp, denom, ms, a, borrow)
	}
}

// divideMany is source Knuth Algorithm D, including estimate correction,
// single-limb multiplication, modular subtraction, and conditional add-back.
func (m *NaturalManager) divideMany(size, n uint32, q limbView) error {
	if n > math.MaxUint32-2 {
		return arena.ErrBounds
	}
	if e := m.ResizeDiv(2, n+1); e != nil {
		return e
	}
	u, v, ms := 0, 1, 2
	count := size - n
	const base = uint64(1) << 32
	for remaining := count; remaining > 0; remaining-- {
		j := remaining - 1
		temp := uint64(m.sourceDivRead(u, j+n))<<32 | uint64(m.sourceDivRead(u, j+n-1))
		qh := temp / uint64(m.sourceDivRead(v, n-1))
		rh := temp % uint64(m.sourceDivRead(v, n-1))
		for {
			if qh < base && qh*uint64(m.sourceDivRead(v, n-2)) <= (rh<<32)+uint64(m.sourceDivRead(u, j+n-2)) {
				break
			}
			qh--
			rh += uint64(m.sourceDivRead(v, n-1))
			if rh >= base {
				break
			}
		}
		if qh >= base {
			panic("mpnum: invalid quotient estimate")
		}
		small := uint32(qh)
		// Source mul(&small,1,v,n,ms), Algorithm M with the zero-digit branch.
		m.sourceDivWrite(ms, 0, 0)
		for i := uint32(0); i < n; i++ {
			digit := m.sourceDivRead(v, i)
			if digit == 0 {
				m.sourceDivWrite(ms, i+1, 0)
				continue
			}
			t := uint64(small)*uint64(digit) + uint64(m.sourceDivRead(ms, i))
			m.sourceDivWrite(ms, i, uint32(t))
			m.sourceDivWrite(ms, i+1, uint32(t>>32))
		}
		var borrow uint32
		for i := uint32(0); i < n+1; i++ {
			a, b := m.sourceDivRead(u, j+i), m.sourceDivRead(ms, i)
			r := a - b
			c1 := r > a
			out := r - borrow
			c2 := out > r
			m.sourceDivWrite(u, j+i, out)
			borrow = 0
			if c1 || c2 {
				borrow = 1
			}
		}
		q.write(j, small)
		if borrow != 0 {
			q.write(j, small-1)
			if e := m.ResizeDiv(3, n+2); e != nil {
				return e
			}
			ab := 3
			var carry uint32
			for i := uint32(0); i < n+1; i++ {
				a := uint32(0)
				if i < n {
					a = m.sourceDivRead(v, i)
				}
				b := m.sourceDivRead(u, j+i)
				r := a + b
				c1 := r < a
				out := r + carry
				c2 := out < r
				m.sourceDivWrite(ab, i, out)
				carry = 0
				if c1 || c2 {
					carry = 1
				}
			}
			m.sourceDivWrite(ab, n+1, carry)
			for i := uint32(0); i < n+1; i++ {
				m.sourceDivWrite(u, j+i, m.sourceDivRead(ab, i))
			}
		}
	}
	return nil
}
