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
		}{
			{false, 512, "2", lbool.Undef},
			{true, 0, "2", lbool.Undef},
			{true, 512, "2", lbool.False},
			{true, 512, "-1", lbool.Undef},
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
					astTestMust(t, theory.ConfigureArithmetic(config))
					for _, formula := range formulas {
						astTestMust(t, s.Assert(formula))
					}
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != tc.want {
						t.Fatal("coupled polynomial equations", status, tc.want)
					}
					if tc.enabled && theory.statistics.grobner == 0 {
						t.Fatal("Grobner engine was not exercised")
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
