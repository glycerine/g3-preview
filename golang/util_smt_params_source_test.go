// Source: src/smt/params/smt_params.h:75-318; src/smt/params/smt_params.cpp:24-164.
// Source author: Leonardo de Moura (2008-02-20).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"os"
	"strings"
	"testing"
)

func TestSMTParamsImplicitBasesDefaultEmptyAndConstructorLocalOnly(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	key := "dack.factor"
	paramsMust(t, c.SetDoubleName(p, &key, 0.73))
	key = "str.binary_search_start"
	paramsMust(t, c.SetUIntName(p, &key, 777))
	key = "random_seed"
	paramsMust(t, c.SetUIntName(p, &key, 81))
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	state, e := s.State()
	paramsMust(t, e)
	if state.RandomSeed != 81 || s.dyn.state.Factor != 0.1 || s.str.state.BinarySearchInitialUpperBound != 64 || !state.DefaultsInitialized {
		t.Fatal("implicit empty bases then local constructor update", state, s.dyn.state, s.str.state)
	}
	paramsMust(t, s.Update(p))
	if s.dyn.state.Factor != 0.73 || s.str.state.BinarySearchInitialUpperBound != 64 {
		t.Fatal("full update dyn included/str omitted")
	}
}
func TestSMTParamsContextOverloadOnlyTwoScalars(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	ctx, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, ctx.Close()) }()
	k := "model"
	paramsMust(t, c.SetBoolName(p, &k, false))
	k = "auto_config"
	paramsMust(t, c.SetBoolName(p, &k, false))
	paramsMust(t, ctx.Update(p))
	before, e := s.State()
	paramsMust(t, e)
	paramsMust(t, s.UpdateContext(ctx))
	after, e := s.State()
	paramsMust(t, e)
	before.Model = false
	before.AutoConfig = false
	if after != before {
		t.Fatal("source context overload exactly two writes", before, after)
	}
}
func TestSMTParamsQIUninitializedDisplayPrefixAndCopiedOwnerGuard(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	sb := arena.NewBudget(1 << 20)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	e = s.Display(out)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("distinct inherited qi_ematching remains uninitialized", e)
	}
	text, e := out.String()
	paramsMust(t, e)
	if len(text) < 15 || text[len(text)-15:] != "m_qi_ematching=" {
		t.Fatal("actual label-before-guard", text)
	}
	copyOwner := *s
	before := b.Used()
	if e := copyOwner.Close(); !errors.Is(e, ErrParamsOwner) || b.Used() != before {
		t.Fatal("copied owning wrapper", e)
	}
}

func smtParamsNamedCold(t *testing.T, s *SMTParams, g *GlobalParamsService, b *arena.Budget) {
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
	if s != nil {
		if s.self != s || s.budget != b {
			t.Fatal("actual containing owner")
		}
		n += s.header
		headers := []uint64{s.preprocess.header, s.preprocess.pattern.header, s.dyn.header, s.qi.header, s.arith.header, s.array.header, s.bv.header, s.str.header, s.pb.header}
		for _, h := range headers {
			if h != 0 {
				t.Fatal("source inline base got second owning header", h)
			}
		}
		owners := []*smtModuleParamsOwner{&s.smtModuleParamsOwner, &s.preprocess.smtModuleParamsOwner, &s.preprocess.pattern.smtModuleParamsOwner, &s.dyn.smtModuleParamsOwner, &s.qi.smtModuleParamsOwner, &s.arith.smtModuleParamsOwner, &s.array.smtModuleParamsOwner, &s.bv.smtModuleParamsOwner, &s.str.smtModuleParamsOwner, &s.pb.smtModuleParamsOwner}
		for _, m := range owners {
			for f := m.helpers; f != nil; f = f.next {
				if f.self != f || f.owner != m || f.invocation == 0 {
					t.Fatal("exact acquired lexical owner")
				}
				n += f.header
			}
		}
		n += uint64(len(s.qi.cost.external) + len(s.qi.cost.pending) + len(s.qi.newGen.external) + len(s.qi.newGen.pending))
	}
	if g.symbols.budget == b {
		n += astTestSymbolGraph(g.symbols) + g.symbols.Stats().Table.RetainedBytes
	}
	if n != b.Used() {
		t.Fatal("before-lift once-only graph", n, b.Used())
	}
}
func smtWarmAllModules(t *testing.T, g *GlobalParamsService) {
	t.Helper()
	for _, name := range []string{"pi", "smt", "rewriter", "model"} {
		s, e := g.symbols.Intern(name)
		paramsMust(t, e)
		_, e = g.current.getParams(s)
		paramsMust(t, e)
	}
	_, e := g.SourceGetValueName("auto_config")
	paramsMust(t, e)
}
func TestSMTParamsEveryWarmConstructorReservationAndCompletedBaseUnwind(t *testing.T) {
	for ordinal := int64(0); ; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		smtWarmAllModules(t, g)
		// Warm real initial/default symbol expressions before this Budget horizon;
		// they belong to the independently audited SymbolManager Budget.
		_, e := g.symbols.Intern("auto")
		paramsMust(t, e)
		_, e = g.symbols.Intern("seq")
		paramsMust(t, e)
		b.FailAfter(ordinal)
		s, e := NewSMTParams(c, p, g, b)
		smtParamsNamedCold(t, s, g, b)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if s != nil {
				if s.endedBases != s.completedBases {
					t.Fatal("reverse completed-base source unwind", ordinal, s.completedBases, s.endedBases)
				}
				before := b.Used()
				if x := s.Update(p); !errors.Is(x, ErrParamsPrecondition) {
					t.Fatal("incomplete ctor call", x)
				}
				if before != b.Used() {
					t.Fatal("guard introduced allocation")
				}
			}
		}
		b.FailAfter(-1)
		if s != nil {
			paramsMust(t, s.Close())
		}
		smtParamsNamedCold(t, nil, g, b)
		if e == nil {
			if ordinal != 15 {
				t.Fatal("actual header/ownedcost and source helper declarations", ordinal)
			}
			break
		}
		if ordinal > 40 {
			t.Fatal("no bounded success")
		}
	}
}
func TestSMTParamsLocalAndFullUpdateActualHelperDeclarationPrefixes(t *testing.T) {
	for _, full := range []bool{false, true} {
		for ordinal := int64(0); ; ordinal++ {
			c, g, p, b := dynAckSourceFixture(t)
			s, e := NewSMTParams(c, p, g, b)
			paramsMust(t, e)
			t.Cleanup(func() { b.FailAfter(-1); paramsMust(t, s.Close()) })
			key := "random_seed"
			paramsMust(t, c.SetUIntName(p, &key, 71))
			key = "compact"
			paramsMust(t, c.SetBoolName(p, &key, true))
			key = "array.weak"
			paramsMust(t, c.SetBoolName(p, &key, true))
			b.FailAfter(ordinal)
			if full {
				e = s.Update(p)
			} else {
				e = s.UpdateLocal(p)
			}
			smtParamsNamedCold(t, s, g, b)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if !full && ordinal == 2 && (s.state.RandomSeed != 71 || s.state.ModelCompact || s.array.state.Weak) {
				t.Fatal("p pubs before model declaration", s.state, s.array.state)
			}
			if !full && ordinal == 3 && (!s.state.ModelCompact || s.array.state.Weak) {
				t.Fatal("model pub before nested Array declaration", s.state, s.array.state)
			}
			if e == nil && (!s.state.ModelCompact || !s.array.state.Weak || s.state.RandomSeed != 71) {
				t.Fatal("completed source local body")
			}
			b.FailAfter(-1)
			paramsMust(t, s.Close())
			smtParamsNamedCold(t, nil, g, b)
			if e == nil {
				want := int64(4)
				if full {
					want = 13
				}
				if ordinal != want {
					t.Fatal("literal declared helper horizon", full, ordinal, want)
				}
				break
			}
			if ordinal > 40 {
				t.Fatal("no bounded success")
			}
		}
	}
}
func TestSMTParamsModelModulePivotPrecedenceAndHiddenDumpReset(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	model, e := g.symbols.Intern("model")
	paramsMust(t, e)
	mp, e := g.current.getParams(model)
	paramsMust(t, e)
	key := "compact"
	paramsMust(t, c.SetBoolName(mp, &key, true))
	key = "arith.greatest_error_pivot"
	paramsMust(t, c.SetBoolName(p, &key, true))
	key = "arith.least_error_pivot"
	paramsMust(t, c.SetBoolName(p, &key, true))
	s.state.DumpBenchmarks = true
	s.state.DumpMinTime = 17
	s.state.DumpRecheck = true
	paramsMust(t, s.UpdateLocal(p))
	if !s.state.ModelCompact || s.arith.state.PivotStrategy != 1 || s.state.DumpBenchmarks || s.state.DumpMinTime != 0.5 || s.state.DumpRecheck {
		t.Fatal("model fallback and source precedence/reset", s.state, s.arith.state)
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("arith.greatest_error_pivot"), false))
	paramsMust(t, s.UpdateLocal(p))
	if s.arith.state.PivotStrategy != 2 {
		t.Fatal("least error branch")
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("arith.least_error_pivot"), false))
	paramsMust(t, s.UpdateLocal(p))
	if s.arith.state.PivotStrategy != 2 {
		t.Fatal("neither hidden flag changes current pivot")
	}
}

func TestSMTParamsAllLocalGetterPublicationsFallbackAndDirectHiddenReads(t *testing.T) {
	cases := []struct {
		key, kind string
		u         uint32
		b         bool
		d         float64
	}{
		{"random_seed", "u", 17, false, 0}, {"relevancy", "u", 7, false, 0}, {"ematching", "b", 0, false, 0}, {"phase_selection", "u", 99, false, 0}, {"restart_strategy", "u", 77, false, 0}, {"restart_factor", "d", 0, false, 2.3}, {"case_split", "u", 61, false, 0}, {"theory_case_split", "b", 0, true, 0}, {"theory_aware_branching", "b", 0, true, 0}, {"delay_units", "b", 0, true, 0}, {"delay_units_threshold", "u", 83, false, 0}, {"timeout", "u", 57, false, 0}, {"rlimit", "u", 29, false, 0}, {"max_conflicts", "u", 91, false, 0}, {"core.validate", "b", 0, true, 0},
	}
	for _, tc := range cases {
		for _, moduleCase := range []bool{false, true} {
			c, g, p, b := dynAckSourceFixture(t)
			s, e := NewSMTParams(c, p, g, b)
			paramsMust(t, e)
			target := p
			if moduleCase {
				mod, e := g.symbols.Intern("smt")
				paramsMust(t, e)
				target, e = g.current.getParams(mod)
				paramsMust(t, e)
			}
			switch tc.kind {
			case "u":
				paramsMust(t, c.SetUIntName(target, &tc.key, tc.u))
			case "b":
				paramsMust(t, c.SetBoolName(target, &tc.key, tc.b))
			case "d":
				paramsMust(t, c.SetDoubleName(target, &tc.key, tc.d))
			}
			paramsMust(t, s.UpdateLocal(p))
			got := s.state
			actualU := map[string]uint32{"random_seed": got.RandomSeed, "relevancy": got.RelevancyLvl, "phase_selection": got.PhaseSelection, "restart_strategy": got.RestartStrategy, "case_split": got.CaseSplitStrategy, "delay_units_threshold": got.DelayUnitsThreshold, "timeout": got.Timeout, "rlimit": got.Rlimit, "max_conflicts": got.MaxConflicts}
			actualB := map[string]bool{"ematching": got.Ematching, "theory_case_split": got.TheoryCaseSplit, "theory_aware_branching": got.TheoryAwareBranching, "delay_units": got.DelayUnits, "core.validate": got.CoreValidate}
			if tc.kind == "u" && actualU[tc.key] != tc.u || tc.kind == "b" && actualB[tc.key] != tc.b || tc.kind == "d" && got.RestartFactor != tc.d {
				t.Fatal(tc, moduleCase, got)
			}
			smtParamsNamedCold(t, s, g, b)
			paramsMust(t, s.Close())
			smtParamsNamedCold(t, nil, g, b)
		}
	}
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	logic, e := g.symbols.Intern("QF_LIA")
	paramsMust(t, e)
	paramsMust(t, c.SetSymbolName(p, dynAckName("logic"), logic))
	paramsMust(t, c.SetBoolName(p, dynAckName("preprocess"), false))
	solver, e := g.symbols.Intern("z3str3")
	paramsMust(t, e)
	paramsMust(t, c.SetSymbolName(p, dynAckName("string_solver"), solver))
	paramsMust(t, s.UpdateLocal(p))
	if s.state.Logic != logic || s.state.Preprocess || s.state.StringSolver != solver {
		t.Fatal(s.state)
	}
	// Direct logic fallback is current-value, whereas generated string_solver
	// constructs and uses its literal seq default on every reached getter.
	paramsMust(t, c.SetUIntName(p, dynAckName("logic"), 9))
	paramsMust(t, c.SetUIntName(p, dynAckName("string_solver"), 9))
	paramsMust(t, s.UpdateLocal(p))
	seq, e := g.symbols.Intern("seq")
	paramsMust(t, e)
	if s.state.Logic != logic || s.state.StringSolver != seq {
		t.Fatal("direct current vs generated literal", s.state)
	}
}

func TestSMTParamsAutoConfigShortCircuitLeavesClosedDefaultBorrowUnread(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	// The actual global default descriptor is a live provider acquired during
	// registration. Only the reached GetValue default read observes its Close.
	paramsMust(t, g.current.globalDescriptors.Close())
	paramsMust(t, c.SetBoolName(p, dynAckName("auto_config"), false))
	paramsMust(t, c.SetUIntName(p, dynAckName("random_seed"), 19))
	paramsMust(t, s.UpdateLocal(p))
	if s.state.AutoConfig || s.state.RandomSeed != 19 {
		t.Fatal("false operand shortcut", s.state)
	}
	before := s.state
	paramsMust(t, c.SetBoolName(p, dynAckName("auto_config"), true))
	e = s.UpdateLocal(p)
	if !errors.Is(e, arena.ErrClosed) || s.state != before {
		t.Fatal("true operand reaches borrowed default before any publication", e, s.state)
	}
	smtParamsNamedCold(t, s, g, b)
}

const smtDefaultLocalDisplay = `m_display_proof=0
m_display_dot_proof=0
m_display_unsat_core=0
m_check_proof=0
m_eq_propagation=1
m_binary_clause_opt=1
m_relevancy_lvl=2
m_relevancy_lemma=0
m_random_seed=0
m_random_var_freq=0.01
m_inv_decay=1.052
m_clause_decay=1
m_random_initial_activity=1
m_phase_selection=3
m_phase_caching_on=400
m_phase_caching_off=100
m_minimize_lemmas=1
m_max_conflicts=4294967295
m_simplify_clauses=1
m_tick=1000
m_display_features=0
m_new_core2th_eq=1
m_ematching=1
m_case_split_strategy=1
m_rel_case_split_order=0
m_lookahead_diseq=0
m_delay_units=0
m_delay_units_threshold=32
m_theory_resolve=0
m_restart_strategy=1
m_restart_initial=100
m_restart_factor=1.1
m_restart_adaptive=1
m_agility_factor=0.9999
m_restart_agility_threshold=0.18
m_lemma_gc_strategy=0
m_lemma_gc_half=0
m_recent_lemmas_size=100
m_lemma_gc_initial=5000
m_lemma_gc_factor=1.1
m_new_old_ratio=16
m_new_clause_activity=10
m_old_clause_activity=500
m_new_clause_relevancy=45
m_old_clause_relevancy=6
m_inv_clause_decay=1
m_smtlib_dump_lemmas=0
m_logic=null
m_profile_res_sub=0
m_display_bool_var2expr=0
m_display_ll_bool_var2expr=0
m_abort_after_preproc=0
m_model=1
m_model_compact=0
m_model_on_timeout=0
m_model_on_final_check=0
m_progress_sampling_freq=0
m_display_installed_theories=0
m_core_validate=0
m_preprocess=1
m_user_theory_preprocess_axioms=0
m_user_theory_persist_axioms=0
m_timeout=4294967295
m_rlimit=0
m_at_labels_cex=0
m_check_at_labels=0
m_dump_goal_as_smt=0
m_auto_config=1
`

func TestSMTParamsAllDisplayChainsIndependentLocalOracleEveryOutputRefusal(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	paramsMust(t, s.qi.SetEmatching(false)) // actual explicit field assignment, not parent constructor synthesis.
	seedBudget := arena.NewBudget(1 << 24)
	seed, e := NewNumericTextStream(NumericTextStreamConfig{Budget: seedBudget})
	paramsMust(t, e)
	paramsMust(t, s.Display(seed))
	full, e := seed.String()
	paramsMust(t, e)
	paramsMust(t, seed.Close())
	if !strings.HasSuffix(full, smtDefaultLocalDisplay) || strings.Count(full, "\n") != 194 || strings.Contains(full, "m_str_") {
		t.Fatal("eight bases then independent68 local literals; Str omitted", full)
	}
	// Every actual output allocation retains its exact emitted prefix, and its
	// concrete storage graph is checked before lifting the active refusal.
	for ordinal := int64(0); ; ordinal++ {
		sb := arena.NewBudget(1 << 24)
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		e = s.Display(out)
		st := out.Stats()
		if sb.Used() != st.OwnerBytes+st.ScratchBytes+st.OutputOwnerBytes+st.OutputOverflowBytes {
			t.Fatal("before-lift stream owner graph")
		}
		text, x := out.String()
		paramsMust(t, x)
		if !strings.HasPrefix(full, text) {
			t.Fatal("literal completed insertion prefix", ordinal, text)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("output refund")
		}
		if e == nil {
			if text != full || ordinal == 0 {
				t.Fatal("full output/success horizon", ordinal)
			}
			break
		}
		if !errors.Is(e, arena.ErrInjectedFailure) || ordinal > 20 {
			t.Fatal(e, ordinal)
		}
	}
	sb := arena.NewBudget(1 << 24)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, s.Display(out))
	text, e := out.String()
	paramsMust(t, e)
	if !strings.Contains(text, "m_display_proof=+0\n") || !strings.Contains(text, "m_random_initial_activity=+1\n") || !strings.Contains(text, "m_random_seed=0\n") || !strings.Contains(text, "m_random_var_freq=+0.01\n") {
		t.Fatal("actual signed bool/enum, unsigned, double flags", text)
	}
}
func TestSMTParamsRealFileBadbitStillReachesQIUninitializedRead(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	f, e := os.CreateTemp(".", "smt-parent-output-")
	paramsMust(t, e)
	defer os.Remove(f.Name())
	paramsMust(t, f.Close())
	sb := arena.NewBudget(1 << 24)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb, BorrowedFile: f})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	e = s.Display(out)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("nonthrowing write failure does not erase later source read", e)
	}
	st := out.Stats()
	if !out.fileBad {
		t.Fatal("real file badbit")
	}
	paramsMust(t, s.qi.SetEmatching(true))
	paramsMust(t, s.Display(out))
	st = out.Stats()
	if sb.Used() != st.OwnerBytes+st.ScratchBytes+st.OutputOwnerBytes+st.OutputOverflowBytes {
		t.Fatal("file graph")
	}
}
func TestSMTParamsConcreteSymbolInsertionsPreserveBorrowedBytesAndNumericFlags(t *testing.T) {
	_, g, _, _ := dynAckSourceFixture(t)
	sb := arena.NewBudget(1 << 24)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	plain, e := g.symbols.Intern("QF_LIA")
	paramsMust(t, e)
	paramsMust(t, out.appendSMTSymbol(g, Symbol{}))
	paramsMust(t, out.AppendByte('|'))
	paramsMust(t, out.appendSMTSymbol(g, plain))
	paramsMust(t, out.AppendByte('|'))
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, out.appendSMTSymbol(g, NumericalSymbol(13)))
	text, e := out.String()
	paramsMust(t, e)
	if text != "null|QF_LIA|k!13" {
		t.Fatal("literal null/C-string/unsigned numeric chain", text)
	}
}

func TestSMTParamsRealYoungerModelDestructorStopProtectsOlderHelperAndAllBases(t *testing.T) {
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
	smt, e := sy.Intern("model")
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
	v, e := NewSMTParams(c, &local, g, b)
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
	older, e := v.beginHelper(&local, "smt")
	paramsMust(t, e)
	younger, e := v.beginHelper(&local, "model")
	paramsMust(t, e)
	paramsMust(t, g.SourceReset())
	olderCount := smtParamsRefCount(t, c, &older.value)
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &younger.value}, {c, &older.value}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	cause := v.finishHelper(younger, nil)
	if !errors.Is(cause, arena.ErrBounds) || !younger.destructorStarted || younger.sourceDone || younger.receipt.invocation == 0 || v.destructorCause != younger.destructorCause {
		t.Fatal("actual younger source stop", cause, younger.receipt)
	}
	if v.endedBases != 0 || older.destructorStarted || smtParamsRefCount(t, c, &older.value) != olderCount {
		t.Fatal("older base lifetime unread after lexical stop")
	}
	if e := v.finishHelper(older, cause); !errors.Is(e, arena.ErrBounds) || older.destructorStarted {
		t.Fatal("older deferred epilogue crossed younger stop", e)
	}
	v.running = false
	v.activeCall = false
	if e := v.Update(&local); !errors.Is(e, ErrParamsPrecondition) || older.destructorStarted || smtParamsRefCount(t, c, &older.value) != olderCount {
		t.Fatal("new Update entered older base through stopped younger owner", e)
	}
	if e := v.UpdateLocal(&local); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("new local mutation through retained owner", e)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("exact consumed raw and retained leaf descriptor", before, after)
	}
	if e := v.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || older.destructorStarted || v.endedBases != 0 {
		t.Fatal("strict parent source stop before older DT", e)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	smtParamsNamedCold(t, v, g, b)
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
	smtParamsNamedCold(t, v, g, b)
}

func smtParamsRefCount(t *testing.T, c *ParamsContext, v *ParamsRef) uint32 {
	t.Helper()
	paramsMust(t, c.check(v))
	if v.slot == 0 {
		return 0
	}
	return c.blocks[v.slot-1].refs
}

func TestSMTParamsActualStringSolverMemberConstructorFailureFrontier(t *testing.T) {
	_, _, q := paramsTestOwner(t)
	bs := arena.NewBudget(^uint64(0))
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2, DirectOnly: true})
	paramsMust(t, e)
	bc := arena.NewBudget(^uint64(0))
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	b := arena.NewBudget(^uint64(0))
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	paramsMust(t, e)
	p := &local
	t.Cleanup(func() {
		paramsMust(t, c.Del(p))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bs.Used() != 0 || bc.Used() != 0 {
			t.Fatal("actual independent owners retained")
		}
	})
	for _, name := range []string{"pi", "smt", "rewriter", "model"} {
		sym, e := g.symbols.Intern(name)
		paramsMust(t, e)
		_, e = g.current.getParams(sym)
		paramsMust(t, e)
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("auto_config"), false))
	bs.FailAfter(0)
	s, e := NewSMTParams(c, p, g, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || s == nil || s.completedBases != 511 || !s.state.DefaultsInitialized || s.state.StringSolverInitialized || s.state.DumpInitialized || s.endedBases != 511 {
		bs.FailAfter(-1)
		if s != nil {
			paramsMust(t, s.Close())
		}
		t.Fatal("symbol auto acquisition occurs after nine bases/all scalar initializers", e)
	}
	smtParamsNamedCold(t, s, g, b)
	ss := g.symbols.Stats()
	if bs.Used() != astTestSymbolGraph(g.symbols)+ss.Table.RetainedBytes {
		t.Fatal("actual foreign symbol namespace graph before faultlift")
	}
	bs.FailAfter(-1)
	paramsMust(t, s.Close())
	smtParamsNamedCold(t, nil, g, b)
}
func TestSMTParamsEveryUnwarmedSharedSymbolModuleAndClassReservation(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 128; ordinal++ {
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
		p, e := c.NewValue()
		paramsMust(t, e)
		// Source auto_config=false erases registration/default-name conversion;
		// all actual first module/symbol/class acquisitions remain in this horizon.
		paramsMust(t, c.SetBoolName(&p, dynAckName("auto_config"), false))
		b.FailAfter(ordinal)
		s, e := NewSMTParams(c, &p, g, b)
		smtParamsNamedCold(t, s, g, b)
		st := c.Stats()
		ts := c.texts.Stats()
		if bc.Used() != st.HeaderBytes+st.BlockBytes+st.ReceiptBytes+st.SourceFrameBytes+st.SourceSetBytes+st.ReplacementBytes+st.PageBytes+st.DirectoryBytes+st.FormatBytes+gparamsVectorGraph(ts) {
			t.Fatal("named actual context storage", ordinal, st, ts, bc.Used())
		}
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		b.FailAfter(-1)
		if s != nil {
			paramsMust(t, s.Close())
		}
		smtParamsNamedCold(t, nil, g, b)
		paramsMust(t, c.Del(&p))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bc.Used() != 0 {
			t.Fatal("exact owner refunds", ordinal, b.Used(), bc.Used())
		}
		if e == nil {
			success = int(ordinal)
			break
		}
	}
	if success != 26 {
		t.Fatal("every actual shared acquisition through first success", success)
	}
}

func TestSMTParamsCompletedSourceFinalizeEndsBasesBeforeOwningHeaderRefund(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	header := s.header
	before := b.Used()
	// Source destruction ends completed bases; the Go placement header remains
	// an explicit owning obligation and borrowed providers remain caller-owned.
	paramsMust(t, s.SourceFinalize())
	if s.endedBases != 511 || !s.closed || s.header != header || !s.qi.closed || !s.preprocess.closed || s.qi.cost.external != nil || s.qi.newGen.external != nil {
		t.Fatal("actual reverse completed source lifetime", s.endedBases, s.header)
	}
	if b.Used() > before {
		t.Fatal("source destruction acquired storage")
	}
	smtParamsNamedCold(t, s, g, b)
	if _, e = s.State(); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("ended ordinary source object", e)
	}
	paramsMust(t, g.open())
	paramsMust(t, c.check(p))
	paramsMust(t, s.Close())
	if s.header != 0 {
		t.Fatal("own header refund")
	}
	paramsMust(t, s.Close())
	smtParamsNamedCold(t, nil, g, b)
}
