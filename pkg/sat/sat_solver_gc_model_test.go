// Source: src/sat/sat_solver.cpp:1378–1670; src/sat/sat_solver.h:380–389. Licensed under the MIT license.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverGCFrozenAllReactivationArities(t *testing.T) {
	for n := 0; n <= 5; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			g, s := satGCFixture(t, 7)
			p := s.analysis.propagation
			l := g.lifecycle()
			x := []Literal{1}
			if n == 0 {
				x = []Literal{1, 3, 5}
			} else {
				for i := 1; i <= n; i++ {
					x = append(x, Literal(2*i))
				}
			}
			if n == 1 {
				x = append(x, 5)
			}
			r := satGCClause(t, g, x, 5)
			satOK(t, l.detachClause(r))
			satOK(t, p.clauses.Freeze(r))
			g.numFrozen = 1
			satOK(t, p.assign(0, satJustification{}))
			if n == 0 {
				satOK(t, p.assign(2, satJustification{}))
			}
			if n < 2 {
				satOK(t, p.assign(4, satJustification{}))
			}
			// Source standalone activate helper returns false for deleted-unit/0/2 routes,
			// but does not itself delete the actual owning clause.
			satOK(t, p.clauses.Unfreeze(r))
			g.numFrozen--
			keep, e := g.activateFrozenClause(r)
			satOK(t, e)
			if keep != (n >= 3) {
				t.Fatal(n, keep)
			}
			if n == 0 && !p.inconsistent {
				t.Fatal("actual zero conflict")
			}
			if n == 1 {
				v, e := p.variables.valueLiteral(2)
				satOK(t, e)
				if v != util.LTrue {
					t.Fatal(v)
				}
			}
			if n >= 3 {
				z, e := p.clauses.Size(r)
				satOK(t, e)
				if z != uint32(n) {
					t.Fatal(z)
				}
			}
		})
	}
}
func TestSATSolverGCDynamicUsedResetAndDeletion(t *testing.T) {
	g, s := satGCFixture(t, 3)
	p := s.analysis.propagation
	cfg := &p.config.state
	cfg.GCStrategy = satGCDynPSM
	cfg.GCSmallLBD = 1
	cfg.GCK = 1
	cfg.Initialized |= satConfigGCSmallLBDMask | satConfigGCKMask
	r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
	satOK(t, p.clauses.IncInactivity(r))
	satOK(t, p.clauses.MarkUsed(r))
	s.conflictsSinceGC = g.gcThreshold + 1
	satOK(t, g.gc())
	n, e := p.clauses.Inactivity(r)
	satOK(t, e)
	if n != 0 {
		t.Fatal("was_used resets before unmark")
	}
	for i := 0; i < 2; i++ {
		s.conflictsSinceGC = g.gcThreshold + 1
		satOK(t, g.gc())
	}
	if p.clauses.IsLive(r) || g.gcClauseStat != 1 || len(g.lifecycle().learned) != 0 {
		t.Fatal("active inactivity delete")
	}
}
func TestSATSolverGCRefusalRootsAndReactivationPrefix(t *testing.T) {
	failures, success := 0, false
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			g, s := satGCFixture(t, 6)
			p := s.analysis.propagation
			cfg := &p.config.state
			cfg.GCStrategy = satGCDynPSM
			cfg.GCSmallLBD = 1
			cfg.GCK = 2
			cfg.Initialized |= satConfigGCSmallLBDMask | satConfigGCKMask
			// First old active clause is deleted, then a frozen successor reactivates and
			// grows the actual unit-assignment trail. Source compaction end has not yet been published.
			old := satGCClause(t, g, []Literal{6, 8, 10}, 5)
			for i := 0; i < 2; i++ {
				satOK(t, p.clauses.IncInactivity(old))
			}
			r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
			satOK(t, g.lifecycle().detachClause(r))
			satOK(t, p.clauses.Freeze(r))
			g.numFrozen = 1
			satOK(t, p.assign(1, satJustification{}))
			satOK(t, p.assign(3, satJustification{}))
			_, e := p.propagateCore(false)
			satOK(t, e)
			satOK(t, p.clauses.ids.ReserveFreeIDs(3))
			s.conflictsSinceGC = g.gcThreshold + 1
			g.budget().FailAfter(fault)
			e = g.gc()
			if e != nil {
				failures++
				satProbingExpectedRefusal(t, p, e)
			} else {
				success = true
				if p.clauses.IsLive(r) || p.clauses.IsLive(old) || len(g.lifecycle().learned) != 0 || g.numFrozen != 0 || g.gcClauseStat != 1 {
					t.Fatal("source result")
				}
				v, err := p.variables.valueLiteral(4)
				satOK(t, err)
				if v != util.LTrue {
					t.Fatal("reactivated unit assignment", v)
				}
			}
			satGCAudit(t, g)
			g.budget().FailAfter(-1)
		})
		if success {
			break
		}
	}
	if failures < 1 || !success {
		t.Fatal(failures, success)
	}
	t.Log("actual gc refusal prefixes", failures)
}
func TestSATSolverGCFloatDomainsAndLubyEdges(t *testing.T) {
	want := []uint32{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8}
	for i, x := range want {
		v, e := satGCLuby(uint32(i + 1))
		satOK(t, e)
		if v != x {
			t.Fatal(i, v, x)
		}
	}
	if v, e := satGCLuby(^uint32(0)); e != nil || v != 0 {
		t.Fatal("literal log0/equality/pow(-Inf) returns0", v, e)
	}
	g, s := satGCFixture(t, 1)
	s.analysis.propagation.config.state.Restart = satRestartGeometric
	s.analysis.propagation.config.state.RestartFactor = -0.5
	g.restartThreshold = 1
	satOK(t, g.restart())
	if g.restartThreshold != 0 {
		t.Fatal("truncation before unsigned conversion")
	}
	s.analysis.propagation.config.state.RestartFactor = -1
	g.restartThreshold = 2
	e := g.restart()
	if !errors.Is(e, arena.ErrBounds) || g.restarts != 2 {
		t.Fatal("real prefix before undefined conversion", e, g.restarts)
	}
}

// Real source learned-clause domain: both original binary clauses imply v0,
// so each learned clause containing v0 is redundant. Enumerate the whole finite
// truth table independently of assignment/propagation and physical GC ordering.
func TestSATSolverGCLearnedModelPreservation(t *testing.T) {
	for _, strategy := range []satGCStrategy{satGCGlue, satGCPSM, satGCGluePSM, satGCPSMGlue, satGCDynPSM} {
		t.Run(fmt.Sprint(strategy), func(t *testing.T) {
			g, s := satGCFixture(t, 4)
			p := s.analysis.propagation
			l := g.lifecycle()
			cfg := &p.config.state
			cfg.GCStrategy = strategy
			cfg.GCSmallLBD, cfg.GCK = 1, 0
			cfg.Initialized |= satConfigGCSmallLBDMask | satConfigGCKMask
			for _, lits := range [][]Literal{{0, 2}, {0, 3}} {
				_, e := l.MkClauseCore(uint32(len(lits)), lits, false)
				satOK(t, e)
			}
			for i := 0; i < 6; i++ {
				satGCClause(t, g, []Literal{0, 4, 6}, uint32(i+2))
			}
			models := func() uint32 {
				var result uint32
				for mask := uint32(0); mask < 16; mask++ {
					litTrue := func(x Literal) bool { return (mask&(1<<x.variable()) != 0) != (x.index()&1 != 0) }
					okay := (litTrue(0) || litTrue(2)) && (litTrue(0) || litTrue(3))
					for _, r := range l.learned {
						n, e := p.clauses.Size(r)
						satOK(t, e)
						clauseTrue := false
						for j := uint32(0); j < n; j++ {
							x, e := p.clauses.Literal(r, j)
							satOK(t, e)
							clauseTrue = clauseTrue || litTrue(x)
						}
						okay = okay && clauseTrue
					}
					if okay {
						result |= 1 << mask
					}
				}
				return result
			}
			before := models()
			s.conflictsSinceGC = g.gcThreshold + 1
			satOK(t, g.gc())
			if got := models(); got != before || got != 0xaaaa {
				t.Fatal("independent complete truth table", before, got)
			}
			satGCAudit(t, g)
		})
	}
}

func TestSATSolverGCSortWorkspaceRefusalAndWarmThreshold(t *testing.T) {
	failed, success := 0, false
	for fault := int64(0); fault < 32; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			g, s := satGCFixture(t, 3)
			p := s.analysis.propagation
			p.config.state.GCStrategy = satGCGluePSM // real source default
			refs := []StoredClauseRef{}
			for i := 0; i < 6; i++ {
				refs = append(refs, satGCClause(t, g, []Literal{0, 2, 4}, uint32(6-i)))
			}
			s.conflictsSinceGC = g.gcThreshold + 1
			g.budget().FailAfter(fault)
			e := g.gc()
			if e != nil {
				failed++
				satProbingExpectedRefusal(t, p, e)
				if len(g.lifecycle().learned) != 6 || g.gcClauseStat != 0 || s.conflictsSinceGC != g.gcThreshold+1 {
					t.Fatal("source unfinished end/counter publication")
				}
				for i, r := range refs {
					want := r
					if g.interrupted {
						want = refs[5-i]
					}
					if g.lifecycle().learned[i] != want {
						t.Fatal("actual sort prefix", i)
					}
				}
				if g.interrupted {
					if g.out != 3 || g.cursor < 3 || g.cursor >= 6 {
						t.Fatal("literal half cursor", g.out, g.cursor)
					}
					for i, r := range g.lifecycle().learned {
						if p.clauses.IsLive(r) != (i < 3 || uint32(i) >= g.cursor) {
							t.Fatal("actual consumed deletion prefix", i)
						}
					}
					if e := g.lifecycle().pushClause(true, refs[0]); !errors.Is(e, arena.ErrClosed) {
						t.Fatal("stopped ownership publication", e)
					}
				}
			} else {
				success = true
				if len(g.lifecycle().learned) != 3 || g.gcClauseStat != 3 || s.conflictsSinceGC != 0 {
					t.Fatal("source half publication")
				}
				for i, r := range g.lifecycle().learned {
					if r != refs[5-i] {
						t.Fatal("actual glue order", i)
					}
				}
			}
			satGCAudit(t, g) // immediate before fault disable
			g.budget().FailAfter(-1)
			if success {
				used := g.budget().Used()
				allocs := testing.AllocsPerRun(20, func() {
					if err := g.gc(); err != nil {
						panic(err)
					}
				})
				if allocs != 0 || g.budget().Used() != used {
					t.Fatal("warm source threshold plateau", allocs, used, g.budget().Used())
				}
			}
		})
		if success {
			break
		}
	}
	if failed == 0 || !success {
		t.Fatal(failed, success)
	}
	t.Log("actual sorting reservation refusals", failed)
}

func TestSATSolverGCFrozenTrueAndInactiveDelete(t *testing.T) {
	g, s := satGCFixture(t, 3)
	p := s.analysis.propagation
	l := g.lifecycle()
	cfg := &p.config.state
	cfg.GCStrategy = satGCDynPSM
	cfg.GCK = 0
	cfg.Initialized |= satConfigGCKMask
	r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
	satOK(t, l.detachClause(r))
	satOK(t, p.clauses.Freeze(r))
	g.numFrozen = 1
	// No assignments since GC: d_tk=vars+1 must not raise retained minimum0.
	g.minDTk = 0
	for i := uint32(0); i < 3; i++ {
		satOK(t, p.variables.phase.Write(i, 0))
		satOK(t, p.variables.previousPhase.Write(i, 0))
	}
	s.conflictsSinceGC = g.gcThreshold + 1
	satOK(t, g.gc())
	if p.clauses.IsLive(r) || g.numFrozen != 0 || g.gcClauseStat != 1 || len(l.learned) != 0 || g.minDTk != 0 {
		t.Fatal("frozen inactivity source prefix")
	}
	satGCAudit(t, g)
	r = satGCClause(t, g, []Literal{1, 2, 4}, 5)
	satOK(t, l.detachClause(r))
	satOK(t, p.clauses.Freeze(r))
	satOK(t, p.assign(0, satJustification{})) // first false; second true returns immediately
	satOK(t, p.assign(2, satJustification{}))
	satOK(t, p.clauses.Unfreeze(r))
	keep, e := g.activateFrozenClause(r)
	satOK(t, e)
	if keep {
		t.Fatal("true early return")
	}
	n, e := p.clauses.Size(r)
	satOK(t, e)
	if n != 3 {
		t.Fatal("true branch does not shrink", n)
	}
	satGCAudit(t, g)
}

// Explicit stopped-merge owner protocol, not an injected getter refusal: getters
// on valid live clauses allocate nothing and cannot hit Budget refusal. The
// workspace and clauses here are genuine owned allocations; only the interruption
// metadata is seeded after a concrete one-element scalar move. No backup list
// supplies ownership: consumed slots are borrowed duplicates of the destination.
func TestSATSolverGCMergeOwnershipProtocol(t *testing.T) {
	for _, fromScratch := range []bool{false, true} {
		t.Run(fmt.Sprint(fromScratch), func(t *testing.T) {
			g, _ := satGCFixture(t, 3)
			l := g.lifecycle()
			for i := 0; i < 4; i++ {
				satGCClause(t, g, []Literal{0, 2, 4}, uint32(i))
			}
			satOK(t, l.propagation.clauses.ids.ReserveFreeIDs(4))
			g.scratchBytes = 4 * uint64(reflect.TypeOf(StoredClauseRef{}).Size())
			satOK(t, g.budget().Reserve(g.scratchBytes))
			g.scratch = make([]StoredClauseRef, 4)
			if fromScratch {
				copy(g.scratch, l.learned)
			}
			g.sortSrcScratch = fromScratch
			g.sortN = 4
			g.sortLeft, g.sortMid, g.sortRight, g.sortEnd = 0, 1, 1, 2
			g.sortDestination()[0] = g.sortSource()[g.sortLeft]
			g.sortLeft++
			g.sortOut = 1
			g.sorting = true
			l.gcContinuation = g
			g.running = false
			satGCAudit(t, g)
			if e := l.pushClause(true, g.sortDestination()[0]); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("stopped merge mutation", e)
			}
			satOK(t, g.Close())
			if l.propagation.clauses.storage.Stats().Live != 0 || len(l.learned) != 0 || g.scratch != nil || l.gcContinuation != nil {
				t.Fatal("actual disjoint ranges must each release once")
			}
			used := g.budget().Used()
			satOK(t, g.Close())
			if g.budget().Used() != used {
				t.Fatal("idempotent cleanup")
			}
		})
	}
}
