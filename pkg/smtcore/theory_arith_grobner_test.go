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
		}{
			{false, 512, "2", lbool.Undef, true},
			{true, 0, "2", lbool.Undef, true},
			{true, 512, "2", lbool.False, true},
			{true, 512, "-1", lbool.Undef, true},
			{true, 512, "-1", lbool.Undef, false},
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
				xz, err := m.MkArithmeticMul(terms[0], terms[2])
				astTestMust(t, err)
				sum, err := m.MkFamilyApp(family, smtArithOpAdd, nil, terms[1:])
				astTestMust(t, err)
				var formulas []ast.ASTHandle
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
