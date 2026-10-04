// Source: src/sat/sat_solver.cpp:75-127. Independent finite CNF copy fixtures.
// Licensed under the MIT license. Source authors: Leonardo de Moura (leonardo), 2011-05-21; Nikolaj Bjorner.
package g3

import (
	"fmt"
	"testing"
)

func TestSATSolverCopyIndependentBinaryCNFsAndDestinationUnion(t *testing.T) {
	for mask := uint32(0); mask < 16; mask++ {
		for route := 0; route < 2; route++ {
			t.Run(fmt.Sprintf("%d/%d", mask, route), func(t *testing.T) {
				src, _, _ := satSolverCheckFixture(t)
				dst, _, _ := satSolverCheckFixture(t)
				for i := 0; i < 2; i++ {
					_, e := src.MkVar(true, true)
					satOK(t, e)
				}
				var clauses [][]satLiteral
				if route == 1 {
					for i := 0; i < 2; i++ {
						_, e := dst.MkVar(true, true)
						satOK(t, e)
					}
					satOK(t, dst.MkClause([]satLiteral{0, 2}))
					clauses = append(clauses, []satLiteral{0, 2})
				}
				for i := uint32(0); i < 4; i++ {
					if mask&(1<<i) != 0 {
						r := []satLiteral{satMakeLiteral(0, i&1 != 0), satMakeLiteral(1, i&2 != 0)}
						clauses = append(clauses, r)
						satOK(t, src.MkClause(append([]satLiteral(nil), r...)))
					}
				}
				satOK(t, dst.copyFrom(src))
				satFullSolverRuntimeAudit(t, src)
				satFullSolverRuntimeAudit(t, dst)
				r, e := dst.Check(nil)
				satOK(t, e)
				want := lFalse
				if len(satConsequenceModels(clauses, nil, 2)) > 0 {
					want = lTrue
				}
				if r != want {
					t.Fatal(r, want, clauses)
				}
				satFullSolverRuntimeAudit(t, dst)
			})
		}
	}
}
func TestSATSolverCopyIndependentNaryAssignmentModels(t *testing.T) {
	for bits := uint32(0); bits < 16; bits++ {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			src, _, _ := satSolverCheckFixture(t)
			dst, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 4; i++ {
				_, e := src.MkVar(true, true)
				satOK(t, e)
			}
			clauses := [][]satLiteral{{0, 3, 4, 7}, {1, 2, 5, 6}}
			for _, r := range clauses {
				satOK(t, src.MkClause(append([]satLiteral(nil), r...)))
			}
			satOK(t, dst.copyFrom(src))
			var asms []satLiteral
			for v := uint32(0); v < 4; v++ {
				asms = append(asms, satMakeLiteral(satBoolVar(v), bits&(1<<v) == 0))
			}
			r, e := dst.Check(asms)
			satOK(t, e)
			want := lFalse
			if len(satConsequenceModels(clauses, asms, 4)) > 0 {
				want = lTrue
			}
			if r != want {
				t.Fatal(r, want, asms)
			}
			satFullSolverRuntimeAudit(t, src)
			satFullSolverRuntimeAudit(t, dst)
		})
	}
}
