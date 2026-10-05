// Source: src/util/params.cpp:550-558,598-610.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"reflect"
)

const (
	paramsFrameAssign uint8 = 1
	paramsFrameInit   uint8 = 2
)

type paramsSourceFrame struct {
	cause                                    error
	invocation                               uint64
	rootGeneration, heldSlot, heldGeneration uint32
	kind                                     uint8
	active, decrementStarted                 bool
}

// ParamsSourceReceipt is a concrete binding to one failed source invocation.
// Fields are private; a caller obtains it only from its current actual value.
type ParamsSourceReceipt struct {
	owner, invocation uint64
	slot, generation  uint32
}

func paramsSourceFrameBytes(n int) uint64 { return paramsBytes(n, reflect.TypeOf(paramsSourceFrame{})) }
func (c *ParamsContext) StoppedSource(v *ParamsRef) (ParamsSourceReceipt, error) {
	if c == nil || c.self != c {
		return ParamsSourceReceipt{}, ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return ParamsSourceReceipt{}, e
	}
	if v.slot == 0 {
		return ParamsSourceReceipt{}, ErrParamsPrecondition
	}
	f := &c.sourceFrames[v.slot-1]
	if !f.active || f.rootGeneration != v.generation || f.cause == nil {
		return ParamsSourceReceipt{}, ErrParamsPrecondition
	}
	return ParamsSourceReceipt{owner: c.owner, invocation: f.invocation, slot: v.slot, generation: v.generation}, nil
}

// CleanupSource discards only the exact reference retained by a failed source
// frame. It does not replay copying, pointer publication, or a consumed DEC.
func (c *ParamsContext) CleanupSource(r ParamsSourceReceipt) error {
	if c == nil || c.self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || r.owner != c.owner || r.slot == 0 || int(r.slot) > len(c.sourceFrames) {
		return ErrParamsOwner
	}
	f := &c.sourceFrames[r.slot-1]
	if !f.active || f.invocation != r.invocation || f.rootGeneration != r.generation || r.invocation == 0 {
		return ErrParamsPrecondition
	}
	return c.cleanupSourceFrame(r.slot)
}
func (c *ParamsContext) cleanupSourceFrame(slot uint32) error {
	f := &c.sourceFrames[slot-1]
	if !f.active || f.invocation == 0 || f.rootGeneration != c.blocks[slot-1].generation || (f.kind != paramsFrameAssign && f.kind != paramsFrameInit) || f.heldSlot == 0 || int(f.heldSlot) > len(c.blocks) {
		return ErrParamsOwner
	}
	b := &c.blocks[f.heldSlot-1]
	if b.generation != f.heldGeneration || !b.live {
		return ErrParamsOwner
	}
	if b.resetActive && (b.resetKind == paramsResetLive || b.resetKind == paramsResetKey) {
		if e := c.continueDecrement(f.heldSlot); e != nil {
			return e
		}
	}
	if !f.decrementStarted {
		if b.refs == 0 {
			return ErrParamsPrecondition
		}
		f.decrementStarted = true
		if e := c.dec(f.heldSlot); e != nil {
			return e
		}
	} else {
		if !b.resetActive || b.refs != 0 {
			return ErrParamsOwner
		}
		if e := c.continueDecrement(f.heldSlot); e != nil {
			return e
		}
	}
	f.active = false
	return nil
}
