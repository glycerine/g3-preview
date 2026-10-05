// Source: src/util/params.cpp:682-690,753-763,805-828.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

const paramsResetKey uint8 = 3

// ParamsResetKeyReceipt is minted only for the matched source entry. An absent
// key cannot acquire authority over an earlier stopped leaf.
type ParamsResetKeyReceipt struct {
	owner, invocation       uint64
	slot, generation, index uint32
}

func (c *ParamsContext) continueResetKey(slot uint32) error {
	b := &c.blocks[slot-1]
	r := &c.decrementReceipts[slot-1]
	if !b.live || !b.resetActive || b.resetKind != paramsResetKey || r.generation != b.generation || r.invocation == 0 || b.resetCursor >= b.size {
		return ErrParamsOwner
	}
	x := c.entry(b, b.resetCursor)
	if b.resetStage == 0 {
		if x.kind == ParamNumeral {
			if e := c.numerals.Del(&x.rat); e != nil {
				if r.cause == nil {
					r.cause = e
				}
				return e
			}
		}
		b.resetStage = 1
	}
	// Explicit owned-string adapter release; source char strings are borrowed.
	if b.resetStage == 1 {
		if x.text.present && !x.text.borrowed {
			if e := c.texts.Free(x.text.ref); e != nil {
				if r.cause == nil {
					r.cause = e
				}
				return e
			}
			x.text.present = false
		}
		b.resetStage = 2
	}
	// Source ascending assignment shifts the suffix before pop_back. There is
	// no fallible allocation in these scalar pointer-free entry assignments.
	for j := b.resetCursor + 1; j < b.size; j++ {
		*c.entry(b, j-1) = *c.entry(b, j)
	}
	b.size--
	*c.entry(b, b.size) = paramEntry{}
	b.resetActive = false
	b.resetKind = 0
	b.resetStage = 0
	b.resetCursor = 0
	return nil
}

func (c *ParamsContext) CompleteResetKey(v *ParamsRef, receipt ParamsResetKeyReceipt) error {
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
	if !b.resetActive || b.resetKind != paramsResetKey || receipt.index != b.resetCursor || r.generation != v.generation || r.invocation != receipt.invocation || r.cause == nil {
		return ErrParamsPrecondition
	}
	return c.continueResetKey(v.slot)
}
