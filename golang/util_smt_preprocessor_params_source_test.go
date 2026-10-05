// Source: src/smt/params/preprocessor_params.h:31-81;
// src/smt/params/preprocessor_params.cpp:22-61; src/smt/params/smt_params_helper.pyg:9-11,19-20.
// Source author: Leonardo de Moura (2008-01-17; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"os"
	"reflect"
	"testing"
)

func TestSMTPreprocessorImplicitPatternBaseGetsEmptyInsteadOfParentArgument(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	pi, e := g.symbols.Intern("pi")
	paramsMust(t, e)
	global, e := g.current.getParams(pi)
	paramsMust(t, e)
	weight := "arith_weight"
	paramsMust(t, c.SetUIntName(global, &weight, 23))
	paramsMust(t, c.SetUIntName(p, &weight, 91))
	macro := "macro_finder"
	paramsMust(t, c.SetBoolName(p, &macro, true))
	v, e := NewSMTPreprocessorParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	base, e := v.PatternState()
	paramsMust(t, e)
	local, e := v.State()
	paramsMust(t, e)
	if base.ArithmeticWeight != 23 || !local.MacroFinder || !local.DefaultsInitialized || v.completedBases != 3 {
		t.Fatal("actual default-empty implicit base, then local update(p)", base, local)
	}
	paramsMust(t, v.Update(p))
	base, e = v.PatternState()
	paramsMust(t, e)
	if base.ArithmeticWeight != 91 {
		t.Fatal("explicit Update later forwards parent p to pattern", base)
	}
}

func preprocessorNamedColdGraph(t *testing.T, v *SMTPreprocessorParams, g *GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.ownerBytes + catalogGraph(g.config.Catalog)
	if g.current != nil {
		n += gparamsGraph(g.current)
	}
	for x := g.failed; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	for x := g.superseded; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	if v != nil {
		if v.self != v || v.budget != b || v.pattern.header != 0 {
			t.Fatal("actual parent / inline base without second header")
		}
		n += v.header
		for f := v.helpers; f != nil; f = f.next {
			if f.self != f || f.owner != &v.smtModuleParamsOwner {
				t.Fatal("actual parent lexical helper")
			}
			n += f.header
		}
		for f := v.pattern.helpers; f != nil; f = f.next {
			if f.self != f || f.owner != &v.pattern.smtModuleParamsOwner {
				t.Fatal("actual pattern lexical helper")
			}
			n += f.header
		}
	}
	if n != b.Used() {
		t.Fatal("immediate exact cold graph", n, b.Used())
	}
}
func TestSMTPreprocessorAllNineteenDefaultsAndFiveLocalGetters(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPreprocessorParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	literal := SMTPreprocessorState{0, 0, false, false, false, false, true, false, true, false, false, true, false, false, false, false, true, true, false, true}
	if got != literal {
		t.Fatal("nineteen declared defaults", got)
	}
	bit, initialized, e := v.BitBlasterState()
	paramsMust(t, e)
	if !initialized || bit.ExtGates || bit.Quantifiers {
		t.Fatal("actual second base after pattern")
	}
	smt, e := g.symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.current.getParams(smt)
	paramsMust(t, e)
	cases := []struct{ key, field string }{{"macro_finder", "MacroFinder"}, {"quasi_macros", "QuasiMacros"}, {"restricted_quasi_macros", "RestrictedQuasiMacros"}, {"pull_nested_quantifiers", "PullNestedQuantifiers"}, {"refine_inj_axioms", "RefineInjectivityAxiom"}}
	for _, tc := range cases {
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(global))
			want := literal
			field := reflect.ValueOf(&want).Elem().FieldByName(tc.field)
			if mode < 3 {
				field.SetBool(!field.Bool())
				paramsMust(t, c.SetBoolName(global, &tc.key, field.Bool()))
			}
			if mode == 1 {
				field.SetBool(!field.Bool())
				paramsMust(t, c.SetBoolName(p, &tc.key, field.Bool()))
			}
			if mode == 2 {
				paramsMust(t, c.SetUIntName(p, &tc.key, 99))
			}
			before, e := v.PatternState()
			paramsMust(t, e)
			paramsMust(t, v.UpdateLocal(p))
			got, e := v.State()
			paramsMust(t, e)
			after, e := v.PatternState()
			paramsMust(t, e)
			if got != want || before != after {
				t.Fatal("actual five local publications, inherited fields unread", tc, mode, got, want)
			}
		}
	}
	preprocessorNamedColdGraph(t, v, g, b)
}
func TestSMTPreprocessorEveryWarmConstructorReservationAndBaseUnwind(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 12; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		for _, name := range []string{"pi", "smt"} {
			_, e := g.symbols.Intern(name)
			paramsMust(t, e)
		}
		b.FailAfter(ordinal)
		v, cause := NewSMTPreprocessorParams(c, p, g, b)
		preprocessorNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		if v != nil && cause != nil {
			if ordinal == 1 {
				if v.completedBases != 0 || v.state.DefaultsInitialized || v.pattern.state.Initialized != 0x180 || v.bitEnded || v.patternEnded {
					t.Fatal("failed first base retains only its actual member prefix", v.state, v.completedBases)
				}
			}
			if ordinal == 2 {
				if v.completedBases != 3 || !v.state.DefaultsInitialized || !v.bitEnded || !v.patternEnded || !v.pattern.closed {
					t.Fatal("later body refusal source unwinds completed bases", v.completedBases)
				}
			}
			copyOwner := *v
			if e := copyOwner.Close(); !errors.Is(e, ErrParamsOwner) {
				t.Fatal("copied parent guard", e)
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		preprocessorNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 3 {
		t.Fatal("one parent header then pi and smt lexical frames", first)
	}
}
func TestSMTPreprocessorUpdatePatternPublicationBeforeLocalDeclarationRefusal(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPreprocessorParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	key := "arith_weight"
	paramsMust(t, c.SetUIntName(p, &key, 73))
	macro := "macro_finder"
	paramsMust(t, c.SetBoolName(p, &macro, true))
	before := v.state
	b.FailAfter(1)
	cause := v.Update(p)
	preprocessorNamedColdGraph(t, v, g, b)
	if !errors.Is(cause, arena.ErrInjectedFailure) || v.pattern.state.ArithmeticWeight != 73 || v.state != before || v.pattern.helpers != nil || v.helpers != nil {
		t.Fatal("pattern call completed before later local declaration", cause, v.state, v.pattern.state)
	}
	b.FailAfter(-1)
	paramsMust(t, v.Update(p))
	if !v.state.MacroFinder || v.pattern.state.ArithmeticWeight != 73 {
		t.Fatal("completed-object valid source retry")
	}
	preprocessorNamedColdGraph(t, v, g, b)
}

const preprocessorDefaultDisplay = patternDefaultDisplay + "m_bb_ext_gates=0\nm_bb_quantifiers=0\nm_lift_ite=0\nm_ng_lift_ite=0\nm_pull_cheap_ite=0\nm_pull_nested_quantifiers=0\nm_eliminate_term_ite=0\nm_macro_finder=0\nm_propagate_values=1\nm_propagate_booleans=0\nm_refine_inj_axiom=1\nm_eliminate_bounds=0\nm_simplify_bit2int=0\nm_nnf_cnf=1\nm_distribute_forall=0\nm_reduce_args=0\nm_quasi_macros=0\nm_restricted_quasi_macros=0\nm_max_bv_sharing=1\nm_pre_simplifier=1\nm_nlquant_elim=0\n"

func TestSMTPreprocessorThirtyOneDisplayChainsEveryOutputPrefixAndBadbit(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPreprocessorParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := v.Display(out)
		got, e := out.String()
		paramsMust(t, e)
		if len(got) > len(preprocessorDefaultDisplay) || got != preprocessorDefaultDisplay[:len(got)] {
			t.Fatal("independent31-chain prefix", ordinal, got, cause)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		if cause == nil && got != preprocessorDefaultDisplay {
			t.Fatal("complete inherited+local output", got)
		}
		stats := out.Stats()
		if sb.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
			t.Fatal("exact output cold graph before lift", ordinal, sb.Used(), stats)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("own output refund")
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 1 {
		t.Fatal("actual output reserves", first)
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	want := "m_pi_max_multi_patterns=0\nm_pi_block_loop_patterns=+1\nm_pi_arith=+1\nm_pi_use_database=+0\nm_pi_arith_weight=5\nm_pi_non_nested_arith_weight=10\nm_pi_pull_quantifiers=+1\nm_pi_nopat_weight=-1\nm_pi_avoid_skolems=+1\nm_pi_warnings=+0\nm_bb_ext_gates=+0\nm_bb_quantifiers=+0\nm_lift_ite=+0\nm_ng_lift_ite=+0\nm_pull_cheap_ite=+0\nm_pull_nested_quantifiers=+0\nm_eliminate_term_ite=+0\nm_macro_finder=+0\nm_propagate_values=+1\nm_propagate_booleans=+0\nm_refine_inj_axiom=+1\nm_eliminate_bounds=+0\nm_simplify_bit2int=+0\nm_nnf_cnf=+1\nm_distribute_forall=+0\nm_reduce_args=+0\nm_quasi_macros=+0\nm_restricted_quasi_macros=+0\nm_max_bv_sharing=+1\nm_pre_simplifier=+1\nm_nlquant_elim=+0\n"
	if got != want {
		t.Fatal("source inherited signed enum/bool/int vs unsigned", got)
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "preprocessor-output-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.fileBad || out.filePosition != 0 {
		t.Fatal("real file badbit continues all chains")
	}
	paramsMust(t, out.Close())
	preprocessorNamedColdGraph(t, v, g, b)
}
func TestSMTPreprocessorEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := NewSymbolManager(SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		bc := arena.NewBudget(^uint64(0))
		c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		local, e := c.NewValue()
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTPreprocessorParams(c, &local, g, b)
		n := g.ownerBytes + catalogGraph(catalog) + gparamsGraph(g.current) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if v != nil {
			n += v.header
			for f := v.pattern.helpers; f != nil; f = f.next {
				n += f.header
			}
			for f := v.helpers; f != nil; f = f.next {
				n += f.header
			}
		}
		if n != b.Used() {
			t.Fatal("exact shared graph before lift", ordinal, n, b.Used())
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("source constructor actual leaf", cause)
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		paramsMust(t, c.Del(&local))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bc.Used() != 0 {
			t.Fatal("all named refund once", ordinal, b.Used(), bc.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 6 {
		t.Fatal("actual shared symbol/module acquisition absent", first)
	}
	t.Log("first successful unwarmed preprocessor constructor", first)
}

func TestSMTPreprocessorFailedConstructorOrdinaryCallsAndOuterGuardUnread(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	_, e := g.symbols.Intern("pi")
	paramsMust(t, e)
	b.FailAfter(1)
	v, cause := NewSMTPreprocessorParams(c, p, g, b)
	if !errors.Is(cause, arena.ErrInjectedFailure) || v == nil {
		t.Fatal(cause)
	}
	b.FailAfter(-1)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	used := b.Used()
	if e = v.Display(out); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("incomplete Display", e)
	}
	if e = v.Update(p); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("incomplete Update", e)
	}
	got, e := out.String()
	paramsMust(t, e)
	if got != "" || b.Used() != used || v.completedBases != 0 {
		t.Fatal("unreached bases/output")
	}
	paramsMust(t, out.Close())
	preprocessorNamedColdGraph(t, v, g, b)
	paramsMust(t, v.Close())
	live, e := NewSMTPreprocessorParams(c, p, g, b)
	paramsMust(t, e)
	copyLive := *live
	used = b.Used()
	if e = copyLive.Close(); !errors.Is(e, ErrParamsOwner) || b.Used() != used || live.bitEnded || live.patternEnded {
		t.Fatal("copied outer owner before base/header mutation", e)
	}
	paramsMust(t, live.Close())
	preprocessorNamedColdGraph(t, nil, g, b)
}
func TestSMTPreprocessorConcreteLexicalStopProtectsBasesAndNextMutation(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("unreached polynomial checkpoint") }, Cooperation: func() error { panic("unreached polynomial cooperation") }})
	paramsMust(t, e)
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	paramsMust(t, e)
	// The concrete lexical fixture uses actual guarded helper acquisition,
	// public module reset and actual last-reference destruction.
	// No callback or source ordinary method interleaving is invented.
	smt, e := sy.Intern("smt")
	paramsMust(t, e)
	sg, e := g.current.getParams(smt)
	paramsMust(t, e)
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
	k, e := sy.Intern("large-real-leaf")
	paramsMust(t, e)
	paramsMust(t, c.SetRat(sg, k, &big))
	paramsMust(t, q.Del(&big))
	v, e := NewSMTPreprocessorParams(c, &local, g, b)
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		b.FailAfter(-1)
		v.running = false
		v.activeCall = false
		paramsDecFinish(t, v.Close)
		paramsDecFinish(t, g.Close)
		paramsDecFinish(t, catalog.Close)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Fatal("all named owner refunds", b.Used(), bn.Used(), bc.Used(), bs.Used())
		}
	})
	v.activeCall = true
	v.running = true
	younger, e := v.beginHelper(&local, "smt")
	paramsMust(t, e)
	paramsMust(t, g.SourceReset())
	patternSerial := v.pattern.serial
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &younger.value}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	cause := v.finishHelper(younger, nil)
	if !errors.Is(cause, arena.ErrBounds) || !younger.destructorStarted || younger.sourceDone || younger.receipt.invocation == 0 || v.destructorCause != younger.destructorCause {
		t.Fatal("actual younger source stop", cause, younger.receipt)
	}
	if v.bitEnded || v.patternEnded || v.pattern.serial != patternSerial {
		t.Fatal("older base lifetime unread after lexical stop")
	}
	v.running = false
	v.activeCall = false
	if e := v.Update(&local); !errors.Is(e, ErrParamsPrecondition) || v.pattern.serial != patternSerial {
		t.Fatal("new Update entered older base through stopped younger owner", e)
	}
	if e := v.UpdateLocal(&local); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("new local mutation through retained owner", e)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("exact consumed raw and retained leaf descriptor", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	preprocessorNamedColdGraph(t, v, g, b)
	v.running = false
	v.activeCall = false
	// Callback/foreign owning requests cannot replace or adopt the exact frame.
	copyOwner := *v
	if e := copyOwner.Close(); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("foreign copied outer owner", e)
	}
	paramsDecFinish(t, v.Close)
	if !errors.Is(younger.destructorCause, arena.ErrBounds) || younger.header != 0 || !younger.sourceDone || !errors.Is(v.destructorCause, arena.ErrBounds) {
		t.Fatal("immutable source cause and matched own completion")
	}
	preprocessorNamedColdGraph(t, v, g, b)
}
