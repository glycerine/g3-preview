// Source: src/smt/params/qi_params.h:31-109; src/smt/params/qi_params.cpp:23-64.
// Source author: Leonardo de Moura (2008-06-15; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"os"
	"reflect"
	"testing"
)

func TestSMTQILiteralDefaultsUninitializedEMatchingAndOwnedText(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	if !got.DefaultsInitialized || got.EmatchingInitialized || got.MaxCexsIncrement != 0 || got.EagerThreshold != 10 || got.LazyThreshold != 20 || got.MaxLazyMultipatternMatching != 2 || !got.LazyQuickChecker || !got.PromoteUnsat || !got.MBQI || got.MaxCexs != 1 || got.MaxIterations != 1000 || got.ForceTemplate != 10 || got.ProfileFrequency != 4294967295 || got.MaxInstances != 4294967295 || !got.MBQIId.Present() {
		t.Fatal("literal defaults and exact generated override", got)
	}
	cost, e := v.CostText()
	paramsMust(t, e)
	text, e := cost.String()
	paramsMust(t, e)
	if text != "(+ weight generation)" {
		t.Fatal("owned default cost", text)
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	if e = v.Display(out); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("invented initialized ematching", e)
	}
	text, e = out.String()
	paramsMust(t, e)
	if text != "m_qi_ematching=" {
		t.Fatal("literal label before uninitialized source read", text)
	}
	paramsMust(t, out.Close())
}

func qiNamedColdGraph(t *testing.T, v *SMTQIParams, g *GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.ownerBytes + catalogGraph(g.config.Catalog)
	if g.current != nil {
		n += gparamsGraph(g.current)
	}
	for x := g.superseded; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	for x := g.failed; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	if v != nil {
		if v.self != v || v.budget != b {
			t.Fatal("exact QI wrapper identity")
		}
		n += v.header + uint64(len(v.cost.external)+len(v.cost.pending)+len(v.newGen.external)+len(v.newGen.pending))
		for f := v.helpers; f != nil; f = f.next {
			if f.self != f || f.owner != &v.smtModuleParamsOwner {
				t.Fatal("actual QI helper")
			}
			n += f.header
		}
	}
	if n != b.Used() {
		t.Fatal("actual once-only cold/payload graph before fault lift", n, b.Used())
	}
}
func TestSMTQIEveryConstructorReservationAndOwnedMemberUnwind(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		_, e := g.symbols.Intern("smt")
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTQIParams(c, p, g, b)
		qiNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual header/cost/helper acquisition", ordinal, cause)
		}
		if v != nil && cause != nil {
			if v.state.DefaultsInitialized != (ordinal >= 2) {
				t.Fatal("actual scalar initialization frontier", ordinal, v.state.DefaultsInitialized)
			}
			if v.constructorReady || v.helpers != nil || len(v.cost.external) != 0 || len(v.newGen.external) != 0 {
				t.Fatal("completed member reverse source unwind", ordinal, v.completed)
			}
			copyOwner := *v
			if e = copyOwner.Close(); !errors.Is(e, ErrParamsOwner) {
				t.Fatal("shallow copied owning wrapper", e)
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		qiNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 3 {
		t.Fatal("wrapper then exact22-byte cost then helper", first)
	}
}
func TestSMTQIUpdateBorrowedIDOwnedCostAndLateAllocationPrefix(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	paramsMust(t, v.SetEmatching(true))
	id := "borrowed-ID"
	key := "mbqi.id"
	paramsMust(t, c.SetStringName(p, &key, &id))
	cost := "0123456789012345678901234567890123456789"
	key = "qi.cost"
	paramsMust(t, c.SetStringName(p, &key, &cost))
	paramsMust(t, c.SetUIntName(p, dynAckName("qi.max_multi_patterns"), 17))
	paramsMust(t, c.SetUIntName(p, dynAckName("mbqi.max_cexs_incr"), 9))
	paramsMust(t, c.SetDoubleName(p, dynAckName("qi.eager_threshold"), 3.25))
	old, e := v.CostText()
	paramsMust(t, e)
	b.FailAfter(1)
	cause := v.Update(p)
	if !errors.Is(cause, arena.ErrInjectedFailure) || v.state.MaxCexsIncrement != 9 || v.state.EagerThreshold != 3.25 || v.state.MaxEagerMultipatterns != 0 || !v.state.MBQIId.Present() || v.helpers != nil {
		t.Fatal("actual earlier getters published, later cost/newpattern skipped", cause, v.state)
	}
	txt, e := old.String()
	paramsMust(t, e)
	if txt != "(+ weight generation)" || v.cost.capacity != 21 {
		t.Fatal("old owned cost before successful replacement", txt, v.cost.capacity)
	}
	qiNamedColdGraph(t, v, g, b)
	b.FailAfter(-1)
	paramsMust(t, v.Update(p))
	if v.cost.capacity != 42 || v.state.MaxEagerMultipatterns != 17 {
		t.Fatal("actual geometric create/dispose/publish and later getter", v.cost.capacity, v.state)
	}
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("released old data pointer lifetime", e)
	}
	owned, e := v.CostText()
	paramsMust(t, e)
	paramsMust(t, c.Reset(p))
	txt, e = owned.String()
	paramsMust(t, e)
	if txt != cost {
		t.Fatal("std::string did not copy actual borrowed chars", txt)
	}
	if _, e = v.state.MBQIId.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("mbqi_id was cloned instead of borrowed", e)
	}
	qiNamedColdGraph(t, v, g, b)
}

const qiDefaultDisplay = `m_qi_ematching=1
m_qi_cost=(+ weight generation)
m_qi_new_gen=cost
m_qi_eager_threshold=10
m_qi_lazy_threshold=20
m_qi_max_eager_multipatterns=0
m_qi_max_lazy_multipattern_matching=2
m_qi_profile=0
m_qi_profile_freq=4294967295
m_qi_quick_checker=0
m_qi_lazy_quick_checker=1
m_qi_promote_unsat=1
m_qi_max_instances=4294967295
m_qi_lazy_instantiation=0
m_qi_conservative_final_check=0
m_mbqi=1
m_mbqi_max_cexs=1
m_mbqi_max_cexs_incr=0
m_mbqi_max_iterations=1000
m_mbqi_trace=0
m_mbqi_force_template=10
m_mbqi_id=
`

func TestSMTQIDisplayAllChainsOutputPrefixesAndNullBadbit(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	paramsMust(t, v.SetEmatching(true))
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := v.Display(out)
		text, e := out.String()
		paramsMust(t, e)
		if len(text) > len(qiDefaultDisplay) || text != qiDefaultDisplay[:len(text)] {
			t.Fatal("independent22-chain prefix", ordinal, text, cause)
		}
		if cause == nil && text != qiDefaultDisplay {
			t.Fatal("complete22-fields", text)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		stats := out.Stats()
		if sb.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
			t.Fatal("actual output cold graph before fault lift", ordinal, sb.Used(), stats)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("output ownership", sb.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 1 {
		t.Fatal("actual output reservations", first)
	}
	key := "mbqi.id"
	paramsMust(t, c.SetStringName(p, &key, nil))
	paramsMust(t, v.Update(p))
	if v.state.MBQIId.Present() {
		t.Fatal("matching null pointer lost")
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, v.Display(out))
	paramsMust(t, out.AppendCString("erased-after-badbit"))
	text, e := out.String()
	paramsMust(t, e)
	if !out.fileBad || text != qiDefaultDisplay[:len(qiDefaultDisplay)-1] {
		t.Fatal("native null CString badbit, final endl and later insertions suppressed", text, out.fileBad)
	}
	paramsMust(t, out.Close())
	qiNamedColdGraph(t, v, g, b)
}

func TestSMTQIAllFourteenGettersIndependentFallbackAndUnchangedFields(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	module, e := g.symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.current.getParams(module)
	paramsMust(t, e)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	paramsMust(t, v.SetEmatching(true))
	cases := []struct {
		key, field string
		kind       uint8
		u          uint32
		d          float64
		b          bool
	}{
		{"mbqi", "MBQI", 0, 0, 0, true}, {"mbqi.max_cexs", "MaxCexs", 1, 1, 0, false}, {"mbqi.max_cexs_incr", "MaxCexsIncrement", 1, 0, 0, false}, {"mbqi.max_iterations", "MaxIterations", 1, 1000, 0, false}, {"mbqi.trace", "Trace", 0, 0, 0, false}, {"mbqi.force_template", "ForceTemplate", 1, 10, 0, false}, {"mbqi.id", "", 3, 0, 0, false}, {"qi.profile", "Profile", 0, 0, 0, false}, {"qi.profile_freq", "ProfileFrequency", 1, 4294967295, 0, false}, {"qi.max_instances", "MaxInstances", 1, 4294967295, 0, false}, {"qi.eager_threshold", "EagerThreshold", 2, 0, 10, false}, {"qi.lazy_threshold", "LazyThreshold", 2, 0, 20, false}, {"qi.cost", "", 4, 0, 0, false}, {"qi.max_multi_patterns", "MaxEagerMultipatterns", 1, 0, 0, false}}
	for _, tc := range cases {
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(global))
			wantB, wantU, wantD, wantText := tc.b, tc.u, tc.d, ""
			if tc.kind == 4 {
				wantText = "(+ weight generation)"
			}
			if mode < 3 {
				switch tc.kind {
				case 0:
					wantB = !tc.b
					paramsMust(t, c.SetBoolName(global, &tc.key, wantB))
				case 1:
					wantU = 77
					paramsMust(t, c.SetUIntName(global, &tc.key, wantU))
				case 2:
					wantD = 2.75
					paramsMust(t, c.SetDoubleName(global, &tc.key, wantD))
				default:
					wantText = "from-module"
					paramsMust(t, c.SetStringName(global, &tc.key, &wantText))
				}
			}
			if mode == 1 {
				switch tc.kind {
				case 0:
					wantB = tc.b
					paramsMust(t, c.SetBoolName(p, &tc.key, wantB))
				case 1:
					wantU = 13
					paramsMust(t, c.SetUIntName(p, &tc.key, wantU))
				case 2:
					wantD = 6.5
					paramsMust(t, c.SetDoubleName(p, &tc.key, wantD))
				default:
					wantText = "from-local"
					paramsMust(t, c.SetStringName(p, &tc.key, &wantText))
				}
			}
			if mode == 2 {
				if tc.kind == 0 {
					paramsMust(t, c.SetUIntName(p, &tc.key, 99))
				} else {
					paramsMust(t, c.SetBoolName(p, &tc.key, true))
				}
			}
			paramsMust(t, v.Update(p))
			state, e := v.State()
			paramsMust(t, e)
			switch tc.kind {
			case 0:
				if reflect.ValueOf(state).FieldByName(tc.field).Bool() != wantB {
					t.Fatal(tc, mode)
				}
			case 1:
				if reflect.ValueOf(state).FieldByName(tc.field).Uint() != uint64(wantU) {
					t.Fatal(tc, mode)
				}
			case 2:
				if reflect.ValueOf(state).FieldByName(tc.field).Float() != wantD {
					t.Fatal(tc, mode)
				}
			case 3:
				n, e := state.MBQIId.length()
				paramsMust(t, e)
				buf := make([]byte, int(n))
				for i := range buf {
					buf[i], e = state.MBQIId.Byte(uint64(i))
					paramsMust(t, e)
				}
				if string(buf) != wantText {
					t.Fatal("borrowed id precedence", tc, mode, string(buf), wantText)
				}
			case 4:
				text, e := v.CostText()
				paramsMust(t, e)
				got, e := text.String()
				paramsMust(t, e)
				if got != wantText {
					t.Fatal("owned cost precedence", tc, mode, got, wantText)
				}
			}
			if !state.EmatchingInitialized || !state.Ematching || state.MaxLazyMultipatternMatching != 2 || state.QuickChecker != 0 || !state.LazyQuickChecker || !state.PromoteUnsat || state.LazyInstantiation || state.ConservativeFinalCheck {
				t.Fatal("update touched absent fields", state)
			}
		}
	}
	qiNamedColdGraph(t, v, g, b)
}
func TestSMTQIEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
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
		v, cause := NewSMTQIParams(c, &local, g, b)
		n := g.ownerBytes + catalogGraph(catalog) + gparamsGraph(g.current) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if v != nil {
			n += v.header + uint64(len(v.cost.external)+len(v.cost.pending)+len(v.newGen.external)+len(v.newGen.pending))
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
	if first < 5 {
		t.Fatal("actual shared symbol/module acquisition absent", first)
	}
	t.Log("first successful unwarmed QI constructor", first)
}

const qiShowPosDisplay = `m_qi_ematching=+1
m_qi_cost=(+ weight generation)
m_qi_new_gen=cost
m_qi_eager_threshold=+10
m_qi_lazy_threshold=+20
m_qi_max_eager_multipatterns=0
m_qi_max_lazy_multipattern_matching=2
m_qi_profile=+0
m_qi_profile_freq=4294967295
m_qi_quick_checker=+0
m_qi_lazy_quick_checker=+1
m_qi_promote_unsat=+1
m_qi_max_instances=4294967295
m_qi_lazy_instantiation=+0
m_qi_conservative_final_check=+0
m_mbqi=+1
m_mbqi_max_cexs=1
m_mbqi_max_cexs_incr=0
m_mbqi_max_iterations=1000
m_mbqi_trace=+0
m_mbqi_force_template=10
m_mbqi_id=
`

func TestSMTQIShowPosAndFileBadbitStillReachesBorrowedCString(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	paramsMust(t, v.SetEmatching(true))
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	if got != qiShowPosDisplay {
		t.Fatal("independent signed bool/enum, unsigned and double profile", got)
	}
	paramsMust(t, out.Close())
	key, id := "mbqi.id", "borrowed-id"
	paramsMust(t, c.SetStringName(p, &key, &id))
	paramsMust(t, v.Update(p))
	file, e := os.CreateTemp(".", "qi-output-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.fileBad || out.filePosition != 0 {
		t.Fatal("real file badbit", out.fileBad, out.filePosition)
	}
	paramsMust(t, c.Reset(p))
	if e = v.Display(out); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("last CString strlen before badbit sentry", e)
	}
	paramsMust(t, out.Close())
	qiNamedColdGraph(t, v, g, b)
}
func TestSMTQIOwnedStringSameCapacityAliasAndNullCostPrefix(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTQIParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	old, e := v.CostText()
	paramsMust(t, e)
	key, text := "qi.cost", "abcd"
	paramsMust(t, c.SetStringName(p, &key, &text))
	paramsMust(t, v.Update(p))
	for i, ch := range []byte(text) {
		got, e := old.Byte(uint64(i))
		paramsMust(t, e)
		if got != ch {
			t.Fatal("captured pointer follows same-capacity mutation", i, got)
		}
	}
	if v.cost.capacity != 21 || v.cost.generation != old.generation {
		t.Fatal("no replacement for same capacity")
	}
	paramsMust(t, c.SetStringName(p, &key, nil))
	maxKey := "qi.max_multi_patterns"
	paramsMust(t, c.SetUIntName(p, &maxKey, 57))
	cause := v.Update(p)
	if !errors.Is(cause, ErrParamsStreamFailure) || v.state.MaxEagerMultipatterns != 0 || v.helpers != nil {
		t.Fatal("source null cost reached after earlier getters, before last getter", cause)
	}
	current, e := v.CostText()
	paramsMust(t, e)
	got, e := current.String()
	paramsMust(t, e)
	if got != text {
		t.Fatal("unread replacement on null pointer", got)
	}
	qiNamedColdGraph(t, v, g, b)
	text = "restored"
	paramsMust(t, c.SetStringName(p, &key, &text))
	paramsMust(t, v.Update(p))
	if v.state.MaxEagerMultipatterns != 57 {
		t.Fatal("completed object source-valid retry")
	}
	paramsMust(t, v.Close())
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("actual ended owned string view", e)
	}
	qiNamedColdGraph(t, nil, g, b)
}

func TestSMTQIEveryOwnedCostReplacementReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		v, e := NewSMTQIParams(c, p, g, b)
		paramsMust(t, e)
		old, e := v.CostText()
		paramsMust(t, e)
		key, text := "qi.cost", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN"
		paramsMust(t, c.SetStringName(p, &key, &text))
		maxKey := "qi.max_multi_patterns"
		paramsMust(t, c.SetUIntName(p, &maxKey, 19))
		b.FailAfter(ordinal)
		cause := v.Update(p)
		qiNamedColdGraph(t, v, g, b)
		if cause != nil {
			if !errors.Is(cause, arena.ErrInjectedFailure) || v.state.MaxEagerMultipatterns != 0 || v.cost.capacity != 21 || v.cost.pending != nil {
				t.Fatal("actual frame then char array refusal prefix", ordinal, cause)
			}
			got, e := old.String()
			paramsMust(t, e)
			if got != "(+ weight generation)" {
				t.Fatal("old payload before replacement publication", got)
			}
		} else {
			if v.cost.capacity != 42 || len(v.cost.external) != 43 || v.state.MaxEagerMultipatterns != 19 {
				t.Fatal("actual geometric array and final getter publication")
			}
			got, e := v.cost.view().String()
			paramsMust(t, e)
			if got != text {
				t.Fatal("ascending copied bytes", got)
			}
			if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("old address invalidated only by replacement", e)
			}
		}
		b.FailAfter(-1)
		paramsMust(t, v.Close())
		qiNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 2 {
		t.Fatal("one actual lexical frame and one actual replacement array", first)
	}
}

func TestSMTQIFailedCostDoesNotPublishLaterScalarDefaults(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	b.FailAfter(1) // Actual wrapper succeeds, first member's external chars refuse.
	v, cause := NewSMTQIParams(c, p, g, b)
	if !errors.Is(cause, arena.ErrInjectedFailure) || v == nil || v.completed != 0 {
		t.Fatal("real first owned-member refusal", v, cause)
	}
	defer func() { b.FailAfter(-1); paramsMust(t, v.Close()) }()
	qiNamedColdGraph(t, v, g, b)
	got, e := v.State()
	paramsMust(t, e)
	// These are observable Go diagnostic storage fields, not values of a valid
	// native incomplete object. No source scalar initializer has been reached.
	if got.DefaultsInitialized || got.MBQI || got.MaxCexs != 0 || got.EagerThreshold != 0 || got.ProfileFrequency != 0 {
		t.Fatal("later scalar default publication preceded failed cost construction", got)
	}
	qiNamedColdGraph(t, v, g, b)
	b.FailAfter(-1)
	paramsMust(t, v.Close())
	qiNamedColdGraph(t, nil, g, b)
}
