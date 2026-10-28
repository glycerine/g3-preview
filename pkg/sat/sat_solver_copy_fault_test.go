package sat

// Source: src/sat/sat_solver.cpp:75-127. Actual destination refusal prefixes.

import (
	"fmt"
	"reflect"
	"testing"
)

func satCopyFaultFixture(t *testing.T) (*Solver, *Solver) {
	src, _, _ := satSolverCheckFixture(t)
	dst, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 5; i++ {
		_, e := src.MkVar(i%2 == 0, true)
		satOK(t, e)
	}
	satOK(t, src.MkClause([]Literal{8}))
	satOK(t, src.MkClause([]Literal{0, 2}))
	satOK(t, src.MkClause([]Literal{1, 3, 4, 6}))
	satOK(t, src.UserPush())
	return src, dst
}
func TestSATSolverCopyEveryActualDestinationRefusalRetainsPublishedGraph(t *testing.T) {
	horizon := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		src, dst := satCopyFaultFixture(t)
		const n = int64(1 << 50)
		dst.budget.FailAfter(n)
		satOK(t, dst.copyFrom(src))
		horizon = n - testFieldBySourceName(reflect.ValueOf(dst.budget).Elem(), "remaining").Int()
		satFullSolverRuntimeAudit(t, src)
		satFullSolverRuntimeAudit(t, dst)
		dst.budget.FailAfter(-1)
	})
	failures, success := int64(0), 0
	for ordinal := int64(0); ordinal <= horizon; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			src, dst := satCopyFaultFixture(t)
			srcUsed := src.budget.Used()
			srcBlocks := src.runtime.memory.Stats().LiveBlocks
			dst.budget.FailAfter(ordinal)
			e := dst.copyFrom(src)
			satFullSolverRuntimeAudit(t, src)
			satFullSolverRuntimeAudit(t, dst)
			satFullSolverUserClauseAudit(t, src)
			satFullSolverUserClauseAudit(t, dst)
			if srcUsed != src.budget.Used() || srcBlocks != src.runtime.memory.Stats().LiveBlocks {
				t.Fatal("source ownership changed", srcUsed, src.budget.Used(), srcBlocks, src.runtime.memory.Stats().LiveBlocks)
			}
			if e != nil {
				satProbingExpectedRefusal(t, dst.propagation, e)
				failures++
			} else {
				n, e := dst.Variables.numVarsChecked()
				satOK(t, e)
				if n != 6 || len(dst.Lifecycle.Original) != 1 {
					t.Fatal(n, len(dst.Lifecycle.Original))
				}
				success++
			}
			dst.budget.FailAfter(-1)
			satOK(t, dst.cleanupConsequences())
			satFullSolverRuntimeAudit(t, dst)
			if er := dst.Close(); er != nil {
				if er != ErrSATAssertion || dst.closeStage != 1 || dst.closeSourceCause != er {
					t.Fatal("actual stopped destructor invariant", er, dst.closeStage, dst.closeSourceCause)
				}
				satFullSolverRuntimeAudit(t, dst)
				satOK(t, dst.CleanupStopped())
				if dst.closeSourceCause != er || !dst.closeGuardAcknowledged {
					t.Fatal("lost immutable source stop history")
				}
			}
		})
	}
	if failures != horizon || success != 1 {
		t.Fatal(horizon, failures, success)
	}
	t.Log("actual destination reserve horizon", horizon)
}
