// Source: src/sat/sat_solver.cpp:3252–3636; src/test/get_consequences.cpp:30–51.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// This ports the Boolean implication input from source test1 into the actual
// low-level SAT solver. Its BV/AST/FD orchestration and printing remain separate
// prerequisites; these assertions do not claim that registered test is complete.
func TestSATSolverConsequencesSourceBooleanImplicationsAppendToOwnedOutput(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{1, 2})) // a => b
	satOK(t, s.MkClause([]Literal{3, 4})) // b => c
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	// Source consequence output is append-only, including a caller-owned row
	// which this query neither interprets nor removes.
	satOK(t, out.appendLiterals([]Literal{6}))
	got, e := s.getConsequences([]Literal{0}, []BoolVar{1, 2, 3}, out)
	satOK(t, e)
	if got != util.LTrue {
		t.Fatal(got)
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 3 {
		t.Fatalf("existing row plus two entailed values: %d", n)
	}
	row, e := out.captureRow(0)
	satOK(t, e)
	v, e := row.read(0)
	satOK(t, e)
	if v != 6 || row.count != 1 {
		t.Fatal("caller prefix changed", v, row.count)
	}
	seen := [4]bool{}
	for i := uint32(1); i < n; i++ {
		row, e = out.captureRow(i)
		satOK(t, e)
		if row.count != 2 {
			t.Fatal("one consequent plus its actual assumption", row.count)
		}
		v, e = row.read(0)
		satOK(t, e)
		premise, e := row.read(1)
		satOK(t, e)
		if (v != 2 && v != 4) || premise != 0 {
			t.Fatal("source explanation", v, premise)
		}
		seen[v.variable()] = true
	}
	if !seen[1] || !seen[2] || seen[3] {
		t.Fatal(seen)
	}
}
