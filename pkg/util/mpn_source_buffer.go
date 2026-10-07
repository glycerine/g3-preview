// Concrete sbuffer<uint32,16> storage from util/buffer.h and util/mpn.h.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// The scalar pending block retains a completed new allocation when copying or
// the old-buffer free fails. It is explicit Go cleanup, not C++ recovery.
// Inline arrays and initialized masks are charged as part of the actual owner.

type SourceLimbBuffer struct {
	inline       [16]uint32
	initialized  uint16
	pos, cap     uint32
	Raw, Pending Block
}

func (s *SourceLimbBuffer) capacity() uint32 {
	if s.cap == 0 {
		return 16
	}
	return s.cap
}
func (s *SourceLimbBuffer) size() uint32 { return s.pos }
func (s *SourceLimbBuffer) read(r *Runtime, i uint32) (uint32, error) {
	if i >= s.pos {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if i >= 16 {
			return 0, arena.ErrBounds
		}
		if s.initialized&(uint16(1)<<i) == 0 {
			return 0, ErrRuntimeUninitialized
		}
		return s.inline[i], nil
	}
	return r.ReadUint32(s.Raw, uint64(i)*4)
}
func (s *SourceLimbBuffer) write(r *Runtime, i, n uint32) error {
	if i >= s.capacity() {
		return arena.ErrBounds
	}
	if s.cap == 0 {
		s.inline[i] = n
		s.initialized |= uint16(1) << i
		return nil
	}
	return r.WriteUint32(s.Raw, uint64(i)*4, n)
}
func (s *SourceLimbBuffer) expand(r *Runtime) error { return s.expandWithPNReceipt(r, nil) }
func (s *SourceLimbBuffer) expandWithPNReceipt(r *Runtime, receipt *naturalPNBufferReceipt) error {
	if s.Pending != (Block{}) {
		return arena.ErrInvalidSlot
	}
	// Source unsigned doubling precedes the real size_t allocation. Wrapped
	// capacities are guarded only at the endangered copy/write, not before it.
	n := s.capacity() << 1
	request := uint64(n) * 4
	if r.SizeTBits() == 32 {
		request = uint64(uint32(request))
	}
	if receipt != nil && receipt.context != nil && receipt.Bound(true) {
		if e := receipt.context.factorHistoryContext.prepareCaller(receipt.context.owner.manager, "constant-history"); e != nil {
			return e
		}
		if e := receipt.context.factorHistoryContext.prepareCaller(receipt.context.owner.manager, "factor-history"); e != nil {
			return e
		}
	}

	if receipt != nil && receipt.rationalSetContext.mpfRationalSet != nil {
		if !receipt.Bound(true) {
			return ErrRuntimeOwner
		}
		if e := receipt.rationalSetContext.prepareRationalSetHistory(receipt.rationalSetContext.mpfRationalSet.manager.integer); e != nil {
			return e
		}
	}
	if receipt != nil && receipt.roundContext.mpfRound != nil {
		if !receipt.Bound(true) {
			return ErrRuntimeOwner
		}
		if e := receipt.roundContext.prepareRoundHistory(receipt.roundContext.mpfRound.Manager.integer); e != nil {
			return e
		}
	}
	block, e := r.Allocate(request)
	if block != (Block{}) {
		s.Pending = block
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
	if s.Raw != (Block{}) {
		old := s.Raw
		e = r.Deallocate(&s.Raw)
		if receipt != nil && receipt.private && s.Raw == (Block{}) && receipt.Published == old {
			receipt.Published = Block{}
		}
		if e != nil {
			return e
		}
	}
	s.Raw = block
	s.Pending = Block{}
	if receipt != nil {
		receipt.publish(s, block)
	}
	s.cap = n
	// cap zero after wrap cannot select inline storage: the endangered write
	// below checks actual allocation size while this pending object remains owned.
	if n == 0 {
		s.Pending = s.Raw
		if receipt != nil {
			if receipt.private && receipt.Published == s.Pending {
				receipt.Published = Block{}
			}
			receipt.acquire(s, s.Pending)
		}
		s.Raw = Block{}
		return arena.ErrBounds
	}
	return nil
}
func (s *SourceLimbBuffer) push(r *Runtime, n uint32) error {
	return s.pushWithPNReceipt(r, n, nil)
}
func (s *SourceLimbBuffer) pushWithPNReceipt(r *Runtime, n uint32, receipt *naturalPNBufferReceipt) error {
	if s.Pending != (Block{}) {
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
func (s *SourceLimbBuffer) resize(r *Runtime, n uint32) error {
	return s.resizeWithPNReceipt(r, n, nil)
}
func (s *SourceLimbBuffer) resizeWithPNReceipt(r *Runtime, n uint32, receipt *naturalPNBufferReceipt) error {
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
func (s *SourceLimbBuffer) reset() { s.pos = 0 }
func (s *SourceLimbBuffer) Finalize(r *Runtime) error {
	if s.Pending != (Block{}) {
		if e := r.Deallocate(&s.Pending); e != nil {
			return e
		}
	}
	if s.Raw != (Block{}) {
		if e := r.Deallocate(&s.Raw); e != nil {
			return e
		}
	}
	s.cap = 0
	s.pos = 0
	s.initialized = 0
	return nil
}
