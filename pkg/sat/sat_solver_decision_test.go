// Source: src/sat/sat_solver.cpp:933–988; src/sat/sat_solver.h:337–338.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/random"
)

func satDecisionFixture(t *testing.T, n uint32) (*satSolverDecision, *satSolverGC) {
	t.Helper()
	g, s := satGCFixture(t, n)
	d, e := newSATSolverDecision(s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, d.Close()) })
	return d, g
}

// A malformed source backing is a Go access-guard fixture, not a valid C++
// allocation failure. num_vars must propagate the actual endangered read.
func TestSATSolverDecisionActualNumVarsReadGuard(t *testing.T) {
	d, g := satDecisionFixture(t, 1)
	p := g.lifecycle().propagation
	p.config.state.RandomFreq = 1
	old := p.variables.level.Raw.Block
	h, e := p.memory.Allocate(8)
	satOK(t, e)
	stale := h
	satOK(t, p.memory.Deallocate(&h))
	p.variables.level.Raw.Block = stale
	_, e = d.nextVar()
	p.variables.level.Raw.Block = old
	if e == nil {
		t.Fatal("source num_vars read guard silently became empty success")
	}
}

func TestSATSolverDecisionEveryActualAllocationPrefix(t *testing.T) {
	firstSuccess := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		d, g := satDecisionFixture(t, 1)
		p := g.lifecycle().propagation
		p.config.state.RandomFreq = 0
		p.config.state.Phase = satPhaseAlwaysTrue
		p.budget.FailAfter(ordinal)
		ok, e := d.decide()
		// All actual named storage is inspected before disabling faults.
		satDecisionAudit(t, g, d)
		if e == nil {
			if !ok || p.variables.scopeLevel != 1 || d.decisions != 1 {
				t.Fatal(ok, d.decisions)
			}
			firstSuccess = int(ordinal)
			p.budget.FailAfter(-1)
			break
		}
		satProbingExpectedRefusal(t, p, e)
		if !p.variables.queue.Empty() {
			t.Fatal("selected queue candidate was already removed")
		}
		if ordinal == 0 {
			if d.decisions != 0 || p.variables.scopeLevel != 0 {
				t.Fatal("scope refusal source prefix")
			}
		} else {
			if d.decisions != 1 || p.variables.scopeLevel != 1 {
				t.Fatal("trail refusal follows scope and decision statistic", ordinal)
			}
			value, e := p.variables.valueVar(0)
			satOK(t, e)
			if value != lbool.True {
				t.Fatal("assignment publication precedes trail growth")
			}
		}
		p.budget.FailAfter(-1)
		satOK(t, d.Close())
		satGCAudit(t, g)
	}
	if firstSuccess < 1 {
		t.Fatal("no complete actual reservation campaign", firstSuccess)
	}
	t.Log("actual decision refusal horizon", firstSuccess)
}

func TestSATSolverDecisionThresholdCastBoundariesAndOwner(t *testing.T) {
	// The conversion truncates before range testing; negative finite frequency
	// is allowed by source and simply chooses queue fallback.
	for _, frequency := range []float64{-1, 0, .01, 1, 2, float64(math.MinInt32) / 32767, float64(math.MaxInt32) / 32767} {
		d, g := satDecisionFixture(t, 2)
		g.lifecycle().propagation.config.state.RandomFreq = frequency
		_, e := d.nextVar()
		satOK(t, e)
	}
	d, g := satDecisionFixture(t, 1)
	before := g.lifecycle().budget.Used()
	copy := *d
	if !errors.Is(copy.Close(), ErrSATOwner) || g.lifecycle().budget.Used() != before {
		t.Fatal("copied owner refunded cold header")
	}
	satOK(t, d.Close())
	if _, e := d.nextVar(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	// A source extension push reaches its real missing binding after publishing
	// scope, before the decision statistic and extension get_phase call.
	d, g = satDecisionFixture(t, 1)
	p := g.lifecycle().propagation
	p.config.state.RandomFreq = 0
	p.extensionPresent = true
	_, e := d.decide()
	p.extensionPresent = false
	if !errors.Is(e, ErrSATMissingSourceExtBinding) || p.variables.scopeLevel != 1 || d.decisions != 0 {
		t.Fatal("actual EXT push prefix", e, d.decisions)
	}
}

func TestSATSolverDecisionEmptyAndRandomDrawOrder(t *testing.T) {
	for _, freq := range []float64{0, 1} {
		d, g := satDecisionFixture(t, 0)
		g.lifecycle().propagation.config.state.RandomFreq = freq
		d.rng.SetSeed(73)
		x, e := d.nextVar()
		satOK(t, e)
		if x != NullBoolVar {
			t.Fatal(x)
		}
		oracle := z3rand.New(73)
		oracle.Next() // The threshold draw precedes both empty return routes.
		if d.rng.Next() != oracle.Next() {
			t.Fatal("empty next_var source RNG prefix")
		}
		ok, e := d.decide()
		satOK(t, e)
		if ok || d.decisions != 0 || g.lifecycle().propagation.variables.scopeLevel != 0 {
			t.Fatal("empty decision changes state")
		}
	}
}

func TestSATSolverDecisionRandomLeavesQueueAndFallbackSkips(t *testing.T) {
	d, g := satDecisionFixture(t, 4)
	p := g.lifecycle().propagation
	v := p.variables
	p.config.state.RandomFreq = 1
	d.rng.SetSeed(17)
	oracle := z3rand.New(17)
	oracle.Next()
	want := BoolVar(oracle.Next() % 4)
	before := satScalarTestUintSize(t, &v.queue.runtimeValues)
	x, e := d.nextVar()
	satOK(t, e)
	if x != want || satScalarTestUintSize(t, &v.queue.runtimeValues) != before || d.rng.Next() != oracle.Next() {
		t.Fatal("selected random candidate must not consume case-split queue", x, want)
	}
	// Highest-activity assigned and eliminated entries are really removed by
	// queue fallback. The valid lower-activity candidate is returned next.
	p.config.state.RandomFreq = 0
	satOK(t, v.activity.Set(0, 30))
	satOK(t, v.queue.ActivityIncreased(0))
	satOK(t, v.activity.Set(1, 20))
	satOK(t, v.queue.ActivityIncreased(1))
	satOK(t, v.activity.Set(2, 10))
	satOK(t, v.queue.ActivityIncreased(2))
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, v.setEliminated(BoolVar(1), true))
	x, e = d.nextVar()
	satOK(t, e)
	if x != 2 || satScalarTestUint(t, &v.queue.runtimeIndices, 0) != 0 || satScalarTestUint(t, &v.queue.runtimeIndices, 1) != 0 || satScalarTestUint(t, &v.queue.runtimeIndices, 2) != 0 {
		t.Fatal("queue skips source assigned/eliminated candidates", x)
	}
}

func TestSATSolverDecisionRejectedRandomCandidateUsesQueue(t *testing.T) {
	for _, eliminated := range []bool{false, true} {
		d, g := satDecisionFixture(t, 3)
		p := g.lifecycle().propagation
		p.config.state.RandomFreq = 1
		d.rng.SetSeed(43)
		oracle := z3rand.New(43)
		oracle.Next()
		rejected := BoolVar(oracle.Next() % 3)
		wanted := BoolVar((uint32(rejected) + 1) % 3)
		if eliminated {
			satOK(t, p.variables.setEliminated(rejected, true))
		} else {
			satOK(t, p.assign(MakeLiteral(rejected, false), satJustification{}))
		}
		satOK(t, p.variables.activity.Set(uint32(wanted), 100))
		satOK(t, p.variables.queue.ActivityIncreased(wanted))
		x, e := d.nextVar()
		satOK(t, e)
		if x != wanted || d.rng.Next() != oracle.Next() {
			t.Fatal("random rejection falls through same source queue", x, wanted)
		}
	}
}

func TestSATSolverDecisionColdConstructionAndCounterWrap(t *testing.T) {
	g, s := satGCFixture(t, 1)
	p := g.lifecycle().propagation
	p.config.state.RandomSeed = 91
	before := p.budget.Used()
	p.budget.FailAfter(0)
	d, e := newSATSolverDecision(s)
	if d != nil || !errors.Is(e, arena.ErrInjectedFailure) || p.budget.Used() != before {
		t.Fatal("real cold header refusal", d, e)
	}
	satGCAudit(t, g)
	p.budget.FailAfter(-1)
	d, e = newSATSolverDecision(s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, d.Close()) })
	oracle := z3rand.New(91)
	if d.rng.Next() != oracle.Next() {
		t.Fatal("source update seed binding")
	}
	p.config.state.RandomFreq = 0
	p.config.state.Phase = satPhaseAlwaysTrue
	d.decisions = math.MaxUint32
	ok, e := d.decide()
	satOK(t, e)
	if !ok || d.decisions != 0 {
		t.Fatal("source unsigned decision counter wraps", ok, d.decisions)
	}
	satDecisionAudit(t, g, d)
}

func TestSATSolverDecisionAllPhasePolicies(t *testing.T) {
	for _, tc := range []struct {
		mode     satPhaseSelection
		cache    bool
		saved    int8
		negative bool
	}{{satPhaseAlwaysTrue, false, 2, false}, {satPhaseAlwaysFalse, false, 2, true},
		{satPhaseCaching, true, 0, false}, {satPhaseCaching, true, 1, true},
		{satPhaseCaching, true, 2, true}, {satPhaseCaching, false, 0, true}} {
		d, g := satDecisionFixture(t, 1)
		p := g.lifecycle().propagation
		p.config.state.RandomFreq = 0
		p.config.state.Phase = tc.mode
		d.conflict.phaseCacheOn = tc.cache
		satOK(t, p.variables.setPhase(BoolVar(0), tc.saved))
		ok, e := d.decide()
		satOK(t, e)
		l, e := p.variables.trailRead(0)
		satOK(t, e)
		if !ok || l.sign() != tc.negative || d.decisions != 1 || p.variables.scopeLevel != 1 {
			t.Fatal("source phase policy", tc, ok, l, d.decisions)
		}
		j, e := p.variables.justification.read(0)
		satOK(t, e)
		if !j.None() {
			t.Fatal("decision justification")
		}
	}
}

func TestSATSolverDecisionRandomPhaseAndInvalidCastPrefix(t *testing.T) {
	d, g := satDecisionFixture(t, 1)
	p := g.lifecycle().propagation
	p.config.state.RandomFreq = 0
	p.config.state.Phase = satPhaseRandom
	d.rng.SetSeed(79)
	oracle := z3rand.New(79)
	oracle.Next()
	negative := oracle.Next()%2 != 0
	ok, e := d.decide()
	satOK(t, e)
	l, e := p.variables.trailRead(0)
	satOK(t, e)
	if !ok || l.sign() != negative || d.rng.Next() != oracle.Next() {
		t.Fatal("phase RNG after next_var threshold draw")
	}
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1e20} {
		d, g := satDecisionFixture(t, 1)
		g.lifecycle().propagation.config.state.RandomFreq = f
		d.rng.SetSeed(21)
		oracle := z3rand.New(21)
		oracle.Next()
		_, e := d.nextVar()
		if !errors.Is(e, arena.ErrBounds) || d.rng.Next() != oracle.Next() || satScalarTestUintSize(t, &g.lifecycle().propagation.variables.queue.runtimeValues) != 2 {
			t.Fatal("guard only at reached undefined int cast; preserve first RNG draw", f, e)
		}
	}
}
