package sat

// Source: src/util/max_cliques.h; src/sat/sat_solver.cpp:3113-3150.

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSATSolverFindMutexesActualBinaryWatchesAppendOutput(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	for _, p := range [][]Literal{{1, 3}, {1, 5}, {3, 5}} {
		satOK(t, s.MkClause(p))
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{6}))
	r, e := s.findMutexes([]Literal{0, 2, 4}, out)
	satOK(t, e)
	if r != lbool.True {
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

func TestSATSolverFindMutexesHasNoCheckpointOrConsistencyPreflight(t *testing.T) {
	s, out := satMutexFixture(t)
	s.Limit.Cancel()
	s.propagation.inconsistent = true
	r, e := s.findMutexes([]Literal{0, 2, 4, 6}, out)
	satOK(t, e)
	if r != lbool.True {
		t.Fatal(r)
	}
	satFullSolverRuntimeAudit(t, s, out)
	s.propagation.inconsistent = false
}

// Checked lexical-owner protocol: actual constructed locals remain named when
// a destructor stops. This does not claim an injected Budget free refusal.
