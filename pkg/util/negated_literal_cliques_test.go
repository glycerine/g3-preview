// Source: src/util/max_cliques.h; src/sat/sat_solver.cpp:3113-3150.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestNegatedLiteralCliquesNegatedLiteralPartitionAndSourceCache(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	for _, p := range [][2]uint32{{1, 3}, {1, 5}, {3, 5}} {
		memoryRuntimeRequire(t, g.AddEdge(p[0], p[1]))
	}
	ps := cliqueTestInput(t, m)
	for _, v := range []uint32{0, 2, 4} {
		memoryRuntimeRequire(t, ps.Push(v))
	}
	out, e := NewRuntimeUnsignedRows(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
	memoryRuntimeRequire(t, g.Cliques(ps, out))
	n, e := out.RowCount()
	memoryRuntimeRequire(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	row, e := cliqueTestCapture(out, 0)
	memoryRuntimeRequire(t, e)
	if row.count != 3 {
		t.Fatal(row.count)
	}
	for i, v := range []uint32{0, 2, 4} {
		got, e := row.read(uint32(i))
		memoryRuntimeRequire(t, e)
		if got != v {
			t.Fatal(got, v)
		}
	}
}
func TestNegatedLiteralCliquesUintSetResizeRefusalKeepsOriginalLogicalSize(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	memoryRuntimeRequire(t, g.seen1.insert(0))
	b.FailAfter(0)
	e = g.seen1.insert(256)
	cliqueTestRefusal(t, e)
	n, er := g.seen1.words.Size()
	memoryRuntimeRequire(t, er)
	if n != 1 {
		t.Error("source resize reserves allcapacity before publishing size", n)
	}
	b.FailAfter(-1)
}

func TestNegatedLiteralCliquesRetainsGoalSpecificSourceCacheAcrossCalls(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	memoryRuntimeRequire(t, g.AddEdge(1, 3))
	memoryRuntimeRequire(t, g.AddEdge(1, 5))
	ps := cliqueTestInput(t, m)
	out, e := NewRuntimeUnsignedRows(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
	for _, v := range []uint32{0, 2} {
		memoryRuntimeRequire(t, ps.Push(v))
	}
	memoryRuntimeRequire(t, g.Cliques(ps, out))
	n, e := out.RowCount()
	memoryRuntimeRequire(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	memoryRuntimeRequire(t, out.SourceFinalize())
	memoryRuntimeRequire(t, ps.Write(1, 4))
	memoryRuntimeRequire(t, g.Cliques(ps, out))
	n, e = out.RowCount()
	memoryRuntimeRequire(t, e)
	if n != 0 {
		t.Fatal("literal source goal-specific cache must remain retained", n)
	}
	cliqueTestAudit(t, m, g, ps, out)
	fresh, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, fresh.Close()) })
	memoryRuntimeRequire(t, fresh.AddEdge(1, 3))
	memoryRuntimeRequire(t, fresh.AddEdge(1, 5))
	memoryRuntimeRequire(t, fresh.Cliques(ps, out))
	n, e = out.RowCount()
	memoryRuntimeRequire(t, e)
	if n != 1 {
		t.Fatal("fresh independent edge implication gives a different second query", n)
	}
}
func TestNegatedLiteralCliquesWarmedSameQueryPhysicalPlateauAndScalarExtent(t *testing.T) {
	if got := cliqueMaxPlusOne(^uint32(0), 0); got != 0 {
		t.Fatal("source UINT max+1 must wrap", got)
	}
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	memoryRuntimeRequire(t, g.AddEdge(1, 3))
	ps := cliqueTestInput(t, m)
	memoryRuntimeRequire(t, ps.Push(0))
	memoryRuntimeRequire(t, ps.Push(2))
	out, e := NewRuntimeUnsignedRows(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
	var used uint64
	for pass := 0; pass < 6; pass++ {
		memoryRuntimeRequire(t, g.Cliques(ps, out))
		cliqueTestAudit(t, m, g, ps, out)
		if pass == 2 {
			used = b.Used()
		}
		if pass > 2 && used != b.Used() {
			t.Fatal("actual warmed graph failed plateau", used, b.Used())
		}
		memoryRuntimeRequire(t, out.SourceFinalize())
	}
	copied := *g
	if e = copied.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	memoryRuntimeRequire(t, g.Close())
	if e = g.AddEdge(1, 3); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestNegatedLiteralCliquesWrappedExtentReachesSecondReserveBeforeSourceRead(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	memoryRuntimeRequire(t, g.next.ReserveEmptyRows(2))
	old := g.next.raw
	b.FailAfter(0)
	e = g.AddEdge(^uint32(0), 0)
	cliqueTestRefusal(t, e)
	n, er := g.next.RowCount()
	memoryRuntimeRequire(t, er)
	if n != 2 || g.next.raw != old {
		t.Fatal("wrapped first reserve0 must leave old rows before second reserve refusal", n, g.next.raw, old)
	}
	cliqueTestAudit(t, m, g, nil)
	b.FailAfter(-1)
}
func TestNegatedLiteralCliquesStoppedLexicalContextRejectsOverwrite(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, g.vars.insert(2))
	memoryRuntimeRequire(t, g.clique.Push(0))
	g.callActive = true
	before := b.Used()
	e = g.AddEdge(1, 3)
	if !errors.Is(e, arena.ErrClosed) {
		t.Error("stopped source locals could be overwritten", e)
	}
	cliqueTestAudit(t, m, g, nil)
	if b.Used() != before {
		t.Error("rejected operation mutated physical owners")
	}
	memoryRuntimeRequire(t, g.Close())
}
