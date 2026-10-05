// Source: src/util/params.cpp:677-680,753-781,799-803.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

const (
	paramsResetDestroy uint8 = 1
	paramsResetLive    uint8 = 2
)

// ParamsResetReceipt binds one live-value reset invocation, without transferring
// the caller's reference or turning its shared block into a zero-ref destructor.
type ParamsResetReceipt struct {
	owner, invocation uint64
	slot, generation  uint32
}

func (c *ParamsContext) continueReset(slot uint32) error {
	b := &c.blocks[slot-1]
	r := &c.decrementReceipts[slot-1]
	if !b.live || !b.resetActive || b.resetKind != paramsResetLive || r.generation != b.generation || r.invocation == 0 {
		return ErrParamsOwner
	}
	// del_values preserves the actual scalar entries until vector.finalize.
	if b.resetStage == 0 {
		for b.resetCursor < b.size {
			x := c.entry(b, b.resetCursor)
			if x.kind == ParamNumeral {
				if e := c.numerals.Del(&x.rat); e != nil {
					if r.cause == nil {
						r.cause = e
					}
					return e
				}
			}
			b.resetCursor++
		}
		b.resetStage = 1
		b.resetCursor = 0
	}
	// Owned copied text is a Go adapter. Its release belongs to final backing
	// disposal, after the source numeral traversal, not to inline-value deletion.
	for b.resetCursor < b.size {
		x := c.entry(b, b.resetCursor)
		if x.text.present && !x.text.borrowed {
			if e := c.texts.Free(x.text.ref); e != nil {
				if r.cause == nil {
					r.cause = e
				}
				return e
			}
			x.text.present = false
		}
		b.resetCursor++
	}
	c.freeEntries(b.entries)
	b.entries = paramsEntriesRef{}
	b.size = 0
	b.capacity = 0
	b.resetActive = false
	b.resetCursor = 0
	b.resetStage = 0
	b.resetKind = 0
	return nil
}

// CompleteReset is an explicit owning continuation of the matching stopped
// action. It preserves caller identity and the live block's reference count.
func (c *ParamsContext) CompleteReset(v *ParamsRef, receipt ParamsResetReceipt) error {
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
	if !b.resetActive || b.resetKind != paramsResetLive || r.generation != v.generation || r.invocation != receipt.invocation || r.cause == nil {
		return ErrParamsPrecondition
	}
	return c.continueReset(v.slot)
}
