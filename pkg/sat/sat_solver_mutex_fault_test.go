// Source: src/sat/sat_solver.cpp:3119-3150; src/util/max_cliques.h all named owners.
package sat

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"
)

func satMutexFixture(t *testing.T) (*Solver, *satSolverLiteralRows) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	for _, p := range [][]Literal{{1, 3}, {1, 5}, {3, 5}, {1, 7}, {3, 7}, {5, 7}} {
		satOK(t, s.MkClause(p))
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{8}))
	return s, out
}
func TestSATSolverFindMutexesEveryActualRefusalPreservesNamedOwners(t *testing.T) {
	horizon := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		s, out := satMutexFixture(t)
		const start = int64(1 << 50)
		s.budget.FailAfter(start)
		r, e := s.findMutexes([]Literal{0, 2, 4, 6}, out)
		satOK(t, e)
		if r != lbool.True {
			t.Fatal(r)
		}
		horizon = start - testFieldBySourceName(reflect.ValueOf(s.budget).Elem(), "remaining").Int()
		satFullSolverRuntimeAudit(t, s, out)
		s.budget.FailAfter(-1)
	})
	failures, success := 0, 0
	for fault := int64(0); fault <= horizon; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, out := satMutexFixture(t)
			before := s.budget.Used()
			s.budget.FailAfter(fault)
			r, e := s.findMutexes([]Literal{0, 2, 4, 6}, out)
			satFullSolverRuntimeAudit(t, s, out)
			satFullSolverUserClauseAudit(t, s)
			row, e2 := out.captureRow(0)
			satOK(t, e2)
			prefix, e2 := row.read(0)
			satOK(t, e2)
			if prefix != 8 {
				t.Fatal(prefix)
			}
			if e != nil {
				satProbingExpectedRefusal(t, s.propagation, e)
				failures++
			} else {
				if r != lbool.True {
					t.Fatal(r)
				}
				success++
			}
			s.budget.FailAfter(-1)
			satOK(t, s.cleanupMutexes())
			satFullSolverRuntimeAudit(t, s, out)
			if s.mutexFrames != nil {
				t.Fatal("retained completed context")
			}
			if e != nil && s.budget.Used() < before {
				t.Fatal("caller/member graph was discarded")
			}
		})
	}
	if int64(failures) != horizon || success != 1 {
		t.Fatal(horizon, failures, success)
	}
	t.Log("actual mutex refusal horizon", horizon)
}
