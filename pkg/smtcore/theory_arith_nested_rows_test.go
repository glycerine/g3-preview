package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNestedCanonicalRowsThroughSeparateAliases(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(integer, "/", proofs), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				logic, kind := "QF_NRA", int32(0)
				if integer {
					logic, kind = "QF_NIA", 1
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
					terms[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("nested-row-", i)), sort)
					astTestMust(t, err)
				}
				x, p, q := terms[0], terms[1], terms[2]
				app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
					h, err := m.MkFamilyApp(family, op, nil, args)
					astTestMust(t, err)
					return h
				}
				two, err := m.ArithmeticNumeralRat(big.NewRat(2, 1), integer)
				astTestMust(t, err)
				for _, pair := range [][2]ast.ASTHandle{{p, app(smtArithOpMul, x, x)}, {q, app(smtArithOpMul, two, x)}} {
					eq, err := m.MkEq(pair[0], pair[1])
					astTestMust(t, err)
					astTestMust(t, s.Assert(eq))
				}
				bad := app(smtArithOpLE, app(smtArithOpAdd, p, two), q)
				ref, err := m.NewRef(bad)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
				config, err := s.kernel.smtParams.ArithmeticState()
				astTestMust(t, err)
				config.NLArithGB, config.NLArithBranching = false, false
				astTestMust(t, theory.ConfigureArithmetic(config))
				for repeat := 0; repeat < 2; repeat++ {
					astTestMust(t, s.Push())
					astTestMust(t, s.Assert(bad))
					before := theory.statistics.crossNested
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.False {
						t.Fatal("canonical nonlinear row conflict", status)
					}
					if theory.statistics.crossNested == before || theory.statistics.grobner != 0 || theory.statistics.nonlinearBranches != 0 {
						t.Fatal("canonical row regression bypassed nested checking")
					}
					if proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("canonical nested row proof")
						}
					}
					astTestMust(t, s.Pop(1))
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("canonical nested row conflict survived pop", status)
					}
				}
			})
		}
	}
}
