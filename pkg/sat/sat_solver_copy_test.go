// Source: src/sat/sat_solver.cpp:75-127; src/sat/sat_solver.h:257.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverCopyActualFlagsTrailClausesAndUserScope(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 5; i++ {
		_, e := src.MkVar(i%2 == 0, i%2 != 0)
		satOK(t, e)
	}
	_, e := dst.MkVar(false, false)
	satOK(t, e)
	satOK(t, src.MkClause([]Literal{8}))
	satOK(t, src.MkClause([]Literal{0, 2}))
	satOK(t, src.MkClause([]Literal{1, 3, 4, 6}))
	// Valid entailed learned binary: the base unit 8 entails (8 or 0).
	satOK(t, src.Lifecycle.mkBinClause(8, 0, true))
	satOK(t, src.UserPush())
	satOK(t, dst.copyFrom(src))
	n, e := dst.Variables.numVarsChecked()
	satOK(t, e)
	if n != 6 {
		t.Fatal(n)
	}
	for v := uint32(1); v < n; v++ {
		a, e := src.Variables.IsExternal(BoolVar(v))
		satOK(t, e)
		b, e := dst.Variables.IsExternal(BoolVar(v))
		satOK(t, e)
		if a != b {
			t.Fatal(v, a, b)
		}
		a, e = src.Variables.decisionEligible(BoolVar(v))
		satOK(t, e)
		b, e = dst.Variables.decisionEligible(BoolVar(v))
		satOK(t, e)
		if a != b {
			t.Fatal(v, a, b)
		}
	}
	ext, e := dst.Variables.IsExternal(BoolVar(0))
	satOK(t, e)
	if ext {
		t.Fatal("existing flag overwritten", ext)
	}
	value, e := dst.Variables.valueLiteral(8)
	satOK(t, e)
	if value != lbool.True {
		t.Fatal(value)
	}
	if dst.Lifecycle.mkBinary != 1 || dst.Lifecycle.mkNary != 1 || len(dst.Lifecycle.learned) != 0 {
		t.Fatal(dst.Lifecycle.mkBinary, dst.Lifecycle.mkNary, len(dst.Lifecycle.learned))
	}
	n, e = dst.conflict.userScopeLiterals.Size()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	l, e := dst.conflict.userScopeLiterals.Read(0)
	satOK(t, e)
	if l != 10 {
		t.Fatal(l)
	}
	satFullSolverRuntimeAudit(t, src)
	satFullSolverRuntimeAudit(t, dst)
	satFullSolverUserClauseAudit(t, src)
	satFullSolverUserClauseAudit(t, dst)
}
func TestSATSolverCopySelfRelocationGuardsReachedOriginalIterator(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2, 4, 6}))
	satOK(t, s.MkClause([]Literal{1, 2, 4, 6}))
	if len(s.Lifecycle.Original) != 2 {
		t.Fatal("native original list", len(s.Lifecycle.Original))
	}
	e := s.copyFrom(s)
	if e != ErrSATOwner {
		t.Fatal("captured source backing guard", e)
	}
	if len(s.Lifecycle.Original) != 3 {
		t.Fatal("completed first append", len(s.Lifecycle.Original))
	}
	satFullSolverRuntimeAudit(t, s)
	satFullSolverUserClauseAudit(t, s)
}
func TestSATSolverCopySelfWithinCapacityPreservesDefinedAppend(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2, 4, 6}))
	satOK(t, s.conflict.userScopeLiterals.Push(0))
	satOK(t, s.copyFrom(s))
	if len(s.Lifecycle.Original) != 2 {
		t.Fatal(len(s.Lifecycle.Original))
	}
	n, e := s.conflict.userScopeLiterals.Size()
	satOK(t, e)
	if n != 0 {
		t.Fatal("source self reset then append sees empty", n)
	}
	satFullSolverRuntimeAudit(t, s)
	satFullSolverUserClauseAudit(t, s)
}
func TestSATSolverCopyUsesOnlySourceBaseTrailAndPopsDestination(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := src.MkVar(true, true)
		satOK(t, e)
		_, e = dst.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, src.propagation.assign(0, satJustification{}))
	_, e := src.Lifecycle.propagate(false)
	satOK(t, e)
	satOK(t, src.Lifecycle.push())
	satOK(t, src.propagation.assign(2, satJustification{}))
	satOK(t, dst.Lifecycle.push())
	satOK(t, dst.propagation.assign(4, satJustification{}))
	satOK(t, dst.copyFrom(src))
	if dst.Variables.scopeLevel != 0 || src.Variables.scopeLevel != 1 {
		t.Fatal(dst.Variables.scopeLevel, src.Variables.scopeLevel)
	}
	for l, w := range map[Literal]lbool.LBool{0: lbool.True, 2: lbool.Undef, 4: lbool.Undef} {
		v, e := dst.Variables.valueLiteral(l)
		satOK(t, e)
		if v != w {
			t.Fatal(l, v, w)
		}
	}
	satFullSolverRuntimeAudit(t, src)
	satFullSolverRuntimeAudit(t, dst)
}
func TestSATSolverCopySourceClosedGuard(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	satOK(t, src.Close())
	if e := dst.copyFrom(src); e != arena.ErrClosed {
		t.Fatal("closed borrowed source", e)
	}
	satFullSolverRuntimeAudit(t, dst)
}
func TestSATSolverCopyDebugEliminatedSourceAssertionAfterEarlierVariable(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := src.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, src.Variables.setEliminated(BoolVar(1), true))
	e := dst.copyFrom(src)
	n, e2 := dst.Variables.numVarsChecked()
	satOK(t, e2)
	if satStorageSourceDebug {
		if e != ErrSATAssertion || n != 1 || dst.Variables.mkVarCount != 1 {
			t.Fatal(e, n, dst.Variables.mkVarCount)
		}
	} else {
		satOK(t, e)
		if n != 2 {
			t.Fatal(n)
		}
	}
	satFullSolverRuntimeAudit(t, src)
	satFullSolverRuntimeAudit(t, dst)
}
func TestSATSolverCopyDebugConverterAssertionAfterDestinationPop(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	_, e := src.MkVar(true, true)
	satOK(t, e)
	_, e = dst.MkVar(true, true)
	satOK(t, e)
	_, e = src.converter.Mk(satModelElimVar, 0)
	satOK(t, e)
	satOK(t, dst.Lifecycle.push())
	satOK(t, dst.propagation.assign(0, satJustification{}))
	e = dst.copyFrom(src)
	if dst.Variables.scopeLevel != 0 {
		t.Fatal(dst.Variables.scopeLevel)
	}
	value, e2 := dst.Variables.valueLiteral(0)
	satOK(t, e2)
	if value != lbool.Undef {
		t.Fatal(value)
	}
	if satStorageSourceDebug {
		if e != ErrSATAssertion {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	satFullSolverRuntimeAudit(t, src)
	satFullSolverRuntimeAudit(t, dst)
}
func TestSATSolverCopyWarmUnitOnlyPhysicalPlateau(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	_, e := src.MkVar(true, true)
	satOK(t, e)
	satOK(t, src.MkClause([]Literal{0}))
	var used uint64
	var blocks uint32
	for pass := 0; pass < 6; pass++ {
		satOK(t, dst.copyFrom(src))
		satFullSolverRuntimeAudit(t, dst)
		if pass == 2 {
			used = dst.budget.Used()
			blocks = dst.runtime.memory.Stats().LiveBlocks
		}
		if pass > 2 && (used != dst.budget.Used() || blocks != dst.runtime.memory.Stats().LiveBlocks) {
			t.Fatal(used, dst.budget.Used(), blocks, dst.runtime.memory.Stats().LiveBlocks)
		}
	}
}

// Packed-word safety-domain fixture: source is_binary_non_learned_clause is
// exactly m_val2==0, even when the kind bits alone still decode as BINARY.
func TestSATSolverCopyReadsExactBinaryNonLearnedPackedWord(t *testing.T) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := src.MkVar(true, true)
		satOK(t, e)
	}
	a, e := src.Variables.watches.Get(0)
	satOK(t, e)
	satOK(t, a.Push(satWatched{val1: 2, val2: 8}))
	b, e := src.Variables.watches.Get(3)
	satOK(t, e)
	satOK(t, b.Push(satWatched{val1: 1, val2: 8}))
	satOK(t, dst.copyFrom(src))
	if dst.Lifecycle.mkBinary != 0 {
		t.Fatal("source exact packed-word filter", dst.Lifecycle.mkBinary)
	}
	satFullSolverRuntimeAudit(t, src)
	satFullSolverRuntimeAudit(t, dst)
	if er := src.Close(); er != nil {
		if src.closeStage != 1 || src.closeSourceCause != er {
			t.Fatal(er, src.closeStage)
		}
		satOK(t, src.CleanupStopped())
	}
}
