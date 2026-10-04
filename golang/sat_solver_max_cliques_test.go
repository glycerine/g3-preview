// Source: src/util/max_cliques.h; src/sat/sat_solver.cpp:3113-3150.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestSATMaxCliquesNegatedLiteralPartitionAndSourceCache(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	for _, p := range [][2]uint32{{1, 3}, {1, 5}, {3, 5}} {
		satOK(t, g.addEdge(p[0], p[1]))
	}
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.vector(s, 0, nil))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	for _, v := range []uint32{0, 2, 4} {
		satOK(t, f.vectors[0].push(v))
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, g.cliques(&f.vectors[0], out))
	n, e := out.rowCount()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	row, e := out.captureRow(0)
	satOK(t, e)
	if row.count != 3 {
		t.Fatal(row.count)
	}
	for i, v := range []satLiteral{0, 2, 4} {
		got, e := row.read(uint32(i))
		satOK(t, e)
		if got != v {
			t.Fatal(got, v)
		}
	}
}
func TestSATSolverFindMutexesActualBinaryWatchesAppendOutput(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	for _, p := range [][]satLiteral{{1, 3}, {1, 5}, {3, 5}} {
		satOK(t, s.MkClause(p))
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]satLiteral{6}))
	r, e := s.findMutexes([]satLiteral{0, 2, 4}, out)
	satOK(t, e)
	if r != lTrue {
		t.Fatal(r)
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	row, e := out.captureRow(1)
	satOK(t, e)
	if row.count != 3 {
		t.Fatal(row.count)
	}
}

func TestSATMaxCliquesUintSetResizeRefusalKeepsOriginalLogicalSize(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	satOK(t, g.seen1.insert(0))
	s.budget.FailAfter(0)
	e = g.seen1.insert(256)
	satProbingExpectedRefusal(t, s.propagation, e)
	n, er := g.seen1.words.size()
	satOK(t, er)
	if n != 1 {
		t.Error("source resize reserves allcapacity before publishing size", n)
	}
	s.budget.FailAfter(-1)
}

func TestSATMaxCliquesRetainsGoalSpecificSourceCacheAcrossCalls(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	satOK(t, g.addEdge(1, 3))
	satOK(t, g.addEdge(1, 5))
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.vector(s, 0, nil))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	for _, v := range []uint32{0, 2} {
		satOK(t, f.vectors[0].push(v))
	}
	satOK(t, g.cliques(&f.vectors[0], out))
	n, e := out.rowCount()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	satOK(t, out.SourceFinalize())
	satOK(t, f.vectors[0].write(1, 4))
	satOK(t, g.cliques(&f.vectors[0], out))
	n, e = out.rowCount()
	satOK(t, e)
	if n != 0 {
		t.Fatal("literal source goal-specific cache must remain retained", n)
	}
	satMutexGraphAudit(t, s, g, out)
	fresh, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, fresh.Close()) })
	satOK(t, fresh.addEdge(1, 3))
	satOK(t, fresh.addEdge(1, 5))
	satOK(t, fresh.cliques(&f.vectors[0], out))
	n, e = out.rowCount()
	satOK(t, e)
	if n != 1 {
		t.Fatal("fresh independent edge implication gives a different second query", n)
	}
}
func TestSATMaxCliquesWarmedSameQueryPhysicalPlateauAndScalarExtent(t *testing.T) {
	if got := satCliqueMaxPlusOne(^uint32(0), 0); got != 0 {
		t.Fatal("source UINT max+1 must wrap", got)
	}
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	satOK(t, g.addEdge(1, 3))
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.vector(s, 0, nil))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	satOK(t, f.vectors[0].push(0))
	satOK(t, f.vectors[0].push(2))
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	var used uint64
	for pass := 0; pass < 6; pass++ {
		satOK(t, g.cliques(&f.vectors[0], out))
		satMutexGraphAudit(t, s, g, out)
		if pass == 2 {
			used = s.budget.Used()
		}
		if pass > 2 && used != s.budget.Used() {
			t.Fatal("actual warmed graph failed plateau", used, s.budget.Used())
		}
		satOK(t, out.SourceFinalize())
	}
	copied := *g
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, g.Close())
	if e = g.addEdge(1, 3); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSATMaxCliquesWrappedExtentReachesSecondReserveBeforeSourceRead(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	satOK(t, g.next.reserveEmptyRows(2))
	old := g.next.raw
	s.budget.FailAfter(0)
	e = g.addEdge(^uint32(0), 0)
	satProbingExpectedRefusal(t, s.propagation, e)
	n, er := g.next.rowCount()
	satOK(t, er)
	if n != 2 || g.next.raw != old {
		t.Fatal("wrapped first reserve0 must leave old rows before second reserve refusal", n, g.next.raw, old)
	}
	satMutexGraphAudit(t, s, g)
	s.budget.FailAfter(-1)
}
func TestSATSolverFindMutexesHasNoCheckpointOrConsistencyPreflight(t *testing.T) {
	s, out := satMutexFixture(t)
	s.limit.cancel()
	s.propagation.inconsistent = true
	r, e := s.findMutexes([]satLiteral{0, 2, 4, 6}, out)
	satOK(t, e)
	if r != lTrue {
		t.Fatal(r)
	}
	satFullSolverRuntimeAudit(t, s, out)
	s.propagation.inconsistent = false
}

// Checked lexical-owner protocol: actual constructed locals remain named when
// a destructor stops. This does not claim an injected Budget free refusal.
func TestSATMaxCliquesStoppedLexicalContextRejectsOverwrite(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	satOK(t, g.vars.insert(2))
	satOK(t, g.clique.push(0))
	g.callActive = true
	before := s.budget.Used()
	e = g.addEdge(1, 3)
	if !errors.Is(e, arena.ErrClosed) {
		t.Error("stopped source locals could be overwritten", e)
	}
	satMutexGraphAudit(t, s, g)
	if s.budget.Used() != before {
		t.Error("rejected operation mutated physical owners")
	}
	satOK(t, g.Close())
}
