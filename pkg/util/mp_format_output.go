// Source lexical sbuffer<char,1024> from util/buffer.h and mpz.cpp display.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

type sourceCharBuffer struct {
	inline            [1024]byte
	initialized       [16]uint64
	pos, cap          uint32
	Raw, PendingField Block
}

func (s *sourceCharBuffer) capacity() uint32 {
	if s.cap == 0 {
		return 1024
	}
	return s.cap
}
func (s *sourceCharBuffer) size() uint32 { return s.pos }
func (s *sourceCharBuffer) read(r *Runtime, i uint32) (byte, error) {
	if i >= s.pos {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if i >= 1024 {
			return 0, arena.ErrBounds
		}
		if s.initialized[i/64]&(uint64(1)<<(i%64)) == 0 {
			return 0, ErrRuntimeUninitialized
		}
		return s.inline[i], nil
	}
	return r.Read(s.Raw, uint64(i))
}
func (s *sourceCharBuffer) write(r *Runtime, i uint32, n byte) error {
	if i >= s.capacity() {
		return arena.ErrBounds
	}
	if s.cap == 0 {
		s.inline[i] = n
		s.initialized[i/64] |= uint64(1) << (i % 64)
		return nil
	}
	return r.Write(s.Raw, uint64(i), n)
}
func (s *sourceCharBuffer) expand(r *Runtime) error {
	if s.PendingField != (Block{}) {
		return arena.ErrInvalidSlot
	}
	n := s.capacity() << 1
	block, e := r.Allocate(uint64(n))
	if block != (Block{}) {
		s.PendingField = block
	}
	if e != nil {
		return e
	}
	for i := uint32(0); i < s.pos; i++ {
		v, e := s.read(r, i)
		if e != nil {
			return e
		}
		if e = r.Write(block, uint64(i), v); e != nil {
			return e
		}
	}
	if s.Raw != (Block{}) {
		if e = r.Deallocate(&s.Raw); e != nil {
			return e
		}
	}
	s.Raw = block
	s.PendingField = Block{}
	s.cap = n
	if n == 0 {
		s.PendingField = s.Raw
		s.Raw = Block{}
		return arena.ErrBounds
	}
	return nil
}
func (s *sourceCharBuffer) push(r *Runtime, n byte) error {
	if s.PendingField != (Block{}) {
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
func (s *sourceCharBuffer) resize(r *Runtime, n uint32) error {
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
func (s *sourceCharBuffer) finalize(r *Runtime) error {
	if s.PendingField != (Block{}) {
		if e := r.Deallocate(&s.PendingField); e != nil {
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
	s.initialized = [16]uint64{}
	return nil
}

// Cptr access is bounded by actual physical storage, not logical constructor
// size. Source mpn multi-limb conversion deliberately ignores lbuf.
func (s *sourceCharBuffer) readPhysical(r *Runtime, i uint64) (byte, error) {
	if i >= uint64(s.capacity()) {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if s.initialized[i/64]&(uint64(1)<<(i%64)) == 0 {
			return 0, ErrRuntimeUninitialized
		}
		return s.inline[int(i)], nil
	}
	return r.Read(s.Raw, i)
}
func (s *sourceCharBuffer) writePhysical(r *Runtime, i uint64, n byte) error {
	if i >= uint64(s.capacity()) {
		return arena.ErrBounds
	}
	return s.write(r, uint32(i), n)
}
