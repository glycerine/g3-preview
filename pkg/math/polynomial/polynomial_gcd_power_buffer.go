// Source: src/math/polynomial/polynomial.cpp:3207-3256,4352-4506;
// src/util/buffer.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Actual sbuffer<power,32>, using the source pair<uint32,uint32> width8.
// No coefficient or monomial mirror supplies the Runtime payload.
type polynomialGCDPowerBuffer struct {
	runtime                             *util.Runtime
	inline                              [32]MonomialPower
	raw, pending                        util.Block
	size, capacity, initialized, copied uint32
	constructed, ended, freeStarted     bool
	cause                               error
}

func (b *polynomialGCDPowerBuffer) construct(r *util.Runtime) {
	*b = polynomialGCDPowerBuffer{runtime: r, capacity: 32, constructed: true}
}
func (b *polynomialGCDPowerBuffer) word(i, field uint32) (uint32, error) {
	if polynomialSourceDebug && i >= b.size {
		return 0, ErrPolynomialAssertion
	}
	if b.raw == (util.Block{}) {
		if i >= b.initialized || i >= 32 {
			return 0, arena.ErrBounds
		}
		if field == 0 {
			return b.inline[i].Var, nil
		}
		return b.inline[i].Degree, nil
	}
	return b.runtime.ReadUint32(b.raw, uint64(i)*8+uint64(field)*4)
}
func (b *polynomialGCDPowerBuffer) setWord(i, field, x uint32) error {
	if polynomialSourceDebug && i >= b.size {
		return ErrPolynomialAssertion
	}
	if b.raw == (util.Block{}) {
		if i >= b.initialized || i >= 32 {
			return arena.ErrBounds
		}
		if field == 0 {
			b.inline[i].Var = x
		} else {
			b.inline[i].Degree = x
		}
		return nil
	}
	return b.runtime.WriteUint32(b.raw, uint64(i)*8+uint64(field)*4, x)
}
func (b *polynomialGCDPowerBuffer) push(x, k uint32) error {
	if !b.constructed || b.ended {
		return arena.ErrClosed
	}
	if b.pending != (util.Block{}) || b.freeStarted {
		return ErrPolynomialPending
	}
	if b.size >= b.capacity {
		capacity := b.capacity * 2
		var e error
		b.pending, e = b.runtime.Allocate(uint64(capacity) * 8)
		if e != nil {
			return e
		}
		for b.copied < b.size {
			x, e := b.word(b.copied, 0)
			if e != nil {
				return e
			}
			k, e := b.word(b.copied, 1)
			if e != nil {
				return e
			}
			if e = b.runtime.WriteUint32(b.pending, uint64(b.copied)*8, x); e != nil {
				return e
			}
			if e = b.runtime.WriteUint32(b.pending, uint64(b.copied)*8+4, k); e != nil {
				return e
			}
			b.copied++
		}
		if b.raw != (util.Block{}) {
			b.freeStarted = true
			if e = b.runtime.Deallocate(&b.raw); e != nil {
				b.cause = e
				return e
			}
		}
		b.raw = b.pending
		b.pending = util.Block{}
		b.capacity = capacity
		b.copied = 0
		b.freeStarted = false
	}
	if b.raw == (util.Block{}) {
		if b.size >= 32 {
			return arena.ErrBounds
		}
		b.inline[b.size] = MonomialPower{x, k}
	} else {
		if e := b.runtime.WriteUint32(b.raw, uint64(b.size)*8, x); e != nil {
			return e
		}
		if e := b.runtime.WriteUint32(b.raw, uint64(b.size)*8+4, k); e != nil {
			return e
		}
	}
	b.size++
	b.initialized = b.size
	return nil
}
func (b *polynomialGCDPowerBuffer) close(owning bool) error { return b.closePhase(owning, owning) }
func (b *polynomialGCDPowerBuffer) closePhase(owning, abort bool) error {
	if !b.constructed || b.ended && !owning {
		return nil
	}
	if b.cause != nil && !owning {
		return b.cause
	}
	if b.raw != (util.Block{}) {
		b.freeStarted = true
		if e := b.runtime.Deallocate(&b.raw); e != nil {
			b.cause = e
			return e
		}
	}
	b.ended = true
	if owning && abort && b.pending != (util.Block{}) {
		if e := b.runtime.Deallocate(&b.pending); e != nil {
			return e
		}
	}
	return nil
}
func polynomialUnsignedReserveFill(c *MultivariatePolynomialCore, v *polynomialVector, n, fill uint32) error {
	size, e := v.size(c)
	if e != nil {
		return e
	}
	if n <= size {
		return nil
	}
	for {
		cap, e := v.capacity(c)
		if e != nil {
			return e
		}
		if cap >= n {
			break
		}
		if e = v.expand(c); e != nil {
			return e
		}
	}
	if e = v.publishSize(c, n); e != nil {
		return e
	}
	for j := size; j < n; j++ {
		if e = c.memory.WriteUint32(v.raw, 8+uint64(j)*4, fill); e != nil {
			return e
		}
		v.initialized = j + 1
	}
	return nil
}
func (m *MultivariatePolynomialManager) variableDegrees(p Polynomial, pws *polynomialGCDPowerBuffer, max bool) error {
	c := m.core
	pws.size = 0
	variables, e := c.monomials.NumVars()
	if e != nil {
		return e
	}
	if e = polynomialUnsignedReserveFill(c, &m.varDegrees, variables, ^uint32(0)); e != nil {
		return e
	}
	size, e := c.readWord(p, 8)
	if e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		term, e := c.monomialCore(p, i)
		if e != nil {
			return e
		}
		count, e := c.monomials.sizeCore(term)
		if e != nil {
			return e
		}
		for j := uint32(0); j < count; j++ {
			x, e := c.monomials.varCore(term, j)
			if e != nil {
				return e
			}
			k, e := c.monomials.degreeAtCore(term, j)
			if e != nil {
				return e
			}
			pos, e := polynomialUnsignedRead(c, &m.varDegrees, x)
			if e != nil {
				return e
			}
			if pos == ^uint32(0) {
				if e = polynomialUnsignedWrite(c, &m.varDegrees, x, pws.size); e != nil {
					return e
				}
				if e = pws.push(x, k); e != nil {
					return e
				}
			} else {
				old, e := pws.word(pos, 1)
				if e != nil {
					return e
				}
				if max && k > old || !max && k < old {
					if e = pws.setWord(pos, 1, k); e != nil {
						return e
					}
				}
			}
		}
	}
	for i := uint32(0); i < pws.size; i++ {
		x, e := pws.word(i, 0)
		if e != nil {
			return e
		}
		if polynomialSourceDebug {
			pos, e := polynomialUnsignedRead(c, &m.varDegrees, x)
			if e != nil {
				return e
			}
			if pos == ^uint32(0) {
				return ErrPolynomialAssertion
			}
		}
		x, e = pws.word(i, 0)
		if e != nil {
			return e
		}
		if e = polynomialUnsignedWrite(c, &m.varDegrees, x, ^uint32(0)); e != nil {
			return e
		}
	}
	if polynomialSourceDebug {
		for i := uint32(0); i < pws.size; i++ {
			for j := i + 1; j < pws.size; j++ {
				x, e := pws.word(i, 0)
				if e != nil {
					return e
				}
				y, e := pws.word(j, 0)
				if e != nil {
					return e
				}
				if x == y {
					return ErrPolynomialAssertion
				}
			}
		}
	}
	return nil
}

func polynomialUnsignedWrite(c *MultivariatePolynomialCore, v *polynomialVector, i, x uint32) error {
	if polynomialSourceDebug {
		n, e := v.size(c)
		if e != nil {
			return e
		}
		if i >= n {
			return ErrPolynomialAssertion
		}
	}
	return c.memory.WriteUint32(v.raw, 8+uint64(i)*4, x)
}

// std::sort adaptation for source power::lt_var: in-place heapsort of the
// actual pair payload. No source comparator reads Degree, and the selected
// host-library comparison/swap schedule is not claimed by this adapter.
func (b *polynomialGCDPowerBuffer) sortVariables() error { return b.sortField(0) }
func (b *polynomialGCDPowerBuffer) sortDegrees() error   { return b.sortField(1) }
func (b *polynomialGCDPowerBuffer) sortField(field uint32) error {
	for end := b.size / 2; end > 0; end-- {
		if e := b.siftField(end-1, b.size, field); e != nil {
			return e
		}
	}
	for end := b.size; end > 1; end-- {
		if e := b.swapPower(0, end-1); e != nil {
			return e
		}
		if e := b.siftField(0, end-1, field); e != nil {
			return e
		}
	}
	return nil
}
func (b *polynomialGCDPowerBuffer) swapPower(i, j uint32) error {
	if i == j {
		return nil
	}
	x, e := b.word(i, 0)
	if e != nil {
		return e
	}
	k, e := b.word(i, 1)
	if e != nil {
		return e
	}
	y, e := b.word(j, 0)
	if e != nil {
		return e
	}
	l, e := b.word(j, 1)
	if e != nil {
		return e
	}
	if e = b.setWord(i, 0, y); e != nil {
		return e
	}
	if e = b.setWord(i, 1, l); e != nil {
		return e
	}
	if e = b.setWord(j, 0, x); e != nil {
		return e
	}
	return b.setWord(j, 1, k)
}
func (b *polynomialGCDPowerBuffer) siftField(root, size, field uint32) error {
	for root < size/2 {
		child := root*2 + 1
		if child+1 < size {
			x, e := b.word(child, field)
			if e != nil {
				return e
			}
			y, e := b.word(child+1, field)
			if e != nil {
				return e
			}
			if x < y {
				child++
			}
		}
		x, e := b.word(root, field)
		if e != nil {
			return e
		}
		y, e := b.word(child, field)
		if e != nil {
			return e
		}
		if x >= y {
			return nil
		}
		if e = b.swapPower(root, child); e != nil {
			return e
		}
		root = child
	}
	return nil
}
