// Source: src/sat/sat_mus.cpp; measured owner-local warm behavior.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverMUSWarmSourceLexicalWorkRetainsBoundedPhysicalGraph(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		s := satMUSFaultFixture(t, recursive)
		// Caller-owned input is prepared before measuring the warmed solver.
		// A fresh slice literal in satMUSFaultCall escapes through Check.
		assumptions := []Literal{0, 2, 4}
		operation := func() (util.LBool, error) {
			if recursive {
				return satMUSFaultCall(s, true)
			}
			return s.Check(assumptions)
		}
		for i := 0; i < 8; i++ {
			_, e := operation()
			satOK(t, e)
		}
		bytes, physical := s.budget.Used(), s.runtime.memory.Stats().TotalBytes
		allocations := testing.AllocsPerRun(32, func() {
			r, e := operation()
			if e != nil {
				t.Fatal(e)
			}
			want := util.LFalse
			if recursive {
				want = util.LTrue
			}
			if r != want {
				t.Fatal(r)
			}
		})
		if s.budget.Used() != bytes || s.runtime.memory.Stats().TotalBytes != physical || s.mus.frames != nil {
			t.Fatal("source local owners failed to plateau", recursive, bytes, s.budget.Used(), physical, s.runtime.memory.Stats())
		}
		if !recursive && allocations != 0 {
			t.Fatal("warmed linear MUS hidden allocation", allocations)
		}
		t.Log("recursive", recursive, "Go allocations/op", allocations, "physical Budget", bytes, "Runtime owner bytes", physical)
		satFullSolverRuntimeAudit(t, s)
	}
}
