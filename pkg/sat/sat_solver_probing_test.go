// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satProbingFixture(t *testing.T, n uint32) (*satSolverProbing, *satSolverClauseLifecycle) {
	t.Helper()
	life, _, _, b := satLifecycleFixture(t, n)
	ctx, _, _, _ := satMainConfigTestRuntime(t)
	params, e := ctx.NewValue()
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ctx.Del(&params)) })
	s, e := newSATSolverProbing(life, ctx, &params, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, life
}
func TestSATSolverProbingSourceDefaultsAndTypo(t *testing.T) {
	s, _ := satProbingFixture(t, 0)
	if !s.enabled || !s.cacheEnabled || !s.binaryEnabled || s.limit != 5000000 || s.cacheLimit != uint64(1024)<<20 || s.stoppedAt != 0 || s.numAssigned != 0 || s.propagation().probingCounter != 0 {
		t.Fatal("source defaults")
	}
	ctx, _, _, _ := satMainConfigTestRuntime(t)
	p, e := ctx.NewValue()
	satOK(t, e)
	defer func() { satOK(t, ctx.Del(&p)) }()
	k := "probing_cache_limit"
	satOK(t, ctx.SetUIntName(&p, &k, 2))
	satOK(t, s.updateParams(ctx, &p))
	if s.cacheLimit != uint64(1024)<<20 {
		t.Fatal("correctly spelled key must be ignored by source")
	}
	k = "probing_chache_limit"
	satOK(t, ctx.SetUIntName(&p, &k, 0))
	satOK(t, s.updateParams(ctx, &p))
	if s.cacheLimit != 0 {
		t.Fatal("source typo key")
	}
	copy := *s
	if e := copy.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if _, e = s.run(false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATSolverProbingCommonConsequenceAndNormalFinalize(t *testing.T) {
	s, life := satProbingFixture(t, 2)
	// Independent truth-table implication: (x OR y) AND (!x OR y) entails y.
	satOK(t, life.mkBinClause(0, 2, false))
	satOK(t, life.mkBinClause(1, 2, false))
	for mask := 0; mask < 4; mask++ {
		x := mask&1 != 0
		y := mask&2 != 0
		if (x || y) && (!x || y) && !y {
			t.Fatal("oracle")
		}
	}
	done, e := s.run(true)
	satOK(t, e)
	if !done {
		t.Fatal("normal completion")
	}
	v, e := life.propagation.variables.valueVar(1)
	satOK(t, e)
	if v != lbool.True || s.numAssigned != 1 || life.propagation.variables.scopeLevel != 0 {
		t.Fatal(v, s.numAssigned)
	}
	size := s.assigned.size()
	if s.cache.blockLive() || size != 0 {
		t.Fatal("normal source finalize")
	}
}
func TestSATSolverProbingFailedLiteralAndTryLiteralCounterDifference(t *testing.T) {
	s, life := satProbingFixture(t, 2)
	// x implies y and !y, so source process asserts !x and counts one.
	satOK(t, life.mkBinClause(1, 2, false))
	satOK(t, life.mkBinClause(1, 3, false))
	satOK(t, s.process(0))
	v, e := life.propagation.variables.valueVar(0)
	satOK(t, e)
	if v != lbool.False || s.numAssigned != 1 || s.propagation().probingCounter != 0 {
		t.Fatal(v, s.numAssigned, s.propagation().probingCounter)
	}
}
func TestSATSolverProbingCacheBorrowAndReset(t *testing.T) {
	s, life := satProbingFixture(t, 3)
	p := life.propagation
	satOK(t, p.assignCore(2, satJustification{}))
	satOK(t, p.assignCore(4, satJustification{}))
	satOK(t, s.cacheBins(0, 0))
	borrow, found, e := s.cachedImpliedLits(0)
	satOK(t, e)
	if !found {
		t.Fatal("cache missing")
	}
	n, e := borrow.size()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	satOK(t, s.resetCache(0))
	n, e = borrow.size()
	satOK(t, e)
	if n != 0 {
		t.Fatal("same entry pointer survives inner finalize", n)
	}
	_, found, e = s.cachedImpliedLits(0)
	satOK(t, e)
	if found {
		t.Fatal("available cleared")
	}
	// Source outer relocation invalidates captured cache_entry pointer.
	satOK(t, s.cacheBins(5, 0))
	if _, e = borrow.size(); e == nil {
		t.Fatal("relocated entry borrower")
	}
}
func TestSATSolverProbingZeroQuotaAndActualCancellation(t *testing.T) {
	s, life := satProbingFixture(t, 3)
	s.limit = 0
	done, e := s.run(true)
	satOK(t, e)
	if done || s.stoppedAt != 1 || s.propagation().probingCounter != 4 {
		t.Fatal(done, s.stoppedAt, s.propagation().probingCounter)
	}
	s.limit = 5000000
	s.propagation().limit.SetCancel(1)
	_, e = s.run(true)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	if life.propagation.variables.scopeLevel != 0 {
		t.Fatal("checkpoint before process")
	}
	s.propagation().limit.ResetCancel()
}
