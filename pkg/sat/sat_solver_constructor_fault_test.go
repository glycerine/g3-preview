package sat

// Source: src/sat/sat_solver.cpp:34–74; src/sat/sat_solver.h declared members.

import (
	"fmt"
	"testing"
)

func TestSATSolverFullConstructorEveryActualRefusalUnwindsRealMembers(t *testing.T) {
	failed, success := 0, 0
	for fault := int64(0); fault < 512; fault++ {
		ok := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			parent, ctx, p := satSolverCheckFixture(t)
			before := parent.budget.Used()
			rawBefore := parent.runtime.memory.Stats().TotalBytes
			paramsBefore := paramsBorrowedPhysicalBytes(ctx.Stats())
			parent.budget.FailAfter(fault)
			child, e := NewSATSolver(SolverConfig{ParamsContext: ctx, Params: p, RLimit: parent.Limit, Budget: parent.budget, Runtime: parent.runtime})
			if e != nil {
				satProbingExpectedRefusal(t, parent.propagation, e)
				failed++
				if child != nil || parent.runtime.liveSolvers != 1 {
					t.Fatal("ordinary ctor abort failed actual cleanup", child, parent.runtime.liveSolvers)
				}
				satFullSolverRuntimeAudit(t, parent)
				// Retained physical Runtime/parameter directories are legitimate real
				// capacity prefixes; every completed cold child and wrapper must refund.
				want := before - rawBefore - paramsBefore + parent.runtime.memory.Stats().TotalBytes + paramsBorrowedPhysicalBytes(ctx.Stats())
				if parent.budget.Used() != want {
					t.Fatal("anonymous cold constructor ownership", parent.budget.Used(), want)
				}
			} else {
				success++
				ok = true
				if child == nil || !child.constructed || parent.runtime.liveSolvers != 2 {
					t.Fatal("constructor completion")
				}
			}
			parent.budget.FailAfter(-1)
			if child != nil {
				satOK(t, child.Close())
			}
		})
		if ok {
			break
		}
	}
	if failed == 0 || success != 1 {
		t.Fatal(failed, success)
	}
	t.Log("actual source construction refusal horizon", failed, success)
}
