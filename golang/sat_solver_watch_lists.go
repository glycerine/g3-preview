// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Scalar tokens represent the actual moved inline list states. One store owns
// all child payloads; relocation copies tokens without duplicating ownership.
type satSolverWatchLists struct {
	self                                *satSolverWatchLists
	store                               *satSolverWatchStore
	budget                              *arena.Budget
	lists                               []satSolverWatchListRef
	ownerBytes, backingBytes, backingID uint64
	closed                              bool
}
type satSolverWatchListsStats struct {
	OwnerBytes, BackingBytes, StoreBytes, TotalBytes uint64
	Size, Capacity                                   uint32
}

func newSATSolverWatchLists(b *arena.Budget, c arena.VectorConfig) (*satSolverWatchLists, error) {
	if c.Descriptors.Budget != b {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(satSolverWatchLists{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	store, e := newSATSolverWatchStore(c)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	w := &satSolverWatchLists{store: store, budget: b, ownerBytes: n, backingID: 1}
	w.self = w
	return w, nil
}
func (w *satSolverWatchLists) open() error {
	if w == nil || w.self != w {
		return ErrSATOwner
	}
	if w.closed {
		return arena.ErrClosed
	}
	return nil
}
func (w *satSolverWatchLists) PushEmpty() error {
	if e := w.open(); e != nil {
		return e
	}
	if len(w.lists) == cap(w.lists) {
		if w.backingID == ^uint64(0) {
			return arena.ErrSlotLimit
		}
		n, bytes, e := satVariableNextCapacity(uint32(cap(w.lists)), uint64(reflect.TypeOf(satSolverWatchListRef{}).Size()), 8, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = w.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satSolverWatchListRef, len(w.lists), int(n))
		copy(next, w.lists)
		w.budget.Release(w.backingBytes)
		w.lists = next
		w.backingBytes = bytes
		w.backingID++
	}
	// The empty temporary source watch_list exists before push without any heap
	// action. Descriptor placement here is an explicit charged Go metadata action.
	r, e := w.store.NewList()
	if e != nil {
		return e
	}
	w.lists = append(w.lists, r)
	return nil
}
func (w *satSolverWatchLists) Stats() satSolverWatchListsStats {
	if w == nil || w.closed {
		return satSolverWatchListsStats{}
	}
	store := w.store.Stats().ChargedBytes
	return satSolverWatchListsStats{w.ownerBytes, w.backingBytes, store, w.ownerBytes + w.backingBytes + store, uint32(len(w.lists)), uint32(cap(w.lists))}
}
func (w *satSolverWatchLists) Get(i uint32) (satSolverWatchListView, error) {
	if e := w.open(); e != nil {
		return satSolverWatchListView{}, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(w.lists)) {
		return satSolverWatchListView{}, ErrSATAssertion
	}
	if uint64(i) >= uint64(len(w.lists)) {
		return satSolverWatchListView{}, arena.ErrBounds
	}
	return satSolverWatchListView{w, i, w.backingID}, nil
}

type satSolverWatchListView struct {
	owner     *satSolverWatchLists
	index     uint32
	backingID uint64
}

func (v satSolverWatchListView) ref() (satSolverWatchListRef, error) {
	if e := v.owner.open(); e != nil {
		return satSolverWatchListRef{}, e
	}
	if v.backingID != v.owner.backingID {
		return satSolverWatchListRef{}, ErrSATOwner
	}
	if uint64(v.index) >= uint64(len(v.owner.lists)) {
		return satSolverWatchListRef{}, arena.ErrBounds
	}
	r := v.owner.lists[v.index]
	if !v.owner.store.IsLive(r) {
		return satSolverWatchListRef{}, arena.ErrInvalidSlot
	}
	return r, nil
}
func (v satSolverWatchListView) Size() (uint32, error) {
	r, e := v.ref()
	if e != nil {
		return 0, e
	}
	return v.owner.store.Size(r)
}
func (v satSolverWatchListView) Push(w satWatched) error {
	r, e := v.ref()
	if e != nil {
		return e
	}
	return v.owner.store.Push(r, w)
}
func (v satSolverWatchListView) Read(i uint32) (satWatched, error) {
	r, e := v.ref()
	if e != nil {
		return satWatched{}, e
	}
	return v.owner.store.Read(r, i)
}
func (v satSolverWatchListView) Write(i uint32, w satWatched) error {
	r, e := v.ref()
	if e != nil {
		return e
	}
	return v.owner.store.Write(r, i, w)
}
func (v satSolverWatchListView) Finalize() error {
	r, e := v.ref()
	if e != nil {
		return e
	}
	return v.owner.store.Finalize(r)
}
func (v satSolverWatchListView) SetEnd(n uint32) error {
	r, e := v.ref()
	if e != nil {
		return e
	}
	return v.owner.store.SetEnd(r, n)
}
func (w *satSolverWatchLists) Finalize() error {
	if e := w.open(); e != nil {
		return e
	}
	if w.backingID == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	for i := range w.lists {
		if w.lists[i] != (satSolverWatchListRef{}) {
			if e := w.store.Free(w.lists[i]); e != nil {
				return e
			}
			w.lists[i] = satSolverWatchListRef{}
		}
	}
	w.budget.Release(w.backingBytes)
	w.backingBytes = 0
	w.lists = nil
	w.backingID++
	return nil
}
func (w *satSolverWatchLists) Close() error {
	if w == nil || w.self != w {
		return ErrSATOwner
	}
	if w.closed {
		return nil
	}
	if e := w.Finalize(); e != nil {
		return e
	}
	if e := w.store.Close(); e != nil {
		return e
	}
	w.budget.Release(w.ownerBytes)
	w.ownerBytes = 0
	w.closed = true
	return nil
}
