package smtcore

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSimpleLogicSetupBeforeFirstPushAndCheck(t *testing.T) {
	for _, logic := range []string{"QF_UF", "QF_DT", "QF_UFLRA"} {
		for _, firstPush := range []bool{false, true} {
			for _, automatic := range []bool{false, true} {
				t.Run(fmt.Sprint(logic, "/", firstPush, "/", automatic), func(t *testing.T) {
					m, cfg := smtModelGeneratorFixture(t)
					solver, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, solver.Close()) })
					if !automatic {
						smtDisableAutoConfig(t, solver)
					}
					var sort ast.ASTHandle
					if logic != "QF_UFLRA" {
						sort, err = m.MkUninterpretedSort(astTestSymbol(t, m, "simple-setup-U"), nil)
					} else {
						zero, e := m.ArithmeticRealNumeralText("0")
						astTestMust(t, e)
						sort, err = m.GetSort(zero)
					}
					astTestMust(t, err)
					left, err := m.MkConst(astTestSymbol(t, m, "simple-setup-a"), sort)
					astTestMust(t, err)
					right, err := m.MkConst(astTestSymbol(t, m, "simple-setup-b"), sort)
					astTestMust(t, err)
					f, err := m.MkFuncDecl(astTestSymbol(t, m, "simple-setup-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
					astTestMust(t, err)
					fa, err := m.MkApp(f, []ast.ASTHandle{left})
					astTestMust(t, err)
					fb, err := m.MkApp(f, []ast.ASTHandle{right})
					astTestMust(t, err)
					formula, err := m.MkEq(fa, fb)
					astTestMust(t, err)
					astTestMust(t, solver.Assert(formula))
					k := solver.kernel
					checkPolicy := func() {
						t.Helper()
						if !automatic {
							if k.smtState.RelevancyLvl != 2 || k.smtState.PhaseSelection != 3 || k.smtState.RandomInitialActivity != 1 || !k.preprocessState.NNFCNF {
								t.Fatal("disabled setup overwrote explicit policy")
							}
							return
						}
						if k.smtParams.state.RelevancyLvl != 0 || k.smtState.RelevancyLvl != 0 || k.relevancy.enabled || k.internalizer.relevancyLevel != 0 || k.preprocessState.NNFCNF || k.smtParams.preprocess.state.NNFCNF {
							t.Fatal("simple profile did not update its owner and consumers")
						}
						if logic != "QF_UFLRA" {
							if k.smtState.RestartStrategy != 2 || k.smtState.PhaseSelection != 4 || k.smtState.RandomInitialActivity != 2 || k.internalizer.randomInitialActivity != 2 {
								t.Fatal("UF search policy was not selected")
							}
						} else if k.arithState.Reflect || k.smtParams.arith.state.Reflect {
							t.Fatal("UFLRA reflection policy was not selected")
						}
					}
					if firstPush {
						astTestMust(t, solver.Push())
						checkPolicy()
						if automatic && logic != "QF_UFLRA" {
							randomized := false
							for _, activity := range k.internalizer.activity[1:] {
								randomized = randomized || activity != 0
							}
							if !randomized {
								t.Fatal("first-push Boolean creation used the old initial-activity mode")
							}
						}
					}
					status, err := smtSetupProfileCheck(solver)
					astTestMust(t, err)
					if status != lbool.True {
						t.Fatal("UF equality did not remain satisfiable", status)
					}
					checkPolicy()
					if firstPush {
						astTestMust(t, solver.Pop(1))
						checkPolicy()
					}
				})
			}
		}
	}
}

func TestSMTUFStaticSetupRejectsArithmetic(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	solver, err := NewSMTSolver(m, nil, "QF_UF", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	integer, err := m.GetSort(zero)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "uf-invalid-arithmetic-x"), integer)
	astTestMust(t, err)
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	formula, err := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{x, zero})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(formula))
	_, err = smtSetupProfileCheck(solver)
	if err == nil || !strings.Contains(err.Error(), "constains arithmetic") || solver.kernel.preprocessingSetupDone {
		t.Fatal("UF feature validation did not reject arithmetic", err)
	}
}
