package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArithmeticInfeasibleRowsUseScopedRelaxedProofPremises(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			for _, relax := range []bool{false, true} {
				t.Run(fmt.Sprint(integer, "/", proofs, "/", relax), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					logic, kind := "QF_LRA", int32(0)
					if integer {
						logic, kind = "QF_LIA", 1
					}
					s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, s.Close()) })
					s.kernel.preprocess = false
					family, err := m.ArithmeticFamilyID()
					astTestMust(t, err)
					sort, err := m.MkFamilySort(family, kind, nil)
					astTestMust(t, err)
					var terms [3]ast.ASTHandle
					for i := range terms {
						terms[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("row-conflict-", i)), sort)
						astTestMust(t, err)
					}
					x, y, unrelated := terms[0], terms[1], terms[2]
					app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
						h, err := m.MkFamilyApp(family, op, nil, args)
						astTestMust(t, err)
						return h
					}
					num := func(value int64) ast.ASTHandle {
						h, err := m.ArithmeticNumeralRat(big.NewRat(value, 1), integer)
						astTestMust(t, err)
						return h
					}
					sum := app(smtArithOpAdd, x, y)
					formulas := []ast.ASTHandle{app(smtArithOpGE, sum, num(10)), app(smtArithOpLE, x, num(1)), app(smtArithOpLE, y, num(2)), app(smtArithOpGE, unrelated, num(77))}
					weaker := []ast.ASTHandle{app(smtArithOpGE, sum, num(5)), app(smtArithOpLE, x, num(4)), app(smtArithOpLE, y, num(4))}
					formulas = append(formulas, weaker...)
					for _, formula := range formulas {
						ref, err := m.NewRef(formula)
						astTestMust(t, err)
						t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
					}
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						for _, formula := range formulas {
							astTestMust(t, s.Assert(formula))
						}
						theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
						config, err := s.kernel.smtParams.ArithmeticState()
						astTestMust(t, err)
						config.StrongerLemmas = relax
						astTestMust(t, theory.ConfigureArithmetic(config))
						before := theory.statistics.pivots
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.False || theory.statistics.pivots == before {
							t.Fatal("infeasible row with pivots", status, theory.statistics)
						}
						if proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							lemma := smtProofTheoryRuleNode(t, m, proof, "farkas")
							if lemma.IsNull() {
								t.Fatal("missing row-conflict farkas rule")
							}
							if smtTestArithmeticExpressionContains(t, m, proof, unrelated) {
								t.Fatal("row conflict cites unrelated assertion")
							}
							found := false
							for _, formula := range weaker {
								found = found || smtTestArithmeticExpressionContains(t, m, proof, formula)
							}
							if found != relax {
								t.Fatal("row explanation ignored stronger-lemmas mode", relax, found)
							}
							decl, err := m.GetDecl(lemma)
							astTestMust(t, err)
							parameters, err := m.GetDeclParametersCopy(decl)
							astTestMust(t, err)
							count, err := m.GetNumArgs(lemma)
							astTestMust(t, err)
							if len(parameters) != int(count)+1 {
								t.Fatal("row proof weights do not align with parent proofs", len(parameters), count)
							}
							for i := range parameters {
								if i > 1 {
									kind, err := m.ParameterKind(parameters[i])
									astTestMust(t, err)
									if kind != ast.ASTParameterRational {
										t.Fatal(kind)
									}
								}
								astTestMust(t, m.DelParameter(&parameters[i]))
							}
						}
						astTestMust(t, s.Pop(1))
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.True {
							t.Fatal("row conflict survived scope pop", status)
						}
					}
				})
			}
		}
	}
}
