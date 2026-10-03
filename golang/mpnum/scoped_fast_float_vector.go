// Copyright (c) 2011 Microsoft Corporation. Concrete scoped_numeral_vector.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type ScopedFastFloatVector struct {
	manager  *FastFloatManager
	values   []FastFloat
	capacity uint32
	self     *ScopedFastFloatVector
	closed   bool
}

// Source vector's 32-bit sizeof(mpff)*capacity+8 header bound is separate
// from the actual Go (widened in checked builds) backing/host-int bound.
func ffVectorAllocation(count, stride, hostMax uint64) (uint64, error) {
	if count > uint64(^uint32(0)) || count > (uint64(^uint32(0))-8)/8 || stride == 0 || count > hostMax/stride {
		return 0, arena.ErrBounds
	}
	return count * stride, nil
}

func NewScopedFastFloatVector(m *FastFloatManager) (*ScopedFastFloatVector, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if e := m.config.Storage.Descriptors.Budget.Reserve(uint64(reflect.TypeOf(ScopedFastFloatVector{}).Size())); e != nil {
		return nil, e
	}
	v := &ScopedFastFloatVector{manager: m}
	v.self = v
	return v, nil
}
func (v *ScopedFastFloatVector) check() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return arena.ErrClosed
	}
	return v.manager.open()
}
func (v *ScopedFastFloatVector) Manager() (*FastFloatManager, error) {
	if e := v.check(); e != nil {
		return nil, e
	}
	return v.manager, nil
}
func (v *ScopedFastFloatVector) Len() (uint32, error) {
	if e := v.check(); e != nil {
		return 0, e
	}
	return uint32(len(v.values)), nil
}
func (v *ScopedFastFloatVector) Capacity() (uint32, error) {
	if e := v.check(); e != nil {
		return 0, e
	}
	return v.capacity, nil
}
func (v *ScopedFastFloatVector) At(i uint32) (*FastFloat, error) {
	if e := v.check(); e != nil {
		return nil, e
	}
	if uint64(i) >= uint64(len(v.values)) {
		return nil, arena.ErrBounds
	}
	return &v.values[i], nil
}
func (v *ScopedFastFloatVector) Back() (*FastFloat, error) {
	if e := v.check(); e != nil {
		return nil, e
	}
	if len(v.values) == 0 {
		return nil, arena.ErrBounds
	}
	return &v.values[len(v.values)-1], nil
}
func (v *ScopedFastFloatVector) reserve(n uint32) error {
	width := uint64(reflect.TypeOf(FastFloat{}).Size())
	for v.capacity < n {
		c := uint64(2)
		if v.capacity != 0 {
			c = (3*uint64(v.capacity) + 1) / 2
		}
		if c > uint64(^uint32(0)) || c <= uint64(v.capacity) {
			return arena.ErrBounds
		}
		allocation, e := ffVectorAllocation(c, width, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		b := v.manager.config.Storage.Descriptors.Budget
		if e := b.Reserve(allocation); e != nil {
			return e
		}
		data := make([]FastFloat, len(v.values), int(c))
		copy(data, v.values)
		old := uint64(v.capacity) * width
		v.values = data
		v.capacity = uint32(c)
		b.Release(old)
	}
	return nil
}
func (v *ScopedFastFloatVector) Resize(n uint32) error {
	if e := v.check(); e != nil {
		return e
	}
	if n <= uint32(len(v.values)) {
		return v.Shrink(n)
	}
	if e := v.reserve(n); e != nil {
		return e
	}
	old := len(v.values)
	v.values = v.values[:n]
	for i := old; i < len(v.values); i++ {
		v.values[i] = FastFloat{}
	}
	return nil
}
func (v *ScopedFastFloatVector) Push(a *FastFloat) error {
	if e := v.check(); e != nil {
		return e
	}
	if e := v.manager.check(a); e != nil {
		return e
	}
	old := uint32(len(v.values))
	if old == ^uint32(0) {
		return arena.ErrBounds
	}
	if e := v.Resize(old + 1); e != nil {
		return e
	}
	return v.manager.Copy(&v.values[old], a)
}
func (v *ScopedFastFloatVector) Shrink(n uint32) error {
	if e := v.check(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.values)) {
		return arena.ErrBounds
	}
	for i := n; i < uint32(len(v.values)); i++ {
		if e := v.manager.Del(&v.values[i]); e != nil {
			return e
		}
		v.values[i] = FastFloat{}
	}
	v.values = v.values[:n]
	return nil
}
func (v *ScopedFastFloatVector) Pop() error {
	if e := v.check(); e != nil {
		return e
	}
	if len(v.values) == 0 {
		return arena.ErrBounds
	}
	return v.Shrink(uint32(len(v.values) - 1))
}
func (v *ScopedFastFloatVector) Reset() error { return v.Shrink(0) }
func (v *ScopedFastFloatVector) Close() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return nil
	}
	if e := v.Reset(); e != nil {
		return e
	}
	b := v.manager.config.Storage.Descriptors.Budget
	b.Release(uint64(v.capacity) * uint64(reflect.TypeOf(FastFloat{}).Size()))
	b.Release(uint64(reflect.TypeOf(ScopedFastFloatVector{}).Size()))
	v.values = nil
	v.capacity = 0
	v.closed = true
	return nil
}
