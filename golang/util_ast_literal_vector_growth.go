// Source: src/util/vector.h:59-100,178-189,350-365; src/util/memory_manager.cpp:274-289.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
)

type runtimeASTLiteralGrowthAcquisition struct {
	owner               *RuntimeASTLiteralVector
	runtime             *memory.Runtime
	next                *runtimeASTLiteralGrowthAcquisition
	block               memory.Block
	header, invocation  uint64
	cause, cleanupCause error
}

func (v *RuntimeASTLiteralVector) matchGrowthColumn(c *satSolverAssignmentColumn) error {
	if v == nil || v.self != v || c != &v.raw || c.self != c || c.runtime == nil || c.runtime.Budget() != v.budget {
		return ErrSATOwner
	}
	if v.closed || !v.constructed && !v.constructionBorrow() || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	if v.growthExecuting || v.copyExecuting {
		return ErrRuntimeUnsignedGrowthExecuting
	}
	return nil
}
func (v *RuntimeASTLiteralVector) startASTGrowth(c *satSolverAssignmentColumn) error {
	if e := v.matchGrowthColumn(c); e != nil {
		return e
	}
	if v.growthSerial == math.MaxUint64 {
		return arena.ErrSlotLimit
	}
	if c.pending != (memory.Block{}) {
		// This token came from a prior NULL-source-pointer allocation. A later
		// Assign may publish a distinct live current pointer; its actual capacity
		// read has already completed. Consumed realloc pointers still reach their
		// lifetime guard there and never gain authority from this archive.
		if c.pendingReplaces || v.growthPendingCause == nil || v.growthInvocation == 0 {
			return ErrSATOwner
		}
		charge := uint64(reflect.TypeOf(runtimeASTLiteralGrowthAcquisition{}).Size())
		if e := v.budget.Reserve(charge); e != nil {
			return e
		}
		node := &runtimeASTLiteralGrowthAcquisition{owner: v, runtime: c.runtime, block: c.pending, header: charge, invocation: v.growthInvocation, cause: v.growthPendingCause, next: v.growthAcquisitions}
		v.growthAcquisitions = node
		c.pending = memory.Block{} // Actual metadata ownership transfer, no source field mutation.
		v.growthPendingCause = nil
	}
	v.growthSerial++
	v.growthInvocation = v.growthSerial
	v.growthExecuting = true
	return nil
}
func (v *RuntimeASTLiteralVector) recordASTGrowth(cause error, retained bool) {
	if v.growthCause == nil {
		v.growthCause = cause
	}
	if retained {
		v.growthPendingCause = cause
	}
}
func (v *RuntimeASTLiteralVector) endASTGrowth() { v.growthExecuting = false }
func (v *RuntimeASTLiteralVector) cleanupGrowthAcquisitions() error {
	for node := v.growthAcquisitions; node != nil; node = v.growthAcquisitions {
		if node.owner != v || node.runtime != v.raw.runtime || node.invocation == 0 || node.invocation > v.growthSerial {
			return ErrSATOwner
		}
		if e := v.raw.release(&node.block); e != nil {
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
