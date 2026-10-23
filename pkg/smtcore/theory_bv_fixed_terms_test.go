package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTBitVectorCompoundFixedEqualityAndNegatedBits(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, negated := range []bool{false, true} {
			t.Run(fmt.Sprint(proofs, "/", negated), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				solver, err := NewSMTSolver(m, nil, "QF_UFBV", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, solver.Close()) })
				// Exercise theory equality discovery without canonicalizing both
				// circuits into the same AST during preprocessing.
				solver.kernel.preprocess = false
				s, err := m.BitVectorSort(2)
				astTestMust(t, err)
				family, err := m.RegisterBitVectorPlugin()
				astTestMust(t, err)
				args := make([]ast.ASTHandle, 4)
				for i := range args {
					args[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("fixed-term-", i)), s)
					astTestMust(t, err)
				}
				var left, right ast.ASTHandle
				if negated {
					left, err = m.MkFamilyApp(family, 32, nil, args[:1])
					astTestMust(t, err)
					right = args[1]
					for i, value := range args[:2] {
						for bit := uint32(0); bit < 2; bit++ {
							atom, err := m.MkBitVectorBit2Bool(bit, value)
							astTestMust(t, err)
							if (i == 0) == (bit == 1) {
								atom, err = m.MkNot(atom)
								astTestMust(t, err)
							}
							astTestMust(t, solver.Assert(atom))
						}
					}
					status, err := solver.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("fixed-bit prefix", status)
					}
				} else {
					left, err = m.MkFamilyApp(family, 30, nil, args)
					astTestMust(t, err)
					ac, err := m.MkFamilyApp(family, 30, nil, []ast.ASTHandle{args[0], args[2]})
					astTestMust(t, err)
					bd, err := m.MkFamilyApp(family, 30, nil, []ast.ASTHandle{args[1], args[3]})
					astTestMust(t, err)
					right, err = m.MkFamilyApp(family, 30, nil, []ast.ASTHandle{ac, bd})
					astTestMust(t, err)
				}
				b, err := m.MkBoolSort()
				astTestMust(t, err)
				p := smtQuickPredicate(t, m, "fixed-term-p", s, b)
				positive, err := m.MkApp(p, []ast.ASTHandle{left})
				astTestMust(t, err)
				negative, err := m.MkApp(p, []ast.ASTHandle{right})
				astTestMust(t, err)
				negative, err = m.MkNot(negative)
				astTestMust(t, err)
				astTestMust(t, solver.Assert(positive))
				astTestMust(t, solver.Assert(negative))
				status, err := solver.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("equal compound values did not merge", status)
				}
				if proofs {
					proof, err := solver.Proof()
					astTestMust(t, err)
					fact, err := m.GetProofFact(proof)
					astTestMust(t, err)
					if !m.IsFalse(fact) {
						t.Fatal("compound equality conflict proof")
					}
				}
			})
		}
	}
}
