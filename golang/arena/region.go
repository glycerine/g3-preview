// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Typed storage adaptation of util/region.h/.cpp; cleanup dispatch order follows
// util/trail.h::undo_trail_stack. Source trail subclass algorithms are separate.
package arena

import (
	"errors"
	"math"
	"reflect"
)

var (
	ErrMark      = errors.New("arena: invalid, foreign, consumed or non-top region mark")
	ErrReentrant = errors.New("arena: mutation during region cleanup")
)

type RegionRef struct {
	Slot       Slot
	Generation Generation
}
type regionIdentity struct{ nonzero byte }

// RegionMark is a cold opaque owner-bound token. Only scalar mark metadata is
// stored in the region. It cannot be used in pointer-free hot record pages.
type RegionMark struct {
	owner  *regionIdentity
	serial uint64
}
type regionMarkState struct {
	serial uint64
	length uint32
}

// Cleanup is a transient caller dispatcher, never stored per record or mark.
// It must not panic or mutate the region. Reads are allowed; all rollback
// records remain live through every cleanup callback. Nil means caller cleanup
// already ran. Callers are responsible for source reference releases and undo.
type Cleanup[T any] func(RegionRef, T)
type RegionStats struct {
	Storage             Stats
	Live, Scopes        uint32
	LogBytes, MarkBytes uint64
}

// Region appends typed records and supports nested marked rollback. It stores
// scalar handle logs and marks; its Slab retains generation history when reused.
// Regions are owner-local and not safe for concurrent use. No borrowed slices
// or pointers to record storage escape. Individual record recycling is absent.
type Region[T any] struct {
	storage      *Slab[T]
	log          []RegionRef
	marks        []regionMarkState
	identity     *regionIdentity
	serial       uint64
	budget       *Budget
	busy, closed bool
}

func NewRegion[T any](config Config) (*Region[T], error) {
	s, e := NewSlab[T](config)
	if e != nil {
		return nil, e
	}
	return &Region[T]{storage: s, budget: config.Budget, identity: &regionIdentity{1}}, nil
}
func (r *Region[T]) mutationError() error {
	if r.closed {
		return ErrClosed
	}
	if r.busy {
		return ErrReentrant
	}
	return nil
}

// reserveRegionLog reserves old+new directory capacity before copying; on a
// predictable failure it leaves the slice and budget untouched. Cold region
// owner headers, caller scratch and cleanup reserves are budgeted by the owner.
func reserveRegionLog[E any](data []E, budget *Budget) ([]E, error) {
	if len(data) < cap(data) {
		return data, nil
	}
	capacity := cap(data)
	if capacity == 0 {
		capacity = 1
	} else {
		if capacity > int(^uint(0)>>1)/2 {
			return data, ErrSlotLimit
		}
		capacity *= 2
	}
	elementBytes := uint64(reflect.TypeFor[E]().Size())
	if uint64(capacity) > uint64(^uint(0)>>1)/elementBytes {
		return data, ErrSlotLimit
	}
	amount := uint64(capacity) * elementBytes
	if e := budget.Reserve(amount); e != nil {
		return data, e
	}
	replacement := make([]E, len(data), capacity)
	copy(replacement, data)
	budget.Release(uint64(cap(data)) * elementBytes)
	return replacement, nil
}

func (r *Region[T]) Allocate(value T) (RegionRef, error) {
	if e := r.mutationError(); e != nil {
		return RegionRef{}, e
	}
	if uint64(len(r.log)) == math.MaxUint32 {
		return RegionRef{}, ErrSlotLimit
	}
	log, e := reserveRegionLog(r.log, r.budget)
	if e != nil {
		return RegionRef{}, e
	}
	r.log = log
	slot, gen, e := r.storage.Allocate()
	if e != nil {
		return RegionRef{}, e
	}
	r.storage.Write(slot, value)
	ref := RegionRef{slot, gen}
	r.log = append(r.log, ref)
	return ref, nil
}
func (r *Region[T]) IsLive(ref RegionRef) bool {
	return !r.closed && r.storage.IsLive(ref.Slot, ref.Generation)
}
func (r *Region[T]) Read(ref RegionRef) (T, error) {
	var zero T
	if r.closed {
		return zero, ErrClosed
	}
	if !r.IsLive(ref) {
		return zero, ErrInvalidSlot
	}
	return r.storage.Read(ref.Slot)
}
func (r *Region[T]) Write(ref RegionRef, value T) error {
	if e := r.mutationError(); e != nil {
		return e
	}
	if !r.IsLive(ref) {
		return ErrInvalidSlot
	}
	return r.storage.Write(ref.Slot, value)
}

func (r *Region[T]) Mark() (RegionMark, error) {
	if e := r.mutationError(); e != nil {
		return RegionMark{}, e
	}
	if r.serial == math.MaxUint64 || uint64(len(r.marks)) == math.MaxUint32 {
		return RegionMark{}, ErrSlotLimit
	}
	marks, e := reserveRegionLog(r.marks, r.budget)
	if e != nil {
		return RegionMark{}, e
	}
	r.serial++
	r.marks = append(marks, regionMarkState{r.serial, uint32(len(r.log))})
	return RegionMark{r.identity, r.serial}, nil
}
func (r *Region[T]) Rollback(mark RegionMark, cleanup Cleanup[T]) error {
	if e := r.mutationError(); e != nil {
		return e
	}
	if mark.owner != r.identity || len(r.marks) == 0 || r.marks[len(r.marks)-1].serial != mark.serial {
		return ErrMark
	}
	return r.rollbackTo(r.marks[len(r.marks)-1].length, len(r.marks)-1, cleanup)
}

// PopScopes explicitly permits popping multiple nested scopes. Ordinary
// Rollback only accepts the top mark, guarding nested scratch owners.
func (r *Region[T]) PopScopes(count uint32, cleanup Cleanup[T]) error {
	if e := r.mutationError(); e != nil {
		return e
	}
	if uint64(count) > uint64(len(r.marks)) {
		return ErrMark
	}
	if count == 0 {
		return nil
	}
	keep := len(r.marks) - int(count)
	return r.rollbackTo(r.marks[keep].length, keep, cleanup)
}
func (r *Region[T]) rollbackTo(length uint32, scopes int, cleanup Cleanup[T]) error {
	r.busy = true
	defer func() { r.busy = false }()
	// No storage mutation before cleanup completes. A callback panic preserves
	// storage and mark state, but external callback effects cannot be undone;
	// retrying cleanup after panic is not guaranteed safe.
	if cleanup != nil {
		for i := len(r.log); i > int(length); i-- {
			ref := r.log[i-1]
			value, _ := r.storage.Read(ref.Slot)
			cleanup(ref, value)
		}
	}
	for i := len(r.log); i > int(length); i-- {
		r.storage.Recycle(r.log[i-1].Slot)
	}
	clear(r.log[int(length):])
	r.log = r.log[:int(length)]
	clear(r.marks[scopes:])
	r.marks = r.marks[:scopes]
	return nil
}

// Reset additionally undoes/cleans records before the first mark (level zero),
// matching trail_stack::reset's explicit base-level undo requirement.
func (r *Region[T]) Reset(cleanup Cleanup[T]) error {
	if e := r.mutationError(); e != nil {
		return e
	}
	return r.rollbackTo(0, 0, cleanup)
}
func (r *Region[T]) TrimFreePages() error {
	if e := r.mutationError(); e != nil {
		return e
	}
	r.storage.TrimFreePages()
	return nil
}
func (r *Region[T]) Close(cleanup Cleanup[T]) error {
	if r.closed {
		return nil
	}
	if r.busy {
		return ErrReentrant
	}
	if e := r.Reset(cleanup); e != nil {
		return e
	}
	r.storage.Close()
	r.budget.Release(uint64(cap(r.log))*uint64(reflect.TypeFor[RegionRef]().Size()) + uint64(cap(r.marks))*uint64(reflect.TypeFor[regionMarkState]().Size()))
	r.log = nil
	r.marks = nil
	r.closed = true
	return nil
}
func (r *Region[T]) Stats() RegionStats {
	return RegionStats{Storage: r.storage.Stats(), Live: uint32(len(r.log)), Scopes: uint32(len(r.marks)), LogBytes: uint64(cap(r.log)) * uint64(reflect.TypeFor[RegionRef]().Size()), MarkBytes: uint64(cap(r.marks)) * uint64(reflect.TypeFor[regionMarkState]().Size())}
}
