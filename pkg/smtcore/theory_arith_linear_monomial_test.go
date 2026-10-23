package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTLinearizedMonomialScopesProofsAndModels(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(integer, "/", proofs), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				logic, sortKind := "QF_NRA", int32(0)
				if integer {
					logic, sortKind = "QF_NIA", 1
				}
				s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				astTestMust(t, s.SetProduceModels(true))
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				sort, err := m.MkFamilySort(family, sortKind, nil)
				astTestMust(t, err)
				x, err := m.MkConst(astTestSymbol(t, m, "linearized-x"), sort)
				astTestMust(t, err)
				y, err := m.MkConst(astTestSymbol(t, m, "linearized-y"), sort)
				astTestMust(t, err)
				product, err := m.MkArithmeticMul(x, y)
				astTestMust(t, err)
				one, err := m.ArithmeticNumeralRat(big.NewRat(1, 1), integer)
				astTestMust(t, err)
				sum, err := m.MkFamilyApp(family, smtArithOpAdd, nil, []ast.ASTHandle{y, one})
				astTestMust(t, err)
				equation, err := m.MkEq(product, sum)
				astTestMust(t, err)
				astTestMust(t, s.Assert(equation))
				// As in source propagate_core, linear monomial propagation
				// remains available when general nonlinear search is disabled.
				astTestMust(t, s.Push())
				theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
				config, err := s.kernel.smtParams.ArithmeticState()
				astTestMust(t, err)
				config.NLArith, config.NLArithGB, config.NLArithBranching = false, false, false
				astTestMust(t, theory.ConfigureArithmetic(config))
				for _, tc := range []struct {
					fixed int64
					model string
				}{{0, "-1"}, {2, "1"}, {3, "1/2"}} {
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						value, err := m.ArithmeticNumeralRat(big.NewRat(tc.fixed, 1), integer)
						astTestMust(t, err)
						fixed, err := m.MkEq(x, value)
						astTestMust(t, err)
						astTestMust(t, s.Assert(fixed))
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						want := lbool.True
						if integer && tc.fixed == 3 {
							want = lbool.False
						}
						if status != want {
							t.Fatal("linearized product", tc.fixed, status, want)
						}
						if status == lbool.True {
							model, err := s.AcquireModel()
							astTestMust(t, err)
							decl, err := m.GetDecl(y)
							astTestMust(t, err)
							value, err := model.ConstantInterpretation(decl)
							astTestMust(t, err)
							text, isInt, numeral, err := m.ArithmeticNumeralText(value)
							astTestMust(t, err)
							if !numeral || isInt != integer || text != tc.model {
								t.Fatal("linearized product model", text, tc.model)
							}
							astTestMust(t, model.DecRef())
						} else if proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							fact, err := m.GetProofFact(proof)
							astTestMust(t, err)
							if !m.IsFalse(fact) {
								t.Fatal("linearized integer product proof")
							}
						}
						astTestMust(t, s.Pop(1))
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.Undef {
							t.Fatal("conditional linear product row survived pop", status)
						}
					}
				}
				astTestMust(t, s.Pop(1))
			})
		}
	}
}
