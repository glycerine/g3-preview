// Source: src/util/scoped_vector.h:23-164.
package util

import "math"

// ScopedVector is a copy-on-write vector with nested rollback scopes. The
// source class is templated; this port stores concrete consumer values as any
// to avoid Go generics.
type ScopedVector struct {
	size       uint32
	elemsStart uint32
	sizes      []uint32
	elems      []any
	elemsLim   []uint32
	index      []uint32
	src        []uint32
	dst        []uint32
	srcLim     []uint32
}

// NewScopedVector constructs an empty vector.
func NewScopedVector() *ScopedVector { return &ScopedVector{} }

// PushScope records the current logical and physical boundaries.
func (vector *ScopedVector) PushScope() {
	vector.requireOwner()
	vector.elemsStart = uint32(len(vector.elems))
	vector.sizes = append(vector.sizes, vector.size)
	vector.srcLim = append(vector.srcLim, uint32(len(vector.src)))
	vector.elemsLim = append(vector.elemsLim, vector.elemsStart)
}

// PopScope restores numScopes nested boundaries in one operation.
func (vector *ScopedVector) PopScope(numScopes uint32) {
	vector.requireOwner()
	if numScopes == 0 {
		return
	}
	if uint64(numScopes) > uint64(len(vector.sizes)) {
		panic("g3: scoped-vector scope underflow")
	}
	newSize := len(vector.sizes) - int(numScopes)
	srcLimit := int(vector.srcLim[newSize])
	for i := len(vector.src); i > srcLimit; {
		i--
		vector.index[vector.src[i]] = vector.dst[i]
	}
	vector.src = vector.src[:srcLimit]
	vector.dst = vector.dst[:srcLimit]
	vector.srcLim = vector.srcLim[:newSize]

	elemLimit := int(vector.elemsLim[newSize])
	clear(vector.elems[elemLimit:])
	vector.elems = vector.elems[:elemLimit]
	vector.elemsLim = vector.elemsLim[:newSize]
	vector.elemsStart = uint32(len(vector.elems))

	vector.size = vector.sizes[newSize]
	vector.sizes = vector.sizes[:newSize]
	if !vector.CheckInvariant() {
		panic("g3: scoped-vector invariant after pop")
	}
}

// At returns the logical element at index.
func (vector *ScopedVector) At(index uint32) any {
	vector.requireIndex(index)
	return vector.elems[vector.index[index]]
}

// Ref returns direct mutable access to the current physical element. Like the
// source ref method, it bypasses copy-on-write; callers must arrange any needed
// rollback themselves and must not retain the pointer across structural edits.
func (vector *ScopedVector) Ref(index uint32) *any {
	vector.requireIndex(index)
	return &vector.elems[vector.index[index]]
}

// Set replaces a logical element, making at most one physical copy in the
// current scope.
func (vector *ScopedVector) Set(index uint32, value any) {
	vector.requireIndex(index)
	physical := vector.index[index]
	if physical >= vector.elemsStart {
		vector.elems[physical] = value
	} else {
		vector.setIndex(index, uint32(len(vector.elems)))
		vector.elems = append(vector.elems, value)
	}
	if !vector.CheckInvariant() {
		panic("g3: scoped-vector invariant after set")
	}
}

// PushBack appends a logical element.
func (vector *ScopedVector) PushBack(value any) {
	vector.requireOwner()
	if vector.size == math.MaxUint32 || uint64(len(vector.elems)) >= uint64(math.MaxUint32) {
		panic("g3: scoped-vector size overflow")
	}
	vector.setIndex(vector.size, uint32(len(vector.elems)))
	vector.elems = append(vector.elems, value)
	vector.size++
	if !vector.CheckInvariant() {
		panic("g3: scoped-vector invariant after push")
	}
}

// PopBack removes the final logical element.
func (vector *ScopedVector) PopBack() {
	vector.requireOwner()
	if vector.size == 0 {
		panic("g3: scoped-vector pop from empty vector")
	}
	physical := vector.index[vector.size-1]
	if int(physical) == len(vector.elems)-1 && uint32(len(vector.elems)) > vector.elemsStart {
		vector.elems[len(vector.elems)-1] = nil
		vector.elems = vector.elems[:len(vector.elems)-1]
	}
	vector.size--
	if !vector.CheckInvariant() {
		panic("g3: scoped-vector invariant after pop")
	}
}

// EraseAndSwap replaces index with the last value and removes the last slot.
func (vector *ScopedVector) EraseAndSwap(index uint32) {
	vector.requireIndex(index)
	if index+1 < vector.size {
		vector.Set(index, vector.At(vector.size-1))
	}
	vector.PopBack()
}

// Size returns the logical element count.
func (vector *ScopedVector) Size() uint32 {
	vector.requireOwner()
	return vector.size
}

// Empty reports whether the logical vector is empty.
func (vector *ScopedVector) Empty() bool { return vector.Size() == 0 }

// Begin returns an iterator at index zero.
func (vector *ScopedVector) Begin() ScopedVectorIterator {
	vector.requireOwner()
	return ScopedVectorIterator{vector: vector}
}

// End returns an iterator at the current logical end.
func (vector *ScopedVector) End() ScopedVectorIterator {
	vector.requireOwner()
	return ScopedVectorIterator{vector: vector, index: vector.size}
}

func (vector *ScopedVector) setIndex(src, dst uint32) {
	for uint64(src) >= uint64(len(vector.index)) {
		vector.index = append(vector.index, 0)
	}
	if src < vector.elemsStart {
		vector.src = append(vector.src, src)
		vector.dst = append(vector.dst, vector.index[src])
	}
	vector.index[src] = dst
}

func (vector *ScopedVector) requireOwner() {
	if vector == nil {
		panic("g3: nil scoped vector")
	}
}

func (vector *ScopedVector) requireIndex(index uint32) {
	vector.requireOwner()
	if index >= vector.size {
		panic("g3: scoped-vector index out of bounds")
	}
}

// CheckInvariant validates every retained index and scope boundary.
func (vector *ScopedVector) CheckInvariant() bool {
	if vector == nil || uint64(vector.size) > uint64(len(vector.elems)) || uint64(vector.elemsStart) > uint64(len(vector.elems)) {
		return false
	}
	if len(vector.src) != len(vector.dst) || len(vector.sizes) != len(vector.srcLim) || len(vector.sizes) != len(vector.elemsLim) {
		return false
	}
	if uint64(vector.size) > uint64(len(vector.index)) {
		return false
	}
	for i := uint32(0); i < vector.size; i++ {
		if uint64(vector.index[i]) >= uint64(len(vector.elems)) {
			return false
		}
	}
	previousSrc, previousElem := uint32(0), uint32(0)
	for i := range vector.sizes {
		if vector.srcLim[i] < previousSrc || uint64(vector.srcLim[i]) > uint64(len(vector.src)) || vector.elemsLim[i] < previousElem || uint64(vector.elemsLim[i]) > uint64(len(vector.elems)) {
			return false
		}
		previousSrc, previousElem = vector.srcLim[i], vector.elemsLim[i]
	}
	return true
}

// ScopedVectorIterator is the source scoped_vector::iterator translated to a
// Go cursor. Iterator equality includes owner identity.
type ScopedVectorIterator struct {
	vector *ScopedVector
	index  uint32
}

func (iterator ScopedVectorIterator) Value() any {
	if iterator.vector == nil || iterator.index >= iterator.vector.size {
		panic("g3: dereference of end scoped-vector iterator")
	}
	return iterator.vector.At(iterator.index)
}

func (iterator *ScopedVectorIterator) Advance() {
	if iterator == nil || iterator.vector == nil || iterator.index >= iterator.vector.size {
		panic("g3: advance of end scoped-vector iterator")
	}
	iterator.index++
}

func (iterator *ScopedVectorIterator) PostAdvance() ScopedVectorIterator {
	previous := *iterator
	iterator.Advance()
	return previous
}

func (iterator ScopedVectorIterator) Equal(other ScopedVectorIterator) bool {
	return iterator.vector == other.vector && iterator.index == other.index
}

func (iterator ScopedVectorIterator) NotEqual(other ScopedVectorIterator) bool {
	return !iterator.Equal(other)
}
