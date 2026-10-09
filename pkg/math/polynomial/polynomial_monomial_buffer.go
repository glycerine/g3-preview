// Source: src/util/buffer.h; src/math/polynomial/polynomial.cpp:3134-3182,7439-7476.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Retained byte-backed buffer used by sparse interpolation and conversion.
type skeletonMonomialBuffer struct {
	initial        [128]Monomial
	raw, pending   util.Block
	size, capacity uint32
}

func skeletonReadMonomial(r *util.Runtime, b util.Block, o uint64) (Monomial, error) {
	lo, e := r.ReadUint32(b, o)
	if e != nil {
		return Monomial{}, e
	}
	hi, e := r.ReadUint32(b, o+4)
	if e != nil {
		return Monomial{}, e
	}
	slot, e := r.ReadUint32(b, o+8)
	if e != nil {
		return Monomial{}, e
	}
	gen, e := r.ReadUint32(b, o+12)
	return Monomial{uint64(lo) | uint64(hi)<<32, slot, gen}, e
}
func skeletonWriteMonomial(r *util.Runtime, b util.Block, o uint64, a Monomial) error {
	for i, v := range [4]uint32{uint32(a.owner), uint32(a.owner >> 32), a.slot, a.generation} {
		if e := r.WriteUint32(b, o+uint64(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}
func (b *skeletonMonomialBuffer) get(m *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	if i >= b.size {
		return Monomial{}, arena.ErrBounds
	}
	if b.raw == (util.Block{}) {
		return b.initial[i], nil
	}
	return skeletonReadMonomial(m.memory, b.raw, uint64(i)*16)
}
func (b *skeletonMonomialBuffer) set(m *MultivariatePolynomialCore, i uint32, a Monomial) error {
	if i >= b.capacity {
		return arena.ErrBounds
	}
	if b.raw == (util.Block{}) {
		b.initial[i] = a
		return nil
	}
	return skeletonWriteMonomial(m.memory, b.raw, uint64(i)*16, a)
}
func (b *skeletonMonomialBuffer) push(m *MultivariatePolynomialCore, a Monomial) error {
	if b.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	if b.size == b.capacity {
		next := b.capacity << 1
		var e error
		b.pending, e = m.memory.Allocate(uint64(next) * 16)
		if e != nil {
			return e
		}
		for i := uint32(0); i < b.size; i++ {
			v, e := b.get(m, i)
			if e != nil {
				return e
			}
			if e = skeletonWriteMonomial(m.memory, b.pending, uint64(i)*16, v); e != nil {
				return e
			}
		}
		if b.raw != (util.Block{}) {
			if e = m.memory.Deallocate(&b.raw); e != nil {
				return e
			}
		}
		b.raw = b.pending
		b.pending = util.Block{}
		b.capacity = next
	}
	if e := b.set(m, b.size, a); e != nil {
		return e
	}
	b.size++
	return nil
}
