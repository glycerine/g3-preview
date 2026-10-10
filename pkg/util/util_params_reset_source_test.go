// Source: src/util/params.cpp:677-680,753-781,799-803.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsReferenceResetRetainsCompletedInlinePrefixAtActualLeafStop(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
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
	scoped, e := unumeral.NewScopedInt(m)
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

	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &v))
	block = &c.Blocks[v.Slot-1]
	beforeSize, beforeRefs := block.Size, block.Refs
	extras := numericPhysicalExtraOwners{RationalServices: []*unumeral.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &alias}}}
	before := numericAuditPhysical(t, m, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	e = c.Reset(&v)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual later numeral release stop", e)
	}
	after := numericAuditPhysical(t, m, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if block.Refs != beforeRefs || block.Size != beforeSize || v != alias {
		t.Fatal("source Reset preserves shared reference and vector extent", *block, v, alias)
	}
	got, e := c.GetUInt(&v, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source inline prefix remains visible before vector finalization", got)
	}
	got, e = c.GetUInt(&alias, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source shared inline prefix remains visible", got)
	}
}

func TestParamsReferenceResetOwnContinuationMatchesKindAndInvocation(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
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
	scoped, e := unumeral.NewScopedInt(m)
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

	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &v))
	block = &c.Blocks[v.Slot-1]
	beforeSize, beforeRefs := block.Size, block.Refs
	extras := numericPhysicalExtraOwners{RationalServices: []*unumeral.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &alias}}}
	before := numericAuditPhysical(t, m, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	var receipt ParamsResetReceipt
	e = c.ResetTracked(&v, &receipt)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual later numeral release stop", e)
	}
	after := numericAuditPhysical(t, m, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if block.Refs != beforeRefs || block.Size != beforeSize || v != alias {
		t.Fatal("source Reset preserves shared reference and vector extent", *block, v, alias)
	}
	got, e := c.GetUInt(&v, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source inline prefix remains visible before vector finalization", got)
	}
	got, e = c.GetUInt(&alias, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source shared inline prefix remains visible", got)
	}

	if receipt.owner != c.Owner || receipt.slot != v.Slot || receipt.generation != v.Generation || receipt.invocation == 0 || block.resetKind != paramsResetLive {
		t.Fatal("actual live reset receipt", receipt, *block)
	}
	wrong := receipt
	wrong.invocation++
	if e := c.CompleteReset(&v, wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("wrong reset invocation", e)
	}
	if e := c.CompleteReset(&v, ParamsResetReceipt{}); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("unstamped reset invocation", e)
	}
	if e := c.CleanupDecrement(&v); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("live reset is not zero-reference destruction", e)
	}
	var decr ParamsDecrementReceipt
	paramsMust(t, c.DelTracked(&alias, &decr))
	if decr != (ParamsDecrementReceipt{}) || block.Refs != 1 || alias != (ParamsRef{}) {
		t.Fatal("defined shared DEC during live reset", decr, *block, alias)
	}
	if e := c.DelTracked(&v, &decr); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("explicit stopped last-reference owner guard", e)
	}
	if decr != (ParamsDecrementReceipt{}) || block.Refs != 1 || block.resetKind != paramsResetLive {
		t.Fatal("last-reference guard cannot overwrite source reset", decr, *block)
	}
	if e := c.Reset(&v); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("ordinary reset cannot replay consumed leaf", e)
	}
	token := v
	paramsMust(t, c.CompleteReset(&v, receipt))
	if v != token || !block.Live || block.Refs != 1 || block.Size != 0 || block.capacity != 0 || block.resetActive {
		t.Fatal("own reset completes live vector only", v, *block)
	}
	if !errors.Is(c.decrementReceipts[v.Slot-1].cause, arena.ErrBounds) {
		t.Fatal("original source stop history lost")
	}
	if e := c.CompleteReset(&v, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("duplicate reset acknowledgment", e)
	}
	numericAuditPhysical(t, m, numericPhysicalExtraOwners{RationalServices: []*unumeral.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.DelTracked(&v, &decr))
	if decr.InvocationField == 0 || decr.InvocationField == receipt.invocation || decr.slot != receipt.slot {
		t.Fatal("new genuine zero-entry protocol after finished reset", decr, receipt)
	}
}

func TestParamsReferenceIndependentResetReceiptsAndGenerationReuse(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
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
	scoped, e := unumeral.NewScopedInt(m)
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
	second, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&second, prefix, 42))
	paramsMust(t, c.SetRat(&second, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.Blocks[v.Slot-1]
	if block.Refs != 1 || block.Size != 2 {
		t.Fatal("actual source reference", block.Refs, block.Size)
	}

	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &v))
	block = &c.Blocks[v.Slot-1]
	beforeSize, beforeRefs := block.Size, block.Refs
	extras := numericPhysicalExtraOwners{RationalServices: []*unumeral.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &alias}, {c, &second}}}
	before := numericAuditPhysical(t, m, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	var firstReceipt ParamsResetReceipt
	e = c.ResetTracked(&v, &firstReceipt)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual later numeral release stop", e)
	}
	after := numericAuditPhysical(t, m, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if block.Refs != beforeRefs || block.Size != beforeSize || v != alias {
		t.Fatal("source Reset preserves shared reference and vector extent", *block, v, alias)
	}
	got, e := c.GetUInt(&v, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source inline prefix remains visible before vector finalization", got)
	}
	got, e = c.GetUInt(&alias, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("source shared inline prefix remains visible", got)
	}

	var secondReceipt ParamsResetReceipt
	if e := c.ResetTracked(&second, &secondReceipt); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("second actual reset stop", e)
	}
	if firstReceipt.invocation == secondReceipt.invocation || firstReceipt.slot == secondReceipt.slot || secondReceipt.invocation == 0 {
		t.Fatal("independent source reset receipts", firstReceipt, secondReceipt)
	}
	both := numericAuditPhysical(t, m, extras)
	if both.Named != before.Named || both.Objects+2 != before.Objects {
		t.Fatal(before, both)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if e := c.CompleteReset(&second, firstReceipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("foreign action adoption", e)
	}
	paramsMust(t, c.CompleteReset(&v, firstReceipt))
	got, e = c.GetUInt(&second, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("unrelated stopped inline prefix", got)
	}
	if !c.Blocks[second.Slot-1].resetActive || !errors.Is(c.decrementReceipts[second.Slot-1].cause, arena.ErrBounds) {
		t.Fatal("unrelated reset receipt overwritten")
	}
	numericAuditPhysical(t, m, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.CompleteReset(&second, secondReceipt))
	paramsMust(t, c.Del(&v))
	paramsMust(t, c.Del(&alias))
	paramsMust(t, c.Del(&second))
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	replacement, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&replacement, k, 9))
	if replacement.Slot != firstReceipt.slot || replacement.Generation == firstReceipt.generation {
		t.Fatal("actual reset slot generation reuse", replacement, firstReceipt)
	}
	if e := c.CompleteReset(&replacement, firstReceipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("stale reset generation", e)
	}
	var emptyReceipt ParamsResetReceipt
	empty, e := c.NewValue()
	paramsMust(t, e)
	emptyReceipt = firstReceipt
	paramsMust(t, c.ResetTracked(&empty, &emptyReceipt))
	if emptyReceipt != firstReceipt {
		t.Fatal("source null reset stamps no receipt")
	}
	paramsMust(t, c.Del(&replacement))
	paramsMust(t, c.Del(&empty))
}
