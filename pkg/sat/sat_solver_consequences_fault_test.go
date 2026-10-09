// Source: src/sat/sat_solver.cpp:3252–3636; actual same-Runtime publication prefixes.
package sat

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"
)

func satConsequenceFaultFixture(t *testing.T, route int) (*Solver, *satSolverLiteralRows) {
	s, ctx, p := satSolverCheckFixture(t)
	for i := 0; i < 5; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	// Binary, ternary and nary antecedents produce the same independent a=>e
	// fixed-value result, through different actual propagation reason shapes.
	for _, row := range [][]Literal{{1, 2}, {1, 3, 4}, {1, 3, 5, 6}, {7, 8}} {
		satOK(t, s.MkClause(row))
	}
	if route == 1 {
		key := "restart.max"
		satOK(t, ctx.SetUIntName(&p, &key, 3))
		satOK(t, s.UpdateParams(&p))
	}
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{0}))
	return s, out
}
func TestSATSolverConsequencesEveryActualRefusalPreservesNamedOwners(t *testing.T) {
	for route := 0; route < 2; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			horizon := int64(0)
			t.Run("calibrate", func(t *testing.T) {
				s, out := satConsequenceFaultFixture(t, route)
				const start = int64(1 << 50)
				s.budget.FailAfter(start)
				r, e := s.getConsequences([]Literal{0}, []BoolVar{1, 2, 3, 4}, out)
				satOK(t, e)
				if r != lbool.True {
					t.Fatal(r)
				}
				horizon = start - testFieldBySourceName(reflect.ValueOf(s.budget).Elem(), "remaining").Int()
				satFullSolverRuntimeAudit(t, s, out)
				satFullSolverUserClauseAudit(t, s)
				s.budget.FailAfter(-1)
			})
			failures, success := 0, 0
			for fault := int64(0); fault <= horizon; fault++ {
				t.Run(fmt.Sprint(fault), func(t *testing.T) {
					s, out := satConsequenceFaultFixture(t, route)
					s.budget.FailAfter(fault)
					r, e := s.getConsequences([]Literal{0}, []BoolVar{1, 2, 3, 4}, out)
					satFullSolverRuntimeAudit(t, s, out)
					satFullSolverUserClauseAudit(t, s)
					// Even an interrupted row construction cannot alter the caller's first row.
					slot, er := s.runtime.memory.ReadUint32(out.storageState().Raw, 8)
					satOK(t, er)
					gen, er := s.runtime.memory.ReadUint32(out.storageState().Raw, 12)
					satOK(t, er)
					if len(out.storageState().OldRows) != 0 {
						// In a stopped move, the original row is owned by precisely one actual
						// current-prefix or old-suffix field; compare its payload below.
						row := out.storageState().Rows[0]
						if row.Block.Slot() != slot || row.Block.Generation() != gen {
							row = out.storageState().OldRows[0]
						}
						v, er := row.Read(0)
						satOK(t, er)
						if v != 0 {
							t.Fatal(v)
						}
					} else {
						v, er := out.storageState().Rows[0].Read(0)
						satOK(t, er)
						if v != 0 {
							t.Fatal(v)
						}
					}
					if e != nil {
						satProbingExpectedRefusal(t, s.propagation, e)
						failures++
					} else {
						success++
						if r != lbool.True {
							t.Fatal(r)
						}
					}
					s.budget.FailAfter(-1)
					satOK(t, out.CloseChecked())
					if er = s.Close(); er != nil {
						if s.closeStage != 1 || s.closeSourceCause == nil {
							t.Fatal(er, s.closeStage)
						}
						satOK(t, s.CleanupStopped())
					}
				})
			}
			if int64(failures) != horizon || success != 1 {
				t.Fatal(horizon, failures, success)
			}
			t.Log("actual allocation refusal horizon", horizon, "first success", success)
		})
	}
}
