// Licensed under the MIT license.
// Explicit host64 size_t storage adapter for clauses wider than SmallBytes.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type satLargeClauseEntry struct {
	data             []uint32
	generation, next uint32
	live, retired    bool
}

func (a *satClauseAllocator) allocateLarge(bytes uint64) (satStoredClauseRef, error) {
	if e := satClauseHostExtent(bytes, uint64(^uint(0)>>1)); e != nil {
		return satStoredClauseRef{}, e
	}
	if bytes%4 != 0 {
		return satStoredClauseRef{}, arena.ErrBounds
	}
	slot := a.largeFree
	newSlot := slot == 0
	if newSlot {
		if uint64(len(a.large)) >= uint64(^uint32(0)) {
			return satStoredClauseRef{}, arena.ErrSlotLimit
		}
		slot = uint32(len(a.large))
	} else {
		slot--
	}
	var dir []satLargeClauseEntry
	dirBytes := uint64(0)
	if newSlot && len(a.large) == cap(a.large) {
		n := uint64(cap(a.large)) * 2
		if n == 0 {
			n = 1
		}
		stride := uint64(reflect.TypeOf(satLargeClauseEntry{}).Size())
		if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/stride {
			return satStoredClauseRef{}, arena.ErrBounds
		}
		dirBytes = n * stride
		if e := a.budget.Reserve(dirBytes); e != nil {
			return satStoredClauseRef{}, e
		}
		dir = make([]satLargeClauseEntry, len(a.large), int(n))
		copy(dir, a.large)
	}
	if e := a.budget.Reserve(bytes); e != nil {
		a.budget.Release(dirBytes)
		return satStoredClauseRef{}, e
	}
	data := make([]uint32, int(bytes/4))
	if dir != nil {
		a.budget.Release(a.largeDirectory)
		a.largeDirectory = dirBytes
		a.large = dir
	}
	if newSlot {
		a.large = append(a.large, satLargeClauseEntry{generation: 1})
	} else {
		a.largeFree = a.large[slot].next
	}
	entry := &a.large[slot]
	entry.next = 0
	entry.live = true
	entry.data = data
	a.largeBytes += bytes
	return satStoredClauseRef{a.origin, slot, entry.generation, 1}, nil
}
func (a *satClauseAllocator) releaseLarge(r satStoredClauseRef) error {
	if e := a.validate(r); e != nil {
		return e
	}
	v := &a.large[r.slot]
	bytes := uint64(len(v.data)) * 4
	a.budget.Release(bytes)
	a.largeBytes -= bytes
	v.data = nil
	v.live = false
	if v.generation == ^uint32(0) {
		v.retired = true
		v.next = 0
	} else {
		v.generation++
		v.next = a.largeFree
		a.largeFree = r.slot + 1
	}
	return nil
}
