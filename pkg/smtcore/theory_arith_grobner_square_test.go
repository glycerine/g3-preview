package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/grobner"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTGrobnerPerfectSquareMonomialsAndCrossTerms(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "square-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "square-y"), sort)
	astTestMust(t, err)
	gb, err := grobner.New(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, gb.Close()) })
	monomial := func(coefficient string, variables ...ast.ASTHandle) *grobner.Monomial {
		value, ok := new(big.Rat).SetString(coefficient)
		if !ok {
			t.Fatal(coefficient)
		}
		result, err := gb.NewMonomial(value, variables)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, gb.DelMonomial(result)) })
		return result
	}
	for _, tc := range []struct {
		coefficient string
		variables   []ast.ASTHandle
		root        string
	}{
		{"9/16", []ast.ASTHandle{x, x, y, y}, "3/4"},
		{"4", nil, "2"},
		{"2/9", []ast.ASTHandle{x, x}, ""},
		{"-1", []ast.ASTHandle{x, x}, ""},
		{"1", []ast.ASTHandle{x, x, x, y}, ""},
		{"1", []ast.ASTHandle{x}, ""},
	} {
		root, square := smtGrobnerPerfectSquare(monomial(tc.coefficient, tc.variables...))
		if square != (tc.root != "") || square && root.RatString() != tc.root {
			t.Fatal("perfect square", tc, root, square)
		}
	}
	first := monomial("9/16", x, x, y, y)
	second := monomial("4/9", x, x)
	a, _ := smtGrobnerPerfectSquare(first)
	b, _ := smtGrobnerPerfectSquare(second)
	if !smtGrobnerSquareCross(first, a, second, b, monomial("-1", x, x, y)) {
		t.Fatal("shared-variable rational square cross term")
	}
	for _, cross := range []*grobner.Monomial{monomial("1", x, x, y), monomial("-2", x, x, y), monomial("-1", x, y, y)} {
		if smtGrobnerSquareCross(first, a, second, b, cross) {
			t.Fatal("non-square cross term accepted")
		}
	}
}

func TestSMTNonlinearGrobnerExpandedSquareScopedProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			for _, preprocess := range []bool{false, true} {
				t.Run(fmt.Sprint(proofs, "/", enabled, "/", preprocess), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, s.Close()) })
					s.kernel.preprocess = preprocess
					family, err := m.ArithmeticFamilyID()
					astTestMust(t, err)
					sort, err := m.MkFamilySort(family, 0, nil)
					astTestMust(t, err)
					x, err := m.MkConst(astTestSymbol(t, m, "expanded-square-x"), sort)
					astTestMust(t, err)
					y, err := m.MkConst(astTestSymbol(t, m, "expanded-square-y"), sort)
					astTestMust(t, err)
					xx, err := m.MkArithmeticMul(x, x)
					astTestMust(t, err)
					yy, err := m.MkArithmeticMul(y, y)
					astTestMust(t, err)
					minusTwo, err := m.ArithmeticRealNumeralText("-2")
					astTestMust(t, err)
					cross, err := m.MkFamilyApp(family, smtArithOpMul, nil, []ast.ASTHandle{minusTwo, x, y})
					astTestMust(t, err)
					polynomial, err := m.MkFamilyApp(family, smtArithOpAdd, nil, []ast.ASTHandle{xx, yy, cross})
					astTestMust(t, err)
					minusOne, err := m.ArithmeticRealNumeralText("-1")
					astTestMust(t, err)
					formula, err := m.MkEq(polynomial, minusOne)
					astTestMust(t, err)
					ref, err := m.NewRef(formula)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
					for repeat := 0; repeat < 2; repeat++ {
						astTestMust(t, s.Push())
						theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
						config, err := s.kernel.smtParams.ArithmeticState()
						astTestMust(t, err)
						config.NLArithGB = enabled
						astTestMust(t, theory.ConfigureArithmetic(config))
						astTestMust(t, s.Assert(formula))
						status, err := s.CheckSat(nil)
						astTestMust(t, err)
						want := lbool.Undef
						if enabled {
							want = lbool.False
						}
						if status != want {
							t.Fatal("expanded square plus one", status, want)
						}
						if enabled && proofs {
							proof, err := s.Proof()
							astTestMust(t, err)
							fact, err := m.GetProofFact(proof)
							astTestMust(t, err)
							if !m.IsFalse(fact) {
								t.Fatal("expanded square conflict proof")
							}
						}
						astTestMust(t, s.Pop(1))
						status, err = s.CheckSat(nil)
						astTestMust(t, err)
						if status != lbool.True {
							t.Fatal("expanded square conflict survived pop", status)
						}
					}
				})
			}
		}
	}
}
