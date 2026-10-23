package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTIntegerEqualityBranchWitnessScopesProofsAndModels(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			for _, rhs := range []int64{1, 5} {
				t.Run(fmt.Sprint(enabled, "/", proofs, "/", rhs), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					s, err := NewSMTSolver(m, nil, "QF_LIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, s.Close()) })
					smtDisableAutoConfig(t, s)
					s.kernel.preprocess = false
					astTestMust(t, s.SetProduceModels(true))
					family, err := m.ArithmeticFamilyID()
					astTestMust(t, err)
					sort, err := m.MkFamilySort(family, 1, nil)
					astTestMust(t, err)
					x, err := m.MkConst(astTestSymbol(t, m, "eq-branch-x"), sort)
					astTestMust(t, err)
					y, err := m.MkConst(astTestSymbol(t, m, "eq-branch-y"), sort)
					astTestMust(t, err)
					num := func(v int64) ast.ASTHandle {
						h, err := m.ArithmeticNumeralRat(big.NewRat(v, 1), true)
						astTestMust(t, err)
						return h
					}
					app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
						h, err := m.MkFamilyApp(family, op, nil, args)
						astTestMust(t, err)
						return h
					}
					eq, err := m.MkEq(app(smtArithOpAdd, app(smtArithOpMul, num(3), x), app(smtArithOpMul, num(2), y)), num(rhs))
					astTestMust(t, err)
					formulas := []ast.ASTHandle{eq, app(smtArithOpGE, x, num(0)), app(smtArithOpLE, x, num(2)), app(smtArithOpGE, y, num(0))}
					for _, formula := range formulas {
						ref, err := m.NewRef(formula)
						astTestMust(t, err)
						t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
					}
					theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
					config, err := s.kernel.smtParams.ArithmeticState()
					astTestMust(t, err)
					config.IntEqBranching = enabled
					astTestMust(t, theory.ConfigureArithmetic(config))
					before := theory.statistics.integerEqualityBranches
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						for _, formula := range formulas {
							astTestMust(t, s.Assert(formula))
						}
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						want := lbool.True
						if rhs == 1 {
							want = lbool.False
						}
						if status != want {
							t.Fatal("integer equality witness split", status, want)
						}
						if rhs == 1 && proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							fact, err := m.GetProofFact(proof)
							astTestMust(t, err)
							if !m.IsFalse(fact) {
								t.Fatal("integer equality split proof fact")
							}
						}
						if rhs == 5 {
							model, err := s.AcquireModel()
							astTestMust(t, err)
							var values []*big.Rat
							for _, term := range []ast.ASTHandle{x, y} {
								decl, err := m.GetDecl(term)
								astTestMust(t, err)
								value, err := model.ConstantInterpretation(decl)
								astTestMust(t, err)
								text, _, matched, err := m.ArithmeticNumeralText(value)
								astTestMust(t, err)
								point, ok := new(big.Rat).SetString(text)
								if !matched || !ok {
									t.Fatal(text)
								}
								values = append(values, point)
							}
							astTestMust(t, model.DecRef())
							if !values[0].IsInt() || !values[1].IsInt() || values[0].Sign() < 0 || values[0].Cmp(big.NewRat(2, 1)) > 0 || values[1].Sign() < 0 || new(big.Rat).Add(new(big.Rat).Mul(big.NewRat(3, 1), values[0]), new(big.Rat).Mul(big.NewRat(2, 1), values[1])).Cmp(big.NewRat(5, 1)) != 0 {
								t.Fatal("equality witness split removed valid model", values)
							}
						}
						astTestMust(t, s.Pop(1))
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.True {
							t.Fatal("integer equality split survived pop", status)
						}
					}
					if enabled && theory.statistics.integerEqualityBranches == before {
						t.Fatal("enabled equality witness path was not exercised")
					}
					if !enabled && theory.statistics.integerEqualityBranches != before {
						t.Fatal("disabled equality witness path was exercised")
					}
				})
			}
		}
	}
}
