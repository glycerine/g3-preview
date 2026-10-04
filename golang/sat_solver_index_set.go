// Source: src/util/hashtable.h:45–64,139–620; src/util/map.h:276.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

const satIndexEntryStride uint64 = 12

func (s *satSolverIndexSet) init(r *memory.Runtime) error {
	s.runtime = r
	s.capacity = 8
	h, e := r.Allocate(8 * satIndexEntryStride)
	s.pending = h
	if e != nil {
		return e
	}
	for i := uint32(0); i < 8; i++ {
		if e = r.WriteUint32(h, uint64(i)*12+4, 0); e != nil {
			return e
		}
	}
	s.raw, s.pending = h, memory.Block{}
	s.size, s.deleted = 0, 0
	return nil
}
func (s *satSolverIndexSet) sync() error {
	if s.parent == (memory.Block{}) {
		return nil
	}
	for i, v := range [5]uint32{s.raw.Slot(), s.raw.Generation(), s.capacity, s.size, s.deleted} {
		if e := s.runtime.WriteUint32(s.parent, s.base+uint64(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}

// Nested source fields are read from the real enclosing Runtime entry at the
// actual member-read site. Cold scalar values are used only for standalone
// lexical hashtables; nested duplicates are constructor/cleanup metadata.
func (s *satSolverIndexSet) sourceHeader(word uint64) (uint32, error) {
	if s.runtime == nil {
		return 0, ErrSATOwner
	}
	if s.parent != (memory.Block{}) {
		return s.runtime.ReadUint32(s.parent, s.base+word*4)
	}
	switch word {
	case 2:
		return s.capacity, nil
	case 3:
		return s.size, nil
	case 4:
		return s.deleted, nil
	}
	return 0, ErrSATOwner
}
func (s *satSolverIndexSet) headerWrite(word uint64, v uint32) error {
	if s.parent != (memory.Block{}) {
		if e := s.runtime.WriteUint32(s.parent, s.base+word*4, v); e != nil {
			return e
		}
	}
	switch word {
	case 2:
		s.capacity = v
	case 3:
		s.size = v
	case 4:
		s.deleted = v
	}
	return nil
}
func (s *satSolverIndexSet) publishPointer(h memory.Block) error {
	if s.parent != (memory.Block{}) {
		if e := s.runtime.WriteUint32(s.parent, s.base, h.Slot()); e != nil {
			return e
		}
		if e := s.runtime.WriteUint32(s.parent, s.base+4, h.Generation()); e != nil {
			return e
		}
	}
	s.raw = h
	return nil
}

// The concrete cold descriptor supplies provenance, never an alternate table
// when actual nested pointer words disagree. Validate at the reached payload read.
func (s *satSolverIndexSet) tableGuard(block memory.Block) error {
	if block != s.raw || s.parent == (memory.Block{}) {
		return nil
	}
	slot, e := s.runtime.ReadUint32(s.parent, s.base)
	if e != nil {
		return e
	}
	gen, e := s.runtime.ReadUint32(s.parent, s.base+4)
	if e != nil {
		return e
	}
	if slot != block.Slot() || gen != block.Generation() {
		return ErrSATOwner
	}
	return nil
}
func (s *satSolverIndexSet) state(block memory.Block, i uint32) (uint32, error) {
	if e := s.tableGuard(block); e != nil {
		return 0, e
	}
	return s.runtime.ReadUint32(block, uint64(i)*12+4)
}
func (s *satSolverIndexSet) key(block memory.Block, i uint32) (uint32, error) {
	if e := s.tableGuard(block); e != nil {
		return 0, e
	}
	return s.runtime.ReadUint32(block, uint64(i)*12+8)
}
func (s *satSolverIndexSet) copyEntry(dst memory.Block, di uint32, src memory.Block, si uint32) error {
	for j := uint64(0); j < 12; j += 4 {
		v, e := s.runtime.ReadUint32(src, uint64(si)*12+j)
		if e != nil {
			return e
		}
		if e = s.runtime.WriteUint32(dst, uint64(di)*12+j, v); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverIndexSet) replacement(capacity uint32) error {
	if s.pending != (memory.Block{}) {
		return arena.ErrClosed
	}
	oldCapacity, e := s.sourceHeader(2)
	if e != nil {
		return e
	}
	h, e := s.runtime.Allocate(uint64(capacity) * 12)
	s.pending = h
	if e != nil {
		return e
	}
	for i := uint32(0); i < capacity; i++ {
		if e = s.runtime.WriteUint32(h, uint64(i)*12+4, 0); e != nil {
			return e
		}
	}
	for i := uint32(0); i < oldCapacity; i++ {
		st, e := s.state(s.raw, i)
		if e != nil {
			return e
		}
		if st != 2 {
			continue
		}
		hash, e := s.runtime.ReadUint32(s.raw, uint64(i)*12)
		if e != nil {
			return e
		}
		begin := hash & (capacity - 1)
		found := false
		for j := uint32(0); j < capacity; j++ {
			k := (begin + j) & (capacity - 1)
			st, e = s.state(h, k)
			if e != nil {
				return e
			}
			if st == 0 {
				if e = s.copyEntry(h, k, s.raw, i); e != nil {
					return e
				}
				found = true
				break
			}
		}
		if !found {
			return ErrSATAssertion
		}
	}
	if e = s.runtime.Deallocate(&s.raw); e != nil {
		return e
	}
	if e = s.publishPointer(h); e != nil {
		return e
	}
	s.pending = memory.Block{}
	if e = s.headerWrite(2, capacity); e != nil {
		return e
	}
	return s.headerWrite(4, 0)
}
func (s *satSolverIndexSet) probe(k uint32) (uint32, bool, error) {
	capacity, e := s.sourceHeader(2)
	if e != nil {
		return 0, false, e
	}
	begin := k & (capacity - 1)
	for segment := 0; segment < 2; segment++ {
		start, end := begin, capacity
		if segment == 1 {
			start, end = 0, begin
		}
		for i := start; i != end; i++ {
			st, e := s.state(s.raw, i)
			if e != nil {
				return 0, false, e
			}
			if st == 0 {
				return i, false, nil
			}
			if st == 2 {
				hash, e := s.runtime.ReadUint32(s.raw, uint64(i)*12)
				if e != nil {
					return 0, false, e
				}
				if hash == k {
					v, e := s.key(s.raw, i)
					if e != nil {
						return 0, false, e
					}
					if v == k {
						return i, true, nil
					}
				}
			}
		}
	}
	return 0, false, nil
}
func (s *satSolverIndexSet) contains(k uint32) (bool, error) { _, yes, e := s.probe(k); return yes, e }
func (s *satSolverIndexSet) insert(k uint32) error {
	size, e := s.sourceHeader(3)
	if e != nil {
		return e
	}
	deletes, e := s.sourceHeader(4)
	if e != nil {
		return e
	}
	capacity, e := s.sourceHeader(2)
	if e != nil {
		return e
	}
	if (size+deletes)<<2 > capacity*3 {
		if e := s.replacement(capacity << 1); e != nil {
			return e
		}
	}
	capacity, e = s.sourceHeader(2)
	if e != nil {
		return e
	}
	begin := k & (capacity - 1)
	var deleted uint32
	hasDeleted := false
	for j := uint32(0); j < capacity; j++ {
		i := (begin + j) & (capacity - 1)
		st, e := s.state(s.raw, i)
		if e != nil {
			return e
		}
		if st == 2 {
			hash, e := s.runtime.ReadUint32(s.raw, uint64(i)*12)
			if e != nil {
				return e
			}
			if hash == k {
				v, e := s.key(s.raw, i)
				if e != nil {
					return e
				}
				if v == k {
					return s.runtime.WriteUint32(s.raw, uint64(i)*12+8, k)
				}
			}
		} else if st == 0 {
			if hasDeleted {
				i = deleted
				deletes, e = s.sourceHeader(4)
				if e != nil {
					return e
				}
				if e = s.headerWrite(4, deletes-1); e != nil {
					return e
				}
			}
			if e = s.runtime.WriteUint32(s.raw, uint64(i)*12+8, k); e != nil {
				return e
			}
			if e = s.runtime.WriteUint32(s.raw, uint64(i)*12+4, 2); e != nil {
				return e
			}
			if e = s.runtime.WriteUint32(s.raw, uint64(i)*12, k); e != nil {
				return e
			}
			size, e = s.sourceHeader(3)
			if e != nil {
				return e
			}
			return s.headerWrite(3, size+1)
		} else {
			if satStorageSourceDebug && st != 1 {
				return ErrSATAssertion
			}
			deleted = i
			hasDeleted = true
		}
	}
	return ErrSATAssertion
}
func (s *satSolverIndexSet) reset() error {
	size, e := s.sourceHeader(3)
	if e != nil {
		return e
	}
	if size == 0 {
		deletes, e := s.sourceHeader(4)
		if e != nil {
			return e
		}
		if deletes == 0 {
			return nil
		}
	}
	capacity, e := s.sourceHeader(2)
	if e != nil {
		return e
	}
	overhead := uint32(0)
	for i := uint32(0); i < capacity; i++ {
		st, e := s.state(s.raw, i)
		if e != nil {
			return e
		}
		if st != 0 {
			if e = s.runtime.WriteUint32(s.raw, uint64(i)*12+4, 0); e != nil {
				return e
			}
		} else {
			overhead++
		}
	}
	if capacity > 16 && overhead<<2 > capacity*3 {
		if e := s.runtime.Deallocate(&s.raw); e != nil {
			return e
		}
		capacity >>= 1
		if e := s.publishPointer(memory.Block{}); e != nil {
			return e
		}
		if e := s.headerWrite(2, capacity); e != nil {
			return e
		}
		h, e := s.runtime.Allocate(uint64(capacity) * 12)
		s.pending = h
		if e != nil {
			return e
		}
		for i := uint32(0); i < capacity; i++ {
			if e = s.runtime.WriteUint32(h, uint64(i)*12+4, 0); e != nil {
				return e
			}
		}
		if e = s.publishPointer(h); e != nil {
			return e
		}
		s.pending = memory.Block{}
	}
	if e = s.headerWrite(3, 0); e != nil {
		return e
	}
	return s.headerWrite(4, 0)
}
func (s *satSolverIndexSet) assign(other *satSolverIndexSet) error {
	if s == other {
		return nil
	}
	if e := s.reset(); e != nil {
		return e
	}
	it, e := other.capture()
	if e != nil {
		return e
	}
	for {
		v, e := it.next()
		if e != nil {
			return e
		}
		if !v.present {
			return nil
		}
		if e = s.insert(v.value); e != nil {
			return e
		}
	}
}
func (s *satSolverIndexSet) union(other *satSolverIndexSet) error {
	if s == other {
		return nil
	}
	it, e := other.capture()
	if e != nil {
		return e
	}
	for {
		v, e := it.next()
		if e != nil {
			return e
		}
		if !v.present {
			return nil
		}
		if e = s.insert(v.value); e != nil {
			return e
		}
	}
}

// Copy constructor allocates exactly the source capacity then rehashes USED
// entries into fresh FREE storage. Assignment instead resets and inserts.
func (s *satSolverIndexSet) copyCtor(other *satSolverIndexSet) error {
	s.runtime = other.runtime
	capacity, e := other.sourceHeader(2)
	if e != nil {
		return e
	}
	s.capacity = capacity
	h, e := s.runtime.Allocate(uint64(s.capacity) * 12)
	s.pending = h
	if e != nil {
		return e
	}
	for i := uint32(0); i < s.capacity; i++ {
		if e = s.runtime.WriteUint32(h, uint64(i)*12+4, 0); e != nil {
			return e
		}
	}
	for i := uint32(0); i < capacity; i++ {
		st, e := other.state(other.raw, i)
		if e != nil {
			return e
		}
		if st != 2 {
			continue
		}
		hash, e := other.runtime.ReadUint32(other.raw, uint64(i)*12)
		if e != nil {
			return e
		}
		found := false
		for j := uint32(0); j < s.capacity; j++ {
			k := ((hash & (s.capacity - 1)) + j) & (s.capacity - 1)
			st, e = s.state(h, k)
			if e != nil {
				return e
			}
			if st == 0 {
				if e = s.copyEntry(h, k, other.raw, i); e != nil {
					return e
				}
				found = true
				break
			}
		}
		if !found {
			return ErrSATAssertion
		}
	}
	s.raw, s.pending = h, memory.Block{}
	size, e := other.sourceHeader(3)
	if e != nil {
		return e
	}
	s.size, s.deleted = size, 0
	return s.sync()
}
func (s *satSolverIndexSet) close() error {
	if s.runtime == nil {
		return nil
	}
	if s.raw != (memory.Block{}) {
		if e := s.runtime.Deallocate(&s.raw); e != nil {
			return e
		}
	}
	if s.pending != (memory.Block{}) {
		if e := s.runtime.Deallocate(&s.pending); e != nil {
			return e
		}
	}
	return nil
}

type satSolverIndexValue struct {
	value   uint32
	present bool
}
type satSolverIndexIterator struct {
	runtime         *memory.Runtime
	block           memory.Block
	capacity, index uint32
}

func (s *satSolverIndexSet) capture() (satSolverIndexIterator, error) {
	capacity, e := s.sourceHeader(2)
	if e != nil {
		return satSolverIndexIterator{}, e
	}
	if e := s.tableGuard(s.raw); e != nil {
		return satSolverIndexIterator{}, e
	}
	return satSolverIndexIterator{runtime: s.runtime, block: s.raw, capacity: capacity}, nil
}
func (it *satSolverIndexIterator) next() (satSolverIndexValue, error) {
	for it.index < it.capacity {
		i := it.index
		it.index++
		st, e := it.runtime.ReadUint32(it.block, uint64(i)*12+4)
		if e != nil {
			return satSolverIndexValue{}, e
		}
		if st == 2 {
			v, e := it.runtime.ReadUint32(it.block, uint64(i)*12+8)
			return satSolverIndexValue{v, true}, e
		}
	}
	return satSolverIndexValue{}, nil
}
