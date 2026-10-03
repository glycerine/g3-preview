// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
// Concrete pointer-free payload adapter for source nested watch vectors.
package g3

import (
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

// Tokens are storage identities, never source Boolean variables or clause IDs.
// Records contain no Go pointers; one cold store owns all payload/ledger pages.
type satSolverWatchListRef struct {
	owner            uint64
	slot, generation uint32
}
type satSolverWatchListRecord struct {
	payload, lifetime                  arena.VectorRef
	length, capacity, generation, next uint32
	live, retired                      bool
}
type satSolverWatchStore struct {
	self                               *satSolverWatchStore
	payloads                           *arena.Vectors64
	lifetimes                          *arena.Words
	budget                             *arena.Budget
	records                            []satSolverWatchListRecord
	origin, ownerBytes, directoryBytes uint64
	free                               uint32 // encoded slot+1; zero means empty.
	closed                             bool
}

func newSATSolverWatchStore(c arena.VectorConfig) (*satSolverWatchStore, error) {
	bytes := uint64(reflect.TypeOf(satSolverWatchStore{}).Size())
	b := c.Descriptors.Budget
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	w, e := arena.NewVectors64(c)
	if e != nil {
		b.Release(bytes)
		return nil, e
	}
	life, e := arena.NewWords(c)
	if e != nil {
		w.Close()
		b.Release(bytes)
		return nil, e
	}
	id, e := satNextOwner()
	if e != nil {
		w.Close()
		life.Close()
		b.Release(bytes)
		return nil, e
	}
	s := &satSolverWatchStore{payloads: w, lifetimes: life, budget: b, origin: id, ownerBytes: bytes}
	s.self = s
	return s, nil
}
func (s *satSolverWatchStore) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverWatchStore) record(r satSolverWatchListRef) (*satSolverWatchListRecord, error) {
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
func (s *satSolverWatchStore) IsLive(r satSolverWatchListRef) bool {
	_, e := s.record(r)
	return e == nil
}
func (s *satSolverWatchStore) NewList() (satSolverWatchListRef, error) {
	if e := s.open(); e != nil {
		return satSolverWatchListRef{}, e
	}
	if s.free != 0 {
		i := s.free - 1
		x := &s.records[i]
		s.free = x.next
		x.next = 0
		x.live = true
		return satSolverWatchListRef{s.origin, i, x.generation}, nil
	}
	if uint64(len(s.records)) >= uint64(^uint32(0)) {
		return satSolverWatchListRef{}, arena.ErrSlotLimit
	}
	if len(s.records) == cap(s.records) {
		stride := uint64(reflect.TypeOf(satSolverWatchListRecord{}).Size())
		n, e := satWatchDirectoryCapacity(uint64(cap(s.records)))
		if e != nil {
			return satSolverWatchListRef{}, e
		}
		bytes := n * stride
		if e = s.budget.Reserve(bytes); e != nil {
			return satSolverWatchListRef{}, e
		}
		a := make([]satSolverWatchListRecord, len(s.records), int(n))
		copy(a, s.records)
		s.budget.Release(s.directoryBytes)
		s.records = a
		s.directoryBytes = bytes
	}
	i := uint32(len(s.records))
	s.records = append(s.records, satSolverWatchListRecord{generation: 1, live: true})
	return satSolverWatchListRef{s.origin, i, 1}, nil
}
func (s *satSolverWatchStore) Size(r satSolverWatchListRef) (uint32, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	return x.length, nil
}
func (s *satSolverWatchStore) Capacity(r satSolverWatchListRef) (uint32, error) {
	x, e := s.record(r)
	if e != nil {
		return 0, e
	}
	return x.capacity, nil
}

func satWatchCodecEncode(w satWatched) [4]uint64 {
	return [4]uint64{uint64(w.val1) | uint64(w.val2)<<32, w.locator.owner, uint64(w.locator.slot) | uint64(w.locator.generation)<<32, uint64(w.locator.storageKind)}
}
func satWatchCodecDecode(a [4]uint64) satWatched {
	return satWatched{val1: uint32(a[0]), val2: uint32(a[0] >> 32), locator: satClauseOffset{owner: a[1], slot: uint32(a[2]), generation: uint32(a[2] >> 32), storageKind: uint32(a[3])}}
}
func (s *satSolverWatchStore) initialized(x *satSolverWatchListRecord, i uint32) (bool, error) {
	if i >= x.capacity {
		return false, arena.ErrBounds
	}
	v, e := s.lifetimes.Read(x.lifetime, i/32)
	return v&(uint32(1)<<(i%32)) != 0, e
}
func (s *satSolverWatchStore) setInitialized(x *satSolverWatchListRecord, i uint32, live bool) error {
	v, e := s.lifetimes.Read(x.lifetime, i/32)
	if e != nil {
		return e
	}
	bit := uint32(1) << (i % 32)
	if live {
		v |= bit
	} else {
		v &^= bit
	}
	return s.lifetimes.Write(x.lifetime, i/32, v)
}
func (s *satSolverWatchStore) decode(x *satSolverWatchListRecord, i uint32) (satWatched, error) {
	var a [4]uint64
	for k := uint32(0); k < 4; k++ {
		v, e := s.payloads.Read(x.payload, i*4+k)
		if e != nil {
			return satWatched{}, e
		}
		a[k] = v
	}
	return satWatchCodecDecode(a), nil
}
func (s *satSolverWatchStore) encode(x *satSolverWatchListRecord, i uint32, w satWatched) error {
	a := satWatchCodecEncode(w)
	for k := uint32(0); k < 4; k++ {
		if e := s.payloads.Write(x.payload, i*4+k, a[k]); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) readPhysical(r satSolverWatchListRef, i uint32) (satWatched, error) {
	x, e := s.record(r)
	if e != nil {
		return satWatched{}, e
	}
	live, e := s.initialized(x, i)
	if e != nil {
		return satWatched{}, e
	}
	if !live {
		return satWatched{}, arena.ErrBounds
	}
	return s.decode(x, i)
}
func (s *satSolverWatchStore) Read(r satSolverWatchListRef, i uint32) (satWatched, error) {
	x, e := s.record(r)
	if e != nil {
		return satWatched{}, e
	}
	if satStorageSourceDebug && i >= x.length {
		return satWatched{}, ErrSATAssertion
	}
	return s.readPhysical(r, i)
}
func (s *satSolverWatchStore) Write(r satSolverWatchListRef, i uint32, w satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && i >= x.length {
		return ErrSATAssertion
	}
	live, e := s.initialized(x, i)
	if e != nil {
		return e
	}
	if !live {
		return arena.ErrBounds
	}
	return s.encode(x, i, w)
}

// Composite charged payload/bitmap preparation is a Go storage adaptation.
// Publish a source expansion only after both allocations succeed. Completed
// arena page/directory growth remains owned even when preparation is freed.
func (s *satSolverWatchStore) allocate(capacity uint32) (arena.VectorRef, arena.VectorRef, error) {
	n, bits, e := satWatchPayloadExtent(capacity)
	if e != nil {
		return arena.VectorRef{}, arena.VectorRef{}, e
	}
	p, e := s.payloads.Allocate(n, n)
	if e != nil {
		return arena.VectorRef{}, arena.VectorRef{}, e
	}
	b, e := s.lifetimes.Allocate(bits, bits)
	if e != nil {
		_ = s.payloads.Free(p)
		return arena.VectorRef{}, arena.VectorRef{}, e
	}
	return p, b, nil
}
func (s *satSolverWatchStore) ReserveCapacity(r satSolverWatchListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	for n > x.capacity {
		c, e := satVectorNextCapacityFor(uint64(x.capacity), 32, 8)
		if e != nil {
			return e
		}
		p, b, e := s.allocate(uint32(c))
		if e != nil {
			return e
		}
		y := satSolverWatchListRecord{payload: p, lifetime: b, capacity: uint32(c)}
		for i := uint32(0); i < x.capacity; i++ {
			w, e := s.decode(x, i)
			if e == nil {
				e = s.encode(&y, i, w)
			}
			if e != nil {
				_ = s.payloads.Free(p)
				_ = s.lifetimes.Free(b)
				return e
			}
		}
		for i := uint32(0); i < uint32((uint64(x.capacity)+31)/32); i++ {
			v, e := s.lifetimes.Read(x.lifetime, i)
			if e != nil {
				_ = s.payloads.Free(p)
				_ = s.lifetimes.Free(b)
				return e
			}
			if e = s.lifetimes.Write(b, i, v); e != nil {
				_ = s.payloads.Free(p)
				_ = s.lifetimes.Free(b)
				return e
			}
		}
		oldP, oldB := x.payload, x.lifetime
		x.payload, x.lifetime, x.capacity = p, b, uint32(c)
		if oldP != (arena.VectorRef{}) {
			if e = s.payloads.Free(oldP); e != nil {
				return e
			}
			if e = s.lifetimes.Free(oldB); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolverWatchStore) Push(r satSolverWatchListRef, w satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if x.length == ^uint32(0) {
		return arena.ErrBounds
	}
	if e = s.ReserveCapacity(r, x.length+1); e != nil {
		return e
	}
	if e = s.encode(x, x.length, w); e != nil {
		return e
	}
	if e = s.setInitialized(x, x.length, true); e != nil {
		return e
	}
	x.length++
	return nil
}
func (s *satSolverWatchStore) Pop(r satSolverWatchListRef) (satWatched, error) {
	x, e := s.record(r)
	if e != nil {
		return satWatched{}, e
	}
	if satStorageSourceDebug && x.length == 0 {
		return satWatched{}, ErrSATAssertion
	}
	if x.length == 0 {
		return satWatched{}, arena.ErrBounds
	}
	w, e := s.readPhysical(r, x.length-1)
	if e != nil {
		return w, e
	}
	if e = s.setInitialized(x, x.length-1, false); e != nil {
		return w, e
	}
	x.length--
	return w, nil
}
func (s *satSolverWatchStore) Shrink(r satSolverWatchListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && n > x.length {
		return ErrSATAssertion
	}
	return s.SetEnd(r, n)
}
func (s *satSolverWatchStore) SetEnd(r satSolverWatchListRef, n uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if x.capacity == 0 {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if n > x.length {
		return arena.ErrBounds
	}
	for i := n; i < x.length; i++ {
		if e = s.setInitialized(x, i, false); e != nil {
			return e
		}
	}
	x.length = n
	return nil
}
func (s *satSolverWatchStore) ResizeFill(r satSolverWatchListRef, n uint32, w satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if n <= x.length {
		return s.Shrink(r, n)
	}
	if e = s.ReserveCapacity(r, n); e != nil {
		return e
	}
	old := x.length
	x.length = n
	for i := old; i < n; i++ {
		if e = s.encode(x, i, w); e != nil {
			return e
		}
		if e = s.setInitialized(x, i, true); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) ReserveFill(r satSolverWatchListRef, n uint32, w satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if n > x.length {
		return s.ResizeFill(r, n, w)
	}
	return nil
}
func (s *satSolverWatchStore) EraseAt(r satSolverWatchListRef, i uint32) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && i >= x.length {
		return ErrSATAssertion
	}
	if i >= x.length {
		return arena.ErrBounds
	}
	for j := i + 1; j < x.length; j++ {
		w, e := s.readPhysical(r, j)
		if e != nil {
			return e
		}
		if e = s.Write(r, j-1, w); e != nil {
			return e
		}
	}
	x.length--
	return nil
}
func (s *satSolverWatchStore) Erase(r satSolverWatchListRef, w satWatched) (bool, error) {
	x, e := s.record(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < x.length; i++ {
		v, e := s.Read(r, i)
		if e != nil {
			return false, e
		}
		if v.Equal(w) {
			return true, s.EraseAt(r, i)
		}
	}
	return false, nil
}
func (s *satSolverWatchStore) EraseClauseWatch(r satSolverWatchListRef, o satClauseOffset) (bool, error) {
	x, e := s.record(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < x.length; i++ {
		w, e := s.Read(r, i)
		if e != nil {
			return false, e
		}
		if w.Clause() {
			q, e := w.ClauseOffset()
			if e != nil {
				return false, e
			}
			if q == o {
				n := x.length - 1
				for j := i + 1; j < x.length; j++ {
					v, e := s.Read(r, j)
					if e != nil {
						return false, e
					}
					if e = s.Write(r, j-1, v); e != nil {
						return false, e
					}
				}
				if e = s.SetEnd(r, n); e != nil {
					return false, e
				}
				return true, nil
			}
		}
	}
	return false, nil
}
func (s *satSolverWatchStore) ResetList(r satSolverWatchListRef) error { return s.SetEnd(r, 0) }
func (s *satSolverWatchStore) Finalize(r satSolverWatchListRef) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if x.payload != (arena.VectorRef{}) {
		if e = s.payloads.Free(x.payload); e != nil {
			return e
		}
		if e = s.lifetimes.Free(x.lifetime); e != nil {
			return e
		}
	}
	x.payload, x.lifetime = arena.VectorRef{}, arena.VectorRef{}
	x.length, x.capacity = 0, 0
	return nil
}
func (s *satSolverWatchStore) Free(r satSolverWatchListRef) error {
	if e := s.Finalize(r); e != nil {
		return e
	}
	x, e := s.record(r)
	if e != nil {
		return e
	}
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
func (s *satSolverWatchStore) CopyList(other *satSolverWatchStore, r satSolverWatchListRef) (satSolverWatchListRef, error) {
	x, e := other.record(r)
	if e != nil {
		return satSolverWatchListRef{}, e
	}
	original := *x
	q, e := s.NewList()
	if e != nil {
		return q, e
	}
	if original.capacity == 0 {
		return q, nil
	}
	p, b, e := s.allocate(original.capacity)
	if e != nil {
		_ = s.Free(q)
		return satSolverWatchListRef{}, e
	}
	out, _ := s.record(q)
	out.payload, out.lifetime, out.capacity = p, b, original.capacity
	out.length = original.length
	for i := uint32(0); i < original.length; i++ {
		w, e := other.readPhysical(r, i)
		if e == nil {
			e = s.encode(out, i, w)
		}
		if e == nil {
			e = s.setInitialized(out, i, true)
		}
		if e != nil {
			_ = s.Free(q)
			return satSolverWatchListRef{}, e
		}
	}
	out.length = original.length
	return q, nil
}
func (s *satSolverWatchStore) CloneList(r satSolverWatchListRef) (satSolverWatchListRef, error) {
	return s.CopyList(s, r)
}
func (s *satSolverWatchStore) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	for i := range s.records {
		if s.records[i].live {
			if e := s.Free(satSolverWatchListRef{s.origin, uint32(i), s.records[i].generation}); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolverWatchStore) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.payloads.Close()
	s.lifetimes.Close()
	s.records = nil
	s.budget.Release(s.directoryBytes + s.ownerBytes)
	s.directoryBytes, s.ownerBytes = 0, 0
	s.closed = true
	return nil
}

func (s *satSolverWatchStore) SetX(r satSolverWatchListRef, i uint32, w, d satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if i >= x.length {
		if e = s.ResizeFill(r, i+1, d); e != nil {
			return e
		}
	}
	live, e := s.initialized(x, i)
	if e != nil {
		return e
	}
	if !live {
		return arena.ErrBounds
	}
	return s.encode(x, i, w)
}

// MoveList implements the watch vector move constructor within this owner.
// A moved-from list object remains valid and empty. No payload is copied.
func (s *satSolverWatchStore) MoveList(r satSolverWatchListRef) (satSolverWatchListRef, error) {
	x, e := s.record(r)
	if e != nil {
		return satSolverWatchListRef{}, e
	}
	q, e := s.NewList()
	if e != nil {
		return q, e
	}
	x, _ = s.record(r)
	y, _ := s.record(q)
	y.payload, y.lifetime, y.length, y.capacity = x.payload, x.lifetime, x.length, x.capacity
	x.payload, x.lifetime = arena.VectorRef{}, arena.VectorRef{}
	x.length, x.capacity = 0, 0
	return q, nil
}
func (s *satSolverWatchStore) AssignList(dst satSolverWatchListRef, other *satSolverWatchStore, src satSolverWatchListRef) error {
	if _, e := s.record(dst); e != nil {
		return e
	}
	x, e := other.record(src)
	if e != nil {
		return e
	}
	if other == s && dst == src {
		return nil
	}
	source := *x
	if e = s.Finalize(dst); e != nil {
		return e
	}
	if source.capacity == 0 {
		return nil
	}
	p, b, e := s.allocate(source.capacity)
	if e != nil {
		return e
	}
	out, _ := s.record(dst)
	out.payload, out.lifetime, out.capacity = p, b, source.capacity
	out.length = source.length
	for i := uint32(0); i < source.length; i++ {
		w, e := other.readPhysical(src, i)
		if e == nil {
			e = s.encode(out, i, w)
		}
		if e == nil {
			e = s.setInitialized(out, i, true)
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) MoveAssignList(dst, src satSolverWatchListRef) error {
	if _, e := s.record(dst); e != nil {
		return e
	}
	x, e := s.record(src)
	if e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e = s.Finalize(dst); e != nil {
		return e
	}
	y, _ := s.record(dst)
	y.payload, y.lifetime, y.length, y.capacity = x.payload, x.lifetime, x.length, x.capacity
	x.payload, x.lifetime = arena.VectorRef{}, arena.VectorRef{}
	x.length, x.capacity = 0, 0
	return nil
}

// Iterators capture actual common-owner backing, not the movable list object.
type satSolverWatchIterator struct {
	store                *satSolverWatchStore
	payload, lifetime    arena.VectorRef
	capacity, index, end uint32
	reverse              bool
}

func (s *satSolverWatchStore) Iter(r satSolverWatchListRef) (satSolverWatchIterator, error) {
	x, e := s.record(r)
	if e != nil {
		return satSolverWatchIterator{}, e
	}
	return satSolverWatchIterator{store: s, payload: x.payload, lifetime: x.lifetime, capacity: x.capacity, end: x.length}, nil
}
func (s *satSolverWatchStore) ReverseIter(r satSolverWatchListRef) (satSolverWatchIterator, error) {
	i, e := s.Iter(r)
	i.reverse = true
	return i, e
}
func (i *satSolverWatchIterator) Next() (satWatched, bool, error) {
	if i == nil || i.store == nil {
		return satWatched{}, false, ErrSATOwner
	}
	if e := i.store.open(); e != nil {
		return satWatched{}, false, e
	}
	if i.index == i.end {
		return satWatched{}, false, nil
	}
	if !i.store.payloads.IsLive(i.payload) || !i.store.lifetimes.IsLive(i.lifetime) {
		return satWatched{}, false, arena.ErrInvalidSlot
	}
	n := i.index
	if i.reverse {
		n = i.end - 1 - i.index
	}
	x := satSolverWatchListRecord{payload: i.payload, lifetime: i.lifetime, capacity: i.capacity}
	live, e := i.store.initialized(&x, n)
	if e != nil {
		return satWatched{}, false, e
	}
	if !live {
		return satWatched{}, false, arena.ErrBounds
	}
	w, e := i.store.decode(&x, n)
	if e != nil {
		return w, false, e
	}
	i.index++
	return w, true, nil
}

func (s *satSolverWatchStore) AppendArray(r satSolverWatchListRef, xs []satWatched) error {
	if _, e := s.record(r); e != nil {
		return e
	}
	for _, w := range xs {
		if e := s.Push(r, w); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) Append(dst satSolverWatchListRef, other *satSolverWatchStore, src satSolverWatchListRef) error {
	if _, e := s.record(dst); e != nil {
		return e
	}
	x, e := other.record(src)
	if e != nil {
		return e
	}
	if s == other && dst == src && x.length != 0 {
		return ErrSATSelfAppend
	}
	n := x.length
	for i := uint32(0); i < n; i++ {
		w, e := other.Read(src, i)
		if e != nil {
			return e
		}
		if e = s.Push(dst, w); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) Fill(r satSolverWatchListRef, w satWatched) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	for i := uint32(0); i < x.length; i++ {
		if e = s.Write(r, i, w); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) Reverse(r satSolverWatchListRef) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	for i := uint32(0); i < x.length/2; i++ {
		a, e := s.Read(r, i)
		if e != nil {
			return e
		}
		b, e := s.Read(r, x.length-i-1)
		if e != nil {
			return e
		}
		if e = s.Write(r, i, b); e != nil {
			return e
		}
		if e = s.Write(r, x.length-i-1, a); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverWatchStore) Contains(r satSolverWatchListRef, w satWatched) (bool, error) {
	x, e := s.record(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < x.length; i++ {
		v, e := s.Read(r, i)
		if e != nil {
			return false, e
		}
		if v.Equal(w) {
			return true, nil
		}
	}
	return false, nil
}
func (s *satSolverWatchStore) SwapLists(a, b satSolverWatchListRef) error {
	x, e := s.record(a)
	if e != nil {
		return e
	}
	y, e := s.record(b)
	if e != nil {
		return e
	}
	if a == b {
		return nil
	}
	x.payload, y.payload = y.payload, x.payload
	x.lifetime, y.lifetime = y.lifetime, x.lifetime
	x.length, y.length = y.length, x.length
	x.capacity, y.capacity = y.capacity, x.capacity
	return nil
}

type satSolverWatchStoreStats struct {
	Lists, Retired                           uint32
	OwnerBytes, DirectoryBytes, ChargedBytes uint64
	Payloads, Lifetimes                      arena.VectorStats
}

func satWatchGraphBytes(v arena.VectorStats) uint64 {
	d := v.Descriptors
	return v.OwnerBytes + v.PayloadBytes + v.BlockMetadataBytes + v.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func (s *satSolverWatchStore) Stats() satSolverWatchStoreStats {
	if s == nil || s.self != s {
		return satSolverWatchStoreStats{}
	}
	out := satSolverWatchStoreStats{OwnerBytes: s.ownerBytes, DirectoryBytes: s.directoryBytes, Payloads: s.payloads.Stats(), Lifetimes: s.lifetimes.Stats()}
	for i := range s.records {
		if s.records[i].live {
			out.Lists++
		}
		if s.records[i].retired {
			out.Retired++
		}
	}
	out.ChargedBytes = out.OwnerBytes + out.DirectoryBytes + satWatchGraphBytes(out.Payloads) + satWatchGraphBytes(out.Lifetimes)
	return out
}

func (v *satSolverWatchStore) DisplayInto(r satSolverWatchListRef, a *satClauseAllocator, s *NumericTextStream) error {
	x, e := v.record(r)
	if e != nil {
		return e
	}
	for i := uint32(0); i < x.length; i++ {
		w, e := v.Read(r, i)
		if e != nil {
			return e
		}
		if i != 0 {
			if e := s.AppendCString(" "); e != nil {
				return e
			}
		}
		switch w.Kind() {
		case 0:
			l, e := w.Literal()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if w.LearnedUnchecked() {
				if e = s.AppendCString("*"); e != nil {
					return e
				}
			}
		case 1:
			if e := s.AppendCString("("); e != nil {
				return e
			}
			l, e := w.Literal1()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(" "); e != nil {
				return e
			}
			l, e = w.Literal2()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(")"); e != nil {
				return e
			}
		case 2:
			if e := s.AppendCString("("); e != nil {
				return e
			}
			l, e := w.BlockedLiteral()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(" "); e != nil {
				return e
			}
			o, e := w.ClauseOffset()
			if e != nil {
				return e
			}
			r, e := a.GetClause(o)
			if e != nil {
				return e
			}
			if e = a.DisplayInto(r, s); e != nil {
				return e
			}
			if e = s.AppendCString(")"); e != nil {
				return e
			}
		case 3:
			n, e := w.ExternalIndex()
			if e != nil {
				return e
			}
			if e = s.AppendUint32(n); e != nil {
				return e
			}
		}
	}
	return nil
}

// SortByKind stably orders the logical range with the source comparator.
func (s *satSolverWatchStore) SortByKind(r satSolverWatchListRef) error {
	return s.sortByKindWithWork(r, nil)
}

func satWatchPayloadExtent(capacity uint32) (uint32, uint32, error) {
	n := uint64(capacity) * 4
	if n > 1<<31 {
		return 0, 0, arena.ErrBounds
	}
	return uint32(n), uint32((uint64(capacity) + 31) / 32), nil
}
func satWatchDirectoryCapacity(old uint64) (uint64, error) {
	// Cold ledger growth follows the source outer watch_list vector's selected
	// LP64 eight-byte element progression, but charges actual scalar records.
	return satVectorNextCapacityFor(old, uint64(reflect.TypeOf(satSolverWatchListRecord{}).Size()), 8)
}
func (s *satSolverWatchStore) NewListFill(n uint32, w satWatched) (satSolverWatchListRef, error) {
	r, e := s.NewList()
	if e != nil {
		return r, e
	}
	if e = s.ResizeFill(r, n, w); e != nil {
		_ = s.Free(r)
		return satSolverWatchListRef{}, e
	}
	return r, nil
}
func (s *satSolverWatchStore) NewListArray(xs []satWatched) (satSolverWatchListRef, error) {
	r, e := s.NewList()
	if e != nil {
		return r, e
	}
	if e = s.AppendArray(r, xs); e != nil {
		_ = s.Free(r)
		return satSolverWatchListRef{}, e
	}
	return r, nil
}

func (s *satSolverWatchStore) GetDefault(r satSolverWatchListRef, i uint32, d satWatched) (satWatched, error) {
	x, e := s.record(r)
	if e != nil {
		return d, e
	}
	if i >= x.length {
		return d, nil
	}
	return s.Read(r, i)
}
func (s *satSolverWatchStore) Back(r satSolverWatchListRef) (satWatched, error) {
	x, e := s.record(r)
	if e != nil {
		return satWatched{}, e
	}
	if satStorageSourceDebug && x.length == 0 {
		return satWatched{}, ErrSATAssertion
	}
	return s.Read(r, x.length-1)
}
func (s *satSolverWatchStore) EraseTernaryWatch(r satSolverWatchListRef, a, b satLiteral) error {
	w, e := newSATTernaryWatch(a, b)
	if e != nil {
		return e
	}
	_, e = s.Erase(r, w)
	return e
}
