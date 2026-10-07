// Licensed under the MIT license.
// Source vector.h pointer specializations: actual runtime-backed lexical values.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Only the borrowed runtime pointer is cold. All element IDs are source pointer
// codecs in the actual backing after two source uint32 header words.

type smallObjectPointerVector struct {
	owner          *SmallAllocator
	raw            Block
	size, capacity uint32
}

func (v *smallObjectPointerVector) header(offset uint64, n uint32) error {
	return smallObjectWriteWord(v.owner.config.Runtime, v.raw, offset, 4, uint64(n))
}
func (v *smallObjectPointerVector) grow() error {
	rt := v.owner.config.Runtime
	width := uint32(v.owner.pointerBytes())
	if v.raw == (Block{}) {
		h, e := rt.Allocate(uint64(8 + 2*width))
		if e != nil {
			v.owner.rememberPending(h)
			return e
		}
		v.raw = h
		if e = v.header(0, 2); e != nil {
			return e
		}
		if e = v.header(4, 0); e != nil {
			return e
		}
		v.capacity = 2
		v.size = 0
		return nil
	}
	old := v.capacity
	next := (3*old + 1) >> 1
	oldBytes := old*width + 8
	nextBytes := next*width + 8
	if next <= old || nextBytes <= oldBytes {
		return v.owner.config.Runtime.emitFailure(FailureVectorOverflow)
	}
	h := v.raw
	e := rt.Reallocate(&h, uint64(nextBytes))
	if e != nil {
		if h != v.raw {
			v.owner.rememberPending(h)
			v.raw = Block{}
		}
		return e
	}
	v.raw = h
	if e = v.header(0, next); e != nil {
		return e
	}
	v.capacity = next
	return nil
}
func (v *smallObjectPointerVector) push(id uint32) error {
	if v.size == v.capacity {
		if e := v.grow(); e != nil {
			return e
		}
	}
	if e := smallObjectWriteWord(v.owner.config.Runtime, v.raw, 8+uint64(v.size)*v.owner.pointerBytes(), v.owner.pointerBytes(), uint64(id)); e != nil {
		return e
	}
	v.size++
	return v.header(4, v.size)
}
func (v *smallObjectPointerVector) reset() error {
	if v.raw == (Block{}) {
		return nil
	}
	v.size = 0
	return v.header(4, 0)
}
func (v *smallObjectPointerVector) get(i uint32) (uint32, error) {
	if i >= v.size {
		return 0, arena.ErrBounds
	}
	n, e := smallObjectReadWord(v.owner.config.Runtime, v.raw, 8+uint64(i)*v.owner.pointerBytes(), v.owner.pointerBytes())
	if e != nil {
		return 0, e
	}
	if n > uint64(^uint32(0)) {
		return 0, ErrAllocatorOwner
	}
	return uint32(n), nil
}
func (v *smallObjectPointerVector) set(i, n uint32) error {
	if i >= v.size {
		return arena.ErrBounds
	}
	return smallObjectWriteWord(v.owner.config.Runtime, v.raw, 8+uint64(i)*v.owner.pointerBytes(), v.owner.pointerBytes(), uint64(n))
}
func (v *smallObjectPointerVector) close() error {
	if v.raw == (Block{}) {
		return nil
	}
	e := v.owner.config.Runtime.Deallocate(&v.raw)
	if e != nil && v.raw != (Block{}) {
		v.owner.retainScratch(v.raw)
		v.raw = Block{}
	}
	return e
}
func (v *smallObjectPointerVector) less(i, j uint32) (bool, error) {
	x, e := v.get(i)
	if e != nil {
		return false, e
	}
	y, e := v.get(j)
	if e != nil {
		return false, e
	}
	a, e := v.owner.location(x)
	if e != nil {
		return false, e
	}
	b, e := v.owner.location(y)
	if e != nil {
		return false, e
	}
	return a.address < b.address, nil
}
func (v *smallObjectPointerVector) swap(i, j uint32) error {
	a, e := v.get(i)
	if e != nil {
		return e
	}
	b, e := v.get(j)
	if e != nil {
		return e
	}
	if e = v.set(i, b); e != nil {
		return e
	}
	return v.set(j, a)
}
func (v *smallObjectPointerVector) sift(root, n uint32) error {
	for uint64(root)*2+1 < uint64(n) {
		child := root*2 + 1
		if child+1 < n {
			yes, e := v.less(child, child+1)
			if e != nil {
				return e
			}
			if yes {
				child++
			}
		}
		yes, e := v.less(root, child)
		if e != nil {
			return e
		}
		if !yes {
			return nil
		}
		if e = v.swap(root, child); e != nil {
			return e
		}
		root = child
	}
	return nil
}

// std::sort is external: concrete no-scratch heapsort preserves the selected
// deterministic scalar-address ordering, not C++ host pointer placements.
func (v *smallObjectPointerVector) sort() error {
	if v.size < 2 {
		return nil
	}
	for i := v.size / 2; i > 0; i-- {
		if e := v.sift(i-1, v.size); e != nil {
			return e
		}
	}
	for n := v.size; n > 1; n-- {
		if e := v.swap(0, n-1); e != nil {
			return e
		}
		if e := v.sift(0, n-1); e != nil {
			return e
		}
	}
	return nil
}
