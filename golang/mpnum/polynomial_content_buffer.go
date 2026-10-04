// Source: src/util/buffer.h:29-59,146-151,196-231;
// src/math/polynomial/polynomial.cpp:3485-3488.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Actual call-local sbuffer<unsigned,128>; enclosing scope accounts the cold
// fields. Runtime storage is constructed only on the source expansion path.
type polynomialContentUnsignedBuffer struct {
	runtime                              *memory.Runtime
	inline                               [128]uint32
	raw, pending                         memory.Block
	count, capacity, initialized, copied uint32
	constructed, finalized, freeStarted  bool
	sourceCause                          error
}

func (b *polynomialContentUnsignedBuffer) construct(r *memory.Runtime) {
	*b = polynomialContentUnsignedBuffer{runtime: r, capacity: 128, constructed: true}
}
func (b *polynomialContentUnsignedBuffer) physical(i uint32) (uint32, error) {
	if i >= b.initialized {
		return 0, arena.ErrBounds
	}
	if b.raw == (memory.Block{}) {
		if i >= 128 {
			return 0, arena.ErrBounds
		}
		return b.inline[i], nil
	}
	return b.runtime.ReadUint32(b.raw, uint64(i)*4)
}
func (b *polynomialContentUnsignedBuffer) get(i uint32) (uint32, error) {
	if !b.constructed || b.finalized {
		return 0, arena.ErrClosed
	}
	if modularSourceDebug && i >= b.count {
		return 0, ErrPolynomialAssertion
	}
	return b.physical(i)
}
func (b *polynomialContentUnsignedBuffer) set(i, w uint32) error {
	if modularSourceDebug && i >= b.count {
		return ErrPolynomialAssertion
	}
	if i >= b.initialized {
		return arena.ErrBounds
	}
	if b.raw == (memory.Block{}) {
		if i >= 128 {
			return arena.ErrBounds
		}
		b.inline[i] = w
		return nil
	}
	return b.runtime.WriteUint32(b.raw, uint64(i)*4, w)
}
func (b *polynomialContentUnsignedBuffer) expand() error {
	nc := b.capacity << 1
	var e error
	b.pending, e = b.runtime.Allocate(uint64(nc) * 4)
	if e != nil {
		return e
	}
	for b.copied < b.count {
		w, e := b.physical(b.copied)
		if e != nil {
			return e
		}
		if e = b.runtime.WriteUint32(b.pending, uint64(b.copied)*4, w); e != nil {
			return e
		}
		b.copied++
	}
	if b.raw != (memory.Block{}) {
		b.freeStarted = true
		if e = b.runtime.Deallocate(&b.raw); e != nil {
			return e
		}
	}
	b.raw = b.pending
	b.pending = memory.Block{}
	b.capacity = nc
	b.copied = 0
	b.freeStarted = false
	return nil
}
func (b *polynomialContentUnsignedBuffer) push(w uint32) (err error) {
	if !b.constructed || b.finalized {
		return arena.ErrClosed
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	defer func() {
		if err != nil {
			b.sourceCause = err
		}
	}()
	if b.count >= b.capacity {
		if e := b.expand(); e != nil {
			return e
		}
	}
	if b.raw == (memory.Block{}) {
		if b.count >= 128 {
			return arena.ErrBounds
		}
		b.inline[b.count] = w
	} else if e := b.runtime.WriteUint32(b.raw, uint64(b.count)*4, w); e != nil {
		return e
	}
	b.count++
	if b.count > b.initialized {
		b.initialized = b.count
	}
	return nil
}
func (b *polynomialContentUnsignedBuffer) resize(n, w uint32) error {
	sz := b.count
	if n > sz {
		for i := sz; i < n; i++ {
			if e := b.push(w); e != nil {
				return e
			}
		}
	} else if n < sz {
		for i := n; i < sz; i++ {
			b.count--
		}
	}
	if modularSourceDebug && b.count != n {
		return ErrPolynomialAssertion
	}
	return nil
}
func (b *polynomialContentUnsignedBuffer) reset() { b.count = 0 }
func (b *polynomialContentUnsignedBuffer) close(owning bool) error {
	if !b.constructed || b.finalized && !owning {
		return nil
	}
	// A failed expansion's allocation is not the source published buffer.
	// Its separate owning abort follows completed source destruction.
	if b.freeStarted && !owning {
		return b.sourceCause
	}
	if b.raw != (memory.Block{}) {
		b.freeStarted = true
		if e := b.runtime.Deallocate(&b.raw); e != nil {
			b.sourceCause = e
			return e
		}
	}
	if owning && b.pending != (memory.Block{}) {
		if e := b.runtime.Deallocate(&b.pending); e != nil {
			return e
		}
	}
	b.finalized = true
	return nil
}
