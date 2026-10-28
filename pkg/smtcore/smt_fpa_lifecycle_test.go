package smtcore

// Source: theory_fpa::mk_fresh and context::copy_plugins.

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTFloatingPointTranslationFreshStateAndSourceIndependence(t *testing.T) {
	for _, sameManager := range []bool{false, true} {
		for _, configured := range []bool{false, true} {
			t.Run(fmt.Sprint(sameManager, "/", configured), func(t *testing.T) {
				manager, config := smtModelGeneratorFixture(t)
				target, targetConfig := manager, config
				if !sameManager {
					target, targetConfig = smtModelGeneratorFixture(t)
				}
				solverConfig := SMTSolverConfig{
					Params:         config.Context,
					ModelEvaluator: config,
					TargetConfig: func(candidate *ast.ASTManager) (SMTSolverConfig, error) {
						if candidate != target {
							return SMTSolverConfig{}, ast.ErrASTHandle
						}
						return SMTSolverConfig{Params: targetConfig.Context, ModelEvaluator: targetConfig}, nil
					},
				}
				source, err := NewSMTSolver(manager, nil, "QF_FP", solverConfig)
				astTestMust(t, err)
				sourceOpen := true
				t.Cleanup(func() {
					if sourceOpen {
						astTestMust(t, source.Close())
					}
				})
				family, err := manager.ExistingFloatingPointFamilyID()
				astTestMust(t, err)
				format, err := manager.FloatingPointSort(2, 2)
				astTestMust(t, err)
				nan, err := manager.MkFPANaNOfSort(format)
				astTestMust(t, err)
				equal, err := manager.MkFamilyApp(family, ast.FPAOpEq, nil, []ast.ASTHandle{nan, nan})
				astTestMust(t, err)
				astTestMust(t, source.Assert(equal))
				original := source.kernel.internalizer.theories[family].(*SMTFloatingPointTheory)
				if configured {
					status, checkErr := source.CheckSat(nil)
					astTestMust(t, checkErr)
					if status != lbool.False {
						t.Fatal("source FPA theory was not exercised", status)
					}
				}

				value, err := source.Translate(target, nil)
				astTestMust(t, err)
				translated := value.(*SMTSolver)
				t.Cleanup(func() { astTestMust(t, translated.Close()) })
				targetFamily, err := target.ExistingFloatingPointFamilyID()
				astTestMust(t, err)
				fresh, ok := translated.kernel.internalizer.theories[targetFamily].(*SMTFloatingPointTheory)
				if !ok || fresh == original || fresh.SMTTheoryBase == original.SMTTheoryBase || len(fresh.conversion) != 0 {
					t.Fatal("translation did not install fresh FPA state")
				}
				astTestMust(t, source.Close())
				sourceOpen = false
				status, err := translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("translated FPA assertion after source close", status)
				}

				value, err = translated.Translate(target, nil)
				astTestMust(t, err)
				chained := value.(*SMTSolver)
				t.Cleanup(func() { astTestMust(t, chained.Close()) })
				chainedTheory, ok := chained.kernel.internalizer.theories[targetFamily].(*SMTFloatingPointTheory)
				if !ok || chainedTheory == fresh || chainedTheory.SMTTheoryBase == fresh.SMTTheoryBase || len(chainedTheory.conversion) != 0 {
					t.Fatal("chained translation shared FPA state")
				}
				status, err = chained.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("chained FPA assertion", status)
				}
			})
		}
	}
}
