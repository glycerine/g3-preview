// Source: src/util/params.cpp:938-944,992-1014.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

// Prepared cold storage retains the actual acquired new_entry value when
// copying or appending throws. The source POD local has no rational destructor.
type paramsSetFrame struct {
	cause, cleanupCause error
	invocation          uint64
	generation          uint32
	active, acquired    bool
	next                paramEntry
}
type ParamsSetReceipt struct {
	owner, invocation uint64
	slot, generation  uint32
}

func paramsSetFrameBytes(n int) uint64 { return paramsBytes(n, reflect.TypeOf(paramsSetFrame{})) }
func (c *ParamsContext) setNewNumeral(slot uint32, key Symbol, value *mpnum.OwnedRational) error {
	frame := &c.setFrames[slot-1]
	if frame.active {
		return ErrParamsPrecondition
	}
	serial, e := c.nextSourceInvocation()
	if e != nil {
		return e
	}
	*frame = paramsSetFrame{generation: c.blocks[slot-1].generation, invocation: serial, next: paramEntry{key: key, kind: ParamNumeral}}
	r, e := c.numerals.NewValue()
	if e != nil {
		frame.cause = e
		return e
	}
	frame.next.rat = r
	frame.acquired = true
	frame.active = true
	if e = c.numerals.Copy(&frame.next.rat, value); e != nil {
		frame.cause = e
		return e
	}
	b := &c.blocks[slot-1]
	if b.size == b.capacity {
		if e = c.grow(b); e != nil {
			frame.cause = e
			return e
		}
	}
	*c.entry(b, b.size) = frame.next
	b.size++
	frame.next = paramEntry{}
	frame.acquired = false
	frame.active = false
	return nil
}
func (c *ParamsContext) StoppedSet(v *ParamsRef) (ParamsSetReceipt, error) {
	if c == nil || c.self != c {
		return ParamsSetReceipt{}, ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return ParamsSetReceipt{}, e
	}
	if v.slot == 0 {
		return ParamsSetReceipt{}, ErrParamsPrecondition
	}
	f := &c.setFrames[v.slot-1]
	if !f.active || !f.acquired || f.generation != v.generation || f.cause == nil {
		return ParamsSetReceipt{}, ErrParamsPrecondition
	}
	return ParamsSetReceipt{owner: c.owner, slot: v.slot, generation: v.generation, invocation: f.invocation}, nil
}
func (c *ParamsContext) CleanupSet(receipt ParamsSetReceipt) error {
	if c == nil || c.self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || receipt.owner != c.owner || receipt.slot == 0 || int(receipt.slot) > len(c.setFrames) {
		return ErrParamsOwner
	}
	f := &c.setFrames[receipt.slot-1]
	if !f.active || !f.acquired || f.invocation != receipt.invocation || receipt.invocation == 0 || f.generation != receipt.generation {
		return ErrParamsPrecondition
	}
	return c.cleanupSetFrame(receipt.slot)
}
func (c *ParamsContext) cleanupSetFrame(slot uint32) error {
	f := &c.setFrames[slot-1]
	if !f.active || !f.acquired || f.invocation == 0 || f.generation != c.blocks[slot-1].generation || f.next.kind != ParamNumeral {
		return ErrParamsOwner
	}
	if e := c.numerals.Del(&f.next.rat); e != nil {
		if f.cleanupCause == nil {
			f.cleanupCause = e
		}
		return e
	}
	f.next = paramEntry{}
	f.active = false
	f.acquired = false
	return nil
}
