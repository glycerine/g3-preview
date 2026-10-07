// Source: src/sat/sat_solver.cpp:2781-2801,3716-3751; src/sat/sat_solver.h:48-70.
package sat

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func satSolverStatisticsFixture(t *testing.T, b *arena.Budget) (*util.Statistics, satSolverStatisticsKeys) {
	st, r := statFixture(t, b)
	var keys satSolverStatisticsKeys
	names := [15]string{"mk bool var", "mk binary clause", "mk ternary clause", "mk clause", "gc clause", "del clause", "conflicts", "propagations", "decisions", "binary propagations", "ternary propagations", "restarts", "minimized lits", "dyn subsumption resolution", "blocked correction sets"}
	for i, n := range names {
		keys.main[i] = statKey(t, r, n)
	}
	for i, n := range [2]string{"elim clauses", "elim literals"} {
		keys.cleaner[i] = statKey(t, r, n)
	}
	for i, n := range [5]string{"subsumed", "subsumption resolution", "elim literals", "elim bool vars", "elim blocked clauses"} {
		keys.simplifier[i] = statKey(t, r, n)
	}
	keys.scc = statKey(t, r, "elim bool vars")
	keys.asymm = statKey(t, r, "elim literals")
	keys.probing = statKey(t, r, "probing assigned")
	return st, keys
}
func satSolverSeedStatistics(s *Solver) {
	s.Variables.mkVarCount = 1
	s.Lifecycle.mkBinary = 2
	s.Lifecycle.mkTernary = 3
	s.Lifecycle.mkNary = 4
	s.gc.gcClauseStat = 5
	s.Lifecycle.deleted = 6
	s.conflict.conflictStat = 7
	s.propagation.clausePropagations = 8
	s.decision.decisions = 9
	s.propagation.binaryPropagations = 10
	s.propagation.ternaryPropagations = 11
	s.gc.restartStat = 12
	s.analysis.minimizedLits = 13
	s.dynamic.eliminated = 14
	s.blockedCorrectionSets = 15
	s.cleaner.elimClauses = 16
	s.cleaner.elimLiterals = 17
	s.simplifier.numSubsumed = 18
	s.simplifier.numSubRes = 19
	s.simplifier.numElimLits = 20
	s.simplifier.numElimVars = 21
	s.simplifier.numBlocked = 22
	s.scc.numElim = 23
	s.asymm.elimLiterals = 24
	s.probing.numAssigned = 25
}
func TestSATSolverStatisticsActualSourceOrderAndResetKeepsSCC(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	st, keys := satSolverStatisticsFixture(t, nil)
	satSolverSeedStatistics(s)
	s.Lifecycle.nonLearnedGeneration = 26
	s.conflict.conflicts = 99
	s.conflict.conflictsSinceRestart = 77
	s.gc.restarts = 44
	s.propagation.cleanerCounter = -17
	satOK(t, s.collectStatistics(st, keys))
	if st.Size() != 25 {
		t.Fatal(st.Size())
	}
	for i := uint32(0); i < 25; i++ {
		v, e := st.UintValue(i)
		satOK(t, e)
		if v != i+1 {
			t.Fatal(i, v)
		}
	}
	satOK(t, s.resetStatistics())
	if s.Lifecycle.nonLearnedGeneration != 0 || s.scc.numElim != 23 || s.conflict.conflicts != 99 || s.conflict.conflictsSinceRestart != 77 || s.gc.restarts != 44 || s.propagation.cleanerCounter != -17 {
		t.Fatal("source reset boundaries")
	}
	satOK(t, st.Reset())
	satOK(t, s.collectStatistics(st, keys))
	if st.Size() != 1 {
		t.Fatal("SCC is intentionally retained", st.Size())
	}
	v, e := st.UintValue(0)
	satOK(t, e)
	if v != 23 {
		t.Fatal(v)
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverStatisticsZeroSkipAndWarmPhysicalPlateau(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	b := arena.NewBudget(^uint64(0))
	st, k := satSolverStatisticsFixture(t, b)
	before := b.Used()
	b.FailAfter(0)
	satOK(t, s.collectStatistics(st, k))
	if b.Used() != before || st.Size() != 0 {
		t.Fatal("all real constructor counters initially zero", b.Used(), before, st.Size())
	}
	b.FailAfter(-1)
	satSolverSeedStatistics(s)
	satOK(t, s.collectStatistics(st, k))
	used := b.Used()
	for pass := 0; pass < 6; pass++ {
		satOK(t, st.Reset())
		b.FailAfter(0)
		satOK(t, s.collectStatistics(st, k))
		if b.Used() != used || st.Size() != 25 {
			t.Fatal("warm actual output backing", b.Used(), used, st.Size())
		}
		b.FailAfter(-1)
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverStatisticsPreservesUint32AndDuplicateKeyContributions(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	st, k := satSolverStatisticsFixture(t, nil)
	s.Variables.mkVarCount = ^uint32(0)
	s.cleaner.elimLiterals = 3
	s.simplifier.numElimLits = 5
	s.asymm.elimLiterals = 7
	// Source update appends contributions rather than coalescing key spellings.
	satOK(t, s.collectStatistics(st, k))
	want := []uint32{^uint32(0), 3, 5, 7}
	if st.Size() != uint32(len(want)) {
		t.Fatal(st.Size())
	}
	for i, w := range want {
		v, e := st.UintValue(uint32(i))
		satOK(t, e)
		if v != w {
			t.Fatal(i, v, w)
		}
	}
	satOK(t, s.collectStatistics(st, k))
	if st.Size() != 8 {
		t.Fatal(st.Size())
	}
	for i, w := range want {
		v, e := st.UintValue(uint32(i + 4))
		satOK(t, e)
		if v != w {
			t.Fatal(i, v, w)
		}
	}
}
func TestSATSolverStatisticsCopiedClosedAndHeaderCharge(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	st, k := satSolverStatisticsFixture(t, nil)
	if s.ownerBytes != uint64(reflect.TypeOf(Solver{}).Size()) || s.blockedCorrectionSets != 0 {
		t.Fatal("actual source field/header")
	}
	copied := *s
	if e := copied.collectStatistics(st, k); e != ErrSATOwner {
		t.Fatal(e)
	}
	if e := copied.resetStatistics(); e != ErrSATOwner {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if e := s.collectStatistics(st, k); e != arena.ErrClosed {
		t.Fatal(e)
	}
	if e := s.resetStatistics(); e != arena.ErrClosed {
		t.Fatal(e)
	}
}
func TestSATSolverStatisticsIndependentSparseCounterMasks(t *testing.T) {
	for mask := uint32(0); mask < 32; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			st, k := satSolverStatisticsFixture(t, nil)
			satSolverSeedStatistics(s)
			positions := [5]int{0, 15, 17, 22, 24}
			var expected [25]uint32
			for i := range expected {
				expected[i] = uint32(i + 1)
			}
			for j, pos := range positions {
				if mask&(1<<uint32(j)) == 0 {
					expected[pos] = 0
				}
			}
			s.Variables.mkVarCount = expected[0]
			s.cleaner.elimClauses = expected[15]
			s.simplifier.numSubsumed = expected[17]
			s.scc.numElim = expected[22]
			s.probing.numAssigned = expected[24]
			satOK(t, s.collectStatistics(st, k))
			index := uint32(0)
			for _, want := range expected {
				if want == 0 {
					continue
				}
				v, e := st.UintValue(index)
				satOK(t, e)
				if v != want {
					t.Fatal(index, v, want, mask)
				}
				index++
			}
			if st.Size() != index {
				t.Fatal(st.Size(), index)
			}
			satFullSolverRuntimeAudit(t, s)
		})
	}
}
