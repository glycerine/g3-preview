// Source: src/util/params.cpp:325-339,510-513.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsReferenceTrackedDecrementGuardedEntryReceipt(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m := q.Manager.Z
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)

		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("remaining named owners", bn.Used(), bc.Used(), bs.Used())
		}
	})
	v, e := c.NewValue()
	paramsMust(t, e)
	k, e := sy.Intern("actual-large-numeral")
	paramsMust(t, e)
	// Separate efficient fixture: public source digit setter and scoped copy.
	// Historical BitwiseNot fixture and its 252-second RED remain unchanged.
	scoped, e := NewScopedInt(m)
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.Blocks[v.Slot-1]
	if block.Refs != 1 || block.Size != 2 {
		t.Fatal("actual source reference", block.Refs, block.Size)
	}
	extras := numericPhysicalExtraOwners{RationalServices: []*RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := numericAuditPhysical(t, m, extras)
	other, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&other, k, 7))
	block = &c.Blocks[v.Slot-1] // actual directory may have relocated
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	var receipt ParamsDecrementReceipt
	e = c.DelTracked(&v, &receipt)
	if !errors.Is(e, arena.ErrBounds) || receipt.owner != c.Owner || receipt.slot != v.Slot || receipt.generation != v.Generation || receipt.InvocationField == 0 || block.Refs != 0 || !block.resetActive {
		t.Fatal("actual zero-count guarded-entry receipt", e, receipt, *block)
	}
	after := numericAuditPhysical(t, m, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	wrong := receipt
	wrong.InvocationField++
	if e := c.CompleteDecrement(&v, wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("different source invocation", e)
	}
	if e := c.CompleteDecrement(&other, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("different actual value", e)
	}
	unchanged := receipt
	empty, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.DelTracked(&empty, &unchanged))
	if unchanged != receipt {
		t.Fatal("null source call stamped receipt")
	}
	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &other))
	paramsMust(t, c.DelTracked(&alias, &unchanged))
	if unchanged != receipt {
		t.Fatal("nonzero reference decrement stamped receipt")
	}
	if GlobalParamsSourceDebug {
		if e := c.DelTracked(&v, &unchanged); !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("source assertion before repeat stamp", e)
		}
		if unchanged != receipt {
			t.Fatal("failed entry replaced receipt")
		}
	}
	if e := c.CompleteDecrement(&v, ParamsDecrementReceipt{}); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("empty receipt adopted stop", e)
	}
	paramsMust(t, c.CompleteDecrement(&v, receipt))
	if block.Live || v != (ParamsRef{}) || block.Refs != 0 {
		t.Fatal("exact callee owning acknowledgment")
	}
	numericAuditPhysical(t, m, numericPhysicalExtraOwners{RationalServices: []*RationalService{q}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if e := c.CompleteDecrement(&v, receipt); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("double acknowledgment", e)
	}
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	replacement, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&replacement, k, 9))
	if replacement.Slot != receipt.slot || replacement.Generation == receipt.generation {
		t.Fatal("real slot generation reuse", replacement, receipt)
	}
	if e := c.CompleteDecrement(&replacement, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("stale generation receipt", e)
	}
	paramsMust(t, c.Del(&replacement))
	paramsMust(t, c.Del(&other))
}
