// Licensed under the MIT license.
// Concrete sbuffer<uint32,16> storage from util/buffer.h and util/mpn.h.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// The scalar pending block retains a completed new allocation when copying or
// the old-buffer free fails. It is explicit Go cleanup, not C++ recovery.
// Inline arrays and initialized masks are charged as part of the actual owner.
type sourceLimbBuffer struct {
	inline       [16]uint32
	initialized  uint16
	pos, cap     uint32
	raw, pending memory.Block
}

func (s *sourceLimbBuffer) capacity() uint32 {
	if s.cap == 0 {
		return 16
	}
	return s.cap
}
func (s *sourceLimbBuffer) size() uint32 { return s.pos }
func (s *sourceLimbBuffer) read(r *memory.Runtime, i uint32) (uint32, error) {
	if i >= s.pos {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if i >= 16 {
			return 0, arena.ErrBounds
		}
		if s.initialized&(uint16(1)<<i) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		return s.inline[i], nil
	}
	return r.ReadUint32(s.raw, uint64(i)*4)
}
func (s *sourceLimbBuffer) write(r *memory.Runtime, i, n uint32) error {
	if i >= s.capacity() {
		return arena.ErrBounds
	}
	if s.cap == 0 {
		s.inline[i] = n
		s.initialized |= uint16(1) << i
		return nil
	}
	return r.WriteUint32(s.raw, uint64(i)*4, n)
}
func (s *sourceLimbBuffer) expand(r *memory.Runtime) error { return s.expandWithPNReceipt(r, nil) }
func (s *sourceLimbBuffer) expandWithPNReceipt(r *memory.Runtime, receipt *naturalPNBufferReceipt) error {
	if s.pending != (memory.Block{}) {
		return arena.ErrInvalidSlot
	}
	// Source unsigned doubling precedes the real size_t allocation. Wrapped
	// capacities are guarded only at the endangered copy/write, not before it.
	n := s.capacity() << 1
	request := uint64(n) * 4
	if r.SizeTBits() == 32 {
		request = uint64(uint32(request))
	}
	if receipt != nil && receipt.context != nil && receipt.bound(true) {
		if e := receipt.context.factorHistoryContext.prepareFactorFrontendConstantHistory(receipt.context.owner.manager); e != nil {
			return e
		}
		if e := receipt.context.factorHistoryContext.prepareFactorHistory(receipt.context.owner.manager); e != nil {
			return e
		}
	}

	if receipt != nil && receipt.roundContext.mpfRound != nil {
		if !receipt.bound(true) {
			return memory.ErrRuntimeOwner
		}
		if e := receipt.roundContext.prepareRoundHistory(receipt.roundContext.mpfRound.manager.integer); e != nil {
			return e
		}
	}
	block, e := r.Allocate(request)
	if block != (memory.Block{}) {
		s.pending = block
		if receipt != nil {
			receipt.acquire(s, block)
		}
	}
	if e != nil {
		return e
	}
	for i := uint32(0); i < s.pos; i++ {
		v, e := s.read(r, i)
		if e != nil {
			return e
		}
		if e = r.WriteUint32(block, uint64(i)*4, v); e != nil {
			return e
		}
	}
	if s.raw != (memory.Block{}) {
		old := s.raw
		e = r.Deallocate(&s.raw)
		if receipt != nil && receipt.private && s.raw == (memory.Block{}) && receipt.published == old {
			receipt.published = memory.Block{}
		}
		if e != nil {
			return e
		}
	}
	s.raw = block
	s.pending = memory.Block{}
	if receipt != nil {
		receipt.publish(s, block)
	}
	s.cap = n
	// cap zero after wrap cannot select inline storage: the endangered write
	// below checks actual allocation size while this pending object remains owned.
	if n == 0 {
		s.pending = s.raw
		if receipt != nil {
			if receipt.private && receipt.published == s.pending {
				receipt.published = memory.Block{}
			}
			receipt.acquire(s, s.pending)
		}
		s.raw = memory.Block{}
		return arena.ErrBounds
	}
	return nil
}
func (s *sourceLimbBuffer) push(r *memory.Runtime, n uint32) error {
	return s.pushWithPNReceipt(r, n, nil)
}
func (s *sourceLimbBuffer) pushWithPNReceipt(r *memory.Runtime, n uint32, receipt *naturalPNBufferReceipt) error {
	if s.pending != (memory.Block{}) {
		return arena.ErrInvalidSlot
	}
	if s.pos >= s.capacity() {
		if e := s.expandWithPNReceipt(r, receipt); e != nil {
			return e
		}
	}
	if e := s.write(r, s.pos, n); e != nil {
		return e
	}
	s.pos++
	return nil
}
func (s *sourceLimbBuffer) resize(r *memory.Runtime, n uint32) error {
	return s.resizeWithPNReceipt(r, n, nil)
}
func (s *sourceLimbBuffer) resizeWithPNReceipt(r *memory.Runtime, n uint32, receipt *naturalPNBufferReceipt) error {
	if n < s.pos {
		s.pos = n
		return nil
	}
	for s.pos < n {
		if e := s.pushWithPNReceipt(r, 0, receipt); e != nil {
			return e
		}
	}
	return nil
}
func (s *sourceLimbBuffer) reset() { s.pos = 0 }
func (s *sourceLimbBuffer) finalize(r *memory.Runtime) error {
	if s.pending != (memory.Block{}) {
		if e := r.Deallocate(&s.pending); e != nil {
			return e
		}
	}
	if s.raw != (memory.Block{}) {
		if e := r.Deallocate(&s.raw); e != nil {
			return e
		}
	}
	s.cap = 0
	s.pos = 0
	s.initialized = 0
	return nil
}
