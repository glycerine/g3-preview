// Source: src/util/mpz.cpp array gcd; src/util/mpzzp.h array gcd. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

// A call-local concrete borrowed view, never a duplicated owning numeral array.
// The slice route retains the existing API's aliasing; the codec route loads
// only the source-reached numeral field from an actual Runtime backing.
type integerArrayView struct {
	allocator      *memory.SmallAllocator
	object         memory.ObjectRef
	values         []Int
	runtime        *memory.Runtime
	raw            memory.Block
	count          uint32
	offset, stride uint64
}

func (v integerArrayView) at(i uint32) (Int, error) {
	if i >= v.count {
		return Int{}, arena.ErrBounds
	}
	if v.runtime == nil && v.allocator == nil {
		if uint64(i) >= uint64(len(v.values)) {
			return Int{}, arena.ErrBounds
		}
		return v.values[i], nil
	}
	if v.stride != 0 && uint64(i) > (math.MaxUint64-v.offset)/v.stride {
		return Int{}, arena.ErrBounds
	}
	o := v.offset + uint64(i)*v.stride
	if o > math.MaxUint64-12 {
		return Int{}, arena.ErrBounds
	}
	var value, lo, hi uint32
	var e error
	if v.allocator != nil {
		value, e = v.allocator.ReadUint32(v.object, o)
	} else {
		value, e = v.runtime.ReadUint32(v.raw, o)
	}
	if e != nil {
		return Int{}, e
	}
	if v.allocator != nil {
		lo, e = v.allocator.ReadUint32(v.object, o+4)
	} else {
		lo, e = v.runtime.ReadUint32(v.raw, o+4)
	}
	if e != nil {
		return Int{}, e
	}
	if v.allocator != nil {
		hi, e = v.allocator.ReadUint32(v.object, o+8)
	} else {
		hi, e = v.runtime.ReadUint32(v.raw, o+8)
	}
	if e != nil {
		return Int{}, e
	}
	ref := uint64(lo) | uint64(hi)<<32
	return Int{val: int32(value), cell: numericCellRef(ref)}, nil
}
func (v integerArrayView) reference(i uint32, local *Int) (*Int, error) {
	if i >= v.count {
		return nil, arena.ErrBounds
	}
	if v.runtime == nil && v.allocator == nil {
		if uint64(i) >= uint64(len(v.values)) {
			return nil, arena.ErrBounds
		}
		return &v.values[i], nil
	}
	a, e := v.at(i)
	if e != nil {
		return nil, e
	}
	*local = a
	return local, nil
}
func (m *IntegerManager) GCDManyIndexed(as integerArrayView, g *Int) error {
	if e := m.check(g); e != nil {
		return e
	}
	var a, b Int
	switch as.count {
	case 0:
		return m.Reset(g)
	case 1:
		pa, e := as.referenceWithAllocatorOwner(m.memory, 0, &a)
		if e != nil {
			return e
		}
		if e = m.Copy(g, pa); e != nil {
			return e
		}
		return m.Abs(g)
	}
	pa, e := as.referenceWithAllocatorOwner(m.memory, 0, &a)
	if e != nil {
		return e
	}
	pb, e := as.referenceWithAllocatorOwner(m.memory, 1, &b)
	if e != nil {
		return e
	}
	if e = m.GCD(pa, pb, g); e != nil {
		return e
	}
	for i := uint32(2); i < as.count; i++ {
		if isOneInteger(g) {
			return nil
		}
		pa, e = as.referenceWithAllocatorOwner(m.memory, i, &a)
		if e != nil {
			return e
		}
		if e = m.GCD(g, pa, g); e != nil {
			return e
		}
	}
	return nil
}
func (m *IntegerModularManager) GCDManyIndexed(as integerArrayView, g *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.GCDManyIndexed(as, g); e != nil {
		return e
	}
	return m.assertNormalized(g)
}

// checkedAt validates the borrowed cohort only when this numeral is reached.
func (v integerArrayView) checkedAt(owner *memory.Runtime, i uint32) (Int, error) {
	if v.allocator != nil && v.allocator.RuntimeOwner() != owner {
		return Int{}, memory.ErrRuntimeOwner
	}
	if v.runtime != nil && v.runtime != owner {
		return Int{}, memory.ErrRuntimeOwner
	}
	return v.at(i)
}
func (v integerArrayView) sameRuntimeElement(owner *memory.Runtime, raw memory.Block, offset uint64, i uint32) bool {
	return v.allocator == nil && v.runtime == owner && v.raw == raw && v.offset+uint64(i)*v.stride == offset && (v.stride == 0 || uint64(i) <= (math.MaxUint64-v.offset)/v.stride)
}
func (v integerArrayView) store(i uint32, a Int) error {
	if i >= v.count {
		return arena.ErrBounds
	}
	if v.runtime == nil && v.allocator == nil {
		if uint64(i) >= uint64(len(v.values)) {
			return arena.ErrBounds
		}
		v.values[i] = a
		return nil
	}
	if v.stride != 0 && uint64(i) > (math.MaxUint64-v.offset)/v.stride {
		return arena.ErrBounds
	}
	o := v.offset + uint64(i)*v.stride
	if o > math.MaxUint64-12 {
		return arena.ErrBounds
	}
	words := [3]uint32{uint32(a.val), uint32(a.cell), uint32(uint64(a.cell) >> 32)}
	for j, w := range words {
		var e error
		if v.allocator != nil {
			e = v.allocator.WriteUint32(v.object, o+uint64(j)*4, w)
		} else {
			e = v.runtime.WriteUint32(v.raw, o+uint64(j)*4, w)
		}
		if e != nil {
			return e
		}
	}
	return nil
}

// Only a source-reached coefficient requires the borrowed object allocator.
func (v integerArrayView) referenceWithAllocatorOwner(owner *memory.Runtime, i uint32, local *Int) (*Int, error) {
	if v.runtime != nil && v.runtime != owner {
		return nil, memory.ErrRuntimeOwner
	}
	if v.allocator != nil && v.allocator.RuntimeOwner() != owner {
		return nil, memory.ErrRuntimeOwner
	}
	return v.reference(i, local)
}
