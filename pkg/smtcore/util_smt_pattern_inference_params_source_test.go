// Source: src/ast/pattern/pattern_inference_params.h:30-50;
// src/ast/pattern/pattern_inference_params.cpp:22-47; src/ast/pattern/pattern_inference_params_helper.pyg:1-12.
package smtcore

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTPatternTenMembersLiteralConstructorAndPIModule(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	pi, e := g.Symbols.Intern("pi")
	paramsMust(t, e)
	global, e := g.Current.GetParams(pi)
	paramsMust(t, e)
	key := "arith_weight"
	paramsMust(t, c.SetUIntName(global, &key, 23))
	v, e := NewSMTPatternInferenceParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	want := SMTPatternInferenceState{0, true, 1, false, 23, 10, true, -1, true, false, 0x3ff}
	if got != want {
		t.Fatal("two explicit initializers then eight generated pi publications", got, want)
	}
}

func patternNamedColdGraph(t *testing.T, v *SMTPatternInferenceParams, g *util.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		n += gparamsGraph(g.Current)
	}
	for x := g.Failed; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	for x := g.Superseded; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	if v != nil {
		if v.self != v || v.Budget != b {
			t.Fatal("actual pattern wrapper")
		}
		n += v.header
		for f := v.Helpers; f != nil; f = f.Next {
			if f.Self != f || f.Owner != &v.ModuleParamsOwner {
				t.Fatal("actual lexical helper")
			}
			n += f.Header
		}
	}
	if n != b.Used() {
		t.Fatal("immediate named cold graph", n, b.Used())
	}
}
func TestSMTPatternEightGettersIndependentFallbackAndUntouchedInitializers(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPatternInferenceParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	pi, e := g.Symbols.Intern("pi")
	paramsMust(t, e)
	global, e := g.Current.GetParams(pi)
	paramsMust(t, e)
	cases := []struct {
		key, field string
		isBool     bool
	}{{"max_multi_patterns", "MaxMultiPatterns", false}, {"block_loop_patterns", "BlockLoopPatterns", true}, {"arith", "Arithmetic", false}, {"use_database", "UseDatabase", true}, {"arith_weight", "ArithmeticWeight", false}, {"non_nested_arith_weight", "NonNestedArithmeticWeight", false}, {"pull_quantifiers", "PullQuantifiers", true}, {"warnings", "Warnings", true}}
	for _, tc := range cases {
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(global))
			want := SMTPatternInferenceState{0, true, 1, false, 5, 10, true, -1, true, false, 0x3ff}
			field := testFieldBySourceName(reflect.ValueOf(&want).Elem(), tc.field)
			if mode < 3 {
				if tc.isBool {
					field.SetBool(!field.Bool())
					paramsMust(t, c.SetBoolName(global, &tc.key, field.Bool()))
				} else {
					n := uint32(77)
					if tc.field == "Arithmetic" {
						n = 2
					}
					field.SetUint(uint64(n))
					paramsMust(t, c.SetUIntName(global, &tc.key, n))
				}
			}
			if mode == 1 {
				if tc.isBool {
					field.SetBool(!field.Bool())
					paramsMust(t, c.SetBoolName(p, &tc.key, field.Bool()))
				} else {
					n := uint32(13)
					if tc.field == "Arithmetic" {
						n = 0
					}
					field.SetUint(uint64(n))
					paramsMust(t, c.SetUIntName(p, &tc.key, n))
				}
			}
			if mode == 2 {
				if tc.isBool {
					paramsMust(t, c.SetUIntName(p, &tc.key, 99))
				} else {
					paramsMust(t, c.SetBoolName(p, &tc.key, true))
				}
			}
			paramsMust(t, v.Update(p))
			got, e := v.State()
			paramsMust(t, e)
			if got != want {
				t.Fatal("independent full ten-field state and initialization mask", tc, mode, got, want)
			}
		}
	}
	patternNamedColdGraph(t, v, g, b)
}
func TestSMTPatternEveryWarmConstructorReservationUninitializedPrefix(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		_, e := g.Symbols.Intern("pi")
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTPatternInferenceParams(c, p, g, b)
		patternNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		if v != nil && cause != nil {
			if v.constructorReady || v.state.Initialized != 0x180 || v.state.NoPatternWeight != -1 || !v.state.AvoidSkolems || v.Helpers != nil {
				t.Fatal("only literal two initializers before failed helper", ordinal, v.state)
			}
			copyOwner := *v
			if e = copyOwner.Close(); !errors.Is(e, util.ErrParamsOwner) {
				t.Fatal("copy guard", e)
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		patternNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 2 {
		t.Fatal("actual wrapper then lexical helper", first)
	}
}

const patternDefaultDisplay = "m_pi_max_multi_patterns=0\nm_pi_block_loop_patterns=1\nm_pi_arith=1\nm_pi_use_database=0\nm_pi_arith_weight=5\nm_pi_non_nested_arith_weight=10\nm_pi_pull_quantifiers=1\nm_pi_nopat_weight=-1\nm_pi_avoid_skolems=1\nm_pi_warnings=0\n"

func TestSMTPatternTenDisplayChainsGoBufferSignedEnumAndBool(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPatternInferenceParams(c, p, g, b)
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
		if len(got) > len(patternDefaultDisplay) || got != patternDefaultDisplay[:len(got)] {
			t.Fatal("ten independent chain prefix", ordinal, got, cause)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		if cause == nil && got != patternDefaultDisplay {
			t.Fatal("complete display", got)
		}
		stats := out.Stats()
		if sb.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
			t.Fatal("actual output graph before fault lift", ordinal, sb.Used(), stats)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("output own refund")
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 0 {
		t.Fatal("Go buffer output did not complete", first)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	want := "m_pi_max_multi_patterns=0\nm_pi_block_loop_patterns=+1\nm_pi_arith=+1\nm_pi_use_database=+0\nm_pi_arith_weight=5\nm_pi_non_nested_arith_weight=10\nm_pi_pull_quantifiers=+1\nm_pi_nopat_weight=-1\nm_pi_avoid_skolems=+1\nm_pi_warnings=+0\n"
	if got != want {
		t.Fatal("signed bool/enum/int vs unsigned source insertion", got)
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "pattern-output-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("real nonthrowing file badbit")
	}
	paramsMust(t, out.Close())
	patternNamedColdGraph(t, v, g, b)
}
func TestSMTPatternFailedConstructorDisplayAndUpdateUnread(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	v, cause := NewSMTPatternInferenceParams(c, p, g, b)
	if util.GlobalParamsSourceDebug {
		if !errors.Is(cause, util.ErrGlobalParamsAssertion) {
			t.Fatal(cause)
		}
	} else if !errors.Is(cause, util.ErrGlobalParamsUninitialized) {
		t.Fatal(cause)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	stats := b.Used()
	if e = v.Display(out); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("incomplete display", e)
	}
	if e = v.Update(p); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("incomplete update", e)
	}
	got, e := out.String()
	paramsMust(t, e)
	if got != "" || b.Used() != stats || v.state.Initialized != 0x180 {
		t.Fatal("no invented initialization or output")
	}
	paramsMust(t, out.Close())
	patternNamedColdGraph(t, v, g, b)
	paramsMust(t, v.Close())
	patternNamedColdGraph(t, nil, g, b)
}
func TestSMTPatternInferenceEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
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
		v, cause := NewSMTPatternInferenceParams(c, &local, g, b)
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
	if first != 4 {
		t.Fatal("actual shared symbol/module acquisition absent", first)
	}
	t.Log("first successful unwarmed pattern constructor", first)
}
