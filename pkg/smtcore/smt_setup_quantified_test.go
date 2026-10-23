package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTQuantifiedLogicSetupProfilesAndConstructorCaches(t *testing.T) {
	for _, tc := range []struct {
		name, logic                string
		mixed, real, pattern, push bool
		eager                      float64
		arrayMode                  uint32
	}{
		{"lia-inferred", "AUFLIA", false, false, false, false, 5, 1},
		{"lia-pattern", "AUFLIA", false, false, true, false, 7, 1},
		{"lia-push", "AUFLIA", false, false, true, true, 66, 1},
		{"lia-alias", "AUFLIA+", false, false, false, false, 66, 1},
		{"nia", "UFNIA", false, false, false, false, 66, 1},
		{"mixed", "AUFLIRA", true, true, false, false, 5, 1},
		{"mixed-nonlinear", "AUFNIRA", true, true, false, false, 5, 1},
		{"mixed-alias", "AUFNIRA-", true, true, true, false, 5, 1},
		{"uflra-check", "UFLRA", true, true, false, false, 5, 3},
		{"uflra-push", "UFLRA", true, true, false, true, 5, 1},
		{"unknown-integer", "ALL", false, false, false, false, 66, 3},
		{"unknown-mixed", "ALL", true, true, false, false, 5, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			params, err := cfg.Context.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&params)) })
			name := "mbqi"
			astTestMust(t, cfg.Context.SetBoolName(&params, &name, false))
			name = "qi.eager_threshold"
			astTestMust(t, cfg.Context.SetDoubleName(&params, &name, 66))
			name = "qi.lazy_threshold"
			astTestMust(t, cfg.Context.SetDoubleName(&params, &name, 1))
			solver, err := NewSMTSolver(m, &params, tc.logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			k := solver.kernel
			manager := k.internalizer.quantifierManager
			costs := manager.registry.costs
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			sort := boolean
			if tc.real {
				zero, e := m.ArithmeticRealNumeralText("0")
				astTestMust(t, e)
				sort, err = m.GetSort(zero)
				astTestMust(t, err)
			}
			predicate := smtQuickPredicate(t, m, "quantified-setup-p", sort, boolean)
			variable, err := m.MkVar(0, sort)
			astTestMust(t, err)
			body, err := m.MkApp(predicate, []ast.ASTHandle{variable})
			astTestMust(t, err)
			spec := ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body, Weight: 1}
			if tc.pattern {
				pattern, err := m.MkPattern([]ast.ASTHandle{body})
				astTestMust(t, err)
				spec.Patterns = []ast.ASTHandle{pattern}
			}
			quantifier, err := m.MkQuantifier(spec)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(quantifier))
			if tc.real {
				// Source feature traversal skips bound variables. Include a
				// real ground term so setup_unknown observes m_has_real.
				ground, err := m.MkConst(astTestSymbol(t, m, "quantified-setup-real"), sort)
				astTestMust(t, err)
				one, err := m.ArithmeticRealNumeralText("1")
				astTestMust(t, err)
				equality, err := m.MkEq(ground, one)
				astTestMust(t, err)
				astTestMust(t, solver.Assert(equality))
			}
			if tc.push {
				astTestMust(t, solver.Push())
			} else {
				astTestMust(t, k.setupAssertedBounds(true))
			}
			qi := k.smtParams.qi.state
			if qi.EagerThreshold != tc.eager || qi.LazyThreshold != 20 || qi.QuickChecker != 1 || !qi.MBQI || !manager.mbqi || manager.modelChecker == nil || !manager.quickUnsat || manager.quickNotSat || manager.registry.schedule.lazyThreshold != 20 {
				t.Fatal("quantifier owner and live manager controls diverged", qi)
			}
			if manager.registry.schedule.eagerThreshold != 66 || manager.registry.costs != costs || manager.builder.costs != costs {
				t.Fatal("logic setup replaced constructor-cached eager threshold or cost expressions")
			}
			preprocess := k.preprocessState
			if !preprocess.EliminateBounds || !preprocess.PropagateBooleans || preprocess.NonGroundLiftITE != smtLiftITEFull || preprocess.MacroFinder != tc.mixed || k.smtState.PhaseSelection != 0 || k.smtParams.array.state.Mode != tc.arrayMode {
				t.Fatal("quantified preprocessing or array policy", preprocess)
			}
			patternState := k.smtParams.preprocess.pattern.state
			arrayFamily, err := m.ArrayFamilyID()
			astTestMust(t, err)
			arrayTheory := smtArrayCore(k.internalizer.theories[arrayFamily])
			if arrayTheory == nil || arrayTheory.arrayConfig.LazyIEq != k.smtParams.array.state.LazyIEq || arrayTheory.arrayConfig.LazyIEqDelay != k.smtParams.array.state.LazyIEqDelay {
				t.Fatal("array backend did not consume quantified setup's interface schedule")
			}
			if tc.mixed {
				if patternState.MaxMultiPatterns != 10 || !k.smtParams.array.state.LazyIEq || k.smtParams.array.state.LazyIEqDelay != 4 {
					t.Fatal("mixed quantified profile omitted pattern or array settings")
				}
			} else if !patternState.UseDatabase || k.smtState.RestartStrategy != 0 || k.smtState.RestartFactor != 1.5 {
				t.Fatal("integer quantified profile omitted pattern database or restart settings")
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("quantified setup failed to validate a satisfiable universal", status)
			}
			if tc.mixed && arrayTheory.finalCheckIndex < 4 {
				t.Fatal("mixed quantified setup skipped the delayed array interface checks")
			}
			if tc.push {
				astTestMust(t, solver.Pop(1))
			}
			// Later parameter updates are allowed to disable MBQI again;
			// one-time logic setup must not silently reenable it on checks.
			astTestMust(t, solver.UpdateParams(&params))
			astTestMust(t, k.setupAssertedBounds(true))
			if manager.mbqi || manager.modelChecker != nil || manager.registry.schedule.eagerThreshold != 66 {
				t.Fatal("later update reran setup or replaced the queue cache")
			}
		})
	}
}

func TestSMTAUFLIAFeatureSetupRejectsRealSorts(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	zero, err := m.ArithmeticRealNumeralText("0")
	astTestMust(t, err)
	real, err := m.GetSort(zero)
	astTestMust(t, err)
	variable, err := m.MkVar(0, real)
	astTestMust(t, err)
	equality, err := m.MkEq(variable, zero)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{real}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: equality, Weight: 1})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(quantifier))
	_, err = solver.CheckSat(nil)
	if err == nil || solver.kernel.preprocessingSetupDone {
		t.Fatal("AUFLIA feature setup accepted real arithmetic", err)
	}
}
