// Source: src/sat/sat_solver.cpp:3157-3249.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATConsequenceChunkPruneAndLastDuplicateRemoval(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var model, lambda RuntimeUintColumn
	model.Bind(s.runtime.memory)
	lambda.Bind(s.runtime.memory)
	defer func() { satOK(t, lambda.Finalize()); satOK(t, model.Finalize()) }()
	for _, v := range []util.LBool{util.LTrue, util.LFalse, util.LUndef} {
		satOK(t, model.Push(uint32(int32(v))))
	}
	for _, v := range []uint32{0, 1, 2, 3, 4, 5} {
		satOK(t, lambda.Push(v))
	}
	satOK(t, satConsequencePruneUnfixed(&lambda, &model))
	n, e := lambda.Size()
	satOK(t, e)
	if n != 3 {
		t.Fatal(n)
	}
	seen := map[uint32]bool{}
	for i := uint32(0); i < n; i++ {
		v, e := lambda.Read(i)
		satOK(t, e)
		seen[v] = true
	}
	if !seen[0] || !seen[3] || !seen[4] {
		t.Fatal(seen)
	}
	satOK(t, lambda.Shrink(0))
	for _, v := range []uint32{2, 4, 2, 6} {
		satOK(t, lambda.Push(v))
	}
	satOK(t, satConsequenceBackRemove(&lambda, 2))
	for i, w := range []uint32{2, 4, 6} {
		v, e := lambda.Read(uint32(i))
		satOK(t, e)
		if v != w {
			t.Fatal(i, v, w)
		}
	}
}
func TestSATConsequenceChunkUsesActualCheckAndAppendsCore(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0}))
	r, e := s.Check(nil)
	satOK(t, e)
	if r != util.LTrue {
		t.Fatal(r)
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	defer func() { satOK(t, out.CloseChecked()) }()
	satOK(t, out.appendLiterals([]Literal{8}))
	r, e = s.coreChunking(&s.model, satConsequenceInput{variables: []BoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, 2)
	satOK(t, e)
	if r != util.LTrue {
		t.Fatal(r)
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	row, e := out.captureRow(1)
	satOK(t, e)
	if row.count != 1 {
		t.Fatal(row.count)
	}
	l, e := row.read(0)
	satOK(t, e)
	if l != 1 {
		t.Fatal("source core contains negated tested literal", l)
	}
	satFullSolverRuntimeAudit(t, s, out)
}

// Malformed source pointer/size input: earlier prune moves remain visible when
// a later model read reaches an absent payload; no prevalidation or repair.
func TestSATConsequenceChunkPruneReachedGuardPreservesSwapPrefix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var model, lambda RuntimeUintColumn
	model.Bind(s.runtime.memory)
	lambda.Bind(s.runtime.memory)
	defer func() { satOK(t, lambda.Finalize()); satOK(t, model.Finalize()) }()
	lf := util.LFalse
	satOK(t, model.Push(uint32(int32(lf))))
	for _, l := range []uint32{0, 4, 1} {
		satOK(t, lambda.Push(l))
	}
	e := satConsequencePruneUnfixed(&lambda, &model)
	if e == nil {
		t.Fatal("expected reached model bounds guard")
	}
	n, e2 := lambda.Size()
	satOK(t, e2)
	if n != 2 {
		t.Fatal(n)
	}
	first, e2 := lambda.Read(0)
	satOK(t, e2)
	if first != 1 {
		t.Fatal(first)
	}
}
func TestSATConsequenceChunkWarmedPhysicalPlateau(t *testing.T) {
	s, model, out := satChunkFixture(t, true)
	var used uint64
	var blocks uint32
	for pass := 0; pass < 6; pass++ {
		r, e := s.coreChunking(model, satConsequenceInput{variables: []BoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, 2)
		satOK(t, e)
		if r != util.LTrue {
			t.Fatal(r)
		}
		satChunkAudit(t, s, model, out)
		stats := s.runtime.memory.Stats()
		if pass == 2 {
			used = s.budget.Used()
			blocks = stats.LiveBlocks
		}
		if pass > 2 && (used != s.budget.Used() || blocks != stats.LiveBlocks) {
			t.Fatal("physical plateau", used, s.budget.Used(), blocks, stats.LiveBlocks)
		}
	}
}
