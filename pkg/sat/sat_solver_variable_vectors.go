// Licensed under the MIT license.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Source capacity/byte arithmetic is unsigned32, independently of actual Go
// record sizes. A source-permitted wrapped undersized allocation is outside the
// full-capacity typed-storage domain; its raw malloc/header prefix is not ported.
func satVariableNextCapacity(old uint32, actualStride, sourceStride, hostMax uint64) (uint32, uint64, error) {
	n := uint32(2)
	if old != 0 {
		n = (3*old + 1) >> 1
		oldBytes := uint32(sourceStride)*old + 8
		newBytes := uint32(sourceStride)*n + 8
		if n <= old || newBytes <= oldBytes {
			return 0, 0, util.NewDefaultException("Overflow encountered when expanding vector")
		}
	}
	if actualStride == 0 || sourceStride == 0 || uint64(n) > (uint64(^uint32(0))-8)/sourceStride || uint64(n) > hostMax/actualStride {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * actualStride, nil
}
func satVariableIndex(length, capacity, initialized uint32, i uint32) error {
	if satStorageSourceDebug && i >= length {
		return ErrSATAssertion
	}
	if i >= capacity || i >= initialized {
		return arena.ErrBounds
	}
	return nil
}

type satVariableCharColumn struct {
	data        []int8
	initialized uint32
}

func (c *satVariableCharColumn) push(b *arena.Budget, v int8) error {
	if uint64(len(c.data)) == uint64(cap(c.data)) {
		n, bytes, e := satVariableNextCapacity(uint32(cap(c.data)), 1, 1, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = b.Reserve(bytes); e != nil {
			return e
		}
		next := make([]int8, len(c.data), int(n))
		copy(next[:cap(c.data)], c.data[:cap(c.data)])
		b.Release(uint64(cap(c.data)))
		c.data = next
	}
	c.data = append(c.data, v)
	if uint32(len(c.data)) > c.initialized {
		c.initialized = uint32(len(c.data))
	}
	return nil
}
func (c *satVariableCharColumn) read(i uint32) (int8, error) {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return 0, e
	}
	return c.data[:cap(c.data)][i], nil
}
func (c *satVariableCharColumn) write(i uint32, v int8) error {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return e
	}
	c.data[:cap(c.data)][i] = v
	return nil
}
func (c *satVariableCharColumn) shrink(n uint32) error {
	if satStorageSourceDebug && n > uint32(len(c.data)) {
		return ErrSATAssertion
	}
	if n > c.initialized || uint64(n) > uint64(cap(c.data)) {
		return arena.ErrBounds
	}
	c.data = c.data[:n]
	return nil
}
func (c *satVariableCharColumn) finalize(b *arena.Budget) {
	b.Release(uint64(cap(c.data)))
	c.data = nil
	c.initialized = 0
}

type satVariableUintColumn struct {
	data        []uint32
	initialized uint32
}

func (c *satVariableUintColumn) push(b *arena.Budget, v uint32) error {
	if len(c.data) == cap(c.data) {
		n, bytes, e := satVariableNextCapacity(uint32(cap(c.data)), 4, 4, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = b.Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint32, len(c.data), int(n))
		copy(next[:cap(c.data)], c.data[:cap(c.data)])
		b.Release(4 * uint64(cap(c.data)))
		c.data = next
	}
	c.data = append(c.data, v)
	if uint32(len(c.data)) > c.initialized {
		c.initialized = uint32(len(c.data))
	}
	return nil
}
func (c *satVariableUintColumn) read(i uint32) (uint32, error) {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return 0, e
	}
	return c.data[:cap(c.data)][i], nil
}
func (c *satVariableUintColumn) write(i uint32, v uint32) error {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return e
	}
	c.data[:cap(c.data)][i] = v
	return nil
}
func (c *satVariableUintColumn) shrink(n uint32) error {
	if satStorageSourceDebug && n > uint32(len(c.data)) {
		return ErrSATAssertion
	}
	if n > c.initialized || uint64(n) > uint64(cap(c.data)) {
		return arena.ErrBounds
	}
	c.data = c.data[:n]
	return nil
}
func (c *satVariableUintColumn) finalize(b *arena.Budget) {
	b.Release(4 * uint64(cap(c.data)))
	c.data = nil
	c.initialized = 0
}

type satVariableJustificationColumn struct {
	data        []satJustification
	initialized uint32
}

func (c *satVariableJustificationColumn) push(b *arena.Budget, v satJustification) error {
	stride := uint64(reflect.TypeOf(satJustification{}).Size())
	if len(c.data) == cap(c.data) {
		n, bytes, e := satVariableNextCapacity(uint32(cap(c.data)), stride, 8, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = b.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satJustification, len(c.data), int(n))
		copy(next[:cap(c.data)], c.data[:cap(c.data)])
		b.Release(stride * uint64(cap(c.data)))
		c.data = next
	}
	c.data = append(c.data, v)
	if uint32(len(c.data)) > c.initialized {
		c.initialized = uint32(len(c.data))
	}
	return nil
}
func (c *satVariableJustificationColumn) read(i uint32) (satJustification, error) {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return satJustification{}, e
	}
	return c.data[:cap(c.data)][i], nil
}
func (c *satVariableJustificationColumn) write(i uint32, v satJustification) error {
	if e := satVariableIndex(uint32(len(c.data)), uint32(cap(c.data)), c.initialized, i); e != nil {
		return e
	}
	c.data[:cap(c.data)][i] = v
	return nil
}
func (c *satVariableJustificationColumn) finalize(b *arena.Budget) {
	b.Release(uint64(reflect.TypeOf(satJustification{}).Size()) * uint64(cap(c.data)))
	c.data = nil
	c.initialized = 0
}
