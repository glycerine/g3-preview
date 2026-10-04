// Source: src/sat/sat_solver.cpp:3157-3249. Actual allocation and owner prefixes.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package g3

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func satChunkFixture(t *testing.T, multi bool) (*satSolver, *satSolverRuntimeUintColumn, *satSolverLiteralRows) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	if multi {
		satOK(t, s.MkClause([]satLiteral{0, 2}))
	} else {
		satOK(t, s.MkClause([]satLiteral{0}))
	}
	r, e := s.Check(nil)
	satOK(t, e)
	if r != lTrue {
		t.Fatal(r)
	}
	model := &satSolverRuntimeUintColumn{}
	model.bind(s.runtime.memory)
	for i := 0; i < 2; i++ {
		satOK(t, model.push(uint32(lTrue)))
	}
	t.Cleanup(func() { satOK(t, model.finalize()) })
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]satLiteral{8}))
	return s, model, out
}
func satChunkAudit(t *testing.T, s *satSolver, m *satSolverRuntimeUintColumn, out *satSolverLiteralRows) {
	roots := satFullSolverRuntimeRoots(s, out)
	roots = append(roots, satMutexColumnRoots(m)...)
	satScalarAuditRoots(t, s.variables, satMutexNonzero(roots)...)
	satFullSolverUserClauseAudit(t, s)
}
func TestSATConsequenceChunkEveryActualRefusalRetainsLocalAndOutputRoots(t *testing.T) {
	for route := 0; route < 2; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			horizon := int64(0)
			t.Run("calibrate", func(t *testing.T) {
				s, m, out := satChunkFixture(t, route == 1)
				const n = int64(1 << 50)
				s.budget.FailAfter(n)
				r, e := s.coreChunking(m, satConsequenceInput{variables: []satBoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, 2)
				satOK(t, e)
				if r != lTrue {
					t.Fatal(r)
				}
				horizon = n - reflect.ValueOf(s.budget).Elem().FieldByName("remaining").Int()
				satChunkAudit(t, s, m, out)
				s.budget.FailAfter(-1)
			})
			failures, success := int64(0), 0
			for ordinal := int64(0); ordinal <= horizon; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					s, m, out := satChunkFixture(t, route == 1)
					s.budget.FailAfter(ordinal)
					r, e := s.coreChunking(m, satConsequenceInput{variables: []satBoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, 2)
					satChunkAudit(t, s, m, out)
					row := &out.rows[0]
					v, e2 := row.read(0)
					satOK(t, e2)
					n, e2 := row.size()
					satOK(t, e2)
					if v != 8 || n != 1 {
						t.Fatal(v, n)
					}
					if e != nil {
						satProbingExpectedRefusal(t, s.propagation, e)
						failures++
					} else {
						if r != lTrue {
							t.Fatal(r)
						}
						success++
					}
					s.budget.FailAfter(-1)
					satOK(t, s.cleanupConsequences())
					satChunkAudit(t, s, m, out)
				})
			}
			if failures != horizon || success != 1 {
				t.Fatal(horizon, failures, success)
			}
			t.Log("actual horizon", horizon)
		})
	}
}
func TestSATConsequenceBruteForceRealCancellationPreservesPrefix(t *testing.T) {
	s, m, out := satChunkFixture(t, false)
	s.limit.cancel()
	e := s.bruteForceConsequences(satConsequenceLiterals(nil), satConsequenceLiterals([]satLiteral{0, 2}), out)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	satChunkAudit(t, s, m, out)
	n, e := out.rowCount()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	row, e := out.captureRow(1)
	satOK(t, e)
	lit, e := row.read(0)
	satOK(t, e)
	if row.count != 1 || lit != 1 {
		t.Fatal(row.count, lit)
	}
}
