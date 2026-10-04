// Licensed under the MIT license.
// Concrete sbuffer<uint32,1024> storage from util/buffer.h and util/mpn.h.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// The scalar pending block retains a completed new allocation when copying or
// the old-buffer free fails. It is explicit Go cleanup, not C++ recovery.
// Inline arrays and initialized masks are charged as part of the actual owner.
type sourceLimbBuffer1024 struct {
	inline       [1024]uint32
	initialized  [16]uint64
	pos, cap     uint32
	raw, pending memory.Block
}

func (s *sourceLimbBuffer1024) capacity() uint32 {
	if s.cap == 0 {
		return 1024
	}
	return s.cap
}
func (s *sourceLimbBuffer1024) size() uint32 { return s.pos }
func (s *sourceLimbBuffer1024) read(r *memory.Runtime, i uint32) (uint32, error) {
	if i >= s.pos {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if i >= 1024 {
			return 0, arena.ErrBounds
		}
		if s.initialized[i/64]&(uint64(1)<<(i%64)) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		return s.inline[i], nil
	}
	return r.ReadUint32(s.raw, uint64(i)*4)
}
func (s *sourceLimbBuffer1024) write(r *memory.Runtime, i, n uint32) error {
	if i >= s.capacity() {
		return arena.ErrBounds
	}
	if s.cap == 0 {
		s.inline[i] = n
		s.initialized[i/64] |= uint64(1) << (i % 64)
		return nil
	}
	return r.WriteUint32(s.raw, uint64(i)*4, n)
}
func (s *sourceLimbBuffer1024) expand(r *memory.Runtime) error {
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
	block, e := r.Allocate(request)
	if block != (memory.Block{}) {
		s.pending = block
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
		if e = r.Deallocate(&s.raw); e != nil {
			return e
		}
	}
	s.raw = block
	s.pending = memory.Block{}
	s.cap = n
	// cap zero after wrap cannot select inline storage: the endangered write
	// below checks actual allocation size while this pending object remains owned.
	if n == 0 {
		s.pending = s.raw
		s.raw = memory.Block{}
		return arena.ErrBounds
	}
	return nil
}
func (s *sourceLimbBuffer1024) push(r *memory.Runtime, n uint32) error {
	if s.pending != (memory.Block{}) {
		return arena.ErrInvalidSlot
	}
	if s.pos >= s.capacity() {
		if e := s.expand(r); e != nil {
			return e
		}
	}
	if e := s.write(r, s.pos, n); e != nil {
		return e
	}
	s.pos++
	return nil
}
func (s *sourceLimbBuffer1024) resize(r *memory.Runtime, n uint32) error {
	if n < s.pos {
		s.pos = n
		return nil
	}
	for s.pos < n {
		if e := s.push(r, 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *sourceLimbBuffer1024) reset() { s.pos = 0 }
func (s *sourceLimbBuffer1024) finalize(r *memory.Runtime) error {
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
	s.initialized = [16]uint64{}
	return nil
}
