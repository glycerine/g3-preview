// Source: src/util/hashtable.h:45-64,139-620; src/util/hash.h:58-63,217-229;
// src/sat/sat_types.h:90-109; src/sat/sat_solver.cpp:3125-3136.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Nikolaj Bjorner.
package g3

import "github.com/glycerine/g3/golang/memory"

type satMutexPairSet struct {
	runtime        *memory.Runtime
	raw, pending   memory.Block
	capacity, size uint32
}

func satMutexPairHash(a, b uint32) uint32 {
	b -= a
	b ^= a << 8
	a -= b
	b ^= a << 16
	b -= a
	b ^= a << 10
	return b
}
func (s *satMutexPairSet) init(r *memory.Runtime) error {
	s.runtime = r
	s.capacity = 8
	return s.allocate(8)
}
func (s *satMutexPairSet) allocate(n uint32) error {
	h, e := s.runtime.Allocate(uint64(n) * 16)
	s.pending = h
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if e = s.runtime.WriteUint32(h, uint64(i)*16+4, 0); e != nil {
			return e
		}
		for _, off := range [2]uint64{8, 12} {
			if e = s.runtime.WriteUint32(h, uint64(i)*16+off, 0xfffffffe); e != nil {
				return e
			}
		}
	}
	if s.raw == (memory.Block{}) {
		s.raw, s.pending = h, memory.Block{}
	}
	return nil
}
func (s *satMutexPairSet) equal(b memory.Block, i, a, c uint32) (bool, error) {
	v, e := s.runtime.ReadUint32(b, uint64(i)*16+8)
	if e != nil || v != a {
		return false, e
	}
	v, e = s.runtime.ReadUint32(b, uint64(i)*16+12)
	return v == c, e
}
func (s *satMutexPairSet) contains(a, b uint32) (bool, error) {
	hash := satMutexPairHash(a, b)
	start := hash & (s.capacity - 1)
	for segment := 0; segment < 2; segment++ {
		begin, end := start, s.capacity
		if segment == 1 {
			begin, end = 0, start
		}
		for i := begin; i != end; i++ {
			st, e := s.runtime.ReadUint32(s.raw, uint64(i)*16+4)
			if e != nil {
				return false, e
			}
			if st == 0 {
				return false, nil
			}
			if st == 2 {
				h, e := s.runtime.ReadUint32(s.raw, uint64(i)*16)
				if e != nil {
					return false, e
				}
				if h == hash {
					yes, e := s.equal(s.raw, i, a, b)
					if e != nil || yes {
						return yes, e
					}
				}
			}
		}
	}
	return false, nil
}
func (s *satMutexPairSet) grow() error {
	n := s.capacity << 1
	if e := s.allocate(n); e != nil {
		return e
	}
	if satStorageSourceDebug && n < s.capacity {
		return ErrSATAssertion
	}
	for i := uint32(0); i < s.capacity; i++ {
		st, e := s.runtime.ReadUint32(s.raw, uint64(i)*16+4)
		if e != nil {
			return e
		}
		if st != 2 {
			continue
		}
		h, e := s.runtime.ReadUint32(s.raw, uint64(i)*16)
		if e != nil {
			return e
		}
		begin := h & (n - 1)
		found := false
		for segment := 0; segment < 2 && !found; segment++ {
			start, end := begin, n
			if segment == 1 {
				start, end = 0, begin
			}
			for k := start; k != end; k++ {
				state, e := s.runtime.ReadUint32(s.pending, uint64(k)*16+4)
				if e != nil {
					return e
				}
				if satStorageSourceDebug && state == 1 {
					return ErrSATAssertion
				}
				if state != 0 {
					continue
				}
				for off := uint64(0); off < 16; off += 4 {
					v, e := s.runtime.ReadUint32(s.raw, uint64(i)*16+off)
					if e != nil {
						return e
					}
					if e = s.runtime.WriteUint32(s.pending, uint64(k)*16+off, v); e != nil {
						return e
					}
				}
				found = true
				break
			}
		}
		if !found {
			return ErrSATAssertion
		}
	}
	if e := s.runtime.Deallocate(&s.raw); e != nil {
		return e
	}
	s.raw, s.pending = s.pending, memory.Block{}
	s.capacity = n
	return nil
}
func (s *satMutexPairSet) insert(a, b uint32) error {
	if s.size<<2 > s.capacity*3 {
		if e := s.grow(); e != nil {
			return e
		}
	}
	hash := satMutexPairHash(a, b)
	for j := uint32(0); j < s.capacity; j++ {
		i := ((hash & (s.capacity - 1)) + j) & (s.capacity - 1)
		st, e := s.runtime.ReadUint32(s.raw, uint64(i)*16+4)
		if e != nil {
			return e
		}
		if st == 2 {
			h, e := s.runtime.ReadUint32(s.raw, uint64(i)*16)
			if e != nil {
				return e
			}
			if h == hash {
				yes, e := s.equal(s.raw, i, a, b)
				if e != nil {
					return e
				}
				if yes {
					return nil
				}
			}
			continue
		}
		if e = s.runtime.WriteUint32(s.raw, uint64(i)*16+8, a); e != nil {
			return e
		}
		if e = s.runtime.WriteUint32(s.raw, uint64(i)*16+12, b); e != nil {
			return e
		}
		if e = s.runtime.WriteUint32(s.raw, uint64(i)*16+4, 2); e != nil {
			return e
		}
		if e = s.runtime.WriteUint32(s.raw, uint64(i)*16, hash); e != nil {
			return e
		}
		s.size++
		return nil
	}
	return ErrSATAssertion
}
func (s *satMutexPairSet) close() error {
	for _, b := range []*memory.Block{&s.raw, &s.pending} {
		if *b != (memory.Block{}) {
			if e := s.runtime.Deallocate(b); e != nil {
				return e
			}
		}
	}
	return nil
}
