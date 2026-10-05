// Source: src/util/params.cpp:550-558,598-610.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"strconv"
	"testing"
)

func TestParamsReferenceAssignmentSourcePointerPublicationAfterDEC(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
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
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 2 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	other, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&other, k, 7))
	block = &c.blocks[v.slot-1] // actual directory may have relocated
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	oldSlot, oldGeneration := v.slot, v.generation
	otherSlot := other.slot
	e = c.Assign(&v, &other)
	if !errors.Is(e, arena.ErrBounds) || v.slot != oldSlot || v.generation != oldGeneration || block.refs != 0 || !block.live || c.blocks[otherSlot-1].refs != 2 {
		t.Fatal("source INC then DEC stop before pointer publication", e, v, *block, c.blocks[otherSlot-1])
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("retained descriptor and actually freed raw object", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if !reflect.ValueOf(&c.entry(block, 1).rat).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").IsValid() {
		t.Fatal("named retained entry")
	}
	receipt, err := c.StoppedSource(&v)
	paramsMust(t, err)
	if receipt.invocation == 0 {
		t.Fatal("actual assignment invocation")
	}
	paramsMust(t, c.CleanupSource(receipt))
	paramsMust(t, c.CleanupDecrement(&v))
	paramsMust(t, c.Del(&other))
}

func TestParamsReferenceInitFailedCopySkipsOldDEC(t *testing.T) {
	success := -1
	for fault := int64(0); fault < 32; fault++ {
		done := false
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			c, sy, _ := paramsTestOwner(t)
			keys := []Symbol{internSymbol(t, sy, "first"), internSymbol(t, sy, "second"), internSymbol(t, sy, "third"), internSymbol(t, sy, "fourth"), internSymbol(t, sy, "fifth"), internSymbol(t, sy, "sixth")}
			v, e := c.NewValue()
			paramsMust(t, e)
			for i, key := range keys[:5] {
				paramsMust(t, c.SetUInt(&v, key, uint32(11*(i+1))))
			}
			shared, e := c.NewValue()
			paramsMust(t, e)
			paramsMust(t, c.Assign(&shared, &v))
			oldSlot := v.slot
			borrowed := c.budget.Used() - paramsBorrowedPhysicalBytes(c.Stats())
			c.budget.FailAfter(fault)
			e = c.SetUInt(&v, keys[5], 66)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			copyIncomplete := v.slot == oldSlot || c.blocks[v.slot-1].size < 5
			if e != nil && copyIncomplete && c.blocks[oldSlot-1].refs != 2 {
				t.Fatal("copy throw must skip old dec_ref", fault, e, v, c.blocks[oldSlot-1])
			}
			if e != nil && copyIncomplete && v.slot != oldSlot {
				receipt, err := c.StoppedSource(&v)
				paramsMust(t, err)
				if receipt.invocation == 0 {
					t.Fatal("actual copy invocation")
				}
				t.Cleanup(func() { paramsMust(t, c.CleanupSource(receipt)) })
			}
			if borrowed+paramsBorrowedPhysicalBytes(c.Stats()) != c.budget.Used() {
				t.Fatal("immediate actual named cold graph")
			}
			for i, key := range keys[:5] {
				got, err := c.GetUInt(&shared, key, 0)
				paramsMust(t, err)
				if got != uint32(11*(i+1)) {
					t.Fatal("source old block changed", got)
				}
			}
			c.budget.FailAfter(-1)
			if e == nil {
				done = true
				success = int(fault)
				got, err := c.GetUInt(&v, keys[5], 0)
				paramsMust(t, err)
				if got != 66 {
					t.Fatal(got)
				}
			}
		})
		if done {
			break
		}
	}
	if success < 0 {
		t.Fatal("no actual first success")
	}
	t.Log("actual first success", success)
}
func TestParamsReferenceInitFrameOwnAbortActualLeafReceipt(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
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
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 2 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	shared, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&shared, &v))
	oldSlot, oldGeneration := v.slot, v.generation
	bc.FailAfter(0)
	e = c.SetUInt(&v, k, 9)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.slot == oldSlot || c.blocks[oldSlot-1].refs != 2 {
		t.Fatal("real published copy-constructor failure", e, v)
	}
	receipt, err := c.StoppedSource(&v)
	paramsMust(t, err)
	frame := &c.sourceFrames[receipt.slot-1]
	if frame.kind != paramsFrameInit || frame.heldSlot != oldSlot || frame.heldGeneration != oldGeneration || frame.decrementStarted {
		t.Fatal("exact source old-pointer publication receipt", *frame)
	}
	paramsMust(t, c.Del(&shared))
	if c.blocks[oldSlot-1].refs != 1 {
		t.Fatal("sole actual frame-held reference")
	}
	extras = upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	wrong := receipt
	wrong.invocation++
	if e := c.CleanupSource(wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("wrong invocation", e)
	}
	e = c.CleanupSource(receipt)
	old := &c.blocks[oldSlot-1]
	if !errors.Is(e, arena.ErrBounds) || old.refs != 0 || !old.resetActive || !frame.active || !frame.decrementStarted || !errors.Is(frame.cause, arena.ErrInjectedFailure) {
		t.Fatal("actual own-abort decrement stop", e, *old, *frame)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.CleanupSource(receipt))
	if old.live || old.refs != 0 || frame.active || !errors.Is(frame.cause, arena.ErrInjectedFailure) {
		t.Fatal("exact consumed child receipt", *old, *frame)
	}
	if e := c.CleanupSource(receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("completed receipt replay", e)
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Del(&v))
}

func TestParamsReferenceTwoFramesAndFreshInvocationReuse(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	keys := []Symbol{internSymbol(t, sy, "one"), internSymbol(t, sy, "two"), internSymbol(t, sy, "three"), internSymbol(t, sy, "four"), internSymbol(t, sy, "five"), internSymbol(t, sy, "six")}
	var current, shared [2]ParamsRef
	for n := range current {
		v, e := c.NewValue()
		paramsMust(t, e)
		current[n] = v
		for i, key := range keys[:5] {
			paramsMust(t, c.SetUInt(&current[n], key, uint32(100*n+i+1)))
		}
		shared[n], e = c.NewValue()
		paramsMust(t, e)
		paramsMust(t, c.Assign(&shared[n], &current[n]))
	}
	// Real completed small blocks prepare directory capacity only; class8 source
	// payloads remain occupied. No field/counter seeds or synthetic reservations.
	holders := make([]ParamsRef, 8)
	for i := range holders {
		v, e := c.NewValue()
		paramsMust(t, e)
		holders[i] = v
		paramsMust(t, c.SetUInt(&holders[i], keys[0], 1))
	}
	for i := range holders {
		paramsMust(t, c.Del(&holders[i]))
	}
	borrowed := c.budget.Used() - paramsBorrowedPhysicalBytes(c.Stats())
	c.budget.FailAfter(0)
	var receipts [2]ParamsSourceReceipt
	for i := range current {
		e := c.SetUInt(&current[i], keys[5], 99)
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("real copy allocation refusal", i, e)
		}
		var err error
		receipts[i], err = c.StoppedSource(&current[i])
		paramsMust(t, err)
		if borrowed+paramsBorrowedPhysicalBytes(c.Stats()) != c.budget.Used() {
			t.Fatal("immediate actual frame cold graph")
		}
		for n := 0; n <= i; n++ {
			f := c.sourceFrames[receipts[n].slot-1]
			if !f.active || f.heldSlot != shared[n].slot || f.heldGeneration != shared[n].generation || c.blocks[f.heldSlot-1].refs != 2 {
				t.Fatal("named actual retained reference", f)
			}
		}
	}
	if receipts[0].invocation == receipts[1].invocation || receipts[0].slot == receipts[1].slot {
		t.Fatal("independent source frames", receipts)
	}
	oldSlot := receipts[0].slot
	paramsMust(t, c.CleanupSource(receipts[0]))
	if !c.sourceFrames[receipts[1].slot-1].active || c.blocks[shared[1].slot-1].refs != 2 {
		t.Fatal("other retained invocation consumed")
	}
	paramsMust(t, c.Del(&current[0]))
	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &shared[0]))
	e = c.SetUInt(&alias, keys[5], 77)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("fresh real copy refusal", e)
	}
	fresh, err := c.StoppedSource(&alias)
	paramsMust(t, err)
	if fresh.slot != oldSlot || fresh.generation == receipts[0].generation || fresh.invocation <= receipts[0].invocation {
		t.Fatal("fresh source invocation", receipts[0], fresh)
	}
	if e := c.CleanupSource(receipts[0]); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("stale source receipt", e)
	}
	if borrowed+paramsBorrowedPhysicalBytes(c.Stats()) != c.budget.Used() {
		t.Fatal("immediate charged source frames")
	}
	for n := range shared {
		for i, key := range keys[:5] {
			got, e := c.GetUInt(&shared[n], key, 0)
			paramsMust(t, e)
			if got != uint32(100*n+i+1) {
				t.Fatal("original source values", n, i, got)
			}
		}
	}
	paramsMust(t, c.CleanupSource(fresh))
	paramsMust(t, c.CleanupSource(receipts[1]))
	c.budget.FailAfter(-1)
}
func TestParamsReferenceNullAssignmentOldDECStop(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
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
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 2 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	other, e := c.NewValue()
	paramsMust(t, e)
	// Source null RHS has no INC and no held source-frame reference.
	block = &c.blocks[v.slot-1] // actual directory may have relocated
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	oldSlot, oldGeneration := v.slot, v.generation
	otherSlot := other.slot
	e = c.Assign(&v, &other)
	if !errors.Is(e, arena.ErrBounds) || v.slot != oldSlot || v.generation != oldGeneration || block.refs != 0 || !block.live || otherSlot != 0 {
		t.Fatal("source INC then DEC stop before pointer publication", e, v, *block, other)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("retained descriptor and actually freed raw object", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if !reflect.ValueOf(&c.entry(block, 1).rat).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").IsValid() {
		t.Fatal("named retained entry")
	}
	if _, err := c.StoppedSource(&v); !errors.Is(err, ErrParamsPrecondition) {
		t.Fatal("null source creates no extra held reference", err)
	}
	paramsMust(t, c.CleanupDecrement(&v))
	paramsMust(t, c.Del(&other))
}

func TestParamsReferenceAssignmentSelfAliasSourceINCThenDEC(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	key := internSymbol(t, sy, "self")
	paramsMust(t, c.SetUInt(&v, key, 23))
	original := v
	used := c.budget.Used()
	c.budget.FailAfter(0)
	paramsMust(t, c.Assign(&v, &v))
	if v != original || c.blocks[v.slot-1].refs != 1 || c.sourceFrames[v.slot-1].active || c.budget.Used() != used {
		t.Fatal("literal source self alias", v)
	}
	got, e := c.GetUInt(&v, key, 0)
	paramsMust(t, e)
	if got != 23 {
		t.Fatal(got)
	}
	c.budget.FailAfter(-1)
}
func TestParamsReferenceContextCloseConsumesExactStoppedFrame(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
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
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 2 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	shared, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&shared, &v))
	oldSlot, oldGeneration := v.slot, v.generation
	bc.FailAfter(0)
	e = c.SetUInt(&v, k, 9)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.slot == oldSlot || c.blocks[oldSlot-1].refs != 2 {
		t.Fatal("real published copy-constructor failure", e, v)
	}
	receipt, err := c.StoppedSource(&v)
	paramsMust(t, err)
	frame := &c.sourceFrames[receipt.slot-1]
	if frame.kind != paramsFrameInit || frame.heldSlot != oldSlot || frame.heldGeneration != oldGeneration || frame.decrementStarted {
		t.Fatal("exact source old-pointer publication receipt", *frame)
	}
	paramsMust(t, c.Del(&shared))
	if c.blocks[oldSlot-1].refs != 1 {
		t.Fatal("sole actual frame-held reference")
	}
	extras = upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	wrong := receipt
	wrong.invocation++
	if e := c.CleanupSource(wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("wrong invocation", e)
	}
	e = c.Close()
	old := &c.blocks[oldSlot-1]
	if !errors.Is(e, arena.ErrBounds) || old.refs != 0 || !old.resetActive || !frame.active || !frame.decrementStarted || !errors.Is(frame.cause, arena.ErrInjectedFailure) {
		t.Fatal("actual own-abort decrement stop", e, *old, *frame)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Close())
	if !c.closed || bc.Used() != 0 {
		t.Fatal("actual owning context final refund", c.closed, bc.Used())
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}})
	paramsMust(t, c.Close())
}

func TestParamsReferenceInitConstructorFailureKeepsCallerOldReference(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	key := internSymbol(t, sy, "ctor")
	v, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, key, 7))
	other, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&other, key, 8))
	shared, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&shared, &v))
	if len(c.blocks) != cap(c.blocks) {
		t.Fatal("actual block directory refusal boundary", len(c.blocks), cap(c.blocks))
	}
	original := v
	before := c.budget.Used()
	c.budget.FailAfter(0)
	e = c.SetUInt(&v, key, 9)
	if !errors.Is(e, arena.ErrInjectedFailure) || v != original || c.blocks[v.slot-1].refs != 2 || c.budget.Used() != before {
		t.Fatal("constructor failure old pointer still caller-owned", e, v)
	}
	for _, frame := range c.sourceFrames {
		if frame.active {
			t.Fatal("unpublished pointer gained frame ownership", frame)
		}
	}
	if _, e := c.StoppedSource(&v); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("no stopped source body frame", e)
	}
	got, e := c.GetUInt(&v, key, 0)
	paramsMust(t, e)
	if got != 7 {
		t.Fatal(got)
	}
	c.budget.FailAfter(-1)
}
