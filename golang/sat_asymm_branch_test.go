// Source: src/sat/sat_asymm_branch.cpp and .h. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func satAsymmFixture(t *testing.T, n uint32) (*satAsymmBranch, *satSolverClauseLifecycle) {
	t.Helper()
	l, _, _, _ := satLifecycleFixture(t, n)
	params, e := l.propagation.config.context.NewValue()
	satOK(t, e)
	a, e := newSATAsymmBranch(l, &params)
	satOK(t, l.propagation.config.context.Del(&params))
	satOK(t, e)
	t.Cleanup(func() { satOK(t, a.Close()) })
	return a, l
}
func satAsymmClause(t *testing.T, l *satSolverClauseLifecycle, x []satLiteral) satStoredClauseRef {
	t.Helper()
	r := satCleanerClause(t, l, x, false, false)
	satOK(t, l.attachClause(r))
	return r
}
func TestSATAsymmDefaultAndSourceEarlyExits(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	p := l.propagation
	if !a.enabled || a.rounds != 32 || a.limit != 100000000 || p.asymmCounter != 0 {
		t.Fatal("defaults")
	}
	a.enabled = false
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, a.run(true))
	if p.variables.qhead != 0 {
		t.Fatal("disabled before propagate")
	}
	a.enabled = true
	p.asymmCounter = 20
	satOK(t, a.run(false))
	if p.variables.qhead != 1 || p.asymmCounter != 20 {
		t.Fatal("propagate then positive budget exit", p.asymmCounter)
	}
	copy := *a
	if !errors.Is(copy.Close(), ErrSATOwner) {
		t.Fatal("copied owner")
	}
	satOK(t, a.Close())
	if !errors.Is(a.run(true), arena.ErrClosed) {
		t.Fatal("closed")
	}
}
func TestSATAsymmProcessActualBinaryReduction(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	p := l.propagation
	_, e := l.mkClauseCore(2, []satLiteral{0, 2}, false)
	satOK(t, e)
	r := satAsymmClause(t, l, []satLiteral{0, 2, 4})
	satOK(t, a.run(true))
	if p.clauses.IsLive(r) || len(l.original) != 0 || a.elimLiterals != 1 || p.variables.scopeLevel != 0 {
		t.Fatal("real binary reduced", a.elimLiterals)
	}
	for i := uint32(0); i < 3; i++ {
		x, e := p.variables.valueVar(satBoolVar(i))
		satOK(t, e)
		if x != lUndef {
			t.Fatal("actual backtrack")
		}
	}
}
func TestSATAsymmSatisfiedAndNoReduction(t *testing.T) {
	a, l := satAsymmFixture(t, 4)
	p := l.propagation
	r := satAsymmClause(t, l, []satLiteral{0, 2, 4})
	satOK(t, a.run(true))
	if !p.clauses.IsLive(r) || len(l.original) != 1 || a.elimLiterals != 0 {
		t.Fatal("unchanged")
	}
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, a.run(true))
	if p.clauses.IsLive(r) || len(l.original) != 0 {
		t.Fatal("satisfied deletion")
	}
}

func TestSATAsymmCompleteFiniteModelAndAllReductionArities(t *testing.T) {
	for want := uint32(1); want <= 4; want++ {
		t.Run(fmt.Sprint(want), func(t *testing.T) {
			a, l := satAsymmFixture(t, 5)
			p := l.propagation
			// Redundant original clause extends a genuinely independent shorter constraint.
			base := []satLiteral{}
			for i := uint32(0); i < want; i++ {
				base = append(base, satLiteral(i*2))
			}
			_, e := l.mkClauseCore(want, base, false)
			satOK(t, e)
			if _, e = p.propagateCore(false); e != nil {
				satOK(t, e)
			}
			x := []satLiteral{0, 2, 4, 6, 8}
			r := satAsymmClause(t, l, x)
			// Exhaust all32 assignments: base logically implies the longer clause.
			model := func(mask uint32, lits []satLiteral) bool {
				for _, lit := range lits {
					if (mask&(1<<lit.variable()) != 0) != (lit.sign()) {
						return true
					}
				}
				return false
			}
			var expected uint32
			for mask := uint32(0); mask < 32; mask++ {
				if model(mask, base) && model(mask, x) {
					expected |= 1 << mask
				}
			}
			satOK(t, a.run(true))
			if want == 1 {
				if p.clauses.IsLive(r) {
					t.Fatal("already satisfied source delete")
				}
			} else if want == 2 {
				if p.clauses.IsLive(r) {
					t.Fatal("binary replacement")
				}
			} else {
				n, e := p.clauses.Size(r)
				satOK(t, e)
				if n != want {
					t.Fatal("nary source shrink", want, n)
				}
			}
			for mask := uint32(0); mask < 32; mask++ {
				okay := model(mask, base)
				if p.clauses.IsLive(r) {
					n, e := p.clauses.Size(r)
					satOK(t, e)
					clauseTrue := false
					for i := uint32(0); i < n; i++ {
						lit, e := p.clauses.Literal(r, i)
						satOK(t, e)
						clauseTrue = clauseTrue || model(mask, []satLiteral{lit})
					}
					okay = okay && clauseTrue
				}
				if okay != (expected&(1<<mask) != 0) {
					t.Fatal("finite implication model", mask)
				}
			}
			if p.variables.scopeLevel != 0 {
				t.Fatal("real pop")
			}
			satAsymmAudit(t, a)
		})
	}
}
func TestSATAsymmSourceSolverCatchAndNoPhaseRestoration(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	p := l.propagation
	satAsymmClause(t, l, []satLiteral{0, 2, 4})
	p.limit.setCancel(1)
	e := a.run(true)
	if !errors.Is(e, ErrSATSolverCanceled) || a.interrupted || a.sorting || len(l.original) != 1 || p.asymmCounter != 1 {
		t.Fatal("actual checkpoint catch", e, p.asymmCounter)
	}
	satAsymmAudit(t, a)
	p.limit.resetCancel()
	// Actual source catch inside process leaves the pushed scope and assignment.
	satOK(t, p.limit.push(2))
	e = a.run(true)
	if !errors.Is(e, ErrSATSolverCanceled) || p.variables.scopeLevel != 1 || a.detachActive || a.interrupted {
		t.Fatal("source exception no automatic pop", e, p.variables.scopeLevel)
	}
	value, e := p.variables.phase.read(0)
	satOK(t, e)
	if value != 1 {
		t.Fatal("catch skips saved-phase restore", value)
	}
	satAsymmAudit(t, a)
	p.limit.pop()
	satOK(t, l.pop(1)) // explicit caller action after audit
}
func TestSATAsymmEveryActualRefusalPrefix(t *testing.T) {
	failures, success := 0, false
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			a, l := satAsymmFixture(t, 5)
			p := l.propagation
			_, e := l.mkClauseCore(2, []satLiteral{0, 2}, false)
			satOK(t, e)
			satAsymmClause(t, l, []satLiteral{0, 2, 4, 6, 8})
			tie1 := satAsymmClause(t, l, []satLiteral{0, 4, 6})
			tie2 := satAsymmClause(t, l, []satLiteral{0, 6, 8})
			l.budget.FailAfter(fault)
			t.Cleanup(func() { l.budget.FailAfter(-1) })
			e = a.run(true)
			if e != nil {
				failures++
				satProbingExpectedRefusal(t, p, e)
			} else {
				success = true
				if a.elimLiterals < 3 || p.variables.scopeLevel != 0 || len(l.original) != 2 || l.original[0] != tie1 || l.original[1] != tie2 {
					t.Fatal("full source result", a.elimLiterals)
				}
			}
			satAsymmAudit(t, a) // exact scalar owners BEFORE fault disable or cleanup
			if e != nil && !a.interrupted && !a.sorting && p.variables.phase.block == (memory.Block{}) {
				// Source assignment already destroyed the old phase vector before
				// allocation refusal. A new operator invocation must not silently
				// treat that interrupted assignment as a fresh empty phase owner.
				if next := a.run(true); !errors.Is(next, arena.ErrClosed) {
					t.Fatal("interrupted phase assignment guard", next)
				}
			}
			l.budget.FailAfter(-1)
			if a.interrupted || a.sorting {
				satOK(t, l.Close())
			}
		})
		if success {
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal(failures, success)
	}
	t.Log("actual refusal horizon", failures)
}

func TestSATAsymmActualNewUnitAndCounterLimit(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	p := l.propagation
	for _, x := range [][]satLiteral{{0, 2}, {0, 3}} {
		_, e := l.mkClauseCore(2, x, false)
		satOK(t, e)
	}
	r := satAsymmClause(t, l, []satLiteral{0, 2, 4})
	before := p.variables.phase.block
	satOK(t, a.run(true))
	value, e := p.variables.valueLiteral(0)
	satOK(t, e)
	if value != lTrue || p.clauses.IsLive(r) || a.elimLiterals != 2 || len(l.original) != 0 || p.variables.scopeLevel != 0 {
		t.Fatal("actual first-assumption conflict creates new root unit")
	}
	if p.variables.phase.block == before {
		t.Fatal("source assignment destroys old phase before copied restore")
	}
	for i := uint32(0); i < 3; i++ {
		x, e := p.variables.phase.read(i)
		satOK(t, e)
		if x != 2 {
			t.Fatal("normal exit restores lexical phase", i, x)
		}
	}
	satAsymmAudit(t, a)
	a, l = satAsymmFixture(t, 3)
	p = l.propagation
	satAsymmClause(t, l, []satLiteral{0, 2, 4})
	a.limit = 0
	// decrement by original count1 already makes counter<0: source no process.
	satOK(t, a.run(true))
	if a.elimLiterals != 0 || len(l.original) != 1 || p.asymmCounter != 1 {
		t.Fatal("zero quota source skips with exact cost", p.asymmCounter)
	}
	used := l.budget.Used()
	for i := 0; i < 3; i++ {
		satOK(t, a.run(true))
	}
	if l.budget.Used() != used {
		t.Fatal("physical plateau", used, l.budget.Used())
	}
	satAsymmAudit(t, a)
	allocs := testing.AllocsPerRun(20, func() {
		if err := a.run(true); err != nil {
			panic(err)
		}
	})
	if l.budget.Used() != used {
		t.Fatal("warmed source lexical physical plateau", used, l.budget.Used())
	}
	t.Log("warm source lexical phase copies/sort Go allocations", allocs)
	a.dec(^uint32(0))
	if p.asymmCounter != 2 {
		t.Fatal("unsigned add/narrow profile")
	}
	p.asymmCounter = -1 << 31
	if !errors.Is(a.negateCounter(), arena.ErrBounds) {
		t.Fatal("source INT_MIN negation undefined")
	}
}
func TestSATAsymmParametersStatsAndColdRefusal(t *testing.T) {
	a, l := satAsymmFixture(t, 1)
	ctx := l.propagation.config.context
	params, e := ctx.NewValue()
	satOK(t, e)
	defer func() { satOK(t, ctx.Del(&params)) }()
	key := "asymm_branch.rounds"
	satOK(t, ctx.SetUIntName(&params, &key, 7))
	key = "asymm_branch.limit"
	satOK(t, ctx.SetUIntName(&params, &key, ^uint32(0)))
	key = "asymm_branch"
	satOK(t, ctx.SetBoolName(&params, &key, false))
	satOK(t, a.Update(&params))
	if a.enabled || a.rounds != 7 || a.limit != 1<<31-1 {
		t.Fatal("source parameter keys/clamp")
	}
	st, r := statFixture(t, nil)
	sk := statKey(t, r, "elim literals")
	a.elimLiterals = 9
	satOK(t, a.collectStatistics(st, sk))
	x, e := st.UintValue(0)
	satOK(t, e)
	if x != 9 {
		t.Fatal(x)
	}
	a.resetStatistics()
	if a.elimLiterals != 0 {
		t.Fatal("resetstats")
	}
	d := pdnew(t, symbolManagerTest(t, nil, false), nil)
	satOK(t, collectAsymmBranchParamDescrs(d))
	n, e := d.Size()
	satOK(t, e)
	if n != 3 {
		t.Fatal("all source descriptors", n)
	}
	before := l.budget.Used()
	l.propagation.asymmCounter = 17
	l.budget.FailAfter(0)
	v, e := newSATAsymmBranch(l, &params)
	if v != nil || !errors.Is(e, arena.ErrInjectedFailure) || l.budget.Used() != before || l.propagation.asymmCounter != 17 {
		t.Fatal("cold reserve before actual constructor", e)
	}
	l.budget.FailAfter(-1)
	if a.ownerBytes != uint64(reflect.TypeOf(satAsymmBranch{}).Size()) {
		t.Fatal("actual cold metadata charge")
	}
}

// A source-undefined malformed char payload reaches copy_core's ascending read
// after allocating and publishing the destination. A failed constructor has no
// source destructor; its retained bytes are drained only by owning Close.
func TestSATAsymmFailedPhaseConstructorUnwindsReportBeforeOwningAbort(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	phase := &l.propagation.variables.phase
	satOK(t, phase.release(&phase.block))
	h, e := phase.runtime.Allocate(12)
	satOK(t, e)
	phase.block = h
	satOK(t, satAssignmentWriteWord(phase.runtime, h, 0, 4))
	satOK(t, satAssignmentWriteWord(phase.runtime, h, 4, 3))
	satOK(t, phase.runtime.Write(h, 8, 1))
	e = a.run(true)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatalf("actual constructor read guard: %v", e)
	}
	if a.reportActive || a.savedPhase.block == (memory.Block{}) {
		t.Fatal("failed constructor must stop older report and retain partial owning copy", a.reportActive, a.savedPhase.block)
	}
	if x, e := phase.runtime.Read(a.savedPhase.block, 8); e != nil || x != 1 {
		t.Fatal("completed ascending copy prefix", x, e)
	}
	if _, e := phase.runtime.Read(a.savedPhase.block, 9); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("unconstructed copy suffix", e)
	}
	if !errors.Is(a.run(true), arena.ErrClosed) {
		t.Fatal("partial constructor owning root must reject reuse")
	}
	satOK(t, a.Close())
}

func TestSATAsymmDebugScopeAssertionPrecedesTrailRead(t *testing.T) {
	if !satStorageSourceDebug {
		return
	}
	a, l := satAsymmFixture(t, 3)
	v := l.propagation.variables
	old := v.trail.values.raw.block
	// An actual released token preserves the checked unsafe read boundary.
	dead, e := l.propagation.memory.Allocate(8)
	satOK(t, e)
	stale := dead
	satOK(t, l.propagation.memory.Deallocate(&dead))
	v.trail.values.raw.block = stale
	v.scopeLevel = 1
	_, e = a.process(satStoredClauseRef{})
	v.trail.values.raw.block = old
	v.scopeLevel = 0
	if !errors.Is(e, ErrSATAssertion) {
		t.Fatalf("scope assertion must precede endangered trail read: %v", e)
	}
}

func TestSATAsymmCompletedLocalDestructionStopsBeforeOlderReport(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	h, e := l.propagation.memory.Allocate(12)
	satOK(t, e)
	stale := h
	satOK(t, l.propagation.memory.Deallocate(&h))
	a.savedPhase.block = stale
	a.savedConstructed = true
	satOK(t, a.report.Start())
	a.reportActive = true
	e = a.finishRun(nil)
	if e == nil || !a.reportActive || !a.savedConstructed || a.savedPhase.block != stale {
		t.Fatal("actual stale-token destruction must retain older report and source-local cursor", e)
	}
	if !errors.Is(a.run(true), arena.ErrClosed) {
		t.Fatal("stopped lexical unwind cannot enter source body")
	}
	// Explicit acknowledgment of the externally freed token, not source repair.
	a.savedPhase.block = memory.Block{}
	satOK(t, a.Close())
	if a.reportActive {
		t.Fatal("owning Close must acknowledge local before report")
	}
}

// Protocol fixture over a real completed local and real public TLS operations.
// The source signed-overflow domain is undefined; the checked free physically
// completes before refusing TLS synchronization. No private counter is written.
func TestSATAsymmPostphysicalLocalFreeRetainsOlderReport(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: l.budget, SizeTBits: 64, ThreadLocal: true})
	satOK(t, e)
	a.savedPhase = satSolverAssignmentColumn{}
	a.savedPhase.bind(rt)
	h, e := rt.Allocate(100001)
	satOK(t, e)
	a.savedPhase.block = h
	a.savedConstructed = true
	first, e := rt.Allocate(8)
	satOK(t, e)
	second, e := rt.Allocate(8)
	satOK(t, e)
	delta := int64(-1000000)
	if e = rt.Reallocate(&first, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual first host extent", e)
	}
	satOK(t, rt.Deallocate(&first))
	st := rt.State()
	if st.LocalSize != 0 || st.GlobalSize >= 0 {
		t.Fatal("actual first synchronization", st)
	}
	target := int64(math.MinInt64) + 1000 - st.GlobalSize + 16
	if e = rt.Reallocate(&second, uint64(target)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual second host extent", e)
	}
	satOK(t, rt.Deallocate(&second))
	if st = rt.State(); st.GlobalSize != math.MinInt64+1000 || st.LocalSize != 0 {
		t.Fatal("real near-minimum TLS prefix", st)
	}
	satOK(t, a.report.Start())
	a.reportActive = true
	e = a.finishRun(nil)
	if !errors.Is(e, arena.ErrBounds) || a.savedPhase.block != (memory.Block{}) || rt.Stats().LiveBlocks != 0 || !a.savedConstructed || !a.reportActive {
		t.Fatal("postphysical completion must retain lexical acknowledgment and older report", e, rt.Stats(), a.savedConstructed, a.reportActive)
	}
	if !errors.Is(a.run(true), arena.ErrClosed) {
		t.Fatal("stopped report cannot be reused")
	}
	satOK(t, a.Close())
	if a.reportActive || a.savedConstructed {
		t.Fatal("explicit owning cleanup acknowledgment")
	}
	satOK(t, rt.Close())
}
