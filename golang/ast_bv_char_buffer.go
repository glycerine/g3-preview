// Source: src/util/buffer.h:25-63,261-268; src/ast/ast_smt2_pp.cpp:185-229.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Actual lexical sbuffer<char> storage. Embedded in the caller's charged value
// frame, with sole inline or Runtime bytes. pending is the exact returned NEW
// allocation before source publication; it is not a second payload copy.
type astBVCharBuffer16 struct {
	inline                  [16]byte
	initialized             uint16
	pos, cap                uint32
	raw, pending            memory.Block
	rawConsumed, sourceDone bool
	sourceCause, ownCause   error
}

func (b *astBVCharBuffer16) capacity() uint32 {
	if b.cap == 0 {
		return 16
	}
	return b.cap
}
func (b *astBVCharBuffer16) read(r *memory.Runtime, i uint32) (byte, error) {
	if b.sourceDone || i >= b.pos {
		return 0, arena.ErrBounds
	}
	if b.cap == 0 {
		if i >= 16 {
			return 0, arena.ErrBounds
		}
		if b.initialized&(uint16(1)<<i) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		return b.inline[i], nil
	}
	return r.Read(b.raw, uint64(i))
}
func (b *astBVCharBuffer16) write(r *memory.Runtime, i uint32, v byte) error {
	if b.sourceDone || i >= b.capacity() {
		return arena.ErrBounds
	}
	if b.cap == 0 {
		b.inline[i] = v
		b.initialized |= uint16(1) << i
		return nil
	}
	return r.Write(b.raw, uint64(i), v)
}
func (b *astBVCharBuffer16) push(r *memory.Runtime, v byte) error {
	if b.sourceDone || b.pending != (memory.Block{}) || b.rawConsumed {
		return arena.ErrInvalidSlot
	}
	if b.pos >= b.capacity() {
		// Literal unsigned capacity shift, followed by actual malloc and ascending
		// memcpy. OLD is deleted before the new source pointer/capacity publication.
		n := b.capacity() << 1
		newBlock, e := r.Allocate(uint64(n))
		if newBlock != (memory.Block{}) {
			b.pending = newBlock
		}
		if e != nil {
			return e
		}
		for i := uint32(0); i < b.pos; i++ {
			x, e := b.read(r, i)
			if e != nil {
				return e
			}
			if e = r.Write(newBlock, uint64(i), x); e != nil {
				return e
			}
		}
		if b.raw != (memory.Block{}) {
			old := b.raw
			e = r.Deallocate(&old)
			if old == (memory.Block{}) {
				b.rawConsumed = true
			}
			if e != nil {
				return e
			}
		}
		b.raw = newBlock
		b.pending = memory.Block{}
		b.rawConsumed = false
		b.cap = n
		if n == 0 {
			return arena.ErrBounds
		} // wrapped source extent cannot hold this write
	}
	if e := b.write(r, b.pos, v); e != nil {
		return e
	}
	b.pos++
	return nil
}
func (b *astBVCharBuffer16) sourceDestroy(r *memory.Runtime) error {
	if b.sourceDone {
		return nil
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.raw != (memory.Block{}) {
		old := b.raw
		e := r.Deallocate(&old)
		if old == (memory.Block{}) {
			b.rawConsumed = true
		}
		if e != nil {
			b.sourceCause = e
			return e
		}
	}
	// Source free_memory never destroys an allocation that did not become m_buffer.
	b.sourceDone = true
	return nil
}
func (b *astBVCharBuffer16) cleanup(r *memory.Runtime) error {
	stop := func(e error) error {
		if e != nil && b.ownCause == nil {
			b.ownCause = e
		}
		return e
	}
	if b.raw != (memory.Block{}) && !b.rawConsumed {
		if e := r.Deallocate(&b.raw); e != nil {
			return stop(e)
		}
		b.rawConsumed = true
	}
	if b.pending != (memory.Block{}) {
		if e := r.Deallocate(&b.pending); e != nil {
			return stop(e)
		}
	}
	return nil
}
