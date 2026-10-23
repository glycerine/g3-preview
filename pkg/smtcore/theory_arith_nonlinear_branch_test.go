package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNonlinearIntegerBranchesScopedProofsAndModels(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			for _, tc := range []struct {
				name                  string
				lower, upper, product string
				want                  lbool.LBool
				roundLimit            uint32
			}{
				{"bounded-model", "1", "4", "6", lbool.True, 1024},
				{"bounded-conflict", "1", "4", "7", lbool.False, 1024},
				{"unbounded-zero-split", "", "", "1", lbool.True, 1024},
				{"lower-split", "0", "", "1", lbool.True, 1024},
				{"upper-split", "", "0", "1", lbool.True, 1024},
				{"round-limit", "1", "9", "35", lbool.Undef, 0},
				{"several-branches", "1", "9", "35", lbool.True, 1024},
			} {
				t.Run(fmt.Sprint(proofs, "/", enabled, "/", tc.name), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					s, err := NewSMTSolver(m, nil, "QF_NIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, s.Close()) })
					s.kernel.preprocess = false
					astTestMust(t, s.SetProduceModels(true))
					family, err := m.ArithmeticFamilyID()
					astTestMust(t, err)
					sort, err := m.MkFamilySort(family, 1, nil)
					astTestMust(t, err)
					x, err := m.MkConst(astTestSymbol(t, m, "integer-branch-x"), sort)
					astTestMust(t, err)
					y, err := m.MkConst(astTestSymbol(t, m, "integer-branch-y"), sort)
					astTestMust(t, err)
					var formulas []ast.ASTHandle
					for _, variable := range []ast.ASTHandle{x, y} {
						for _, bound := range []struct {
							text string
							op   int32
						}{{tc.lower, smtArithOpGE}, {tc.upper, smtArithOpLE}} {
							if bound.text == "" {
								continue
							}
							value, err := m.ArithmeticIntegerNumeralText(bound.text)
							astTestMust(t, err)
							formula, err := m.MkFamilyApp(family, bound.op, nil, []ast.ASTHandle{variable, value})
							astTestMust(t, err)
							formulas = append(formulas, formula)
						}
					}
					product, err := m.MkArithmeticMul(x, y)
					astTestMust(t, err)
					value, err := m.ArithmeticIntegerNumeralText(tc.product)
					astTestMust(t, err)
					equality, err := m.MkEq(product, value)
					astTestMust(t, err)
					formulas = append(formulas, equality)
					for _, formula := range formulas {
						ref, err := m.NewRef(formula)
						astTestMust(t, err)
						t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
					}
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
						config, err := s.kernel.smtParams.ArithmeticState()
						astTestMust(t, err)
						config.NLArithBranching = enabled
						config.NLArithGB = false
						config.NLArithRounds = tc.roundLimit
						astTestMust(t, theory.ConfigureArithmetic(config))
						for _, formula := range formulas {
							astTestMust(t, s.Assert(formula))
						}
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						want := tc.want
						if !enabled && tc.want != lbool.False {
							want = lbool.Undef
						}
						if status != want {
							t.Fatal("nonlinear integer split", status, want)
						}
						if enabled && status == lbool.True && theory.statistics.nonlinearBranches == 0 {
							t.Fatal("integer nonlinear branching was not exercised")
						}
						if !enabled && status == lbool.Undef {
							state := s.kernel.internalizer
							previous := len(state.booleanData)
							branched, err := theory.branchNonlinearIntegerVariable()
							astTestMust(t, err)
							preferred := false
							for variable := previous; variable < len(state.booleanData); variable++ {
								if state.booleanData[variable].TryTrueFirst() {
									assignment, err := state.Assignment(NewSMTLiteral(int32(variable), false))
									astTestMust(t, err)
									if assignment != lbool.Undef {
										t.Fatal("nonlinear case split was asserted as a unit")
									}
									preferred = true
								}
							}
							if !branched || !preferred {
								t.Fatal("nonlinear case split lacks the source true-first preference")
							}
						}
						if status == lbool.False && proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							fact, err := m.GetProofFact(proof)
							astTestMust(t, err)
							if !m.IsFalse(fact) {
								t.Fatal("integer nonlinear split proof")
							}
						}
						if status == lbool.True {
							model, err := s.AcquireModel()
							astTestMust(t, err)
							actual := big.NewRat(1, 1)
							for _, variable := range []ast.ASTHandle{x, y} {
								decl, err := m.GetDecl(variable)
								astTestMust(t, err)
								value, err := model.ConstantInterpretation(decl)
								astTestMust(t, err)
								text, integer, numeral, err := m.ArithmeticNumeralText(value)
								astTestMust(t, err)
								valueRat, ok := new(big.Rat).SetString(text)
								if !integer || !numeral || !ok || !valueRat.IsInt() {
									t.Fatal("non-integral nonlinear model", text)
								}
								actual.Mul(actual, valueRat)
							}
							if actual.RatString() != tc.product {
								t.Fatal("nonlinear model product", actual, tc.product)
							}
							astTestMust(t, model.DecRef())
						}
						astTestMust(t, s.Pop(1))
						if theory.nonlinearRounds != 0 {
							t.Fatal("nonlinear round budget survived scope pop", theory.nonlinearRounds)
						}
						if len(theory.derivedBounds) != 0 || theory.nonlinearStrategy != 0 {
							t.Fatal("nonlinear strategy facts survived scope pop")
						}
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.True {
							t.Fatal("integer nonlinear branch survived scope pop", status)
						}
					}
				})
			}
		}
	}
}
