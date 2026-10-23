package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArithmeticSharedEqualitySplitsScopedProofsAndModels(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(integer, "/", proofs), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				logic, kind := "QF_UFLRA", int32(0)
				if integer {
					logic, kind = "QF_UFLIA", 1
				}
				s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				astTestMust(t, s.SetProduceModels(true))
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				sort, err := m.MkFamilySort(family, kind, nil)
				astTestMust(t, err)
				var variables [3]ast.ASTHandle
				for i := range variables {
					variables[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("interface-", i)), sort)
					astTestMust(t, err)
				}
				x, y, z := variables[0], variables[1], variables[2]
				f, err := m.MkFuncDecl(astTestSymbol(t, m, "interface-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
				astTestMust(t, err)
				fx, err := m.MkApp(f, []ast.ASTHandle{x})
				astTestMust(t, err)
				fz, err := m.MkApp(f, []ast.ASTHandle{z})
				astTestMust(t, err)
				eq, err := m.MkEq(fx, fz)
				astTestMust(t, err)
				neq, err := m.MkNot(eq)
				astTestMust(t, err)
				// Retain the nonlinear-free equality cycle outside solver scopes.
				var cycle []ast.ASTHandle
				for _, pair := range [][2]ast.ASTHandle{{x, y}, {y, z}, {z, x}} {
					bound, err := m.MkFamilyApp(family, smtArithOpLE, nil, pair[:])
					astTestMust(t, err)
					cycle = append(cycle, bound)
				}
				for _, expr := range append(cycle, neq) {
					ref, err := m.NewRef(expr)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				}
				for repeat := 0; repeat < 2; repeat++ {
					astTestMust(t, s.Push())
					astTestMust(t, s.Assert(neq))
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("shared arithmetic split should permit distinct values", status)
					}
					model, err := s.AcquireModel()
					astTestMust(t, err)
					var values [2]string
					for i, term := range []ast.ASTHandle{x, z} {
						decl, err := m.GetDecl(term)
						astTestMust(t, err)
						value, err := model.ConstantInterpretation(decl)
						astTestMust(t, err)
						text, _, matched, err := m.ArithmeticNumeralText(value)
						astTestMust(t, err)
						if !matched {
							t.Fatal("missing model numeral")
						}
						values[i] = text
					}
					astTestMust(t, model.DecRef())
					if values[0] == values[1] {
						t.Fatal("UF disequality has equal arithmetic arguments in returned model", values)
					}
					if s.kernel.internalizer.statistics.NumInterfaceEqs == 0 {
						t.Fatal("shared model-value equality was not split")
					}
					astTestMust(t, s.Push())
					for _, formula := range cycle {
						astTestMust(t, s.Assert(formula))
					}
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.False {
						t.Fatal("arithmetic equality cycle and UF disequality", status)
					}
					if proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("interface-equality conflict proof")
						}
					}
					astTestMust(t, s.Pop(1))
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("interface equality decision survived pop", status)
					}
					astTestMust(t, s.Pop(1))
					theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
					if len(theory.assumeEqualityCandidates) != 0 || theory.assumeEqualityHead != 0 {
						t.Fatal("shared equality candidates survived scope pop")
					}
				}
			})
		}
	}
}
