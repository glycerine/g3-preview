// Source: src/smt/params/theory_str_params.h:25-143;
// src/smt/params/theory_str_params.cpp:21-40; src/smt/params/smt_params_helper.pyg:70-87.
package smtcore

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTStrSeventeenLiteralDefaultsAndMandatoryConstructorUpdate(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	key := "str.binary_search_start"
	paramsMust(t, c.SetUIntName(p, &key, 123))
	v, e := NewSMTStrParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	want := SMTStrState{true, false, false, true, false, true, true, false, false, 123, -0.1, true, 1000, 1000, 10, 10, 10}
	if got != want {
		t.Fatal("seventeen literal initializers, actual constructor update", got)
	}
	if v.Serial != 1 || v.Helpers != nil || !v.constructorReady {
		t.Fatal("one actual lexical helper completion")
	}
}

func strNamedColdGraph(t *testing.T, v *SMTStrParams, g *uparams.GlobalParamsService, b *arena.Budget) {
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
			t.Fatal("exact string params wrapper")
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
		t.Fatal("immediate actual cold graph", n, b.Used())
	}
}
func TestSMTStrAllSeventeenGettersIndependentFallbackAndWholeState(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTStrParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	module, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.Current.GetParams(module)
	paramsMust(t, e)
	cases := []struct {
		key, field string
		kind       uint8
	}{
		{"str.strong_arrangements", "StrongArrangements", 0}, {"str.aggressive_length_testing", "AggressiveLengthTesting", 0}, {"str.aggressive_value_testing", "AggressiveValueTesting", 0}, {"str.aggressive_unroll_testing", "AggressiveUnrollTesting", 0}, {"str.fast_length_tester_cache", "UseFastLengthTesterCache", 0}, {"str.fast_value_tester_cache", "UseFastValueTesterCache", 0}, {"str.string_constant_cache", "StringConstantCache", 0}, {"str.finite_overlap_models", "FiniteOverlapModels", 0}, {"str.use_binary_search", "UseBinarySearch", 0}, {"str.binary_search_start", "BinarySearchInitialUpperBound", 1}, {"str.overlap_priority", "OverlapTheoryAwarePriority", 2}, {"str.regex_automata", "RegexAutomata", 0}, {"str.regex_automata_difficulty_threshold", "RegexAutomataDifficultyThreshold", 1}, {"str.regex_automata_intersection_difficulty_threshold", "RegexAutomataIntersectionDifficultyThreshold", 1}, {"str.regex_automata_failed_automaton_threshold", "RegexAutomataFailedAutomatonThreshold", 1}, {"str.regex_automata_failed_intersection_threshold", "RegexAutomataFailedIntersectionThreshold", 1}, {"str.regex_automata_length_attempt_threshold", "RegexAutomataLengthAttemptThreshold", 1}}
	for _, tc := range cases {
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(global))
			want := SMTStrState{true, false, false, true, false, true, true, false, false, 64, -0.1, true, 1000, 1000, 10, 10, 10}
			field := testFieldBySourceName(reflect.ValueOf(&want).Elem(), tc.field)
			if mode < 3 {
				switch tc.kind {
				case 0:
					field.SetBool(!field.Bool())
					paramsMust(t, c.SetBoolName(global, &tc.key, field.Bool()))
				case 1:
					field.SetUint(77)
					paramsMust(t, c.SetUIntName(global, &tc.key, 77))
				case 2:
					field.SetFloat(2.75)
					paramsMust(t, c.SetDoubleName(global, &tc.key, 2.75))
				}
			}
			if mode == 1 {
				switch tc.kind {
				case 0:
					field.SetBool(!field.Bool())
					paramsMust(t, c.SetBoolName(p, &tc.key, field.Bool()))
				case 1:
					field.SetUint(13)
					paramsMust(t, c.SetUIntName(p, &tc.key, 13))
				case 2:
					field.SetFloat(6.5)
					paramsMust(t, c.SetDoubleName(p, &tc.key, 6.5))
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
			got, e := v.State()
			paramsMust(t, e)
			if got != want {
				t.Fatal("independent complete scalar state", tc, mode, got, want)
			}
		}
	}
	strNamedColdGraph(t, v, g, b)
}
func TestSMTStrEveryConstructorReservationAndOwnedMemberUnwind(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		_, e := g.Symbols.Intern("smt")
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTStrParams(c, p, g, b)
		strNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual header/cost/helper acquisition", ordinal, cause)
		}
		if v != nil && cause != nil {
			if v.constructorReady || v.Helpers != nil {
				t.Fatal("completed member reverse source unwind", ordinal, v.Serial)
			}
			copyOwner := *v
			if e = copyOwner.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
				t.Fatal("shallow copied owning wrapper", e)
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		strNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 2 {
		t.Fatal("wrapper then actual lexical helper", first)
	}
}
func TestSMTStrEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		bc := arena.NewBudget(^uint64(0))
		c, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := uparams.NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		local, e := c.NewValue()
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTStrParams(c, &local, g, b)
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
	// Symbol spellings use Go storage; count only remaining arena acquisitions.
	if first != 2 {
		t.Fatal("actual shared symbol/module acquisition absent", first)
	}
	t.Log("first successful unwarmed string-theory constructor", first)
}

func TestSMTStrNullGlobalIncompleteConstructorAndOwnerEligibility(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	v, cause := NewSMTStrParams(c, p, g, b)
	if v == nil || v.constructorReady || v.Serial != 1 || v.Helpers != nil {
		t.Fatal("actual first lexical declaration before publication", v, cause)
	}
	if uparams.GlobalParamsSourceDebug {
		if !errors.Is(cause, uparams.ErrGlobalParamsAssertion) {
			t.Fatal(cause)
		}
	} else if !errors.Is(cause, uparams.ErrGlobalParamsUninitialized) {
		t.Fatal(cause)
	}
	if e := v.Update(p); !errors.Is(e, uparams.ErrParamsPrecondition) {
		t.Fatal("uncompleted source constructor not callable", e)
	}
	copyOwner := *v
	if e := copyOwner.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("actual outer identity", e)
	}
	strNamedColdGraph(t, v, g, b)
	paramsMust(t, v.Close())
	strNamedColdGraph(t, nil, g, b)
}
