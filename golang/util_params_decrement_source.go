// Source: src/util/params.cpp:325-339,510-513.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// Cold error metadata is prepared with the block directory. Actual leaf owners
// remain in their entries; this receipt never manufactures an owning numeral.
type paramsDecrementReceipt struct {
	cause      error
	invocation uint64
	generation uint32
}

func paramsReceiptBytes(n int) uint64 {
	return paramsBytes(n, reflect.TypeOf(paramsDecrementReceipt{}))
}

// continueDecrement is an explicit Go owning continuation, never a second
// source dec_ref. Its scalar cursor names the exact retained entry/leaf token.
func (c *ParamsContext) continueDecrement(slot uint32) error {
	b := &c.blocks[slot-1]
	receipt := &c.decrementReceipts[slot-1]
	if !b.live || !b.resetActive || receipt.generation != b.generation {
		return ErrParamsOwner
	}
	for b.resetCursor < b.size {
		entry := c.entry(b, b.resetCursor)
		if b.resetStage == 0 {
			if entry.kind == ParamNumeral {
				if e := c.numerals.Del(&entry.rat); e != nil {
					if receipt.cause == nil {
						receipt.cause = e
					}
					return e
				}
			}
			b.resetStage = 1
		}
		if b.resetStage == 1 {
			if entry.text.present && !entry.text.borrowed {
				if e := c.texts.Free(entry.text.ref); e != nil {
					if receipt.cause == nil {
						receipt.cause = e
					}
					return e
				}
			}
			b.resetStage = 2
		}
		*entry = paramEntry{}
		b.resetCursor++
		b.resetStage = 0
	}
	c.freeEntries(b.entries)
	b.entries = paramsEntriesRef{}
	b.size = 0
	b.capacity = 0
	b.live = false
	b.resetActive = false
	return nil
}

// CleanupDecrement acknowledges only the exact zero-reference block whose
// source destruction stopped. It retains its original cause in cold metadata
// until block reuse, and consumes the caller token only after all leaves finish.
func (c *ParamsContext) CleanupDecrement(v *ParamsRef) error {
	if c == nil || c.self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot == 0 {
		return ErrParamsPrecondition
	}
	b := &c.blocks[v.slot-1]
	if !b.resetActive || b.refs != 0 {
		return ErrParamsPrecondition
	}
	if e := c.continueDecrement(v.slot); e != nil {
		return e
	}
	*v = ParamsRef{}
	return nil
}

// A stopped destructor no longer denotes a valid ordinary source params value.
// Mutations cannot replace or resurrect its exact retained leaf owner. Reads
// retain their reached source guards; dec_ref retains its own DEBUG assertion.
func (c *ParamsContext) mutable(v *ParamsRef) error {
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot != 0 && (c.blocks[v.slot-1].resetActive || c.sourceFrames[v.slot-1].active) {
		return ErrParamsPrecondition
	}
	return nil
}

// ParamsDecrementReceipt binds a parent to the exact reached child destruction.
// It carries no authority derived from a block's reference count or registry.
type ParamsDecrementReceipt struct {
	owner, invocation uint64
	slot, generation  uint32
}

// CompleteDecrement consumes only a matching stopped invocation. It never
// repeats dec_ref, and leaves the original source failure in cold history.
func (c *ParamsContext) CompleteDecrement(v *ParamsRef, receipt ParamsDecrementReceipt) error {
	if c == nil || c.self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot == 0 || receipt.owner != c.owner || receipt.slot != v.slot || receipt.generation != v.generation || receipt.invocation == 0 {
		return ErrParamsPrecondition
	}
	b := &c.blocks[v.slot-1]
	r := &c.decrementReceipts[v.slot-1]
	if !b.resetActive || b.refs != 0 || r.generation != v.generation || r.invocation != receipt.invocation {
		return ErrParamsPrecondition
	}
	if e := c.continueDecrement(v.slot); e != nil {
		return e
	}
	*v = ParamsRef{}
	return nil
}

func (c *ParamsContext) nextSourceInvocation() (uint64, error) {
	if c.sourceSerial == math.MaxUint64 {
		return 0, arena.ErrBounds
	}
	c.sourceSerial++
	return c.sourceSerial, nil
}
