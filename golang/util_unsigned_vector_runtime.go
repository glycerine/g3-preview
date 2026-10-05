// Source: src/util/vector.h:43-125,178-189,215-240,303-310,350-365,389-433,531-546.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// RuntimeUnsignedVector owns actual unsigned svector bytes. Embedded instances
// have no separate header charge; the concrete enclosing owner charges them.
type RuntimeUnsignedVector struct {
	self                              *RuntimeUnsignedVector
	column                            satSolverRuntimeUintColumn
	budget                            *arena.Budget
	header                            uint64
	destructorCause                   error
	constructorCause, assignmentCause error
	copyAcquisitions                  *runtimeUnsignedCopyAcquisition
	copySerial                        uint64
	copyCleanupCause                  error
	assignmentDestroyed, constructed  bool
	sourceDone, closed                bool
}

func (v *RuntimeUnsignedVector) bind(r *memory.Runtime, b *arena.Budget) {
	v.self = v
	v.budget = b
	v.column.bind(r)
}
func NewRuntimeUnsignedVector(r *memory.Runtime, b *arena.Budget) (*RuntimeUnsignedVector, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(RuntimeUnsignedVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &RuntimeUnsignedVector{header: n, constructed: true}
	v.bind(r, b)
	return v, nil
}
func (v *RuntimeUnsignedVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed || !v.constructed || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	return v.column.raw.open()
}
func (v *RuntimeUnsignedVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.column.size()
}
func (v *RuntimeUnsignedVector) Capacity() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.column.capacity()
}
func (v *RuntimeUnsignedVector) At(i uint32) (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.column.read(i)
}
func (v *RuntimeUnsignedVector) Set(i, n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.column.write(i, n)
}
func (v *RuntimeUnsignedVector) Push(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.column.push(n)
}
func (v *RuntimeUnsignedVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.column.raw.block == (memory.Block{}) {
		return nil
	}
	return satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 4, 0)
}

// Resize follows source capacity-loop, SIZE publication, ascending placement.
func (v *RuntimeUnsignedVector) Resize(n, fill uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	old, e := v.column.size()
	if e != nil {
		return e
	}
	if n <= old {
		return v.column.shrink(n)
	}
	for {
		cap, e := v.column.capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = v.column.raw.expandStride(4); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 8+4*uint64(i), fill); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeUnsignedVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed || v.sourceDone || !v.constructed {
		return nil
	}
	if v.destructorCause != nil {
		return v.destructorCause
	}
	if v.column.raw.block != (memory.Block{}) {
		if e := v.column.raw.runtime.Deallocate(&v.column.raw.block); e != nil {
			v.destructorCause = e
			return e
		}
	}
	v.sourceDone = true
	return nil
}
func (v *RuntimeUnsignedVector) closeOwning() error {
	if v.assignmentDestroyed {
		v.column.raw.block = memory.Block{}
		v.assignmentDestroyed = false
	}
	if e := v.column.finalize(); e != nil {
		if v.copyCleanupCause == nil {
			v.copyCleanupCause = e
		}
		return e
	}
	if e := v.cleanupCopyAcquisitions(); e != nil {
		return e
	}
	v.budget.Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}
func (v *RuntimeUnsignedVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	if e := v.SourceFinalize(); e != nil {
		return e
	}
	return v.closeOwning()
}
func (v *RuntimeUnsignedVector) CleanupStopped() error {
	if v == nil || v.self != v || v.destructorCause == nil {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	return v.closeOwning()
}
