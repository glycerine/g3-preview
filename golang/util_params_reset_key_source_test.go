// Source: src/util/params.cpp:682-690,805-828.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"strconv"
	"testing"
)

func TestParamsReferenceResetKeyRejectsOrdinaryReplayAfterActualLeafStop(t *testing.T) {
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
	tail, e := sy.Intern("tail")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, tail, 84))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 3 {
		t.Fatal("actual source reference", block.refs, block.size)
	}

	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &v))
	block = &c.blocks[v.slot-1]
	beforeSize, beforeRefs := block.size, block.refs
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &alias}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	e = c.ResetKey(&v, k)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual later numeral release stop", e)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if block.refs != beforeRefs || block.size != beforeSize || v != alias {
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

	got, e = c.GetUInt(&v, tail, 0)
	paramsMust(t, e)
	if got != 84 {
		t.Fatal("source unshifted suffix", got)
	}
	if e := c.ResetKey(&v, k); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("ordinary reset-key cannot acknowledge consumed source destruction", e)
	}
	if block.size != beforeSize || block.refs != beforeRefs {
		t.Fatal("ordinary replay changed source vector", *block)
	}
}

func TestParamsReferenceResetKeyMatchedOwnShiftAndForeignStop(t *testing.T) {
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
	second, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&second, prefix, 42))
	paramsMust(t, c.SetRat(&second, k, &big))
	paramsMust(t, q.Del(&big))
	tail, e := sy.Intern("tail")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, tail, 84))
	paramsMust(t, c.SetUInt(&second, tail, 84))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 3 {
		t.Fatal("actual source reference", block.refs, block.size)
	}

	alias, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(&alias, &v))
	block = &c.blocks[v.slot-1]
	beforeSize, beforeRefs := block.size, block.refs
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &second}}})
	missingKey, e := sy.Intern("never-found")
	paramsMust(t, e)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	var receipt ParamsResetKeyReceipt
	e = c.ResetKeyTracked(&v, k, &receipt)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual later numeral release stop", e)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &second}}})
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal(before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if block.refs != beforeRefs || block.size != beforeSize || v != alias {
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

	got, e = c.GetUInt(&v, tail, 0)
	paramsMust(t, e)
	if got != 84 {
		t.Fatal("source unshifted suffix", got)
	}
	if e := c.ResetKey(&v, k); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("ordinary reset-key cannot acknowledge consumed source destruction", e)
	}
	if block.size != beforeSize || block.refs != beforeRefs {
		t.Fatal("ordinary replay changed source vector", *block)
	}

	if receipt.index != 1 || receipt.invocation == 0 || block.resetKind != paramsResetKey {
		t.Fatal("matched source index and kind", receipt, *block)
	}
	stateBefore := r.State()
	cursorBefore, stageBefore := block.resetCursor, block.resetStage
	out := receipt
	paramsMust(t, c.ResetKeyTracked(&v, missingKey, &out))
	missingName := "never-found"
	paramsMust(t, c.ResetKeyNameTracked(&v, &missingName, &out))
	if out != receipt || r.State() != stateBefore || block.resetCursor != cursorBefore || block.resetStage != stageBefore || block.size != beforeSize || block.refs != beforeRefs {
		t.Fatal("defined absent-key source shortcut changed stopped owner", out, *block)
	}
	noopRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &second}}})
	if noopRoots != after {
		t.Fatal("defined no-op changed physical root census", after, noopRoots)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	wrong := receipt
	wrong.index = 0
	if e := c.CompleteResetKey(&v, wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("different index", e)
	}
	wrong = receipt
	wrong.invocation++
	if e := c.CompleteResetKey(&v, wrong); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("different invocation", e)
	}
	if e := c.CompleteResetKey(&v, ParamsResetKeyReceipt{}); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("empty receipt", e)
	}
	if e := c.CompleteReset(&v, ParamsResetReceipt{owner: receipt.owner, slot: receipt.slot, generation: receipt.generation, invocation: receipt.invocation}); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("full reset cannot consume reset-key", e)
	}
	if e := c.SetUInt(&v, prefix, 99); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("mutation cannot replace retained leaf/index", e)
	}
	var secondReceipt ParamsResetKeyReceipt
	keyName := "actual-large-numeral"
	if e := c.ResetKeyNameTracked(&second, &keyName, &secondReceipt); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("second named key actual source stop", e)
	}
	both := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &second}}})
	if both.Named != before.Named || both.Objects+2 != before.Objects {
		t.Fatal(before, both)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if e := c.CompleteResetKey(&second, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("foreign stopped key cannot be adopted", e)
	}
	var dec ParamsDecrementReceipt
	paramsMust(t, c.DelTracked(&alias, &dec))
	if dec != (ParamsDecrementReceipt{}) || block.refs != beforeRefs-1 {
		t.Fatal("shared scalar DEC during key stop", dec, *block)
	}
	if e := c.DelTracked(&v, &dec); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("qualified last-reference key stop guard", e)
	}
	if dec != (ParamsDecrementReceipt{}) || block.refs != beforeRefs-1 || block.resetKind != paramsResetKey {
		t.Fatal("zeroDEC cannot overwrite key receipt", dec, *block)
	}
	paramsMust(t, c.CompleteResetKey(&v, receipt))
	if block.refs != beforeRefs-1 || block.size != beforeSize-1 || c.entry(block, 0).key != prefix || c.entry(block, 1).key != tail || c.entry(block, 1).bits != 84 {
		t.Fatal("source ascending suffix shift then pop", *block)
	}
	got, e = c.GetUInt(&v, tail, 0)
	paramsMust(t, e)
	if got != 84 {
		t.Fatal("shared shifted suffix", got)
	}
	got, e = c.GetUInt(&second, prefix, 0)
	paramsMust(t, e)
	if got != 42 || !c.blocks[second.slot-1].resetActive {
		t.Fatal("other pending key remains source visible", got)
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &second}}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.CompleteResetKey(&second, secondReceipt))
	if c.blocks[second.slot-1].size != 2 || c.entry(&c.blocks[second.slot-1], 1).key != tail {
		t.Fatal("second actual name route shift")
	}
	if e := c.CompleteResetKey(&v, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("double key acknowledgment", e)
	}
	if !errors.Is(c.decrementReceipts[v.slot-1].cause, arena.ErrBounds) {
		t.Fatal("original source key stop history lost")
	}

	paramsMust(t, c.Del(&v))
	paramsMust(t, c.Del(&second))
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	replacement, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&replacement, prefix, 12))
	if replacement.slot != receipt.slot || replacement.generation == receipt.generation {
		t.Fatal("actual key slot reuse", replacement, receipt)
	}
	if e := c.CompleteResetKey(&replacement, receipt); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("stale key generation", e)
	}
	sentinel := receipt
	missing, e := sy.Intern("missing-key")
	paramsMust(t, e)
	paramsMust(t, c.ResetKeyTracked(&replacement, missing, &sentinel))
	if sentinel != receipt {
		t.Fatal("unmatched key cannot stamp/adopt prior action")
	}
	paramsMust(t, c.Del(&replacement))
}

func TestParamsReferenceResetKeyFiniteAscendingShiftAndNoReservation(t *testing.T) {
	for route := 0; route < 2; route++ {
		for n := 0; n <= 6; n++ {
			for target := 0; target <= n; target++ {
				t.Run(strconv.Itoa(route)+"/"+strconv.Itoa(n)+"/"+strconv.Itoa(target), func(t *testing.T) {
					c, sy, _ := paramsTestOwner(t)
					v, e := c.NewValue()
					paramsMust(t, e)
					keys := make([]Symbol, n+1)
					names := make([]string, n+1)
					for i := range keys {
						names[i] = "key" + strconv.Itoa(i)
						keys[i], e = sy.Intern(names[i])
						paramsMust(t, e)
						if i < n {
							paramsMust(t, c.SetUInt(&v, keys[i], uint32(i*37+11)))
						}
					}
					alias, e := c.NewValue()
					paramsMust(t, e)
					paramsMust(t, c.Assign(&alias, &v))
					before := c.budget.Used()
					c.budget.FailAfter(0)
					t.Cleanup(func() { c.budget.FailAfter(-1) })
					var r ParamsResetKeyReceipt
					if route == 0 {
						e = c.ResetKeyTracked(&v, keys[target], &r)
					} else {
						e = c.ResetKeyNameTracked(&v, &names[target], &r)
					}
					paramsMust(t, e)
					if c.budget.Used() != before {
						t.Fatal("source scalar shift reserved storage", before, c.budget.Used())
					}
					if target == n {
						if r != (ParamsResetKeyReceipt{}) {
							t.Fatal("unmatched key acquired invocation", r)
						}
					} else {
						if r.index != uint32(target) || r.invocation == 0 {
							t.Fatal("actual matched index", r)
						}
					}
					want := n
					if target < n {
						want--
					}
					if v.slot == 0 {
						if want != 0 {
							t.Fatal("source null vector")
						}
						return
					}
					block := &c.blocks[v.slot-1]
					if int(block.size) != want || block.refs != 2 || v != alias {
						t.Fatal("source size/refcount", *block)
					}
					j := 0
					for i := 0; i < n; i++ {
						if i == target {
							continue
						}
						x := c.entry(block, uint32(j))
						if x.key != keys[i] || x.kind != ParamUInt || x.bits != uint64(i*37+11) {
							t.Fatal("independent ascending removal", i, j, *x)
						}
						j++
					}
				})
			}
		}
	}
}
