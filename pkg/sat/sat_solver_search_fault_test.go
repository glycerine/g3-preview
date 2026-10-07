// Source: src/sat/sat_solver.cpp:723–802,993–1034,1191–1249.
package sat

import (
	"fmt"
	"testing"
)

func TestSATSolverDefaultCheckEveryActualRefusalKeepsNamedRuntimeRoots(t *testing.T) {
	failed, success := 0, 0
	for fault := int64(0); fault < 1024; fault++ {
		ok := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 3; i++ {
				_, e := s.MkVar(true, true)
				satOK(t, e)
			}
			original := [][]Literal{{0, 2}, {1, 4}, {3, 5}}
			for _, c := range original {
				satOK(t, s.MkClause(append([]Literal(nil), c...)))
			}
			s.budget.FailAfter(fault)
			got, e := s.Check(nil)
			satFullSolverRuntimeAudit(t, s)
			// This binary-only source family has no main clause owners. Do not infer
			// ownership from occupied allocator slots or use a formula snapshot to repair.
			satFullSolverUserClauseAudit(t, s)
			if e != nil {
				satProbingExpectedRefusal(t, s.propagation, e)
				failed++
			} else {
				success++
				ok = true
				if got != satSolverFiniteCNFOracle(original, 3) {
					t.Fatal(got)
				}
			}
			s.budget.FailAfter(-1)
			if e = s.Close(); e != nil {
				if s.closeStage != 1 || s.closeSourceCause == nil {
					t.Fatal("unfinished owning release", e, s.closeStage)
				}
				satFullSolverRuntimeAudit(t, s)
				satOK(t, s.CleanupStopped())
			}
		})
		if ok {
			break
		}
	}
	if failed == 0 || success != 1 {
		t.Fatal(failed, success)
	}
	t.Log("actual source Check refusal horizon", failed, success)
}
