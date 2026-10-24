package core

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func simplifyMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func simplifyFixture(t *testing.T) (*ast.ASTManager, *uparams.ParamsContext, *uparams.GlobalParamsService) {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	simplifyMust(t, err)
	numbers, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	simplifyMust(t, err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numbers, Budget: budget})
	simplifyMust(t, err)
	context, err := uparams.NewParamsContext(symbols, numbers, uparams.ParamsConfig{Budget: budget})
	simplifyMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	simplifyMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, context, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	simplifyMust(t, err)
	simplifyMust(t, globals.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	simplifyMust(t, err)
	t.Cleanup(func() {
		_, err := uparams.BindProcessGlobalParamsService(previous)
		simplifyMust(t, err)
		simplifyMust(t, globals.Close())
		simplifyMust(t, catalog.Close())
		simplifyMust(t, context.Close())
		simplifyMust(t, manager.Close())
		simplifyMust(t, numbers.Close())
		simplifyMust(t, symbols.Close())
		if budget.Used() != 0 {
			t.Fatal("simplifier retained charged memory", budget.Used())
		}
	})
	return manager, context, globals
}

func simplifyPin(t *testing.T, manager *ast.ASTManager, value ast.ASTHandle) {
	t.Helper()
	ref, err := manager.NewRef(value)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, manager.DelRef(&ref)) })
}

func simplifyFalseEquality(t *testing.T, manager *ast.ASTManager) ast.ASTHandle {
	t.Helper()
	one, err := manager.ArithmeticIntegerNumeralText("1")
	simplifyMust(t, err)
	two, err := manager.ArithmeticIntegerNumeralText("2")
	simplifyMust(t, err)
	formula, err := manager.MkEq(one, two)
	simplifyMust(t, err)
	simplifyPin(t, manager, formula)
	return formula
}

func TestSimplifyTacticProofDependenciesDepthAndCleanup(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	simplifyMust(t, manager.ToggleProofMode(ast.ASTProofEnabled))
	simplifier, err := NewSimplifyTactic(manager, context, globals, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, simplifier.Close()) })
	goal, err := tactic.NewGoal(manager, true, true, true)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	formula := simplifyFalseEquality(t, manager)
	proof, err := manager.MkProofAsserted(formula)
	simplifyMust(t, err)
	dependency, err := manager.DependencyLeaf(formula)
	simplifyMust(t, err)
	simplifyMust(t, manager.DependencyInc(dependency))
	t.Cleanup(func() { simplifyMust(t, manager.DependencyDec(dependency)) })
	simplifyMust(t, goal.AssertWithProof(formula, proof, dependency))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, simplifier.Apply(goal, &result))
	if len(result.Goals) != 1 || result.Goals[0] != goal || !goal.IsDecidedUnsat() || result.ModelConverter != nil || result.ProofConverter != nil || !result.Core.IsNull() {
		t.Fatal("simplifier did not return its in-place inconsistent goal")
	}
	depth, err := goal.Depth()
	simplifyMust(t, err)
	if depth != 1 {
		t.Fatal("simplifier depth", depth)
	}
	steps, err := simplifier.NumSteps()
	simplifyMust(t, err)
	if steps == 0 {
		t.Fatal("simplifier did not count its rewrite steps")
	}
	derived, err := goal.Proof(0)
	simplifyMust(t, err)
	fact, err := manager.GetProofFact(derived)
	simplifyMust(t, err)
	if !manager.IsFalse(fact) {
		t.Fatal("rewritten proof did not prove false")
	}
	gotDependency, err := goal.Dependency(0)
	simplifyMust(t, err)
	if gotDependency != dependency {
		t.Fatal("rewriting discarded the assertion dependency")
	}
	simplifyMust(t, simplifier.Cleanup())
	steps, err = simplifier.NumSteps()
	simplifyMust(t, err)
	if steps != 0 {
		t.Fatal("cleanup retained previous rewrite counters")
	}
	simplifyMust(t, simplifier.Apply(goal, &result))
	depth, err = goal.Depth()
	simplifyMust(t, err)
	steps, err = simplifier.NumSteps()
	simplifyMust(t, err)
	if depth != 2 || steps != 0 {
		t.Fatal("already inconsistent goal was rewritten or did not advance depth", depth, steps)
	}
}

func TestSimplifyTacticConvertsRewriteExceptionAndRecovers(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	params, err := context.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, context.Del(&params)) })
	key := "max_steps"
	simplifyMust(t, context.SetUIntName(&params, &key, 0))
	simplifier, err := NewSimplifyTactic(manager, context, globals, &params)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, simplifier.Close()) })
	goal, err := tactic.NewGoal(manager, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(simplifyFalseEquality(t, manager)))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	err = simplifier.Apply(goal, &result)
	var exception tactic.TacticException
	if !errors.As(err, &exception) || exception.Message() != tactic.TacticMaxStepsMessage {
		t.Fatal("rewriter exception did not cross the tactic boundary correctly", err)
	}
	if len(result.Goals) != 0 {
		t.Fatal("failed rewrite published a subgoal")
	}
	simplifyMust(t, context.SetUIntName(&params, &key, math.MaxUint32))
	simplifyMust(t, simplifier.UpdateParams(&params))
	simplifyMust(t, simplifier.Cleanup())
	simplifyMust(t, simplifier.Apply(goal, &result))
	if !goal.IsDecidedUnsat() {
		t.Fatal("simplifier did not recover after its rewrite limit was updated")
	}
}

func TestEliminateAndFactoryStickyParamsAndTranslation(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	value, err := context.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, context.Del(&value)) })
	key := "elim_and"
	simplifyMust(t, context.SetBoolName(&value, &key, false))
	wrapped, err := MakeEliminateAndTactic(manager, context, globals, &value)
	simplifyMust(t, err)
	simplifyMust(t, wrapped.IncRef())
	t.Cleanup(func() { simplifyMust(t, wrapped.DecRef()) })
	simplifyMust(t, wrapped.UpdateParams(&value))
	copy, err := wrapped.Translate(manager)
	simplifyMust(t, err)
	simplifyMust(t, copy.IncRef())
	t.Cleanup(func() { simplifyMust(t, copy.DecRef()) })
	boolean, err := manager.MkBoolSort()
	simplifyMust(t, err)
	var atoms []ast.ASTHandle
	for _, name := range []string{"elim-p", "elim-q", "elim-r"} {
		symbol, err := manager.Symbols.Intern(name)
		simplifyMust(t, err)
		atom, err := manager.MkConst(symbol, boolean)
		simplifyMust(t, err)
		simplifyPin(t, manager, atom)
		atoms = append(atoms, atom)
	}
	conjunction, err := manager.MkAnd(atoms[0], atoms[1])
	simplifyMust(t, err)
	formula, err := manager.MkOr(conjunction, atoms[2])
	simplifyMust(t, err)
	simplifyPin(t, manager, formula)
	for _, runner := range []tactic.Tactic{wrapped, copy, wrapped} {
		goal, err := tactic.NewGoal(manager, false, true, false)
		simplifyMust(t, err)
		simplifyMust(t, goal.Assert(formula))
		var result tactic.TacticResult
		func() {
			defer func() { simplifyMust(t, result.Close()); simplifyMust(t, goal.Close()) }()
			simplifyMust(t, runner.Apply(goal, &result))
			root, err := goal.Form(0)
			simplifyMust(t, err)
			var todo = []ast.ASTHandle{root}
			for len(todo) != 0 {
				node := todo[len(todo)-1]
				todo = todo[:len(todo)-1]
				kind, err := manager.GetDeclKind(node)
				simplifyMust(t, err)
				if kind == ast.ASTOpAnd {
					t.Fatal("elim-and parameter was overridden or lost by cleanup/translation")
				}
				count, err := manager.GetNumArgs(node)
				simplifyMust(t, err)
				for i := uint32(0); i < count; i++ {
					child, err := manager.GetArg(node, i)
					simplifyMust(t, err)
					todo = append(todo, child)
				}
			}
		}()
		simplifyMust(t, runner.Cleanup())
	}
}

func TestSimplifyTacticTranslationOwnsParamsAcrossManagers(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	settings, err := context.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, context.Del(&settings)) })
	key := "max_steps"
	simplifyMust(t, context.SetUIntName(&settings, &key, 0))
	source, err := NewSimplifyTactic(manager, context, globals, &settings)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, source.Close()) })
	targetManager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: manager.Symbols, Numerals: manager.NumeralService(), Budget: manager.Budget()})
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, targetManager.Close()) })
	translated, err := source.Translate(targetManager)
	simplifyMust(t, err)
	simplifyMust(t, translated.IncRef())
	t.Cleanup(func() { simplifyMust(t, translated.DecRef()) })
	// Mutating the original parameter owner and closing its tactic must not
	// alter the translated tactic's stored max_steps setting.
	simplifyMust(t, context.SetUIntName(&settings, &key, math.MaxUint32))
	simplifyMust(t, source.UpdateParams(&settings))
	simplifyMust(t, source.Close())
	goal, err := tactic.NewGoal(targetManager, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(simplifyFalseEquality(t, targetManager)))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	err = translated.Apply(goal, &result)
	var exception tactic.TacticException
	if !errors.As(err, &exception) || exception.Message() != tactic.TacticMaxStepsMessage {
		t.Fatal("translation lost its copied rewrite limit", err)
	}
	simplifyMust(t, translated.UpdateParams(&settings))
	simplifyMust(t, translated.Cleanup())
	simplifyMust(t, translated.Apply(goal, &result))
	if !goal.IsDecidedUnsat() {
		t.Fatal("translated tactic did not recover with updated parameters")
	}
	descriptions, err := uparams.NewParamDescrs(manager.Symbols, uparams.ParamDescrsConfig{Budget: manager.Budget()})
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, descriptions.Close()) })
	simplifyMust(t, translated.CollectParamDescrs(descriptions))
	for _, name := range []string{"elim_and", "max_steps", "som", "sort_store"} {
		found, err := descriptions.ContainsName(name)
		simplifyMust(t, err)
		if !found {
			t.Fatal("simplifier parameter description missing", name)
		}
	}
}

func TestSimplifyTacticSolverCompositionScopesAndUndecided(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	child, err := MakeSimplifyTactic(manager, context, globals, nil)
	simplifyMust(t, err)
	backend, err := solver.NewTacticSolver(manager, child, nil, false, true, false, "QF_UF", solver.TacticSolverConfig{Params: context})
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, backend.Close()) })
	status, err := backend.CheckSat(nil)
	simplifyMust(t, err)
	if status != lbool.True {
		t.Fatal("empty simplified goal was not SAT", status)
	}
	generated, err := backend.AcquireModel()
	simplifyMust(t, err)
	if generated == nil {
		t.Fatal("simplified SAT goal did not publish its requested model")
	}
	simplifyMust(t, generated.DecRef())
	simplifyMust(t, backend.Push())
	simplifyMust(t, backend.Assert(simplifyFalseEquality(t, manager)))
	status, err = backend.CheckSat(nil)
	simplifyMust(t, err)
	if status != lbool.False {
		t.Fatal("simplifier did not discharge a contradictory equality", status)
	}
	simplifyMust(t, backend.Pop(1))
	status, err = backend.CheckSat(nil)
	simplifyMust(t, err)
	if status != lbool.True {
		t.Fatal("tactic solver retained the popped contradiction", status)
	}
	boolean, err := manager.MkBoolSort()
	simplifyMust(t, err)
	name, err := manager.Symbols.Intern("simplify-undecided")
	simplifyMust(t, err)
	atom, err := manager.MkConst(name, boolean)
	simplifyMust(t, err)
	simplifyMust(t, backend.Assert(atom))
	status, err = backend.CheckSat(nil)
	simplifyMust(t, err)
	if status != lbool.Undef {
		t.Fatal("simplification alone claimed to solve an undecided goal", status)
	}
}
