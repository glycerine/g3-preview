package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNonlinearNumericProgressAndScopedStrategies(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			if proofs {
				astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			}
			s, err := NewSMTSolver(m, nil, "QF_NIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = false
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			sort, err := m.MkFamilySort(family, 1, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "numeric-progress-x"), sort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "numeric-progress-y"), sort)
			astTestMust(t, err)
			var formulas []ast.ASTHandle
			for _, variable := range []ast.ASTHandle{x, y} {
				for _, bound := range []struct {
					op   int32
					text string
				}{{smtArithOpGE, "1"}, {smtArithOpLE, "10"}} {
					value, err := m.ArithmeticIntegerNumeralText(bound.text)
					astTestMust(t, err)
					formula, err := m.MkFamilyApp(family, bound.op, nil, []ast.ASTHandle{variable, value})
					astTestMust(t, err)
					formulas = append(formulas, formula)
				}
			}
			product, err := m.MkArithmeticMul(x, y)
			astTestMust(t, err)
			eleven, err := m.ArithmeticIntegerNumeralText("11")
			astTestMust(t, err)
			equality, err := m.MkEq(product, eleven)
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
				config.NLArithGB, config.NLArithBranching = false, false
				astTestMust(t, theory.ConfigureArithmetic(config))
				for _, formula := range formulas {
					astTestMust(t, s.Assert(formula))
				}
				// Flush every input atom and initial bound before recording the
				// search counters. Only numeric facts can advance this check.
				astTestMust(t, s.Push())
				before := s.kernel.internalizer.searchProgress()
				status, err := s.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("iterated nonlinear interval conflict", status)
				}
				after := s.kernel.internalizer.searchProgress()
				if after.theory <= before.theory || after.variables != before.variables || len(theory.derivedBounds) == 0 {
					t.Fatal("numeric-only progress was not retained", before, after, len(theory.derivedBounds))
				}
				if theory.statistics.grobner != 0 || theory.statistics.nonlinearBranches != 0 {
					t.Fatal("interval conflict used a disabled strategy")
				}
				if proofs {
					proof, err := s.Proof()
					astTestMust(t, err)
					fact, err := m.GetProofFact(proof)
					astTestMust(t, err)
					if !m.IsFalse(fact) {
						t.Fatal("numeric-only progress proof")
					}
				}
				astTestMust(t, s.Pop(1))
				if len(theory.derivedBounds) != 0 || theory.nonlinearStrategy != 0 {
					t.Fatal("derived numeric facts survived pop")
				}
				astTestMust(t, s.Pop(1))
				status, err = s.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("numeric interval conflict survived outer pop", status)
				}
			}
		})
	}
}
