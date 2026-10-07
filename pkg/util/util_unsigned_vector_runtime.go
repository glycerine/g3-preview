// Source: src/util/vector.h:43-125,178-189,215-240,303-310,350-365,389-433,449-451,531-546.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// RuntimeUnsignedVector owns actual unsigned svector bytes. Embedded instances
// have no separate header charge; the concrete enclosing owner charges them.
type RuntimeUnsignedVector struct {
	self                              *RuntimeUnsignedVector
	Column                            RuntimeUintColumn
	budget                            *arena.Budget
	header                            uint64
	destructorCause                   error
	constructorCause, assignmentCause error
	copyAcquisitions                  *runtimeUnsignedCopyAcquisition
	copySerial                        uint64
	copyCleanupCause                  error
	assignmentDestroyed, Constructed  bool
	sourceDone, closed                bool
	growthAcquisitions                *runtimeUnsignedGrowthAcquisition
	growthSerial, growthInvocation    uint64
	growthCause, growthPendingCause   error
	growthExecuting                   bool
}

func (v *RuntimeUnsignedVector) Bind(r *Runtime, b *arena.Budget) {
	v.self = v
	v.budget = b
	v.Column.Bind(r)
}
func NewRuntimeUnsignedVector(r *Runtime, b *arena.Budget) (*RuntimeUnsignedVector, error) {
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
	v := &RuntimeUnsignedVector{header: n, Constructed: true}
	v.Bind(r, b)
	return v, nil
}
func (v *RuntimeUnsignedVector) open() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.growthExecuting {
		return ErrRuntimeUnsignedGrowthExecuting
	}
	if v.closed || !v.Constructed || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	return v.Column.Raw.Open()
}
func (v *RuntimeUnsignedVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.Column.Size()
}
func (v *RuntimeUnsignedVector) Capacity() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.Column.Capacity()
}
func (v *RuntimeUnsignedVector) At(i uint32) (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.Column.Read(i)
}
func (v *RuntimeUnsignedVector) Set(i, n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.Column.Write(i, n)
}
func (v *RuntimeUnsignedVector) Push(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.pushUnsignedSource(n, nil)
}
func (v *RuntimeUnsignedVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.Column.Raw.Block == (Block{}) {
		return nil
	}
	return WriteRuntimeWord(v.Column.Raw.Runtime, v.Column.Raw.Block, 4, 0)
}

// Resize follows source capacity-loop, SIZE publication, ascending placement.
func (v *RuntimeUnsignedVector) Resize(n, fill uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	old, e := v.Column.Size()
	if e != nil {
		return e
	}
	if n <= old {
		return v.Column.Shrink(n)
	}
	for {
		cap, e := v.Column.Capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = v.expandUnsignedSource(nil); e != nil {
			return e
		}
	}
	if e = WriteRuntimeWord(v.Column.Raw.Runtime, v.Column.Raw.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = WriteRuntimeWord(v.Column.Raw.Runtime, v.Column.Raw.Block, 8+4*uint64(i), fill); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeUnsignedVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.growthExecuting {
		return ErrRuntimeUnsignedGrowthExecuting
	}
	if v.closed || v.sourceDone || !v.Constructed {
		return nil
	}
	if v.destructorCause != nil {
		return v.destructorCause
	}
	if v.Column.Raw.Block != (Block{}) {
		if e := v.Column.Raw.Runtime.Deallocate(&v.Column.Raw.Block); e != nil {
			v.destructorCause = e
			return e
		}
	}
	v.sourceDone = true
	return nil
}
func (v *RuntimeUnsignedVector) closeOwning() error {
	if v.assignmentDestroyed {
		v.Column.Raw.Block = Block{}
		v.assignmentDestroyed = false
	}
	if e := v.Column.Finalize(); e != nil {
		if v.copyCleanupCause == nil {
			v.copyCleanupCause = e
		}
		return e
	}
	if e := v.cleanupGrowthAcquisitions(); e != nil {
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
		return ErrRuntimeVectorOwner
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
		return ErrRuntimeVectorOwner
	}
	if v.closed {
		return nil
	}
	return v.closeOwning()
}

// CPtr reads the actual current member pointer only. Logical SIZE/CAP headers
// and borrowed provider/payload guards are reached only by later dereferences.
func (v *RuntimeUnsignedVector) CPtr() (RuntimeUint32View, error) {
	if v == nil || v.self != v {
		return RuntimeUint32View{}, ErrRuntimeVectorOwner
	}
	if !v.Constructed || v.closed || v.sourceDone || v.destructorCause != nil {
		return RuntimeUint32View{}, arena.ErrClosed
	}
	return BorrowRuntimeUint32(v.Column.Raw.Runtime, v.Column.Raw.Block, 8), nil
}
