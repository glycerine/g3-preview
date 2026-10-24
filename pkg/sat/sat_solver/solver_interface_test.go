package sat_solver_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	incsat "github.com/glycerine/g3/pkg/sat/sat_solver"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestIncrementalSolverTranslationOriginalRootsAndTargetServices(t *testing.T) {
	f := newEncoderFixture(t, false)
	target := newEncoderFixture(t, false)
	resolved := 0
	s, err := incsat.NewIncrementalSolverWithConfig(f.manager, &f.params, incsat.IncrementalSolverConfig{Services: f.services, TargetServices: func(m *ast.ASTManager) (model.ModelEvaluatorConfig, error) {
		resolved++
		if m != target.manager {
			return model.ModelEvaluatorConfig{}, ast.ErrASTHandle
		}
		return target.services, nil
	}})
	incMust(t, err)
	t.Cleanup(func() { incMust(t, s.Close()) })
	p := solverForm(t, f, `p`)
	h := solverForm(t, f, `(= x #x1)`)
	incMust(t, s.AssertTracked(h, p))
	got, err := s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal(got)
	}
	incMust(t, s.Push())
	_, err = s.Translate(target.manager, &target.params)
	var refusal z3exception.DefaultException
	if !errors.As(err, &refusal) || err.Error() != "Cannot translate sat solver at non-base level" {
		t.Fatal("source scoped translation refusal", err)
	}
	incMust(t, s.Pop(1))
	translated, err := s.Translate(target.manager, &target.params)
	incMust(t, err)
	t.Cleanup(func() { incMust(t, translated.Close()) })
	if resolved != 1 {
		t.Fatal("target services were not resolved", resolved)
	}
	n, err := translated.NumAssertions()
	incMust(t, err)
	if n != 1 {
		t.Fatal("translated rewritten SAT state instead of original queue", n)
	}
	n, err = translated.NumAssumptions()
	incMust(t, err)
	if n != 1 {
		t.Fatal("lost tracker", n)
	}
	incMust(t, s.Close())
	incMust(t, f.services.Cooperation.Close())
	got, err = translated.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal(got)
	}
	concrete, ok := translated.(*incsat.Solver)
	if !ok {
		t.Fatal("unexpected translated solver type")
	}
	h = solverForm(t, target, `(= x #x1)`)
	assertSolverModel(t, target, concrete, []ast.ASTHandle{h})
	incMust(t, translated.Push())
	incMust(t, translated.Assert(solverForm(t, target, `(= x #x2)`)))
	got, err = translated.CheckSat(nil)
	incMust(t, err)
	if got != lbool.False {
		t.Fatal("translated assertion/tracker not active", got)
	}
	incMust(t, translated.Pop(1))
	var display strings.Builder
	incMust(t, translated.Display(&display, nil))
	if !strings.Contains(display.String(), "assert") {
		t.Fatal("missing default solver display", display.String())
	}
	got, cores, err := translated.PreferredSAT([]ast.ASTHandle{solverForm(t, target, `(not p)`)})
	incMust(t, err)
	if got != lbool.True || len(cores) != 0 {
		t.Fatal("source default preferred-sat", got, cores)
	}
}

func TestIncrementalSolverTranslationAllocationFrontiers(t *testing.T) {
	for _, sameManager := range []bool{true, false} {
		t.Run(fmt.Sprint(sameManager), func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 8192 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f, s := newSolverFixture(t)
					incMust(t, s.AssertTracked(solverForm(t, f, `(= (bvadd x y) #x3)`), solverForm(t, f, `p`)))
					manager := f.manager
					if !sameManager {
						var err error
						manager, err = ast.NewASTManager(ast.ASTManagerConfig{Symbols: f.manager.Symbols, Numerals: f.manager.NumeralService(), Budget: f.manager.Budget()})
						incMust(t, err)
						incMust(t, manager.CopyFamiliesPlugins(f.manager))
						t.Cleanup(func() { incMust(t, manager.Close()) })
					}
					// The source owns partial translated targets, so it must close
					// before the target manager after a failed translation.
					t.Cleanup(func() { incMust(t, s.Close()) })
					f.manager.Budget().FailAfter(stop)
					translated, err := s.Translate(manager, &f.params)
					f.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
						incMust(t, translated.Close())
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if t.Failed() {
					t.Fatal("translation reservation frontier", sameManager, stop)
				}
			}
			if !success {
				t.Fatal("no successful translation frontier")
			}
		})
	}
}

func TestIncrementalSolverSolverTacticIntegration(t *testing.T) {
	for _, unsat := range []bool{false, true} {
		t.Run(fmt.Sprint(unsat), func(t *testing.T) {
			f, s := newSolverFixture(t)
			a, err := solver.NewSolverTactic(s, solver.SolverTacticConfig{Params: f.services.Context})
			incMust(t, err)
			t.Cleanup(func() { incMust(t, a.Close()) })
			g, err := tactic.NewGoal(f.manager, false, true, unsat)
			incMust(t, err)
			t.Cleanup(func() { incMust(t, g.Close()) })
			h := solverForm(t, f, `(= x #x1)`)
			p := solverForm(t, f, `p`)
			q := solverForm(t, f, `q`)
			if unsat {
				dp, e := f.manager.DependencyLeaf(p)
				incMust(t, e)
				dq, e := f.manager.DependencyLeaf(q)
				incMust(t, e)
				incMust(t, g.AssertWithProof(h, ast.ASTHandle{}, dp))
				incMust(t, g.AssertWithProof(solverForm(t, f, `(= x #x2)`), ast.ASTHandle{}, dq))
			} else {
				incMust(t, g.Assert(h))
			}
			var result tactic.TacticResult
			t.Cleanup(func() { incMust(t, result.Close()) })
			incMust(t, a.Apply(g, &result))
			incMust(t, a.Close())
			if unsat {
				if !g.IsDecidedUnsat() {
					t.Fatal("solver2tactic did not publish contradiction")
				}
				labels, err := f.manager.DependencyLinearize(result.Core, nil)
				incMust(t, err)
				seen := map[ast.ASTHandle]bool{}
				for _, label := range labels {
					seen[label] = true
				}
				if len(labels) != 2 || !seen[p] || !seen[q] {
					t.Fatal("solver2tactic core labels", labels)
				}
				return
			}
			if !g.IsDecidedSat() || result.ModelConverter == nil {
				t.Fatal("solver2tactic did not publish model")
			}
			var out tactic.ModelRef
			t.Cleanup(func() { incMust(t, out.Close()) })
			incMust(t, result.ModelConverter.ApplyModel(&out, 0))
			md, err := out.Get()
			incMust(t, err)
			ev, err := model.NewModelEvaluator(md, f.services)
			incMust(t, err)
			t.Cleanup(func() { incMust(t, ev.Close()) })
			value, err := f.manager.NewRef(ast.ASTHandle{})
			incMust(t, err)
			t.Cleanup(func() { incMust(t, f.manager.DelRef(&value)) })
			incMust(t, ev.Evaluate(h, value))
			v, err := f.manager.RefGet(value)
			incMust(t, err)
			if !f.manager.IsTrue(v) {
				t.Fatal("adapter model violates original input after backend destruction")
			}
		})
	}
}
