package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNonlinearMaxMinBoundsMovementAndScopedProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, test := range []struct{ conflict, shared bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
			conflict, shared := test.conflict, test.shared
			t.Run(fmt.Sprint(proofs, "/", conflict, "/", shared), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				logic := "QF_NRA"
				if shared {
					logic = "QF_UFNRA"
				}
				s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				astTestMust(t, s.SetProduceModels(true))
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				sort, err := m.MkFamilySort(family, 0, nil)
				astTestMust(t, err)
				variables := make([]ast.ASTHandle, 3)
				for i := range variables {
					variables[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("max-min-", i)), sort)
					astTestMust(t, err)
				}
				x, y, z := variables[0], variables[1], variables[2]
				app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
					h, err := m.MkFamilyApp(family, op, nil, args)
					astTestMust(t, err)
					return h
				}
				numeral := func(value int64) ast.ASTHandle {
					h, err := m.ArithmeticRealNumeralText(fmt.Sprint(value))
					astTestMust(t, err)
					return h
				}
				equality, err := m.MkEq(x, app(smtArithOpMul, numeral(2), y))
				astTestMust(t, err)
				product := app(smtArithOpMul, x, z)
				formulas := []ast.ASTHandle{equality, app(smtArithOpGE, y, numeral(1)), app(smtArithOpLE, y, numeral(2)), app(smtArithOpGE, z, numeral(-1)), app(smtArithOpLE, z, numeral(1))}
				if conflict {
					formulas = append(formulas, app(smtArithOpGT, product, numeral(4)))
				} else {
					equality, err := m.MkEq(product, numeral(-2))
					astTestMust(t, err)
					formulas = append(formulas, equality)
				}
				if shared {
					f, err := m.MkFuncDecl(astTestSymbol(t, m, "max-min-shared-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
					astTestMust(t, err)
					fz, err := m.MkApp(f, []ast.ASTHandle{z})
					astTestMust(t, err)
					bound, err := m.MkEq(fz, numeral(0))
					astTestMust(t, err)
					formulas = append(formulas, bound)
				}
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
					config.NLArithGB, config.NLArithBranching = false, false
					astTestMust(t, theory.ConfigureArithmetic(config))
					for _, formula := range formulas {
						astTestMust(t, s.Assert(formula))
					}
					before := theory.statistics.maxMin
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					want := lbool.True
					if conflict {
						want = lbool.False
					}
					if status != want {
						t.Fatal("max/min nonlinear result", status, want)
					}
					if theory.statistics.maxMin == before {
						t.Fatal("source max/min path was not exercised")
					}
					if theory.statistics.grobner != 0 || theory.statistics.nonlinearBranches != 0 {
						t.Fatal("max/min regression used disabled strategies")
					}
					if conflict && proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("max/min conflict proof")
						}
					}
					if !conflict {
						if shared && theory.liberalFinalCheck {
							t.Fatal("shared model movement skipped conservative revalidation")
						}
						if shared {
							old := theory.solver
							zColumn, err := theory.ensureVariable(z)
							astTestMust(t, err)
							yColumn, err := theory.ensureVariable(y)
							astTestMust(t, err)
							replay, status, err := theory.buildSimplex()
							astTestMust(t, err)
							if status != lbool.True {
								t.Fatal("logical replay", status)
							}
							zVariable, found := theory.theoryVariable(zColumn)
							if !found {
								t.Fatal("shared theory column disappeared")
							}
							zNode, err := theory.Enode(zVariable)
							astTestMust(t, err)
							isShared, err := theory.sharedArithmeticNode(zNode)
							astTestMust(t, err)
							if !isShared {
								t.Fatal("regression did not share the moved arithmetic term")
							}
							yValue, zValue := big.NewRat(1, 1), big.NewRat(-1, 1)
							before := replay.Value(zColumn)
							if before.Main.Cmp(zValue) == 0 && before.Epsilon.Sign() == 0 {
								yValue, zValue = big.NewRat(2, 1), big.NewRat(-1, 2)
							}
							candidate, status, err := theory.buildSimplexWithValues(map[uint32]*big.Rat{yColumn: yValue, zColumn: zValue})
							astTestMust(t, err)
							if status != lbool.True {
								t.Fatal("valid nonlinear point rejected", status)
							}
							theory.solver = replay
							published, err := theory.publishOptimizedArithmeticAssignment(candidate)
							astTestMust(t, err)
							if !published || theory.solver != candidate {
								t.Fatal("valid shared movement was discarded")
							}
							valid, err := theory.checkNonlinearMonomialAssignments()
							astTestMust(t, err)
							if !valid {
								t.Fatal("published nonlinear point violates products")
							}
							theory.solver = old
						}
						model, err := s.AcquireModel()
						astTestMust(t, err)
						values := make([]*big.Rat, 3)
						for i, variable := range variables {
							decl, err := m.GetDecl(variable)
							astTestMust(t, err)
							value, err := model.ConstantInterpretation(decl)
							astTestMust(t, err)
							text, _, matched, err := m.ArithmeticNumeralText(value)
							astTestMust(t, err)
							var ok bool
							values[i], ok = new(big.Rat).SetString(text)
							if !matched || !ok {
								t.Fatal("max/min model numeral", text)
							}
						}
						if values[0].Cmp(new(big.Rat).Mul(big.NewRat(2, 1), values[1])) != 0 || new(big.Rat).Mul(values[0], values[2]).Cmp(big.NewRat(-2, 1)) != 0 {
							t.Fatal("max/min model does not satisfy original equations", values)
						}
						astTestMust(t, model.DecRef())
					}
					astTestMust(t, s.Pop(1))
					if len(theory.derivedBounds) != 0 {
						t.Fatal("optimized bounds survived pop")
					}
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("optimized state survived pop", status)
					}
				}
			})
		}
	}
}
