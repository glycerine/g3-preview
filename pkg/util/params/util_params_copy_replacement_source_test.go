// Source: src/util/params.cpp:559-595,753-763,938-962.
package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsReferenceCopyStopsAtMatchedNumeralDestruction(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, runtime.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("retained owners", bn.Used(), bc.Used(), bs.Used())
		}
	})

	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	before := internSymbol(t, sy, "before")
	key := internSymbol(t, sy, "number")
	after := internSymbol(t, sy, "after")
	paramsMust(t, c.SetUInt(&src, before, 42))
	paramsMust(t, c.SetBool(&src, key, true))
	paramsMust(t, c.SetUInt(&src, after, 77))

	// An actual large numeral gives the matched destination entry a Runtime
	// payload whose source destructor can stop after the earlier copy step.
	scoped, e := unumeral.NewScopedInt(q.Manager.Z)
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, q.Manager.Z.SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(&dst, key, &big))
	paramsMust(t, q.Del(&big))

	paramsDecTLSNearMinimum(t, runtime)
	bn.FailAfter(0)
	e = c.Copy(&dst, &src)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("matched numeral destructor did not stop copy", e)
	}
	if got, err := c.GetUInt(&dst, before, 0); err != nil || got != 42 {
		t.Fatal("earlier copy prefix", got, err)
	}
	if present, err := c.Contains(&dst, after); err != nil || present {
		t.Fatal("later source entry copied after stop", present, err)
	}
	if got, err := c.Display(&src); err != nil || got != "(params before 42 number true after 77)" {
		t.Fatal("source changed during stopped copy", got, err)
	}
	receipt, e := c.StoppedSetReplacement(&dst)
	paramsMust(t, e)
	if receipt.invocation == 0 || !errors.Is(c.Copy(&dst, &src), ErrParamsPrecondition) {
		t.Fatal("stopped replacement accepted ordinary copy replay")
	}
	bn.FailAfter(-1)
	paramsMust(t, c.CompleteSetReplacement(&dst, receipt))
	if got, err := c.Display(&dst); err != nil || got != "(params number true before 42)" {
		t.Fatal("owning completion replayed the later copy step", got, err)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&src))
}

// A failed copy owns the destination's partially published entries. Its
// matched numeral replacement can be aborted without replaying the source
// setter, after which the caller releases that destination explicitly.
func TestParamsReferenceCopyStoppedReplacementOwningAbort(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, runtime.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("retained owners", bn.Used(), bc.Used(), bs.Used())
		}
	})
	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	before := internSymbol(t, sy, "before")
	key := internSymbol(t, sy, "number")
	after := internSymbol(t, sy, "after")
	paramsMust(t, c.SetUInt(&src, before, 42))
	paramsMust(t, c.SetBool(&src, key, true))
	paramsMust(t, c.SetUInt(&src, after, 77))

	scoped, e := unumeral.NewScopedInt(q.Manager.Z)
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, q.Manager.Z.SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(&dst, key, &big))
	paramsMust(t, q.Del(&big))

	paramsDecTLSNearMinimum(t, runtime)
	bn.FailAfter(0)
	e = c.Copy(&dst, &src)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("matched numeral destructor did not stop copy", e)
	}
	receipt, e := c.StoppedSetReplacement(&dst)
	paramsMust(t, e)
	bn.FailAfter(-1)
	paramsMust(t, c.AbortSetReplacement(&dst, receipt))
	if e = c.CompleteSetReplacement(&dst, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("aborted replacement accepted replay", e)
	}
	if got, err := c.Display(&src); err != nil || got != "(params before 42 number true after 77)" {
		t.Fatal("source changed during owning abort", got, err)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&src))
}

// A matched rational copy writes into the existing entry. If its numerator
// allocation refuses, the destination owns the failed numeral until deletion;
// the source and an earlier completed copy step remain intact.
func TestParamsReferenceCopyMatchedNumeralAllocationRefusal(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	before := internSymbol(t, sy, "before")
	key := internSymbol(t, sy, "number")
	paramsMust(t, c.SetBool(&src, before, true))

	scoped, e := unumeral.NewScopedInt(q.Manager.Z)
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, q.Manager.Z.SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(&src, key, &big))
	paramsMust(t, q.Del(&big))
	zero, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetRat(&dst, key, &zero))
	paramsMust(t, q.Del(&zero))
	sourceNumerator := c.Entry(&c.Blocks[src.Slot-1], 1).Rat.Raw.Num.Cell

	q.Manager.Z.BudgetField.FailAfter(0)
	e = c.Copy(&dst, &src)
	q.Manager.Z.BudgetField.FailAfter(-1)
	if e == nil {
		t.Fatal("matched numeral did not reach numerator allocation", e)
	}
	entry := c.Entry(&c.Blocks[dst.Slot-1], 0)
	if entry.Kind != ParamNumeral || entry.Rat.Raw.Num.Cell != q.Manager.Z.FailedNullRef {
		t.Fatal("refusal did not retain the matched numeral's failed numerator", entry.Kind, entry.Rat.Raw.Num.Cell)
	}
	if got, err := c.GetBool(&dst, before, false); err != nil || !got {
		t.Fatal("completed copy prefix", got, err)
	}
	sourceEntry := c.Entry(&c.Blocks[src.Slot-1], 1)
	if sourceEntry.Rat.Raw.Num.Cell != sourceNumerator {
		t.Fatal("source numeral changed during failed copy")
	}
	if rational, err := q.IsRational(&sourceEntry.Rat); err != nil || !rational {
		t.Fatal("source numeral became unreadable", rational, err)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&src))
}
