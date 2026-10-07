// Source: src/sat/sat_mus.cpp:52–209; actual allocation-prefix ownership campaign.
package sat

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func satMUSFaultFixture(t *testing.T, recursive bool) *Solver {
	s, ctx, params := satSolverCheckFixture(t)
	n := 3
	if recursive {
		n = 5
	}
	var clause, assumptions []Literal
	for i := 0; i < n; i++ {
		v, e := s.MkVar(true, true)
		satOK(t, e)
		clause = append(clause, MakeLiteral(v, true))
		assumptions = append(assumptions, MakeLiteral(v, false))
	}
	satOK(t, s.MkClause(clause))
	if recursive {
		// Direct source mus2 fixture uses the actual solver-produced five-literal
		// core. Dispatch threshold >64 is independently covered by the65 fixture.
		r, e := s.Check(assumptions)
		satOK(t, e)
		if r != util.LFalse {
			t.Fatal(r)
		}
	} else {
		key := "core.minimize"
		satOK(t, ctx.SetBoolName(&params, &key, true))
		satOK(t, s.UpdateParams(&params))
	}
	return s
}
func satMUSFaultCall(s *Solver, recursive bool) (util.LBool, error) {
	if recursive {
		// Reproduce the operator's completed is_active flet before invoking
		// this private callee on a smaller independently valid source core.
		old := s.mus.isActive
		s.mus.isActive = true
		defer func() { s.mus.isActive = old }()
		return s.mus.mus2()
	}
	return s.Check([]Literal{0, 2, 4})
}
func TestSATSolverMUSEveryActualRefusalPreservesNamedPhysicalRoots(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		t.Run(fmt.Sprint(recursive), func(t *testing.T) {
			horizon := int64(0)
			t.Run("calibrate", func(t *testing.T) {
				s := satMUSFaultFixture(t, recursive)
				const start = int64(1 << 50)
				s.budget.FailAfter(start)
				r, e := satMUSFaultCall(s, recursive)
				satOK(t, e)
				want := util.LFalse
				if recursive {
					want = util.LTrue
				}
				if r != want {
					t.Fatal(r, want)
				}
				horizon = start - testFieldBySourceName(reflect.ValueOf(s.budget).Elem(), "remaining").Int()
				satFullSolverRuntimeAudit(t, s)
				satFullSolverUserClauseAudit(t, s)
				s.budget.FailAfter(-1)
			})
			failures, success := 0, 0
			for fault := int64(0); fault <= horizon; fault++ {
				t.Run(fmt.Sprint(fault), func(t *testing.T) {
					s := satMUSFaultFixture(t, recursive)
					restart := s.config.state.RestartMax
					s.budget.FailAfter(fault)
					result, e := satMUSFaultCall(s, recursive)
					satFullSolverRuntimeAudit(t, s)
					satFullSolverUserClauseAudit(t, s)
					if s.mus.isActive || s.config.state.RestartMax != restart {
						t.Fatal("source scoped fields not restored")
					}
					if e != nil {
						satProbingExpectedRefusal(t, s.propagation, e)
						failures++
					} else {
						success++
						want := util.LFalse
						if recursive {
							want = util.LTrue
						}
						if result != want {
							t.Fatal(result, want)
						}
						count, e := s.conflict.core.Size()
						satOK(t, e)
						wantCount := uint32(3)
						if recursive {
							wantCount = 5
						}
						if count != wantCount {
							t.Fatal("necessary original assumptions", count)
						}
					}
					s.budget.FailAfter(-1)
					if e = s.Close(); e != nil {
						if s.closeStage != 1 || s.closeSourceCause == nil {
							t.Fatal("unfinished true owner", e, s.closeStage)
						}
						satFullSolverRuntimeAudit(t, s)
						satOK(t, s.CleanupStopped())
					}
				})
			}
			if failures == 0 || success != 1 || int64(failures) != horizon {
				t.Fatal("actual calibrated refusal horizon", horizon, failures, success)
			}
			t.Log("actual Budget refusals including explicit cold lexical headers", horizon, "first success", success)
		})
	}
}
