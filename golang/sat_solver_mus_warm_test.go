// Source: src/sat/sat_mus.cpp; measured owner-local warm behavior.
// Licensed under the MIT license. Source author: Nikolaj Bjorner, 2014.
package g3

import "testing"

func TestSATSolverMUSWarmSourceLexicalWorkRetainsBoundedPhysicalGraph(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		s := satMUSFaultFixture(t, recursive)
		for i := 0; i < 8; i++ {
			_, e := satMUSFaultCall(s, recursive)
			satOK(t, e)
		}
		bytes, physical := s.budget.Used(), s.runtime.memory.Stats().TotalBytes
		allocations := testing.AllocsPerRun(32, func() {
			r, e := satMUSFaultCall(s, recursive)
			if e != nil {
				t.Fatal(e)
			}
			want := lFalse
			if recursive {
				want = lTrue
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
