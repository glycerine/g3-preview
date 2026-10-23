package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTIntegerFreedomPatchingScopedModels(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(mixed, "/", proofs), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				logic := "QF_LIA"
				if mixed {
					logic = "QF_LIRA"
				}
				s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				astTestMust(t, s.SetProduceModels(true))
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				integer, err := m.MkFamilySort(family, 1, nil)
				astTestMust(t, err)
				x, err := m.MkConst(astTestSymbol(t, m, "patch-x"), integer)
				astTestMust(t, err)
				y, err := m.MkConst(astTestSymbol(t, m, "patch-y"), integer)
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
				eq, err := m.MkEq(app(smtArithOpMul, num(2), x), y)
				astTestMust(t, err)
				formulas := []ast.ASTHandle{eq, app(smtArithOpGE, y, num(1)), app(smtArithOpLE, y, num(3))}
				var r ast.ASTHandle
				if mixed {
					real, err := m.MkFamilySort(family, 0, nil)
					astTestMust(t, err)
					r, err = m.MkConst(astTestSymbol(t, m, "patch-real"), real)
					astTestMust(t, err)
					lo, err := m.ArithmeticNumeralRat(big.NewRat(1, 4), false)
					astTestMust(t, err)
					hi, err := m.ArithmeticNumeralRat(big.NewRat(3, 4), false)
					astTestMust(t, err)
					formulas = append(formulas, app(smtArithOpGE, r, lo), app(smtArithOpLE, r, hi))
				}
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
					before := theory.statistics.branches
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True || theory.statistics.branches != before {
						t.Fatal("freedom-interval integer patch did not solve without branching", status, theory.statistics.branches-before)
					}
					model, err := s.AcquireModel()
					astTestMust(t, err)
					terms := []ast.ASTHandle{x, y}
					if mixed {
						terms = append(terms, r)
					}
					var values []*big.Rat
					for _, term := range terms {
						decl, err := m.GetDecl(term)
						astTestMust(t, err)
						value, err := model.ConstantInterpretation(decl)
						astTestMust(t, err)
						text, _, matched, err := m.ArithmeticNumeralText(value)
						astTestMust(t, err)
						point, ok := new(big.Rat).SetString(text)
						if !matched || !ok {
							t.Fatal("integer-patched model numeral", text)
						}
						values = append(values, point)
					}
					astTestMust(t, model.DecRef())
					if !values[0].IsInt() || !values[1].IsInt() || new(big.Rat).Mul(values[0], big.NewRat(2, 1)).Cmp(values[1]) != 0 || values[1].Cmp(big.NewRat(1, 1)) < 0 || values[1].Cmp(big.NewRat(3, 1)) > 0 {
						t.Fatal("integer-patched model violates original constraints", values)
					}
					if mixed && (values[2].Cmp(big.NewRat(1, 4)) < 0 || values[2].Cmp(big.NewRat(3, 4)) > 0) {
						t.Fatal("integer patch rounded unrestricted real column", values)
					}
					astTestMust(t, s.Pop(1))
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("integer patch survived pop", status)
					}
				}
			})
		}
	}
}
