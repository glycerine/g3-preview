// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
// Source lexical sbuffer<char,1024> from util/buffer.h and mpz.cpp display.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

type sourceCharBuffer struct {
	inline       [1024]byte
	initialized  [16]uint64
	pos, cap     uint32
	raw, pending memory.Block
}

func (s *sourceCharBuffer) capacity() uint32 {
	if s.cap == 0 {
		return 1024
	}
	return s.cap
}
func (s *sourceCharBuffer) size() uint32 { return s.pos }
func (s *sourceCharBuffer) read(r *memory.Runtime, i uint32) (byte, error) {
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
	return r.Read(s.raw, uint64(i))
}
func (s *sourceCharBuffer) write(r *memory.Runtime, i uint32, n byte) error {
	if i >= s.capacity() {
		return arena.ErrBounds
	}
	if s.cap == 0 {
		s.inline[i] = n
		s.initialized[i/64] |= uint64(1) << (i % 64)
		return nil
	}
	return r.Write(s.raw, uint64(i), n)
}
func (s *sourceCharBuffer) expand(r *memory.Runtime) error {
	if s.pending != (memory.Block{}) {
		return arena.ErrInvalidSlot
	}
	n := s.capacity() << 1
	block, e := r.Allocate(uint64(n))
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
		if e = r.Write(block, uint64(i), v); e != nil {
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
	if n == 0 {
		s.pending = s.raw
		s.raw = memory.Block{}
		return arena.ErrBounds
	}
	return nil
}
func (s *sourceCharBuffer) push(r *memory.Runtime, n byte) error {
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
func (s *sourceCharBuffer) resize(r *memory.Runtime, n uint32) error {
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
func (s *sourceCharBuffer) finalize(r *memory.Runtime) error {
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

// Cptr access is bounded by actual physical storage, not logical constructor
// size. Source mpn multi-limb conversion deliberately ignores lbuf.
func (s *sourceCharBuffer) readPhysical(r *memory.Runtime, i uint64) (byte, error) {
	if i >= uint64(s.capacity()) {
		return 0, arena.ErrBounds
	}
	if s.cap == 0 {
		if s.initialized[i/64]&(uint64(1)<<(i%64)) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		return s.inline[int(i)], nil
	}
	return r.Read(s.raw, i)
}
func (s *sourceCharBuffer) writePhysical(r *memory.Runtime, i uint64, n byte) error {
	if i >= uint64(s.capacity()) {
		return arena.ErrBounds
	}
	return s.write(r, uint32(i), n)
}
