// Source: src/sat/sat_solver.cpp:3252–3636; src/test/cnf_backbones.cpp:248–268.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package g3

import (
	"fmt"
	"testing"
)

// Enumeration is a test oracle only. Production consequence discovery must
// execute actual propagation, assumption explanations, conflict/search and GC.
func satConsequenceModels(clauses [][]satLiteral, asms []satLiteral, n uint32) []uint32 {
	var models []uint32
	for bits := uint32(0); bits < uint32(1)<<n; bits++ {
		yes := true
		value := func(l satLiteral) bool { return (bits&(1<<uint32(l.variable())) != 0) != l.sign() }
		for _, l := range asms {
			yes = yes && value(l)
		}
		for _, row := range clauses {
			holds := false
			for _, l := range row {
				holds = holds || value(l)
			}
			yes = yes && holds
		}
		if yes {
			models = append(models, bits)
		}
	}
	return models
}
func TestSATSolverConsequencesIndependentAllTwoVariableBinaryCNFs(t *testing.T) {
	for mask := uint32(0); mask < 16; mask++ {
		for route := 0; route < 3; route++ {
			t.Run(fmt.Sprintf("%d/%d", mask, route), func(t *testing.T) {
				s, _, _ := satSolverCheckFixture(t)
				for i := 0; i < 2; i++ {
					_, e := s.MkVar(true, true)
					satOK(t, e)
				}
				var clauses [][]satLiteral
				for i := uint32(0); i < 4; i++ {
					if mask&(1<<i) != 0 {
						row := []satLiteral{satMakeLiteral(0, i&1 != 0), satMakeLiteral(1, i&2 != 0)}
						clauses = append(clauses, row)
						satOK(t, s.MkClause(append([]satLiteral(nil), row...)))
					}
				}
				var asms []satLiteral
				if route == 1 {
					asms = []satLiteral{0}
				}
				if route == 2 {
					asms = []satLiteral{1}
				}
				models := satConsequenceModels(clauses, asms, 2)
				out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
				satOK(t, e)
				t.Cleanup(func() { satOK(t, out.CloseChecked()) })
				got, e := s.getConsequences(asms, []satBoolVar{0, 1}, out)
				satOK(t, e)
				n, e := out.rowCount()
				satOK(t, e)
				if len(models) == 0 {
					if got != lFalse || n != 0 {
						t.Fatal("unsatisfiable source query", got, n)
					}
					return
				}
				if got != lTrue {
					t.Fatal("satisfiable source query", got)
				}
				seen := map[satLiteral]bool{}
				for i := uint32(0); i < n; i++ {
					row, e := out.captureRow(i)
					satOK(t, e)
					if row.count == 0 {
						t.Fatal("missing consequent")
					}
					lit, e := row.read(0)
					satOK(t, e)
					if lit.variable() > 1 || seen[lit] {
						t.Fatal("unexpected/duplicate consequent", lit)
					}
					seen[lit] = true
					var premises []satLiteral
					for j := uint32(1); j < row.count; j++ {
						premise, e := row.read(j)
						satOK(t, e)
						member := false
						for _, original := range asms {
							member = member || original == premise
						}
						if !member {
							t.Fatal("explanation is not a supplied assumption", premise, asms)
						}
						premises = append(premises, premise)
					}
					for _, bits := range satConsequenceModels(clauses, premises, 2) {
						if (bits&(1<<uint32(lit.variable())) != 0) == lit.sign() {
							t.Fatal("unentailed explanation", clauses, premises, lit, bits)
						}
					}
				}
				for variable := uint32(0); variable < 2; variable++ {
					bit := models[0]&(1<<variable) != 0
					fixed := true
					for _, model := range models {
						fixed = fixed && (model&(1<<variable) != 0) == bit
					}
					lit := satMakeLiteral(satBoolVar(variable), !bit)
					if seen[lit] != fixed {
						t.Fatal("missing or spurious fixed value", variable, models, seen)
					}
				}
			})
		}
	}
}
