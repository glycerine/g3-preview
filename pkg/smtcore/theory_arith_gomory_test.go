package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTIntegerBranchAndGomoryScopesProofsAndModels(t *testing.T) {
	for _, cuts := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			for _, unsat := range []bool{false, true} {
				t.Run(fmt.Sprint(cuts, "/", proofs, "/", unsat), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					logic := "QF_LIA"
					if unsat {
						logic = "QF_LIRA"
					}
					s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, s.Close()) })
					s.kernel.preprocess = false
					astTestMust(t, s.SetProduceModels(true))
					family, err := m.ArithmeticFamilyID()
					astTestMust(t, err)
					sort, err := m.MkFamilySort(family, 1, nil)
					astTestMust(t, err)
					x, err := m.MkConst(astTestSymbol(t, m, "gomory-x"), sort)
					astTestMust(t, err)
					y, err := m.MkConst(astTestSymbol(t, m, "gomory-y"), sort)
					astTestMust(t, err)
					z, err := m.MkConst(astTestSymbol(t, m, "gomory-unrelated"), sort)
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
					eq, err := m.MkEq(app(smtArithOpAdd, app(smtArithOpMul, num(2), x), y), num(1))
					astTestMust(t, err)
					formulas := []ast.ASTHandle{eq, app(smtArithOpGE, y, num(0)), app(smtArithOpLE, y, num(1)), app(smtArithOpGE, z, num(9876543))}
					if unsat {
						real, err := m.MkFamilySort(family, 0, nil)
						astTestMust(t, err)
						r, err := m.MkConst(astTestSymbol(t, m, "gomory-real"), real)
						astTestMust(t, err)
						half, err := m.ArithmeticNumeralRat(big.NewRat(1, 2), false)
						astTestMust(t, err)
						formulas = append(formulas, app(smtArithOpGE, r, num(0)), app(smtArithOpLE, r, half), app(smtArithOpLE, y, r))
					}
					for _, expr := range formulas {
						ref, err := m.NewRef(expr)
						astTestMust(t, err)
						t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
					}
					theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
					config, err := s.kernel.smtParams.ArithmeticState()
					astTestMust(t, err)
					config.BranchCutRatio = ^uint32(0)
					if cuts {
						config.BranchCutRatio = 1
					}
					astTestMust(t, theory.ConfigureArithmetic(config))
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						for _, expr := range formulas {
							astTestMust(t, s.Assert(expr))
						}
						before := theory.statistics.gomoryCuts
						beforeBranches := theory.statistics.branches
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						want := lbool.True
						if unsat {
							want = lbool.False
						}
						if status != want {
							t.Fatal("integer Gomory result", status, want)
						}
						if cuts && theory.statistics.gomoryCuts == before {
							t.Fatal("Gomory path was not exercised")
						}
						if !cuts && theory.statistics.branches == beforeBranches {
							t.Fatal("source integer Boolean case split was not exercised")
						}
						if unsat && proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							fact, err := m.GetProofFact(proof)
							astTestMust(t, err)
							if !m.IsFalse(fact) || cuts && !smtProofHasTheoryRule(t, m, proof, "gomory-cut") {
								t.Fatal("Gomory proof fact or rule metadata")
							}
							if cuts {
								lemma := smtProofTheoryRuleNode(t, m, proof, "gomory-cut")
								decl, err := m.GetDecl(lemma)
								astTestMust(t, err)
								parameters, err := m.GetDeclParametersCopy(decl)
								astTestMust(t, err)
								if len(parameters) < 3 {
									t.Fatal("Gomory lemma omitted bound weights")
								}
								for i := range parameters {
									kind, err := m.ParameterKind(parameters[i])
									astTestMust(t, err)
									if i > 1 && kind != ast.ASTParameterRational {
										t.Fatal("non-rational Gomory weight", kind)
									}
									astTestMust(t, m.DelParameter(&parameters[i]))
								}
								if smtTestArithmeticExpressionContains(t, m, lemma, z) {
									t.Fatal("Gomory proof cites unrelated assertion")
								}
							}
						}
						if !unsat {
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
							if !values[0].IsInt() || !values[1].IsInt() || new(big.Rat).Add(new(big.Rat).Mul(big.NewRat(2, 1), values[0]), values[1]).Cmp(big.NewRat(1, 1)) != 0 || values[1].Sign() < 0 || values[1].Cmp(big.NewRat(1, 1)) > 0 {
								t.Fatal("Gomory model violates original constraints", values)
							}
						}
						astTestMust(t, s.Pop(1))
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.True {
							t.Fatal("Gomory bounds survived pop", status)
						}
					}
				})
			}
		}
	}
}
