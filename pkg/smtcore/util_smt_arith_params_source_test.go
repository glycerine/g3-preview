// Source: src/smt/params/theory_arith_params.h:53-164;
// src/smt/params/theory_arith_params.cpp:23-94.
// Source author: Leonardo de Moura (2008-05-06).
// Licensed under the MIT license; see LICENSE.txt.
package smtcore

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSMTArithAllFortySixLiteralDefaults(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTArithParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	want := SMTArithState{Eq2Ineq: false, ProcessAllEqs: false, Mode: 2, AutoConfigSimplex: false, BlandsRuleThreshold: 1000, PropagateEqs: true, BoundProp: 2, StrongerLemmas: true, SkipRowsWithBigCoeffs: true, MaxLemmaSize: 128, SmallLemmaSize: 16, Reflect: true, IgnoreInt: false, LazyPivotingLevel: 0, RandomSeed: 0, RandomInitialValue: false, RandomLower: -1000, RandomUpper: 1000, Adaptive: false, AdaptiveAssertionThreshold: 0.2, AdaptivePropagationThreshold: 0.4, DumpLemmas: false, EagerEqAxioms: true, BranchCutRatio: 2, IntEqBranching: false, EnumConstMod: false, GCDTest: true, EagerGCD: false, AdaptiveGCD: false, PropagationThreshold: 4294967295, PivotStrategy: 0, AddBinaryBounds: false, PropagationStrategy: 1, EqBounds: false, LazyAdapter: false, Fixnum: false, IntOnly: false, NLArith: true, NLArithGB: true, NLArithGBThreshold: 512, NLArithGBEqs: false, NLArithGBPerturbate: true, NLArithMaxDegree: 6, NLArithBranching: true, NLArithRounds: 1024, EuclideanSolver: false}
	if got != want {
		t.Fatal("46 literal defaults and generated accessor defaults", got, want)
	}
}

func arithNamedColdGraph(t *testing.T, v *SMTArithParams, g *util.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		n += gparamsGraph(g.Current)
	}
	for p := g.Superseded; p != nil; p = p.RetainedNext {
		n += gparamsGraph(p)
	}
	for p := g.Failed; p != nil; p = p.RetainedNext {
		n += gparamsGraph(p)
	}
	if v != nil {
		if v.self != v || v.Budget != b || v.ModuleParamsOwner.Self != &v.ModuleParamsOwner {
			t.Fatal("exact arithmetic wrapper")
		}
		n += v.header
		for f := v.Helpers; f != nil; f = f.Next {
			if f.Self != f || f.Owner != &v.ModuleParamsOwner || f.Invocation == 0 || f.Invocation > v.Serial {
				t.Fatal("actual helper frame")
			}
			n += f.Header
		}
	}
	if n != b.Used() {
		t.Fatal("all named cold bytes before fault lift", n, b.Used())
	}
}
func TestSMTArithSixteenGettersIndependentLocalModuleLiteralPaths(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTArithParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	smt, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	sg, e := g.Current.GetParams(smt)
	paramsMust(t, e)
	rw, e := g.Symbols.Intern("rewriter")
	paramsMust(t, e)
	rg, e := g.Current.GetParams(rw)
	paramsMust(t, e)
	cases := []struct {
		key, field  string
		uintKind    bool
		defaultUInt uint32
		defaultBool bool
	}{
		{"arith.random_initial_value", "RandomInitialValue", false, 0, false}, {"random_seed", "RandomSeed", true, 0, false}, {"arith.solver", "Mode", true, 2, false},
		{"arith.nl", "NLArith", false, 0, true}, {"arith.nl.gb", "NLArithGB", false, 0, true}, {"arith.nl.branching", "NLArithBranching", false, 0, true}, {"arith.nl.rounds", "NLArithRounds", true, 1024, false},
		{"arith.euclidean_solver", "EuclideanSolver", false, 0, false}, {"arith.propagate_eqs", "PropagateEqs", false, 0, true}, {"arith.branch_cut_ratio", "BranchCutRatio", true, 2, false},
		{"arith.int_eq_branch", "IntEqBranching", false, 0, false}, {"arith.ignore_int", "IgnoreInt", false, 0, false}, {"arith.propagation_mode", "BoundProp", true, 2, false}, {"arith.dump_lemmas", "DumpLemmas", false, 0, false}, {"arith.reflect", "Reflect", false, 0, true}, {"eq2ineq", "Eq2Ineq", false, 0, false}}
	for _, tc := range cases {
		target := sg
		if tc.key == "eq2ineq" {
			target = rg
		}
		// Independent four local modes: absent, correct local, wrong local type,
		// and both absent with literal fallback. Other fields stay source-owned.
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(target))
			wantUInt, wantBool := tc.defaultUInt, tc.defaultBool
			if mode < 3 {
				if tc.uintKind {
					wantUInt = 0xfedcba98
					if tc.field == "Mode" {
						wantUInt = 3
					}
					if tc.field == "BoundProp" {
						wantUInt = 1
					}
					paramsMust(t, c.SetUIntName(target, &tc.key, wantUInt))
				} else {
					wantBool = !tc.defaultBool
					paramsMust(t, c.SetBoolName(target, &tc.key, wantBool))
				}
			}
			if mode == 1 {
				if tc.uintKind {
					wantUInt = 17
					if tc.field == "Mode" {
						wantUInt = 1
					}
					if tc.field == "BoundProp" {
						wantUInt = 0
					}
					paramsMust(t, c.SetUIntName(p, &tc.key, wantUInt))
				} else {
					wantBool = tc.defaultBool
					paramsMust(t, c.SetBoolName(p, &tc.key, wantBool))
				}
			}
			if mode == 2 {
				if tc.uintKind {
					paramsMust(t, c.SetBoolName(p, &tc.key, true))
				} else {
					paramsMust(t, c.SetUIntName(p, &tc.key, 8))
				}
			}
			before, _ := v.State()
			paramsMust(t, v.Update(p))
			after, e := v.State()
			paramsMust(t, e)
			field := testFieldBySourceName(reflect.ValueOf(after), tc.field)
			if tc.uintKind {
				if uint32(field.Uint()) != wantUInt {
					t.Fatal("uint precedence", tc, mode, field.Uint(), wantUInt)
				}
			} else if field.Bool() != wantBool {
				t.Fatal("bool precedence", tc, mode, field.Bool(), wantBool)
			}
			// None of the thirty non-getter fields may be changed by an update.
			a, z := reflect.ValueOf(before), reflect.ValueOf(after)
			for i := 0; i < a.NumField(); i++ {
				name := a.Type().Field(i).Name
				assigned := false
				for _, other := range cases {
					if name == other.field {
						assigned = true
						break
					}
				}
				if !assigned && a.Field(i).Interface() != z.Field(i).Interface() {
					t.Fatal("non-updated source member", name)
				}
			}
		}
	}
	arithNamedColdGraph(t, v, g, b)
}
func TestSMTArithSecondDeclarationRefusalKeepsFifteenFieldPrefix(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	for _, k := range []string{"smt", "rewriter"} {
		_, e := g.Symbols.Intern(k)
		paramsMust(t, e)
	}
	// Source p getters complete before the source ap declaration is attempted.
	for _, k := range []string{"arith.random_initial_value", "arith.euclidean_solver", "arith.int_eq_branch", "arith.ignore_int", "arith.dump_lemmas", "eq2ineq"} {
		paramsMust(t, c.SetBoolName(p, &k, true))
	}
	for _, k := range []string{"arith.nl", "arith.nl.gb", "arith.nl.branching", "arith.propagate_eqs", "arith.reflect"} {
		paramsMust(t, c.SetBoolName(p, &k, false))
	}
	for _, k := range []string{"random_seed", "arith.nl.rounds", "arith.branch_cut_ratio"} {
		paramsMust(t, c.SetUIntName(p, &k, 91))
	}
	keyMode, keyBound := "arith.solver", "arith.propagation_mode"
	paramsMust(t, c.SetUIntName(p, &keyMode, 3))
	paramsMust(t, c.SetUIntName(p, &keyBound, 1))
	b.FailAfter(2)
	v, cause := NewSMTArithParams(c, p, g, b)
	if v == nil || !errors.Is(cause, arena.ErrInjectedFailure) || v.Serial != 1 || v.Helpers != nil || v.constructorReady {
		t.Fatal("real second declaration reservation", v, cause)
	}
	arithNamedColdGraph(t, v, g, b)
	got, e := v.State()
	paramsMust(t, e)
	if !got.RandomInitialValue || got.RandomSeed != 91 || got.Mode != 3 || got.NLArith || got.NLArithGB || got.NLArithBranching || got.NLArithRounds != 91 || !got.EuclideanSolver || got.PropagateEqs || got.BranchCutRatio != 91 || !got.IntEqBranching || !got.IgnoreInt || got.BoundProp != 1 || !got.DumpLemmas || got.Reflect || got.Eq2Ineq {
		t.Fatal("exact first fifteen publications, untouched last eq2ineq", got)
	}
	sb := arena.NewBudget(^uint64(0))
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	sb.FailAfter(0)
	if e = v.Display(out); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("failed source constructor output entered", e)
	}
	if e = v.Update(p); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("failed source ctor reused", e)
	}
	text, e := out.String()
	paramsMust(t, e)
	if text != "" {
		t.Fatal("failed ctor output", text)
	}
	sb.FailAfter(-1)
	paramsMust(t, out.Close())
	b.FailAfter(-1)
	paramsMust(t, v.Close())
	arithNamedColdGraph(t, nil, g, b)
	// A completed object's partial update has the same live scalar prefix, and
	// ordinary source-valid retry after this prephysical metadata refusal works.
	v, e = NewSMTArithParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	k := "eq2ineq"
	paramsMust(t, c.SetBoolName(p, &k, false))
	b.FailAfter(1)
	cause = v.Update(p)
	if !errors.Is(cause, arena.ErrInjectedFailure) || !v.state.Eq2Ineq || v.Helpers != nil {
		t.Fatal("completed-object old final field and source unwind", cause, v.state)
	}
	arithNamedColdGraph(t, v, g, b)
	b.FailAfter(-1)
	paramsMust(t, v.Update(p))
	if v.state.Eq2Ineq {
		t.Fatal("literal retry final getter")
	}
}
func TestSMTArithEveryWarmConstructorReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		for _, k := range []string{"smt", "rewriter"} {
			_, e := g.Symbols.Intern(k)
			paramsMust(t, e)
		}
		b.FailAfter(ordinal)
		v, cause := NewSMTArithParams(c, p, g, b)
		arithNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("real constructor leaf", ordinal, cause)
		}
		if v != nil && (v.header != uint64(reflect.TypeOf(SMTArithParams{}).Size()) || v.Helpers != nil) {
			t.Fatal("exact actual header and completed unwind", ordinal)
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		arithNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 3 {
		t.Fatal("wrapper and two declaration frames", first)
	}
}

const arithDefaultDisplay = `m_arith_eq2ineq=0
m_arith_process_all_eqs=0
m_arith_mode=2
m_arith_auto_config_simplex=0
m_arith_blands_rule_threshold=1000
m_arith_propagate_eqs=1
m_arith_bound_prop=2
m_arith_stronger_lemmas=1
m_arith_skip_rows_with_big_coeffs=1
m_arith_max_lemma_size=128
m_arith_small_lemma_size=16
m_arith_reflect=1
m_arith_ignore_int=0
m_arith_lazy_pivoting_lvl=0
m_arith_random_seed=0
m_arith_random_initial_value=0
m_arith_random_lower=-1000
m_arith_random_upper=1000
m_arith_adaptive=0
m_arith_adaptive_assertion_threshold=0.2
m_arith_adaptive_propagation_threshold=0.4
m_arith_dump_lemmas=0
m_arith_eager_eq_axioms=1
m_arith_branch_cut_ratio=2
m_arith_int_eq_branching=0
m_arith_enum_const_mod=0
m_arith_gcd_test=1
m_arith_eager_gcd=0
m_arith_adaptive_gcd=0
m_arith_propagation_threshold=4294967295
m_arith_pivot_strategy=0
m_arith_add_binary_bounds=0
m_arith_propagation_strategy=1
m_arith_eq_bounds=0
m_arith_lazy_adapter=0
m_arith_fixnum=0
m_arith_int_only=0
m_nl_arith=1
m_nl_arith_gb=1
m_nl_arith_gb_threshold=512
m_nl_arith_gb_eqs=0
m_nl_arith_gb_perturbate=1
m_nl_arith_max_degree=6
m_nl_arith_branching=1
m_nl_arith_rounds=1024
m_arith_euclidean_solver=0
`
const arithShowPosDisplay = `m_arith_eq2ineq=+0
m_arith_process_all_eqs=+0
m_arith_mode=+2
m_arith_auto_config_simplex=+0
m_arith_blands_rule_threshold=1000
m_arith_propagate_eqs=+1
m_arith_bound_prop=+2
m_arith_stronger_lemmas=+1
m_arith_skip_rows_with_big_coeffs=+1
m_arith_max_lemma_size=128
m_arith_small_lemma_size=16
m_arith_reflect=+1
m_arith_ignore_int=+0
m_arith_lazy_pivoting_lvl=0
m_arith_random_seed=0
m_arith_random_initial_value=+0
m_arith_random_lower=-1000
m_arith_random_upper=+1000
m_arith_adaptive=+0
m_arith_adaptive_assertion_threshold=+0.2
m_arith_adaptive_propagation_threshold=+0.4
m_arith_dump_lemmas=+0
m_arith_eager_eq_axioms=+1
m_arith_branch_cut_ratio=2
m_arith_int_eq_branching=+0
m_arith_enum_const_mod=+0
m_arith_gcd_test=+1
m_arith_eager_gcd=+0
m_arith_adaptive_gcd=+0
m_arith_propagation_threshold=4294967295
m_arith_pivot_strategy=+0
m_arith_add_binary_bounds=+0
m_arith_propagation_strategy=+1
m_arith_eq_bounds=+0
m_arith_lazy_adapter=+0
m_arith_fixnum=+0
m_arith_int_only=+0
m_nl_arith=+1
m_nl_arith_gb=+1
m_nl_arith_gb_threshold=512
m_nl_arith_gb_eqs=+0
m_nl_arith_gb_perturbate=+1
m_nl_arith_max_degree=6
m_nl_arith_branching=+1
m_nl_arith_rounds=1024
m_arith_euclidean_solver=+0
`

func TestSMTArithFortySixDisplayChainsAllOutputPrefixesAndFlags(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTArithParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := v.Display(out)
		got, e := out.String()
		paramsMust(t, e)
		if !strings.HasPrefix(arithDefaultDisplay, got) {
			t.Fatal("independent 46-field insertion prefix", ordinal, got, cause)
		}
		if cause == nil && got != arithDefaultDisplay {
			t.Fatal("source default complete display", got)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual output reservation", cause)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("exact output once-only owner", sb.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 1 {
		t.Fatal("output no reservation horizon", first)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	if got != arithShowPosDisplay {
		t.Fatal("signed bool/enum/int/double and unsigned flag branches", got)
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "smt-arith-display-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("real nonthrowing ostream badbit", out.FileBad, out.FilePosition)
	}
	paramsMust(t, out.Close())
	arithNamedColdGraph(t, v, g, b)
}
func TestSMTArithEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		bc := arena.NewBudget(^uint64(0))
		c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := util.NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		local, e := c.NewValue()
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTArithParams(c, &local, g, b)
		n := g.OwnerBytes + catalogGraph(catalog) + gparamsGraph(g.Current) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if v != nil {
			n += v.header
			for f := v.Helpers; f != nil; f = f.Next {
				n += f.Header
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
	t.Log("first successful unwarmed arithmetic constructor", first)
}
func TestSMTArithNullGlobalConstructorAndForeignLocalReachedGuards(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	v, cause := NewSMTArithParams(c, p, g, b)
	if v == nil || v.constructorReady || v.Helpers != nil || v.Serial != 1 {
		t.Fatal("actual helper constructor source prefix", v, cause)
	}
	if util.GlobalParamsSourceDebug {
		if !errors.Is(cause, util.ErrGlobalParamsAssertion) {
			t.Fatal(cause)
		}
	} else if !errors.Is(cause, util.ErrGlobalParamsUninitialized) {
		t.Fatal(cause)
	}
	arithNamedColdGraph(t, v, g, b)
	copyOwner := *v
	if e := copyOwner.Close(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("copy cannot consume original", e)
	}
	paramsMust(t, v.Close())
	paramsMust(t, g.InitializeSource())
	v, e := NewSMTArithParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	fc, _, _ := paramsTestOwner(t)
	foreign, e := fc.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, fc.Del(&foreign)) }()
	before, _ := v.State()
	serial := v.Serial
	if e = v.Update(&foreign); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("actual first getter foreign local guard", e)
	}
	after, _ := v.State()
	if before != after || v.Helpers != nil || v.Serial != serial+1 {
		t.Fatal("foreign first getter changed fields/reached later declaration", after, v.Serial)
	}
	paramsMust(t, v.Update(p))
	arithNamedColdGraph(t, v, g, b)
}
