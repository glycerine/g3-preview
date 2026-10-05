// Source: src/util/params.cpp:992-1014.
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

func TestParamsReferenceSetRatAppendRefusalRetainsActualLocal(t *testing.T) {
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
	// Fill actual published vector capacity through ordinary source setters.
	for c.blocks[v.slot-1].size < c.blocks[v.slot-1].capacity {
		key, e := sy.Intern("fill" + strconv.Itoa(int(c.blocks[v.slot-1].size)))
		paramsMust(t, e)
		paramsMust(t, c.SetUInt(&v, key, 84))
	}
	before, e := q.Stats()
	paramsMust(t, e)
	token := v
	size := c.blocks[v.slot-1].size
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	beforeRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	bc.FailAfter(0)
	e = c.SetRat(&v, k, &big)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual destination append refusal after numeral copy", e)
	}
	after, e := q.Stats()
	paramsMust(t, e)
	if v != token || c.blocks[v.slot-1].size != size {
		t.Fatal("source vector publication before failed append", v, token, c.blocks[v.slot-1].size, size)
	}
	got, e := c.GetUInt(&v, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("published old scalar prefix", got)
	}
	if after.LiveCells != before.LiveCells+1 {
		t.Fatal("source copied untransferred rational local must remain owned before explicit abort", before.LiveCells, after.LiveCells)
	}
	afterRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if afterRoots.Named != beforeRoots.Named+1 || afterRoots.Objects != beforeRoots.Objects+1 {
		t.Fatal("actual declared copied local physical ownership", beforeRoots, afterRoots)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	receipt, e := c.StoppedSet(&v)
	paramsMust(t, e)
	wrong := receipt
	wrong.invocation++
	if !errors.Is(c.CleanupSet(wrong), ErrParamsPrecondition) {
		t.Fatal("foreign invocation accepted")
	}
	wrong = receipt
	wrong.generation++
	if !errors.Is(c.CleanupSet(wrong), ErrParamsPrecondition) {
		t.Fatal("foreign generation accepted")
	}
	if !errors.Is(c.SetUInt(&v, prefix, 99), ErrParamsPrecondition) {
		t.Fatal("stopped local overwritten by ordinary mutation")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	// Explicit owning abort releases the actual copied local, with the
	// destination allocation refusal still active. It never repeats append.
	paramsMust(t, c.CleanupSet(receipt))
	if !errors.Is(c.setFrames[v.slot-1].cause, arena.ErrInjectedFailure) {
		t.Fatal("source body cause lost")
	}
	if !errors.Is(c.CleanupSet(receipt), ErrParamsPrecondition) {
		t.Fatal("double acknowledgment")
	}
	afterAbort := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if afterAbort.Named != beforeRoots.Named || afterAbort.Objects != beforeRoots.Objects {
		t.Fatal("owning abort did not release only the real local", beforeRoots, afterAbort)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, q.Del(&big))
}

// The public TLS recipe reaches the signed-overflow guarded domain only.
// This is an owning continuation proof, not valid-domain allocation failure.
func TestParamsReferenceSetRatOwningAbortActualConsumedLeaf(t *testing.T) {
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
	// Fill actual published vector capacity through ordinary source setters.
	for c.blocks[v.slot-1].size < c.blocks[v.slot-1].capacity {
		key, e := sy.Intern("fill" + strconv.Itoa(int(c.blocks[v.slot-1].size)))
		paramsMust(t, e)
		paramsMust(t, c.SetUInt(&v, key, 84))
	}
	before, e := q.Stats()
	paramsMust(t, e)
	token := v
	size := c.blocks[v.slot-1].size
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	beforeRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	bc.FailAfter(0)
	if e = c.SetRat(&v, k, &big); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	receipt, e := c.StoppedSet(&v)
	paramsMust(t, e)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	e = c.CleanupSet(receipt)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual postphysical TLS stop", e)
	}
	f := &c.setFrames[v.slot-1]
	if !f.active || !f.acquired || !errors.Is(f.cause, arena.ErrInjectedFailure) || !errors.Is(f.cleanupCause, arena.ErrBounds) {
		t.Fatal("retained exact source/own causes", *f)
	}
	stopped := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if stopped.Named != beforeRoots.Named+1 || stopped.Objects != beforeRoots.Objects {
		t.Fatal("consumed raw physical frontier retained descriptor", beforeRoots, stopped)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.CleanupSet(receipt))
	drained := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if drained.Named != beforeRoots.Named || drained.Objects != beforeRoots.Objects {
		t.Fatal("no leaf replay or raw double release", beforeRoots, drained)
	}
	if !errors.Is(f.cause, arena.ErrInjectedFailure) || !errors.Is(f.cleanupCause, arena.ErrBounds) {
		t.Fatal("source/cleanup history changed")
	}
	if v != token || c.blocks[v.slot-1].size != size {
		t.Fatal("published destination changed")
	}
	after, e := q.Stats()
	paramsMust(t, e)
	if after.LiveCells != before.LiveCells {
		t.Fatal("retained descriptor not consumed")
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	bn.FailAfter(-1)
	paramsDecFinish(t, func() error { return q.Del(&big) })
}

func TestParamsReferenceSetRatIndependentLocalsAndDeadDestination(t *testing.T) {
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
	// Fill actual published vector capacity through ordinary source setters.
	for c.blocks[v.slot-1].size < c.blocks[v.slot-1].capacity {
		key, e := sy.Intern("fill" + strconv.Itoa(int(c.blocks[v.slot-1].size)))
		paramsMust(t, e)
		paramsMust(t, c.SetUInt(&v, key, 84))
	}
	before, e := q.Stats()
	paramsMust(t, e)
	token := v
	size := c.blocks[v.slot-1].size
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	beforeRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	w, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&w, prefix, 21))
	for c.blocks[w.slot-1].size < c.blocks[w.slot-1].capacity {
		key, e := sy.Intern("second-fill" + strconv.Itoa(int(c.blocks[w.slot-1].size)))
		paramsMust(t, e)
		paramsMust(t, c.SetUInt(&w, key, 63))
	}
	extras.ParamsValues = append(extras.ParamsValues, paramsPhysicalValueOwner{c, &w})
	beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	bc.FailAfter(0)
	for _, dst := range []*ParamsRef{&v, &w} {
		if e := c.SetRat(dst, k, &big); !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
	}
	first, e := c.StoppedSet(&v)
	paramsMust(t, e)
	second, e := c.StoppedSet(&w)
	paramsMust(t, e)
	if first.invocation == second.invocation {
		t.Fatal("independent invocation reused")
	}
	both := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if both.Named != beforeRoots.Named+2 || both.Objects != beforeRoots.Objects+2 {
		t.Fatal(beforeRoots, both)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Del(&v))
	if c.blocks[first.slot-1].live || !c.setFrames[first.slot-1].active {
		t.Fatal("retained local confused with dead destination")
	}
	paramsMust(t, c.CleanupSet(first))
	extras.ParamsValues = []paramsPhysicalValueOwner{{c, &w}}
	one := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if one.Named != beforeRoots.Named+1 || one.Objects != beforeRoots.Objects+1 || !c.setFrames[second.slot-1].active {
		t.Fatal("other local consumed", one)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.CleanupSet(second))
	if !errors.Is(c.CleanupSet(first), ErrParamsPrecondition) {
		t.Fatal("stale invocation accepted")
	}
	bc.FailAfter(-1)
	replacement, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&replacement, prefix, 9))
	if replacement.slot != first.slot || replacement.generation != first.generation+1 {
		t.Fatal("actual reusable generation", replacement, first)
	}
	if !errors.Is(c.CleanupSet(first), ErrParamsPrecondition) {
		t.Fatal("old generation acquired new local")
	}
	paramsMust(t, c.Del(&replacement))
	if c.blocks[token.slot-1].size != 0 || size == 0 || before.LiveCells == 0 {
		t.Fatal("fixture scalar authority")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, q.Del(&big))
}

func TestParamsReferenceSetRatAllActualCopyAndAppendPrefixes(t *testing.T) {
	for _, route := range []string{"copy", "append"} {
		t.Run(route, func(t *testing.T) {
			success := false
			for fault := 0; fault < 64; fault++ {
				t.Run(strconv.Itoa(fault), func(t *testing.T) {
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
					// Fill actual published vector capacity through ordinary source setters.
					for c.blocks[v.slot-1].size < c.blocks[v.slot-1].capacity {
						key, e := sy.Intern("fill" + strconv.Itoa(int(c.blocks[v.slot-1].size)))
						paramsMust(t, e)
						paramsMust(t, c.SetUInt(&v, key, 84))
					}
					before, e := q.Stats()
					paramsMust(t, e)
					token := v
					size := c.blocks[v.slot-1].size
					extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
					beforeRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
					if route == "copy" {
						bn.FailAfter(int64(fault))
					} else {
						bc.FailAfter(int64(fault))
					}
					e = c.SetRat(&v, k, &big)
					if e != nil {
						var failure memory.Failure
						if !errors.Is(e, arena.ErrInjectedFailure) && !(errors.As(e, &failure) && failure.ErrorCode() == 101 && r.State().OutOfMemory) {
							t.Fatal("unexpected actual source refusal", e)
						}
						if v != token || c.blocks[v.slot-1].size != size {
							t.Fatal("append failed publication", v, token)
						}
						got, e := c.GetUInt(&v, prefix, 0)
						paramsMust(t, e)
						if got != 42 {
							t.Fatal("old published prefix", got)
						}
						f := &c.setFrames[v.slot-1]
						if !f.active || !f.acquired || f.cause == nil {
							t.Fatal("untransferred acquired source local absent")
						}
						upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
						paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
						// The actual failure graph is audited before either provider is lifted.
						bn.FailAfter(-1)
						bc.FailAfter(-1)
						receipt, e := c.StoppedSet(&v)
						paramsMust(t, e)
						paramsMust(t, c.CleanupSet(receipt))
						end := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
						if end.Named != beforeRoots.Named || end.Objects != beforeRoots.Objects {
							t.Fatal("actual local abort left an owner", beforeRoots, end)
						}
					} else {
						success = true
						if v != token || c.blocks[v.slot-1].size != size+1 || c.setFrames[v.slot-1].active {
							t.Fatal("true entry publication/transfer")
						}
						next := c.entry(&c.blocks[v.slot-1], size)
						if next.key != k || next.kind != ParamNumeral {
							t.Fatal("actual appended entry")
						}
						upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
						paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
						bn.FailAfter(-1)
						bc.FailAfter(-1)
						equal, e := q.Equal(&next.rat, &big)
						paramsMust(t, e)
						if !equal {
							t.Fatal("literal numeric copy changed value")
						}
						after, e := q.Stats()
						paramsMust(t, e)
						if after.LiveCells != before.LiveCells+1 {
							t.Fatal("published actual numeric owner")
						}
					}
					paramsMust(t, q.Del(&big))
				})
				if success {
					t.Logf("first actual success %d", fault)
					break
				}
			}
			if !success {
				t.Fatal("actual source campaign did not reach success")
			}
		})
	}
}
