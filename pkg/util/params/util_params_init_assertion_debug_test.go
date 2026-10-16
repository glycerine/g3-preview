//go:build g3debug

package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func paramsCallWithClosedProcessDebug(t *testing.T, call func() error) error {
	t.Helper()
	d, _, _ := debugFixture(t)
	old, e := udiag.BindProcessDebugContext(d)
	paramsMust(t, e)
	paramsMust(t, d.Close())
	result := call()
	_, e = udiag.BindProcessDebugContext(old)
	paramsMust(t, e)
	return result
}

// Source: src/util/params.cpp:598-611. The assertion is reached after init,
// before the caller's requested entry mutation, including the unique no-op.
func TestParamsInitSourcePostconditionAfterFirstAndUnique(t *testing.T) {
	c, symbols, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	first := internSymbol(t, symbols, "first")
	second := internSymbol(t, symbols, "second")
	if e := paramsCallWithClosedProcessDebug(t, func() error { return c.SetUInt(&v, first, 7) }); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("first init postcondition", e)
	}
	if v.Slot == 0 || c.Blocks[v.Slot-1].Refs != 1 || c.Blocks[v.Slot-1].Size != 0 {
		t.Fatal("first init must publish its empty block before assertion", v)
	}
	paramsMust(t, c.SetUInt(&v, first, 7))
	firstSlot := v.Slot
	if e := paramsCallWithClosedProcessDebug(t, func() error { return c.SetUInt(&v, second, 9) }); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("unique init postcondition", e)
	}
	if v.Slot != firstSlot || c.Blocks[firstSlot-1].Refs != 1 || c.Blocks[firstSlot-1].Size != 1 {
		t.Fatal("unique path changed entries before assertion", v)
	}
	if value, e := c.GetUInt(&v, second, 0); e != nil || value != 0 {
		t.Fatal("requested SET must follow assertion", value, e)
	}
	paramsMust(t, c.SetUInt(&v, second, 9))
	paramsMust(t, c.Del(&v))
}
