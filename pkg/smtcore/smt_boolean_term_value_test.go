package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTBooleanTermTruthValueCongruenceAndProof(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, value := range []bool{false, true} {
			for _, late := range []bool{false, true} {
				m, solver := smtAssertedNNFFixture(t, proofs)
				solver.kernel.preprocess = false
				b, err := m.MkBoolSort()
				astTestMust(t, err)
				a, err := m.MkConst(astTestSymbol(t, m, "truth-term-a"), b)
				astTestMust(t, err)
				f := smtQuickPredicate(t, m, "truth-term-f", b, b)
				constant, err := m.MkFalse()
				astTestMust(t, err)
				assigned := a
				if value {
					constant, err = m.MkTrue()
				} else {
					assigned, err = m.MkNot(a)
				}
				astTestMust(t, err)
				fa, err := m.MkApp(f, []ast.ASTHandle{a})
				astTestMust(t, err)
				fc, err := m.MkApp(f, []ast.ASTHandle{constant})
				astTestMust(t, err)
				notFC, err := m.MkNot(fc)
				astTestMust(t, err)
				formulas := []ast.ASTHandle{assigned, fa, notFC}
				if late {
					formulas = []ast.ASTHandle{fa, notFC, assigned}
				}
				for _, formula := range formulas {
					astTestMust(t, solver.Assert(formula))
				}
				status, err := solver.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("truth assignment did not imply function congruence", proofs, value, late, status)
				}
				if proofs {
					proof, err := solver.Proof()
					astTestMust(t, err)
					fact, err := m.GetProofFact(proof)
					astTestMust(t, err)
					if !m.IsFalse(fact) {
						t.Fatal("Boolean term congruence lost its proof", value)
					}
				}
			}
		}
	}
}
