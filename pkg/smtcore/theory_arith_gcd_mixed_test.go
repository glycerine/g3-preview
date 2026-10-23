package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTMixedGCDWithFixedFractionalReal(t *testing.T) {
	for _, preprocess := range []bool{false, true} {
		t.Run(fmt.Sprint(preprocess), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			s, err := NewSMTSolver(m, nil, "QF_LIRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = preprocess
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			integer, err := m.MkFamilySort(family, 1, nil)
			astTestMust(t, err)
			real, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "mixed-gcd-x"), integer)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "mixed-gcd-y"), integer)
			astTestMust(t, err)
			r, err := m.MkConst(astTestSymbol(t, m, "mixed-gcd-r"), real)
			astTestMust(t, err)
			app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
				h, err := m.MkFamilyApp(family, op, nil, args)
				astTestMust(t, err)
				return h
			}
			num := func(value *big.Rat) ast.ASTHandle {
				h, err := m.ArithmeticNumeralRat(value, false)
				astTestMust(t, err)
				return h
			}
			left := app(smtArithOpAdd, app(smtArithOpMul, num(big.NewRat(2, 1)), app(smtArithOpToReal, x)), app(smtArithOpMul, num(big.NewRat(4, 1)), app(smtArithOpToReal, y)), r)
			eq, err := m.MkEq(left, num(big.NewRat(1, 1)))
			astTestMust(t, err)
			astTestMust(t, s.Assert(eq))
			initial, err := s.CheckSat(nil)
			astTestMust(t, err)
			if initial != lbool.True {
				t.Fatal(initial)
			}
			astTestMust(t, s.Push())
			fixed, err := m.MkEq(r, num(big.NewRat(1, 2)))
			astTestMust(t, err)
			astTestMust(t, s.Assert(fixed))
			status, err := s.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("fixed fractional real in integer row", status)
			}
			if !preprocess {
				theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
				if theory.statistics.gcdConflicts == 0 || theory.statistics.branches != 0 {
					t.Fatal("mixed GCD contradiction bypassed source check", theory.statistics)
				}
			}
			astTestMust(t, s.Pop(1))
			status, err = s.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("fixed real premise survived pop", status)
			}
		})
	}
}
