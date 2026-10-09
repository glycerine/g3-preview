// Source: src/util/small_object_allocator.cpp:172-242; src/util/vector.h.
package util

import "github.com/glycerine/g3/pkg/arena"

// smallObjectPointerVector is temporary consolidation scratch. Allocation IDs
// are scalar values, so ordinary typed slice storage replaces the source vector
// header and raw pointer array.
type smallObjectPointerVector struct {
	owner  *SmallAllocator
	values []uint32
}

func (v *smallObjectPointerVector) push(id uint32) error {
	if uint64(len(v.values)) == uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	v.values = append(v.values, id)
	return nil
}

func (v *smallObjectPointerVector) reset() error {
	v.values = v.values[:0]
	return nil
}

func (v *smallObjectPointerVector) get(i uint32) (uint32, error) {
	if uint64(i) >= uint64(len(v.values)) {
		return 0, arena.ErrBounds
	}
	return v.values[i], nil
}

func (v *smallObjectPointerVector) set(i, n uint32) error {
	if uint64(i) >= uint64(len(v.values)) {
		return arena.ErrBounds
	}
	v.values[i] = n
	return nil
}

func (v *smallObjectPointerVector) close() error {
	v.values = nil
	return nil
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

// The source uses std::sort. This no-scratch heapsort preserves deterministic
// scalar-address ordering without depending on Go map or pointer order.
func (v *smallObjectPointerVector) sort() error {
	n := uint32(len(v.values))
	if n < 2 {
		return nil
	}
	for i := n / 2; i > 0; i-- {
		if e := v.sift(i-1, n); e != nil {
			return e
		}
	}
	for n > 1 {
		if e := v.swap(0, n-1); e != nil {
			return e
		}
		if e := v.sift(0, n-1); e != nil {
			return e
		}
		n--
	}
	return nil
}
