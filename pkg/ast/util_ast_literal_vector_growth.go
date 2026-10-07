// Source: src/util/vector.h:59-100,178-189,350-365; src/util/memory_manager.cpp:274-289.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type runtimeASTLiteralGrowthAcquisition struct {
	owner               *RuntimeASTLiteralVector
	runtime             *util.Runtime
	next                *runtimeASTLiteralGrowthAcquisition
	block               util.Block
	header, invocation  uint64
	cause, cleanupCause error
}

func (v *RuntimeASTLiteralVector) MatchVectorGrowthColumn(c *util.RuntimeVectorColumn) error {
	if v == nil || v.self != v || c != &v.raw || c.Self != c || c.Runtime == nil || c.Runtime.Budget() != v.budget {
		return util.ErrRuntimeVectorOwner
	}
	if v.closed || !v.constructed && !v.constructionBorrow() || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	if v.growthExecuting || v.copyExecuting {
		return util.ErrRuntimeUnsignedGrowthExecuting
	}
	return nil
}
func (v *RuntimeASTLiteralVector) StartVectorGrowth(c *util.RuntimeVectorColumn) error {
	if e := v.MatchVectorGrowthColumn(c); e != nil {
		return e
	}
	if v.growthSerial == math.MaxUint64 {
		return arena.ErrSlotLimit
	}
	if c.Pending != (util.Block{}) {
		// This token came from a prior NULL-source-pointer allocation. A later
		// Assign may publish a distinct live current pointer; its actual capacity
		// read has already completed. Consumed realloc pointers still reach their
		// lifetime guard there and never gain authority from this archive.
		if c.PendingReplaces || v.growthPendingCause == nil || v.growthInvocation == 0 {
			return util.ErrRuntimeVectorOwner
		}
		charge := uint64(reflect.TypeOf(runtimeASTLiteralGrowthAcquisition{}).Size())
		if e := v.budget.Reserve(charge); e != nil {
			return e
		}
		node := &runtimeASTLiteralGrowthAcquisition{owner: v, runtime: c.Runtime, block: c.Pending, header: charge, invocation: v.growthInvocation, cause: v.growthPendingCause, next: v.growthAcquisitions}
		v.growthAcquisitions = node
		c.Pending = util.Block{} // Actual metadata ownership transfer, no source field mutation.
		v.growthPendingCause = nil
	}
	v.growthSerial++
	v.growthInvocation = v.growthSerial
	v.growthExecuting = true
	return nil
}
func (v *RuntimeASTLiteralVector) RecordVectorGrowth(cause error, retained bool) {
	if v.growthCause == nil {
		v.growthCause = cause
	}
	if retained {
		v.growthPendingCause = cause
	}
}
func (v *RuntimeASTLiteralVector) EndVectorGrowth() { v.growthExecuting = false }
func (v *RuntimeASTLiteralVector) cleanupGrowthAcquisitions() error {
	for node := v.growthAcquisitions; node != nil; node = v.growthAcquisitions {
		if node.owner != v || node.runtime != v.raw.Runtime || node.invocation == 0 || node.invocation > v.growthSerial {
			return util.ErrRuntimeVectorOwner
		}
		if e := v.raw.Release(&node.block); e != nil {
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
