// Source: src/util/params.cpp:753-763,946-1014,1017-1033.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

// leaf is a borrowed scalar identity receipt, never a second owning numeral.
// The sole old leaf remains in the actual matched entry. next owns only any
// prepared Go copied text; native SET_STR borrows a C-string instead.
type paramsReplacementFrame struct {
	cause, cleanupCause error
	invocation          uint64
	generation, index   uint32
	oldKind             ParamKind
	phase               uint8
	active              bool
	leaf                mpnum.OwnedRational
	next                paramEntry
}
type ParamsSetReplacementReceipt struct {
	owner, invocation       uint64
	slot, generation, index uint32
}

func paramsReplacementFrameBytes(n int) uint64 {
	return paramsBytes(n, reflect.TypeOf(paramsReplacementFrame{}))
}
func (c *ParamsContext) beginReplacement(slot, index uint32, next paramEntry, serial uint64) error {
	b := &c.blocks[slot-1]
	if c.replacementFrames[slot-1].active || index >= b.size {
		return ErrParamsPrecondition
	}
	entry := c.entry(b, index)
	if entry.kind != ParamNumeral {
		return ErrParamsPrecondition
	}
	if serial == 0 {
		return ErrParamsPrecondition
	}
	// Actual prepared text ownership transfers at this assignment; no root is
	// inferred from a refcount, allocation count, or descriptor registry.
	c.replacementFrames[slot-1] = paramsReplacementFrame{invocation: serial, generation: b.generation, index: index, oldKind: entry.kind, active: true, leaf: entry.rat, next: next}
	return c.continueReplacement(slot, true)
}
func (c *ParamsContext) continueReplacement(slot uint32, publish bool) error {
	f := &c.replacementFrames[slot-1]
	b := &c.blocks[slot-1]
	if !f.active || !b.live || f.generation != b.generation || f.invocation == 0 || f.index >= b.size || f.oldKind != ParamNumeral {
		return ErrParamsOwner
	}
	entry := c.entry(b, f.index)
	if f.phase == 0 {
		if entry.kind != f.oldKind || entry.rat != f.leaf {
			return ErrParamsPrecondition
		}
		e := c.numerals.Del(&entry.rat)
		f.leaf = entry.rat // The real leaf call may consume only a completed prefix.
		if e != nil {
			if f.cause == nil {
				f.cause = e
			} else if f.cleanupCause == nil {
				f.cleanupCause = e
			}
			return e
		}
		f.phase = 1
	}
	if publish {
		// Source SET_SIMPLE/SET_SYM publishes kind before its scalar field. The
		// following cold stores cannot allocate or invoke a borrowed service.
		entry.kind = f.next.kind
		entry.bits = f.next.bits
		entry.symbol = f.next.symbol
		entry.rat = f.next.rat
		entry.text = f.next.text
	} else {
		// Context owning abort is distinct from resuming the failed setter body.
		// It discards only the actual prepared argument and consumed old leaf.
		if f.next.text.present && !f.next.text.borrowed {
			if e := c.texts.Free(f.next.text.ref); e != nil {
				if f.cleanupCause == nil {
					f.cleanupCause = e
				}
				return e
			}
		}
		*entry = paramEntry{}
	}
	f.next = paramEntry{}
	f.active = false
	f.phase = 2
	return nil
}
func (c *ParamsContext) StoppedSetReplacement(v *ParamsRef) (ParamsSetReplacementReceipt, error) {
	if c == nil || c.self != c {
		return ParamsSetReplacementReceipt{}, ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return ParamsSetReplacementReceipt{}, e
	}
	if v.slot == 0 {
		return ParamsSetReplacementReceipt{}, ErrParamsPrecondition
	}
	f := &c.replacementFrames[v.slot-1]
	if !f.active || f.cause == nil || f.generation != v.generation {
		return ParamsSetReplacementReceipt{}, ErrParamsPrecondition
	}
	return ParamsSetReplacementReceipt{owner: c.owner, slot: v.slot, generation: v.generation, index: f.index, invocation: f.invocation}, nil
}
func (c *ParamsContext) CompleteSetReplacement(v *ParamsRef, r ParamsSetReplacementReceipt) error {
	if c == nil || c.self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot == 0 || r.owner != c.owner || r.slot != v.slot || r.generation != v.generation || r.invocation == 0 {
		return ErrParamsPrecondition
	}
	f := &c.replacementFrames[v.slot-1]
	if !f.active || f.invocation != r.invocation || f.index != r.index || f.generation != r.generation {
		return ErrParamsPrecondition
	}
	return c.continueReplacement(v.slot, true)
}
