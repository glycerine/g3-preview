// Licensed under the MIT license.
// Concrete pointer-free payload adapter for source nested literal vectors.
package g3

import (
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

// Tokens are storage identities, never source Boolean variables or clause IDs.
// Records contain no Go pointers; one cold store owns all payload/ledger pages.
type satSolverLiteralListRef struct {
	owner            uint64
	slot, generation uint32
}
type satSolverLiteralListRecord struct {
	payload                            arena.VectorRef
	length, capacity, generation, next uint32
	live, retired                      bool
}
type satSolverLiteralStore struct {
	self                               *satSolverLiteralStore
	words                              *arena.Words
	budget                             *arena.Budget
	records                            []satSolverLiteralListRecord
	origin, ownerBytes, directoryBytes uint64
	free                               uint32 // encoded slot+1; zero means empty.
	closed                             bool
}

func newSATSolverLiteralStore(c arena.VectorConfig) (*satSolverLiteralStore, error) {
	bytes := uint64(reflect.TypeOf(satSolverLiteralStore{}).Size())
	b := c.Descriptors.Budget
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	w, e := arena.NewWords(c)
	if e != nil {
		b.Release(bytes)
		return nil, e
	}
	id, e := satNextOwner()
	if e != nil {
		w.Close()
		b.Release(bytes)
		return nil, e
	}
	s := &satSolverLiteralStore{words: w, budget: b, origin: id, ownerBytes: bytes}
	s.self = s
	return s, nil
}
func (s *satSolverLiteralStore) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverLiteralStore) record(r satSolverLiteralListRef) (*satSolverLiteralListRecord, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	if r.owner != s.origin {
		return nil, ErrSATOwner
	}
	if uint64(r.slot) >= uint64(len(s.records)) {
		return nil, arena.ErrInvalidSlot
	}
	x := &s.records[r.slot]
	if !x.live || x.generation != r.generation {
		return nil, arena.ErrInvalidSlot
	}
	return x, nil
}
func (s *satSolverLiteralStore) IsLive(r satSolverLiteralListRef) bool {
	_, e := s.record(r)
	return e == nil
}
func (s *satSolverLiteralStore) NewList() (satSolverLiteralListRef, error) {
	if e := s.open(); e != nil {
		return satSolverLiteralListRef{}, e
	}
	if s.free != 0 {
		i := s.free - 1
		x := &s.records[i]
		s.free = x.next
		x.next = 0
		x.live = true
		return satSolverLiteralListRef{s.origin, i, x.generation}, nil
	}
	if uint64(len(s.records)) >= uint64(^uint32(0)) {
		return satSolverLiteralListRef{}, arena.ErrSlotLimit
	}
	if len(s.records) == cap(s.records) {
		stride := uint64(reflect.TypeOf(satSolverLiteralListRecord{}).Size())
		n, e := satVectorNextCapacityFor(uint64(cap(s.records)), stride, stride)
		if e != nil {
			return satSolverLiteralListRef{}, e
		}
		bytes := n * stride
		if e = s.budget.Reserve(bytes); e != nil {
			return satSolverLiteralListRef{}, e
		}
		a := make([]satSolverLiteralListRecord, len(s.records), int(n))
		copy(a, s.records)
		s.budget.Release(s.directoryBytes)
		s.records = a
		s.directoryBytes = bytes
	}
	i := uint32(len(s.records))
	s.records = append(s.records, satSolverLiteralListRecord{generation: 1, live: true})
	return satSolverLiteralListRef{s.origin, i, 1}, nil
}
func (s *satSolverLiteralStore) Size(r satSolverLiteralListRef) (uint32, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	return x.length, nil
}
func (s *satSolverLiteralStore) Capacity(r satSolverLiteralListRef) (uint32, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	return x.capacity, nil
}
func (s *satSolverLiteralStore) readPhysical(r satSolverLiteralListRef, i uint32) (satLiteral, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	if i >= x.capacity {
		return 0, arena.ErrBounds
	}
	v, e := s.words.Read(x.payload, i)
	return satLiteral(v), e
}
func (s *satSolverLiteralStore) Read(r satSolverLiteralListRef, i uint32) (satLiteral, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	if satStorageSourceDebug && i >= x.length {
		return 0, ErrSATAssertion
	}
	return s.readPhysical(r, i)
}
func (s *satSolverLiteralStore) Write(r satSolverLiteralListRef, i uint32, l satLiteral) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && i >= x.length {
		return ErrSATAssertion
	}
	if i >= x.capacity {
		return arena.ErrBounds
	}
	return s.words.Write(x.payload, i, uint32(l))
}

// One source capacity expansion allocates a new actual arena payload, copies
// FULL prior capacity (trivial source reallocate), publishes that expansion,
// then frees old payload. Later expansion failure retains completed prefixes.
// Arena free blocks/pages remain physically owned and charged, an explicit
// pooling adaptation; source logical capacities remain exactly 2,3,5,8,... .
func (s *satSolverLiteralStore) Reserve(r satSolverLiteralListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	for n > x.capacity {
		cap64, e := satVectorNextCapacity(uint64(x.capacity), 4)
		if e != nil {
			return e
		}
		c := uint32(cap64)
		p, e := s.words.Allocate(c, c)
		if e != nil {
			return e
		}
		for i := uint32(0); i < x.capacity; i++ {
			v, e := s.words.Read(x.payload, i)
			if e != nil {
				_ = s.words.Free(p)
				return e
			}
			if e = s.words.Write(p, i, v); e != nil {
				_ = s.words.Free(p)
				return e
			}
		}
		old := x.payload
		x.payload = p
		x.capacity = c
		if old != (arena.VectorRef{}) {
			if e = s.words.Free(old); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolverLiteralStore) Push(r satSolverLiteralListRef, l satLiteral) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if x.length == ^uint32(0) {
		return arena.ErrBounds
	}
	if e = s.Reserve(r, x.length+1); e != nil {
		return e
	}
	if e = s.words.Write(x.payload, x.length, uint32(l)); e != nil {
		return e
	}
	x.length++
	return nil
}
func (s *satSolverLiteralStore) Shrink(r satSolverLiteralListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && n > x.length {
		return ErrSATAssertion
	}
	if x.capacity == 0 {
		return nil
	}
	if n > x.capacity {
		return arena.ErrBounds
	}
	x.length = n
	return nil
}
func (s *satSolverLiteralStore) Resize(r satSolverLiteralListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if n <= x.length {
		return s.Shrink(r, n)
	}
	if e = s.Reserve(r, n); e != nil {
		return e
	}
	for i := x.length; i < n; i++ {
		if e = s.words.Write(x.payload, i, uint32(satDefaultLiteral())); e != nil {
			return e
		}
	}
	x.length = n
	return nil
}
func (s *satSolverLiteralStore) CloneList(r satSolverLiteralListRef) (satSolverLiteralListRef, error) {
	x, e := s.record(r)
	if e != nil {
		return satSolverLiteralListRef{}, e
	}
	original := *x // NewList may relocate descriptor directory.
	result, e := s.NewList()
	if e != nil {
		return satSolverLiteralListRef{}, e
	}
	if original.capacity == 0 {
		return result, nil
	}
	p, e := s.words.Allocate(original.capacity, original.capacity)
	if e != nil {
		_ = s.Free(result)
		return satSolverLiteralListRef{}, e
	}
	for i := uint32(0); i < original.length; i++ {
		v, e := s.words.Read(original.payload, i)
		if e != nil {
			_ = s.words.Free(p)
			_ = s.Free(result)
			return satSolverLiteralListRef{}, e
		}
		if e = s.words.Write(p, i, v); e != nil {
			_ = s.words.Free(p)
			_ = s.Free(result)
			return satSolverLiteralListRef{}, e
		}
	}
	out, _ := s.record(result)
	out.payload = p
	out.capacity = original.capacity
	out.length = original.length
	return result, nil
}
func (s *satSolverLiteralStore) Free(r satSolverLiteralListRef) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if x.payload != (arena.VectorRef{}) {
		if e = s.words.Free(x.payload); e != nil {
			return e
		}
	}
	x.payload = arena.VectorRef{}
	x.length = 0
	x.capacity = 0
	x.live = false
	if x.generation == ^uint32(0) {
		x.retired = true
		return nil
	}
	x.generation++
	x.next = s.free
	s.free = r.slot + 1
	return nil
}
func (s *satSolverLiteralStore) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	for i := range s.records {
		if s.records[i].live {
			if e := s.Free(satSolverLiteralListRef{s.origin, uint32(i), s.records[i].generation}); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolverLiteralStore) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.words.Close()
	s.records = nil
	s.budget.Release(s.directoryBytes + s.ownerBytes)
	s.directoryBytes = 0
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Cross-owner copy bridges actual words, never foreign raw arena handles.
// It is the concrete nested literal_vector copy-constructor adapter.
func (s *satSolverLiteralStore) CopyList(source *satSolverLiteralStore, r satSolverLiteralListRef) (satSolverLiteralListRef, error) {
	if e := s.open(); e != nil {
		return satSolverLiteralListRef{}, e
	}
	x, e := source.record(r)
	if e != nil {
		return satSolverLiteralListRef{}, e
	}
	original := *x
	result, e := s.NewList()
	if e != nil {
		return satSolverLiteralListRef{}, e
	}
	if original.capacity == 0 {
		return result, nil
	}
	p, e := s.words.Allocate(original.capacity, original.capacity)
	if e != nil {
		_ = s.Free(result)
		return satSolverLiteralListRef{}, e
	}
	for i := uint32(0); i < original.length; i++ {
		v, e := source.words.Read(original.payload, i)
		if e != nil {
			_ = s.words.Free(p)
			_ = s.Free(result)
			return satSolverLiteralListRef{}, e
		}
		if e = s.words.Write(p, i, v); e != nil {
			_ = s.words.Free(p)
			_ = s.Free(result)
			return satSolverLiteralListRef{}, e
		}
	}
	out, _ := s.record(result)
	out.payload, out.capacity, out.length = p, original.capacity, original.length
	return result, nil
}
