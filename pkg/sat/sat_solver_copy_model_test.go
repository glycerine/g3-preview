// Source: src/sat/sat_solver.cpp:75-127. Independent finite CNF copy fixtures.
package sat

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
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
				var clauses [][]Literal
				if route == 1 {
					for i := 0; i < 2; i++ {
						_, e := dst.MkVar(true, true)
						satOK(t, e)
					}
					satOK(t, dst.MkClause([]Literal{0, 2}))
					clauses = append(clauses, []Literal{0, 2})
				}
				for i := uint32(0); i < 4; i++ {
					if mask&(1<<i) != 0 {
						r := []Literal{MakeLiteral(0, i&1 != 0), MakeLiteral(1, i&2 != 0)}
						clauses = append(clauses, r)
						satOK(t, src.MkClause(append([]Literal(nil), r...)))
					}
				}
				satOK(t, dst.copyFrom(src))
				satFullSolverRuntimeAudit(t, src)
				satFullSolverRuntimeAudit(t, dst)
				r, e := dst.Check(nil)
				satOK(t, e)
				want := util.LFalse
				if len(satConsequenceModels(clauses, nil, 2)) > 0 {
					want = util.LTrue
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
			clauses := [][]Literal{{0, 3, 4, 7}, {1, 2, 5, 6}}
			for _, r := range clauses {
				satOK(t, src.MkClause(append([]Literal(nil), r...)))
			}
			satOK(t, dst.copyFrom(src))
			var asms []Literal
			for v := uint32(0); v < 4; v++ {
				asms = append(asms, MakeLiteral(BoolVar(v), bits&(1<<v) == 0))
			}
			r, e := dst.Check(asms)
			satOK(t, e)
			want := util.LFalse
			if len(satConsequenceModels(clauses, asms, 4)) > 0 {
				want = util.LTrue
			}
			if r != want {
				t.Fatal(r, want, asms)
			}
			satFullSolverRuntimeAudit(t, src)
			satFullSolverRuntimeAudit(t, dst)
		})
	}
}
