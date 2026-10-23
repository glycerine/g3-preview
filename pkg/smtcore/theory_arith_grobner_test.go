package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNonlinearGrobnerCoupledEquationsScopedProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, tc := range []struct {
			enabled       bool
			limit         uint32
			secondProduct string
			want          lbool.LBool
			perturbate    bool
			fixedFactor   bool
		}{
			{false, 512, "2", lbool.Undef, true, false},
			{true, 0, "2", lbool.Undef, true, false},
			{true, 512, "2", lbool.False, true, false},
			{true, 512, "-1", lbool.Undef, true, false},
			{true, 512, "-1", lbool.Undef, false, false},
			{true, 512, "2", lbool.False, true, true},
		} {
			t.Run(fmt.Sprint(proofs, "/", tc), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				sort, err := m.MkFamilySort(family, 0, nil)
				astTestMust(t, err)
				unrelated, err := m.MkConst(astTestSymbol(t, m, "gb-unrelated"), sort)
				astTestMust(t, err)
				terms := make([]ast.ASTHandle, 3)
				for i := range terms {
					terms[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("gb-", i)), sort)
					astTestMust(t, err)
				}
				// x*y = 1, x*z = 2, y+z = 0 implies 1+2 = 0.
				// No variable has a unary bound, so interval propagation alone
				// cannot establish this contradiction.
				xy, err := m.MkArithmeticMul(terms[0], terms[1])
				astTestMust(t, err)
				var fixedGuard ast.ASTHandle
				if tc.fixedFactor {
					fixed, err := m.MkConst(astTestSymbol(t, m, "gb-fixed-factor"), sort)
					astTestMust(t, err)
					two, err := m.ArithmeticRealNumeralText("2")
					astTestMust(t, err)
					fixedGuard, err = m.MkEq(fixed, two)
					astTestMust(t, err)
					xy, err = m.MkFamilyApp(family, smtArithOpMul, nil, []ast.ASTHandle{fixed, terms[0], terms[1]})
					astTestMust(t, err)
				}
				xz, err := m.MkArithmeticMul(terms[0], terms[2])
				astTestMust(t, err)
				sum, err := m.MkFamilyApp(family, smtArithOpAdd, nil, terms[1:])
				astTestMust(t, err)
				var formulas []ast.ASTHandle
				unrelatedValue, err := m.ArithmeticRealNumeralText("77")
				astTestMust(t, err)
				unrelatedBound, err := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{unrelated, unrelatedValue})
				astTestMust(t, err)
				unrelatedRef, err := m.NewRef(unrelatedBound)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, m.DelRef(&unrelatedRef)) })
				formulas = append(formulas, unrelatedBound)
				if !fixedGuard.IsNull() {
					formulas = append(formulas, fixedGuard)
					ref, err := m.NewRef(fixedGuard)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				}
				for i, expression := range []ast.ASTHandle{xy, xz, sum} {
					value, err := m.ArithmeticRealNumeralText([]string{"1", tc.secondProduct, "0"}[i])
					astTestMust(t, err)
					formula, err := m.MkEq(expression, value)
					astTestMust(t, err)
					formulas = append(formulas, formula)
					ref, err := m.NewRef(formula)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				}
				for repeat := 0; repeat < 2; repeat++ {
					astTestMust(t, s.Push())
					theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
					config, err := s.kernel.smtParams.ArithmeticState()
					astTestMust(t, err)
					config.NLArithGB = tc.enabled
					config.NLArithGBThreshold = tc.limit
					config.NLArithGBPerturbate = tc.perturbate
					astTestMust(t, theory.ConfigureArithmetic(config))
					for _, formula := range formulas {
						astTestMust(t, s.Assert(formula))
					}
					before := theory.statistics.grobner
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != tc.want {
						t.Fatal("coupled polynomial equations", status, tc.want)
					}
					if tc.enabled && theory.statistics.grobner == 0 {
						t.Fatal("Grobner engine was not exercised")
					}
					if tc.enabled && tc.limit == 0 && !theory.nonlinearGBExhausted {
						t.Fatal("Grobner equation limit did not mark the search exhausted")
					}
					if tc.enabled && tc.secondProduct == "-1" {
						rounds := theory.statistics.grobner - before
						if tc.perturbate && rounds < 2 || !tc.perturbate && rounds != 1 {
							t.Fatal("Grobner perturbation rounds", tc.perturbate, rounds)
						}
					}
					if tc.enabled && tc.limit == 0 {
						config.NLArithGBThreshold = 512
						astTestMust(t, theory.ConfigureArithmetic(config))
						retried, err := s.CheckSat(nil)
						astTestMust(t, err)
						if retried != lbool.False || theory.nonlinearGBExhausted {
							t.Fatal("new search did not reset Grobner exhaustion", retried, theory.nonlinearGBExhausted)
						}
					}
					if status == lbool.False && proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("Grobner conflict proof")
						}
						if smtTestArithmeticExpressionContains(t, m, proof, unrelated) {
							t.Fatal("Grobner proof cites unrelated assertion")
						}
						for _, formula := range formulas[1:] {
							if !smtTestArithmeticExpressionContains(t, m, proof, formula) {
								t.Fatal("Grobner proof lost a necessary equation")
							}
						}
					}
					astTestMust(t, s.Pop(1))
					if theory.nonlinearGBExhausted {
						t.Fatal("Grobner exhaustion survived pop")
					}
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("Grobner conflict survived pop", status)
					}
				}
			})
		}
	}
}
