// Source: src/util/vector.h:45–201; actual nested output ownership and lifetime.
package sat

import (
	"testing"
)

func TestSATConsequenceRowsCapturedInnerPayloadSurvivesOuterMoveAndDiesAtClose(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{0, 2}))
	oldOuter := out.storage().StorageState().Raw
	row, e := out.captureRow(0)
	satOK(t, e)
	satOK(t, out.appendLiterals([]Literal{4}))
	satOK(t, out.appendLiterals([]Literal{6}))
	if out.storage().StorageState().Raw == oldOuter {
		t.Fatal("outer did not relocate")
	}
	if _, e = s.runtime.memory.Read(oldOuter, 8); e == nil {
		t.Fatal("captured outer still alive")
	}
	for i, want := range []Literal{0, 2} {
		got, e := row.read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.CloseChecked())
	if _, e = row.read(0); e == nil {
		t.Fatal("captured inner still alive after actual owner close")
	}
}
