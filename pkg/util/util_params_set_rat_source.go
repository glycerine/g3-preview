// Source: src/util/params.cpp:938-944,992-1014.
package util

import (
	"reflect"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Prepared cold storage retains the actual acquired new_entry value when
// copying or appending throws. The source POD local has no rational destructor.
type paramsSetFrame struct {
	Cause, CleanupCause error
	Invocation          uint64
	Generation          uint32
	Active, Acquired    bool
	Next                paramEntry
}
type ParamsSetReceipt struct {
	owner, InvocationField uint64
	Slot, Generation       uint32
}

func paramsSetFrameBytes(n int) uint64 { return paramsBytes(n, reflect.TypeOf(paramsSetFrame{})) }
func (c *ParamsContext) setNewNumeral(slot uint32, key usymbol.Symbol, value *unumeral.OwnedRational) error {
	frame := &c.SetFrames[slot-1]
	if frame.Active {
		return ErrParamsPrecondition
	}
	serial, e := c.nextSourceInvocation()
	if e != nil {
		return e
	}
	*frame = paramsSetFrame{Generation: c.Blocks[slot-1].Generation, Invocation: serial, Next: paramEntry{key: key, Kind: ParamNumeral}}
	r, e := c.NumeralsField.NewValue()
	if e != nil {
		frame.Cause = e
		return e
	}
	frame.Next.Rat = r
	frame.Acquired = true
	frame.Active = true
	if e = c.NumeralsField.Copy(&frame.Next.Rat, value); e != nil {
		frame.Cause = e
		return e
	}
	b := &c.Blocks[slot-1]
	if b.Size == b.capacity {
		if e = c.grow(b); e != nil {
			frame.Cause = e
			return e
		}
	}
	*c.Entry(b, b.Size) = frame.Next
	b.Size++
	frame.Next = paramEntry{}
	frame.Acquired = false
	frame.Active = false
	return nil
}
func (c *ParamsContext) StoppedSet(v *ParamsRef) (ParamsSetReceipt, error) {
	if c == nil || c.Self != c {
		return ParamsSetReceipt{}, ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return ParamsSetReceipt{}, e
	}
	if v.Slot == 0 {
		return ParamsSetReceipt{}, ErrParamsPrecondition
	}
	f := &c.SetFrames[v.Slot-1]
	if !f.Active || !f.Acquired || f.Generation != v.Generation || f.Cause == nil {
		return ParamsSetReceipt{}, ErrParamsPrecondition
	}
	return ParamsSetReceipt{owner: c.Owner, Slot: v.Slot, Generation: v.Generation, InvocationField: f.Invocation}, nil
}
func (c *ParamsContext) CleanupSet(receipt ParamsSetReceipt) error {
	if c == nil || c.Self != c {
		return ErrParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Closed || receipt.owner != c.Owner || receipt.Slot == 0 || int(receipt.Slot) > len(c.SetFrames) {
		return ErrParamsOwner
	}
	f := &c.SetFrames[receipt.Slot-1]
	if !f.Active || !f.Acquired || f.Invocation != receipt.InvocationField || receipt.InvocationField == 0 || f.Generation != receipt.Generation {
		return ErrParamsPrecondition
	}
	return c.cleanupSetFrame(receipt.Slot)
}
func (c *ParamsContext) cleanupSetFrame(slot uint32) error {
	f := &c.SetFrames[slot-1]
	if !f.Active || !f.Acquired || f.Invocation == 0 || f.Generation != c.Blocks[slot-1].Generation || f.Next.Kind != ParamNumeral {
		return ErrParamsOwner
	}
	if e := c.NumeralsField.Del(&f.Next.Rat); e != nil {
		if f.CleanupCause == nil {
			f.CleanupCause = e
		}
		return e
	}
	f.Next = paramEntry{}
	f.Active = false
	f.Acquired = false
	return nil
}

func (r ParamsSetReceipt) Invocation() uint64 { return r.InvocationField }
