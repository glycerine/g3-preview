// Source: src/sat/sat_solver.h:74–136; src/sat/sat_solver.cpp:34–58; src/sat/sat_var_queue.h:35.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/random"
)

func TestSATSolverVariableColdPreparationHasNoPrematureQueue(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	v, e := newSATColdVariableState(todo, b, arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	if v.queue != nil || v.activity != nil || v.watches != nil || v.trail != nil || r.Stats().LiveBlocks != 0 {
		t.Fatal("source future member constructed early")
	}
	if v.memory != r || v.level != nil {
		t.Fatal("cold source borrows not real Runtime")
	}
	satOK(t, v.Close())
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATSolverDecisionColdDefaultSeedAndOwningUnwind(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	d, e := newSATColdDecision(b)
	satOK(t, e)
	// Source random_gen(unsigned seed=0) is initialized at its member site,
	// before later child constructors and before final updt_params reseeding.
	var expected z3rand.Generator
	if got, want := d.rng.Next(), expected.Next(); got != want {
		t.Fatal(got, want)
	}
	if _, e = d.nextVar(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("unconstructed dependent member admitted", e)
	}
	satOK(t, d.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATSolverConverterColdBorrowBeforeSourceConstruction(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	m, e := newSATColdModelConverter(b)
	satOK(t, e)
	if m.constructed {
		t.Fatal("source model member initialized early")
	}
	satOK(t, m.constructionBorrow())
	if e = m.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary access before source member", e)
	}
	copied := *m
	if e = copied.constructionBorrow(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, m.initializeSource())
	satOK(t, m.open())
	satOK(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATSolverConverterColdUnwindAndReserveRefusal(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	b.FailAfter(0)
	if m, e := newSATColdModelConverter(b); m != nil || !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(m, e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b.FailAfter(-1)
	m, e := newSATColdModelConverter(b)
	satOK(t, e)
	satOK(t, m.Close())
	if e = m.constructionBorrow(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATSolverProbingConstructionBorrowDoesNotPermitRun(t *testing.T) {
	l, p, _, b := satLifecycleFixture(t, 0)
	probe, e := newSATColdProbing(l, b)
	satOK(t, e)
	defer func() { satOK(t, probe.Close()) }()
	satOK(t, probe.constructionBorrow())
	if _, e = probe.run(false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("unconstructed probe ran", e)
	}
	a, e := newSATSolverConflictAnalysis(p, b)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	d, e := newSATSolverDynamicSubsumption(a, probe, b)
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	f, e := newSATSolverConflictResolution(a, d)
	satOK(t, e)
	defer func() { satOK(t, f.Close()) }()
	decision, e := newSATColdDecision(b)
	satOK(t, e)
	decision.conflict = f
	defer func() { satOK(t, decision.Close()) }()
	satOK(t, decision.constructionBorrow())
	if e = decision.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cold chain admitted ordinary decision", e)
	}
	ctx := p.config.context
	params, e := ctx.NewValue()
	satOK(t, e)
	defer func() { satOK(t, ctx.Del(&params)) }()
	cleaner, e := newSATCleaner(l, b)
	satOK(t, e)
	defer func() { satOK(t, cleaner.Close()) }()
	simp, e := newSATSimplifier(l, cleaner, decision, &params)
	satOK(t, e)
	defer func() { satOK(t, simp.Close()) }()
	if e = simp.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("simplifier ordinary operation before future probe member", e)
	}
	satOK(t, probe.initializeSource(ctx, &params))
	satOK(t, decision.open())
	satOK(t, simp.open())
	if !probe.enabled || !probe.cacheEnabled || probe.propagation().probingCounter != 0 {
		t.Fatal("actual source probing constructor omitted")
	}
}

func TestSATSolverColdVariableOrdinaryGuardBeforeAnyPublication(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	v, e := newSATColdVariableState(todo, b, arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	defer func() {
		satOK(t, v.Close())
		satOK(t, todo.Close())
		satOK(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	if e = v.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cold variable owner accepted ordinary entry", e)
	}

	satOK(t, v.constructionBorrow())
	if _, e = v.mkVar(false, true); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if v.modelKnown || v.modelIsCurrent || v.mkVarCount != 0 || r.Stats().LiveBlocks != 0 {
		t.Fatal("ordinary call published before source construction")
	}
}

func TestSATSolverInterveningChildFailureBeforeQueueConstruction(t *testing.T) {
	old, _, b := satPropagationFixture(t, 0)
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	defer func() { satOK(t, todo.Close()) }()
	v, e := newSATColdVariableState(todo, b, arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, old.memory)
	satOK(t, e)
	defer func() { satOK(t, v.Close()) }()
	p, e := newSATSolverPropagationState(v, old.clauses, old.converter, old.limit, old.memory, old.config, b)
	satOK(t, e)
	defer func() { satOK(t, p.Close()) }()
	l, e := newSATSolverClauseLifecycle(p, b)
	satOK(t, e)
	defer func() { satOK(t, l.Close()) }()
	cleaner, e := newSATCleaner(l, b)
	satOK(t, e)
	defer func() { satOK(t, cleaner.Close()) }()
	a, e := newSATSolverConflictAnalysis(p, b)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	if e = p.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary propagation before queue", e)
	}
	if e = l.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary lifecycle before queue", e)
	}
	if e = a.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary analysis before queue", e)
	}
	ctx := p.config.context
	live, e := ctx.NewValue()
	satOK(t, e)
	key := "scc"
	satOK(t, ctx.SetBoolName(&live, &key, true))
	invalid := live
	satOK(t, ctx.Del(&live))
	before := b.Used()
	rawBefore := old.memory.Stats().LiveBlocks
	// Both source child parameter reads happen before the later case split
	// queue member. An actual dead ParamsRef must fail there, not at a fake
	// ordinary solver readiness gate or premature heap reserve128.
	if child, e := newSATSCC(l, &invalid); child != nil || !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("SCC source parameter site", child, e)
	}
	if child, e := newSATAsymmBranch(l, &invalid); child != nil || !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("asymm source parameter site", child, e)
	}
	if v.queue != nil || v.watches != nil || v.activity != nil || v.trail != nil || b.Used() != before || old.memory.Stats().LiveBlocks != rawBefore {
		t.Fatal("later source member allocation/prefix changed")
	}
}
