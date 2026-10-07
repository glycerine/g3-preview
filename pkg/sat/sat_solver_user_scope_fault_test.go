// Source: src/sat/sat_solver.cpp:2630–2645; src/util/id_gen.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverUserGCDeletionRefusalOwnsExactCompactionRanges(t *testing.T) {
	failed, success := 0, 0
	for fault := int64(0); fault < 64; fault++ {
		ok := false
		t.Run(string(rune('A'+fault)), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 5; i++ {
				_, e := s.MkVar(false, true)
				satOK(t, e)
			}
			// Alternating keep/delete causes actual owning ranges to differ from the
			// full pre-compaction vector after the first successful deletion.
			for _, c := range [][]Literal{{0, 2, 4, 6}, {0, 2, 4, 8}, {1, 2, 4, 7}, {1, 2, 4, 9}, {0, 3, 5, 8}} {
				_, e := s.Lifecycle.mkNaryClauseSpan(4, satSolverLiteralSpan{literals: c}, false)
				satOK(t, e)
			}
			s.budget.FailAfter(fault)
			e := s.gcLit(false, 8)
			satFullSolverRuntimeAudit(t, s)
			satFullSolverUserClauseAudit(t, s)
			if e != nil {
				satProbingExpectedRefusal(t, s.propagation, e)
				failed++
				if e = s.MkClause([]Literal{0, 2}); !errors.Is(e, arena.ErrClosed) {
					t.Fatal("stopped captured cohort permits mutation", e)
				}
			} else {
				success++
				ok = true
				if len(s.Lifecycle.Original) != 2 {
					t.Fatal(len(s.Lifecycle.Original))
				}
			}
			// Only after all immediate named scalar/physical checks may the explicit
			// caller disable refusal and invoke owning teardown of the real ranges.
			s.budget.FailAfter(-1)
			if e = s.Close(); e != nil {
				if !satStorageSourceDebug || s.closeStage != 1 || s.closeSourceCause == nil {
					t.Fatal("unexpected stopped source teardown", e, s.closeStage)
				}
				satFullSolverRuntimeAudit(t, s)
				satFullSolverUserClauseAudit(t, s)
				cause := s.closeSourceCause
				if again := s.Close(); again != cause {
					t.Fatal("source Close suppressed/replaced stopped cause", again, cause)
				}
				satOK(t, s.CleanupStopped())
				if s.closeSourceCause != cause || !s.closeGuardAcknowledged {
					t.Fatal("lost source stopped history")
				}
			}
			if s.runtime.liveSolvers != 0 || s.runtime.memory.Stats().LiveBlocks != 0 {
				t.Fatal("actual final owner roots", s.runtime.liveSolvers, s.runtime.memory.Stats())
			}
		})
		if ok {
			break
		}
	}
	if failed == 0 || success != 1 {
		t.Fatal(failed, success)
	}
	t.Log("actual refusal/success", failed, success)
}
