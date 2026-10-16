//go:build g3debug

package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Source: src/util/params.cpp:799-803. The assertion follows vector finalize,
// and a stopped return retains the actual reset invocation for owning finish.
func TestParamsResetSourcePostconditionAfterFinalization(t *testing.T) {
	c, symbols, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	k := internSymbol(t, symbols, "value")
	paramsMust(t, c.SetUInt(&v, k, 17))
	slot := v.Slot
	var receipt ParamsResetReceipt
	if e := paramsCallWithClosedProcessDebug(t, func() error { return c.ResetTracked(&v, &receipt) }); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("reset postcondition after finalization", e)
	}
	b := &c.Blocks[slot-1]
	if receipt.owner != c.Owner || receipt.slot != slot || receipt.invocation == 0 || !b.Live || !b.resetActive || b.resetKind != paramsResetLive || b.Size != 0 || b.capacity != 0 || b.entries != (paramsEntriesRef{}) || b.Refs != 1 {
		t.Fatal("stopped reset must retain completed body and exact receipt", receipt, *b)
	}
	paramsMust(t, c.CompleteReset(&v, receipt))
	if b.resetActive || b.Size != 0 || b.Refs != 1 {
		t.Fatal("owning continuation did not complete source reset", *b)
	}
	paramsMust(t, c.SetUInt(&v, k, 19))
	if value, e := c.GetUInt(&v, k, 0); e != nil || value != 19 {
		t.Fatal("completed owner did not resume", value, e)
	}
	paramsMust(t, c.Del(&v))
}
