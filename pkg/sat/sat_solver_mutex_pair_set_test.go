package sat

// Source: src/sat/sat_solver.cpp:3119-3150; native literal-pair membership.

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverNativeMutexDedupGrowthWithoutBackingReservations(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	const variables = 32
	input := make([]Literal, variables)
	for i := range input {
		v, e := s.MkVar(true, true)
		satOK(t, e)
		input[i] = Literal(uint32(v) * 2)
	}
	// A complete graph has exactly one maximal clique: every input variable.
	// Ordinary clauses are collected in both source passes, so all 496 pairs
	// reach membership twice without relying on hash-table implementation details.
	for i, a := range input {
		for _, b := range input[i+1:] {
			satOK(t, s.MkClause([]Literal{a.Negated(), b.Negated()}))
		}
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{DefaultLiteral()}))
	before := s.budget.Used()
	// Permit only the actual cold frame header; all pair, graph and result
	// backing is native Go storage. Repeating the query must release the frame.
	for repeat := 0; repeat < 2; repeat++ {
		s.budget.FailAfter(1)
		result, e := s.findMutexes(input, out)
		s.budget.FailAfter(-1)
		satOK(t, e)
		if result != lbool.True || s.budget.Used() != before || s.mutexFrames != nil {
			t.Fatal(result, s.budget.Used(), before, s.mutexFrames)
		}
		if len(s.userBinClauses) != variables*(variables-1) {
			t.Fatal("actual duplicated source traversal", len(s.userBinClauses))
		}
		count, e := out.rowCount()
		satOK(t, e)
		if count != uint32(repeat+2) {
			t.Fatal("append one clique per query", count)
		}
		row, e := out.captureRow(count - 1)
		satOK(t, e)
		if row.count != variables {
			t.Fatal("complete graph clique", row.count)
		}
		seen := [variables]bool{}
		for i := uint32(0); i < row.count; i++ {
			l, e := row.read(i)
			satOK(t, e)
			if l.sign() || l.variable() >= variables || seen[l.variable()] {
				t.Fatal("unexpected or repeated clique vertex", l)
			}
			seen[l.variable()] = true
		}
		satFullSolverRuntimeAudit(t, s, out)
	}
	row, e := out.captureRow(0)
	satOK(t, e)
	prefix, e := row.read(0)
	satOK(t, e)
	if row.count != 1 || prefix != DefaultLiteral() {
		t.Fatal("caller output prefix changed", row.count, prefix)
	}
}
