package smtcore

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArrayLogicSetupStaticAndFirstPushProfiles(t *testing.T) {
	for _, logic := range []string{"QF_AX", "QF_AUFLIA"} {
		for _, profile := range []string{"units", "clause", "first-push", "disabled"} {
			t.Run(fmt.Sprint(logic, "/", profile), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				solver, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, solver.Close()) })
				if profile == "disabled" {
					smtDisableAutoConfig(t, solver)
				}
				boolean, err := m.MkBoolSort()
				astTestMust(t, err)
				indexSort := boolean
				if logic == "QF_AUFLIA" {
					zero, e := m.ArithmeticIntegerNumeralText("0")
					astTestMust(t, e)
					indexSort, err = m.GetSort(zero)
					astTestMust(t, err)
				}
				arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
				astTestMust(t, err)
				array, err := m.MkConst(astTestSymbol(t, m, "array-setup-a"), arraySort)
				astTestMust(t, err)
				index, err := m.MkConst(astTestSymbol(t, m, "array-setup-i"), indexSort)
				astTestMust(t, err)
				family, err := m.ArrayFamilyID()
				astTestMust(t, err)
				formula, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{array, index})
				astTestMust(t, err)
				if profile == "clause" {
					p := smtRelevancyBool(t, m, "array-setup-clause-p")
					formula, err = m.MkOr(formula, p)
					astTestMust(t, err)
				}
				astTestMust(t, solver.Assert(formula))
				if profile == "first-push" {
					astTestMust(t, solver.Push())
				}
				status, err := smtSetupProfileCheck(solver)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("array setup changed satisfiability", status)
				}
				k := solver.kernel
				if profile == "disabled" {
					if k.smtParams.array.state.Mode != 3 || !k.preprocessState.NNFCNF || k.smtState.RelevancyLvl != 2 {
						t.Fatal("disabled automatic setup applied the array profile")
					}
					return
				}
				if k.smtParams.array.state.Mode != 1 || k.preprocessState.NNFCNF || k.smtParams.preprocess.state.NNFCNF {
					t.Fatal("array mode or preprocessing owner was not updated")
				}
				level, phase := uint32(2), uint32(3)
				restart, factor, activity := uint32(1), 1.1, uint32(1)
				if profile == "units" {
					level, phase = 0, 0
				} else if logic == "QF_AUFLIA" {
					phase, restart, factor = 4, 0, 1.5
					if profile == "clause" {
						level, activity = 0, 0
					}
				}
				if k.smtState.RelevancyLvl != level || k.smtParams.state.RelevancyLvl != level || k.internalizer.relevancyLevel != level || k.relevancy.enabled != (level > 0) || k.smtState.PhaseSelection != phase || k.smtState.RestartStrategy != restart || k.smtState.RestartFactor != factor || k.internalizer.randomInitialActivity != activity {
					t.Fatal("array profile differs from the selected source overload", k.smtState)
				}
				if profile == "first-push" {
					astTestMust(t, solver.Pop(1))
					if k.smtState.RelevancyLvl != level {
						t.Fatal("array setup was lost on pop")
					}
				}
			})
		}
	}
}

func TestSMTQFAUFLIAStaticSetupRejectsRealSorts(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	solver, err := NewSMTSolver(m, nil, "QF_AUFLIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	zero, err := m.ArithmeticRealNumeralText("0")
	astTestMust(t, err)
	real, err := m.GetSort(zero)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "array-logic-real-x"), real)
	astTestMust(t, err)
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	formula, err := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{x, zero})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(formula))
	_, err = smtSetupProfileCheck(solver)
	if err == nil || !strings.Contains(err.Error(), "real variables") || solver.kernel.preprocessingSetupDone {
		t.Fatal("QF_AUFLIA feature validation did not reject real arithmetic", err)
	}
}
