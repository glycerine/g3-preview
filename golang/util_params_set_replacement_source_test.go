// Source: src/util/params.cpp:946-1014,1017-1033.
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

func TestParamsReferenceMatchedReplacementRejectsConsumedNUMReplay(t *testing.T) {
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
	paramsMust(t, c.SetRat(&v, k, &big))
	beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	e = c.SetUInt(&v, k, 99)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual NUMDT consumed leaf", e)
	}
	if v != token || c.blocks[v.slot-1].size != size+1 || c.entry(&c.blocks[v.slot-1], size).kind != ParamNumeral {
		t.Fatal("replacement published past stopped leaf")
	}
	stopped := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if stopped.Named != beforeRoots.Named || stopped.Objects+1 != beforeRoots.Objects {
		t.Fatal("actual consumed payload retained descriptor", beforeRoots, stopped)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if e = c.SetUInt(&v, k, 99); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("ordinary replacement replay consumed a stopped NUMDT", e)
	}
	receipt, e := c.StoppedSetReplacement(&v)
	paramsMust(t, e)
	wrong := receipt
	wrong.invocation++
	if !errors.Is(c.CompleteSetReplacement(&v, wrong), ErrParamsPrecondition) {
		t.Fatal("foreign owning invocation")
	}
	paramsMust(t, c.CompleteSetReplacement(&v, receipt))
	got, e := c.GetUInt(&v, k, 0)
	paramsMust(t, e)
	if got != 99 {
		t.Fatal("literal scalar publication after exact owning ack", got)
	}
	if !errors.Is(c.CompleteSetReplacement(&v, receipt), ErrParamsPrecondition) {
		t.Fatal("double owning ack")
	}
	after, e := q.Stats()
	paramsMust(t, e)
	if after.LiveCells != before.LiveCells {
		t.Fatal("unexpected source numeral roots")
	}
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	paramsDecFinish(t, func() error { return q.Del(&big) })
}

func TestParamsReferenceMatchedReplacementSourceKindsAndLiveNoops(t *testing.T) {
	for _, kind := range []string{"bool", "uint", "double", "symbol", "owned-string", "borrowed-string"} {
		for _, byName := range []bool{false, true} {
			if kind == "borrowed-string" && byName {
				continue
			}
			t.Run(kind+"/"+strconv.FormatBool(byName), func(t *testing.T) {
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
				paramsMust(t, c.SetRat(&v, k, &big))
				replacement, e := sy.Intern("replacement-symbol")
				paramsMust(t, e)
				var setter func() error
				name := "actual-large-numeral"
				text := "literal copied text with real backing"
				switch kind {
				case "bool":
					if byName {
						setter = func() error { return c.SetBoolName(&v, &name, true) }
					} else {
						setter = func() error { return c.SetBool(&v, k, true) }
					}
				case "uint":
					if byName {
						setter = func() error { return c.SetUIntName(&v, &name, 99) }
					} else {
						setter = func() error { return c.SetUInt(&v, k, 99) }
					}
				case "double":
					if byName {
						setter = func() error { return c.SetDoubleName(&v, &name, 2.5) }
					} else {
						setter = func() error { return c.SetDouble(&v, k, 2.5) }
					}
				case "symbol":
					if byName {
						setter = func() error { return c.SetSymbolName(&v, &name, replacement) }
					} else {
						setter = func() error { return c.SetSymbol(&v, k, replacement) }
					}
				case "owned-string":
					if byName {
						setter = func() error { return c.SetStringName(&v, &name, &text) }
					} else {
						setter = func() error { return c.SetString(&v, k, &text) }
					}
				case "borrowed-string":
					setter = func() error { return c.SetStringSymbol(&v, k, replacement) }
				}
				beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				paramsDecTLSNearMinimum(t, r)
				bn.FailAfter(0)
				if e = setter(); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("actual NUMDT leaf stop", e)
				}
				bc.FailAfter(0)
				f := &c.replacementFrames[v.slot-1]
				if !f.active || f.index != size || f.oldKind != ParamNumeral || f.phase != 0 || f.leaf != c.entry(&c.blocks[v.slot-1], size).rat {
					t.Fatal("actual guarded old leaf binding")
				}
				receipt, e := c.StoppedSetReplacement(&v)
				paramsMust(t, e)
				stopped := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				if stopped.Named != beforeRoots.Named || stopped.Objects+1 != beforeRoots.Objects {
					t.Fatal(beforeRoots, stopped)
				}
				paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
				absent := Symbol{} // Numerical key 17 never appears in this actual fixture.
				absent = NumericalSymbol(17)
				oldSerial := c.sourceSerial
				sentinel := ParamsResetKeyReceipt{invocation: 123, index: 456}
				paramsMust(t, c.ResetKeyTracked(&v, absent, &sentinel))
				missing := "never-added-replacement-key"
				paramsMust(t, c.ResetKeyNameTracked(&v, &missing, &sentinel))
				if sentinel.invocation != 123 || sentinel.index != 456 || c.sourceSerial != oldSerial {
					t.Fatal("defined absent RESET touched receipt/frame")
				}
				found, e := c.Contains(&v, k)
				paramsMust(t, e)
				if !found {
					t.Fatal("source Contains reads key only")
				}
				found, e = c.ContainsName(&v, &missing)
				paramsMust(t, e)
				if found {
					t.Fatal("absent Contains")
				}
				old, e := c.GetUInt(&v, prefix, 0)
				paramsMust(t, e)
				if old != 42 {
					t.Fatal("unrelated scalar read")
				}
				if !errors.Is(setter(), ErrParamsPrecondition) {
					t.Fatal("ordinary stopped setter replay")
				}
				wrong := receipt
				wrong.index++
				if !errors.Is(c.CompleteSetReplacement(&v, wrong), ErrParamsPrecondition) {
					t.Fatal("foreign index")
				}
				upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
				paramsMust(t, c.CompleteSetReplacement(&v, receipt))
				x := c.entry(&c.blocks[v.slot-1], size)
				switch kind {
				case "bool":
					if x.kind != ParamBool || x.bits != 1 {
						t.Fatal("source bool field", x)
					}
				case "uint":
					if x.kind != ParamUInt || x.bits != 99 {
						t.Fatal("source unsigned field", x)
					}
				case "double":
					if x.kind != ParamDouble || x.bits != 0x4004000000000000 {
						t.Fatal("source double field", x)
					}
				case "symbol":
					if x.kind != ParamSymbol || x.symbol != replacement {
						t.Fatal("source symbol field", x)
					}
				case "owned-string", "borrowed-string":
					upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
					paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
					bc.FailAfter(-1) // Caller-owned Go string extraction has its own charged workspace.
					value, present, e := c.GetString(&v, k, nil)
					paramsMust(t, e)
					want := text
					if kind == "borrowed-string" {
						want = "replacement-symbol"
					}
					if x.kind != ParamString || !present || value != want {
						t.Fatal("source string publication", value, present)
					}
				}
				if v != token || c.blocks[v.slot-1].size != size+1 || f.active || !errors.Is(f.cause, arena.ErrBounds) {
					t.Fatal("matched publication source history")
				}
				after, e := q.Stats()
				paramsMust(t, e)
				if after.LiveCells != before.LiveCells {
					t.Fatal("exact old NUM release")
				}
				upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
				bn.FailAfter(-1)
				bc.FailAfter(-1)
				paramsDecFinish(t, func() error { return q.Del(&big) })
			})
		}
	}
}

func TestParamsReferenceMatchedReplacementContextAbortDoesNotPublish(t *testing.T) {
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
	older, e := sy.Intern("older-numeral")
	paramsMust(t, e)
	paramsMust(t, c.SetRat(&v, older, &big))
	paramsMust(t, c.SetRat(&v, k, &big))
	beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	text := "actual retained copied argument to discard"
	if e = c.SetString(&v, k, &text); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual first NUMDT stop", e)
	}
	bc.FailAfter(0)
	receipt, e := c.StoppedSetReplacement(&v)
	paramsMust(t, e)
	f := &c.replacementFrames[v.slot-1]
	if !f.next.text.present || f.next.text.borrowed || c.texts.Stats().Live != 1 {
		t.Fatal("actual prepared argument absent")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if e = c.Del(&v); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("last DEC adopted setter leaf", e)
	}
	if e = c.Close(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual older NUMDT stops context owning close", e)
	}
	b := &c.blocks[token.slot-1]
	if !b.live || !b.resetActive || b.resetKind != paramsResetLive || b.size != size+2 || f.active || f.phase != 2 || c.entry(b, size+1).kind == ParamString || *c.entry(b, size+1) != (paramEntry{}) {
		t.Fatal("own abort published replacement setter body")
	}
	if f.invocation != receipt.invocation || !errors.Is(f.cause, arena.ErrBounds) || c.texts.Stats().Live != 0 {
		t.Fatal("exact abort history/text refund")
	}
	afterRoots := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if afterRoots.Named+1 != beforeRoots.Named || afterRoots.Objects+2 != beforeRoots.Objects {
		t.Fatal("two actual physical releases, target descriptor ack only", beforeRoots, afterRoots)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Close())
	if e = c.CompleteSetReplacement(&v, receipt); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("closed context reused receipt", e)
	}
	after, e := q.Stats()
	paramsMust(t, e)
	if after.LiveCells != before.LiveCells {
		t.Fatal("old copied roots not drained")
	}
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	paramsDecFinish(t, func() error { return q.Del(&big) })
}

func TestParamsReferenceMatchedReplacementAllPreparedTextPrefixes(t *testing.T) {
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
			paramsMust(t, c.SetRat(&v, k, &big))
			beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
			text := "prepared copied text before the real old numeral destructor"
			bc.FailAfter(int64(fault))
			e = c.SetString(&v, k, &text)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("actual copied-string reservation", e)
				}
				if c.replacementFrames[v.slot-1].active || c.entry(&c.blocks[v.slot-1], size).kind != ParamNumeral || c.texts.Stats().Live != 0 {
					t.Fatal("preparation failure reached old NUMDT or lost acquired text")
				}
				end := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				if end.Named != beforeRoots.Named || end.Objects != beforeRoots.Objects {
					t.Fatal("prepare failure changed old root", beforeRoots, end)
				}
			} else {
				success = true
				if c.entry(&c.blocks[v.slot-1], size).kind != ParamString || c.replacementFrames[v.slot-1].active {
					t.Fatal("actual text publication")
				}
				end := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
				if end.Named+1 != beforeRoots.Named || end.Objects+1 != beforeRoots.Objects {
					t.Fatal("old numeral not retired", beforeRoots, end)
				}
			}
			if v != token || c.blocks[v.slot-1].size != size+1 {
				t.Fatal("actual matched vector size")
			}
			paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
			bc.FailAfter(-1)
			if success {
				value, present, e := c.GetString(&v, k, nil)
				paramsMust(t, e)
				if !present || value != text {
					t.Fatal("source text value", value, present)
				}
			}
			after, e := q.Stats()
			paramsMust(t, e)
			want := before.LiveCells + 1
			if success {
				want = before.LiveCells
			}
			if after.LiveCells != want {
				t.Fatal("actual numeral roots", after.LiveCells, want)
			}
			paramsMust(t, q.Del(&big))
		})
		if success {
			t.Logf("first actual text success %d", fault)
			break
		}
	}
	if !success {
		t.Fatal("actual text preparation sweep did not succeed")
	}
}

func TestParamsReferenceMatchedReplacementIndependentFramesAndReuse(t *testing.T) {
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
	paramsMust(t, c.SetRat(&v, k, &big))
	w, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&w, prefix, 21))
	paramsMust(t, c.SetRat(&w, k, &big))
	replacement, e := sy.Intern("another-replacement")
	paramsMust(t, e)
	extras.ParamsValues = append(extras.ParamsValues, paramsPhysicalValueOwner{c, &w})
	beforeRoots = upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	if e = c.SetUInt(&v, k, 99); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e = c.SetSymbol(&w, k, replacement); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	first, e := c.StoppedSetReplacement(&v)
	paramsMust(t, e)
	second, e := c.StoppedSetReplacement(&w)
	paramsMust(t, e)
	if first.invocation == second.invocation {
		t.Fatal("distinct source calls share authority")
	}
	both := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if both.Named != beforeRoots.Named || both.Objects+2 != beforeRoots.Objects {
		t.Fatal("two actual consumed leaves", beforeRoots, both)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	foreign := first
	foreign.invocation = second.invocation
	if !errors.Is(c.CompleteSetReplacement(&v, foreign), ErrParamsPrecondition) || !errors.Is(c.CompleteSetReplacement(&v, second), ErrParamsPrecondition) {
		t.Fatal("foreign frame adopted")
	}
	paramsMust(t, c.CompleteSetReplacement(&v, first))
	one := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if one.Named+1 != beforeRoots.Named || one.Objects != both.Objects || !c.replacementFrames[w.slot-1].active {
		t.Fatal("unrelated frame consumed", one)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Del(&v))
	extras.ParamsValues = []paramsPhysicalValueOwner{{c, &w}}
	bc.FailAfter(-1)
	fresh, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&fresh, prefix, 9))
	if fresh.slot != first.slot || fresh.generation != first.generation+1 {
		t.Fatal("actual replacement generation", fresh, first)
	}
	if !errors.Is(c.CompleteSetReplacement(&fresh, first), ErrParamsPrecondition) {
		t.Fatal("stale completed frame acquired replacement")
	}
	extras.ParamsValues = append(extras.ParamsValues, paramsPhysicalValueOwner{c, &fresh})
	paramsMust(t, c.CompleteSetReplacement(&w, second))
	if !errors.Is(c.CompleteSetReplacement(&w, second), ErrParamsPrecondition) {
		t.Fatal("double completion")
	}
	got, e := c.GetSymbol(&w, k, Symbol{})
	paramsMust(t, e)
	if got != replacement {
		t.Fatal("actual second symbol field")
	}
	if c.blocks[token.slot-1].size != 1 || size == 0 {
		t.Fatal("generation fixture prefix")
	}
	after, e := q.Stats()
	paramsMust(t, e)
	if after.LiveCells != before.LiveCells {
		t.Fatal("both exact old leaf descriptors not consumed")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	bn.FailAfter(-1)
	paramsDecFinish(t, func() error { return q.Del(&big) })
}
