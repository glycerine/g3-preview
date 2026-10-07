// Source: src/util/vector.h:59-100,178-189,350-365; src/util/memory_manager.cpp:274-289.
package util

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrRuntimeUnsignedGrowthExecuting = errors.New("g3: unsigned-vector acquisition is executing")

// A named archive owns only a token explicitly moved from this exact column's
// prior returned acquisition. It never owns the current source m_data field.
type runtimeUnsignedGrowthAcquisition struct {
	owner               *RuntimeUnsignedVector
	runtime             *Runtime
	next                *runtimeUnsignedGrowthAcquisition
	block               Block
	header, invocation  uint64
	cause, cleanupCause error
}

func (v *RuntimeUnsignedVector) matchGrowthColumn(c *RuntimeVectorColumn) error {
	if v == nil || v.self != v || c != &v.Column.Raw || c.Self != c || c.Runtime == nil || c.Runtime.Budget() != v.budget {
		return ErrRuntimeVectorOwner
	}
	if v.closed || !v.Constructed || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	if v.growthExecuting {
		return ErrRuntimeUnsignedGrowthExecuting
	}
	return nil
}
func (v *RuntimeUnsignedVector) startUnsignedGrowth(c *RuntimeVectorColumn) error {
	if e := v.matchGrowthColumn(c); e != nil {
		return e
	}
	if v.growthSerial == math.MaxUint64 {
		return arena.ErrSlotLimit
	}
	if c.Pending != (Block{}) {
		// This token came from a prior NULL-source-pointer allocation. A later
		// Assign may publish a distinct live current pointer; its actual capacity
		// read has already completed. Consumed realloc pointers still reach their
		// lifetime guard there and never gain authority from this archive.
		if c.PendingReplaces || v.growthPendingCause == nil || v.growthInvocation == 0 {
			return ErrRuntimeVectorOwner
		}
		charge := uint64(reflect.TypeOf(runtimeUnsignedGrowthAcquisition{}).Size())
		if e := v.budget.Reserve(charge); e != nil {
			return e
		}
		node := &runtimeUnsignedGrowthAcquisition{owner: v, runtime: c.Runtime, block: c.Pending, header: charge, invocation: v.growthInvocation, cause: v.growthPendingCause, next: v.growthAcquisitions}
		v.growthAcquisitions = node
		c.Pending = Block{} // Actual metadata ownership transfer, no source field mutation.
		v.growthPendingCause = nil
	}
	v.growthSerial++
	v.growthInvocation = v.growthSerial
	v.growthExecuting = true
	return nil
}
func (v *RuntimeUnsignedVector) recordUnsignedGrowth(cause error, retained bool) {
	if v.growthCause == nil {
		v.growthCause = cause
	}
	if retained {
		v.growthPendingCause = cause
	}
}
func (v *RuntimeUnsignedVector) endUnsignedGrowth() { v.growthExecuting = false }
func (v *RuntimeUnsignedVector) expandUnsignedSource(d *DebugContext) error {
	return v.Column.Raw.ExpandStrideUnsignedOwner(4, d, v)
}
func (v *RuntimeUnsignedVector) pushUnsignedSource(x uint32, d *DebugContext) error {
	if e := v.open(); e != nil {
		return e
	}
	n, e := v.Column.Size()
	if e != nil {
		return e
	}
	cap, e := v.Column.Capacity()
	if e != nil {
		return e
	}
	if n == cap {
		if e = v.expandUnsignedSource(d); e != nil {
			return e
		}
	}
	if e = WriteRuntimeWord(v.Column.Raw.Runtime, v.Column.Raw.Block, 8+4*uint64(n), x); e != nil {
		return e
	}
	return WriteRuntimeWord(v.Column.Raw.Runtime, v.Column.Raw.Block, 4, n+1)
}
func (v *RuntimeUnsignedVector) cleanupGrowthAcquisitions() error {
	for node := v.growthAcquisitions; node != nil; node = v.growthAcquisitions {
		if node.owner != v || node.runtime != v.Column.Raw.Runtime || node.invocation == 0 || node.invocation > v.growthSerial {
			return ErrRuntimeVectorOwner
		}
		if e := v.Column.Raw.Release(&node.block); e != nil {
			if node.cleanupCause == nil {
				node.cleanupCause = e
			}
			return e
		}
		v.growthAcquisitions = node.next
		v.budget.Release(node.header)
		node.header = 0
	}
	return nil
}
