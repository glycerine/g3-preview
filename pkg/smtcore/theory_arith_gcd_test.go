package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTIntegerGCDAndExtendedGCDScopedProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, extended := range []bool{false, true} {
			t.Run(fmt.Sprint(proofs, "/", extended), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				s, err := NewSMTSolver(m, nil, "QF_LIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				s.kernel.preprocess = false
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				sort, err := m.MkFamilySort(family, 1, nil)
				astTestMust(t, err)
				x, err := m.MkConst(astTestSymbol(t, m, "gcd-x"), sort)
				astTestMust(t, err)
				y, err := m.MkConst(astTestSymbol(t, m, "gcd-y"), sort)
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
				coefficient, rhs := int64(4), int64(1)
				if extended {
					coefficient, rhs = 5, 0
				}
				eq, err := m.MkEq(app(smtArithOpAdd, app(smtArithOpMul, num(2), x), app(smtArithOpMul, num(coefficient), y)), num(rhs))
				astTestMust(t, err)
				formulas := []ast.ASTHandle{eq}
				if extended {
					formulas = append(formulas, app(smtArithOpGE, x, num(1)), app(smtArithOpLE, x, num(2)))
				}
				for _, expr := range formulas {
					ref, err := m.NewRef(expr)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				}
				theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
				config, err := s.kernel.smtParams.ArithmeticState()
				astTestMust(t, err)
				config.GCDTest = true
				astTestMust(t, theory.ConfigureArithmetic(config))
				for repeat := 0; repeat < 2; repeat++ {
					astTestMust(t, s.Push())
					before, branches := theory.statistics.gcdConflicts, theory.statistics.branches
					for _, expr := range formulas {
						astTestMust(t, s.Assert(expr))
					}
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.False {
						t.Fatal("integer GCD contradiction", status)
					}
					if theory.statistics.gcdConflicts == before || theory.statistics.branches != branches {
						t.Fatal("GCD contradiction required branch-and-bound", theory.statistics)
					}
					if proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("GCD conflict proof")
						}
						if !smtProofHasTheoryRule(t, m, proof, "gcd-test") {
							t.Fatal("GCD conflict proof lost gcd-test rule")
						}
					}
					astTestMust(t, s.Pop(1))
					status, err = s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("GCD contradiction survived pop", status)
					}
				}
				config.IgnoreInt = true
				astTestMust(t, theory.ConfigureArithmetic(config))
				astTestMust(t, s.Push())
				for _, expr := range formulas {
					astTestMust(t, s.Assert(expr))
				}
				before := theory.statistics.gcdConflicts
				status, err := s.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.Undef || theory.statistics.gcdConflicts != before {
					t.Fatal("IgnoreInt did not give up before integer checks", status)
				}
				astTestMust(t, s.Pop(1))
			})
		}
	}
}

func smtProofHasTheoryRule(t *testing.T, m *ast.ASTManager, proof ast.ASTHandle, rule string) bool {
	t.Helper()
	stack := []ast.ASTHandle{proof}
	seen := make(map[ast.ASTHandle]bool)
	for len(stack) != 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[node] {
			continue
		}
		seen[node] = true
		kind, err := m.GetKind(node)
		astTestMust(t, err)
		if kind != ast.ASTApp {
			continue
		}
		decl, err := m.GetDecl(node)
		astTestMust(t, err)
		parameters, err := m.GetDeclParametersCopy(decl)
		astTestMust(t, err)
		matched := false
		for i := range parameters {
			kind, err := m.ParameterKind(parameters[i])
			astTestMust(t, err)
			if kind == ast.ASTParameterSymbol {
				symbol, err := m.ParameterSymbol(parameters[i])
				astTestMust(t, err)
				text, _, err := m.Symbols.BareString(symbol)
				astTestMust(t, err)
				matched = matched || text == rule
			}
			astTestMust(t, m.DelParameter(&parameters[i]))
		}
		if matched {
			return true
		}
		count, err := m.GetNumArgs(node)
		astTestMust(t, err)
		for i := uint32(0); i < count; i++ {
			arg, err := m.GetArg(node, i)
			astTestMust(t, err)
			stack = append(stack, arg)
		}
	}
	return false
}
