package solver

import (
	"errors"
	"runtime"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type routedTestSolver struct {
	Solver
	status lbool.LBool
	err    error
	checks int
	reason string
}

func (s *routedTestSolver) ReasonUnknown() (string, error) { return s.reason, nil }

func (s *routedTestSolver) CheckSat([]ast.ASTHandle) (lbool.LBool, error) {
	s.checks++
	return s.status, s.err
}

func combinedTestPair(t *testing.T, m *ast.ASTManager, pctx *uparams.ParamsContext, firstStatus, secondStatus lbool.LBool) (*routedTestSolver, *routedTestSolver) {
	t.Helper()
	firstBase, err := NewTacticSolver(m, tactic.NewSkipTactic(), nil, false, false, false, "", TacticSolverConfig{Params: pctx})
	solverMust(t, err)
	secondBase, err := NewTacticSolver(m, tactic.NewSkipTactic(), nil, false, false, false, "", TacticSolverConfig{Params: pctx})
	solverMust(t, err)
	return &routedTestSolver{Solver: firstBase, status: firstStatus}, &routedTestSolver{Solver: secondBase, status: secondStatus}
}

func TestCombinedSolverDelayedInitializationAndIncrementalRouting(t *testing.T) {
	m := solverTestManager(t)
	pctx := solverTestParams(t, m)
	globals := combinedSolverGlobals(t, pctx)
	first, second := combinedTestPair(t, m, pctx, lbool.True, lbool.False)
	c, err := NewCombinedSolver(first, second, nil, CombinedSolverConfig{Params: pctx, Globals: globals})
	solverMust(t, err)
	defer func() { solverMust(t, c.Close()) }()
	a := solverTacticBool(t, m, "combined-a")
	b := solverTacticBool(t, m, "combined-b")
	solverMust(t, c.Assert(a))
	if n, _ := second.NumAssertions(); n != 0 {
		t.Fatal("second solver initialized early", n)
	}
	status, err := c.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.True || first.checks != 1 || second.checks != 0 {
		t.Fatal("initial routing", status, first.checks, second.checks)
	}
	solverMust(t, c.Assert(b))
	if n, _ := second.NumAssertions(); n != 2 {
		t.Fatal("delayed assertion copy", n)
	}
	status, err = c.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.False || first.checks != 1 || second.checks != 1 {
		t.Fatal("incremental routing", status, first.checks, second.checks)
	}
	first.reason, second.reason = "first-result", "second-result"
	if reason, e := c.ReasonUnknown(); e != nil || reason != "second-result" {
		t.Fatal("second result routing", reason, e)
	}
	if c.useFirst {
		t.Fatal("second result was not selected")
	}
}

func TestCombinedSolverRealTimeoutCancellationIsScoped(t *testing.T) {
	m := solverTestManager(t)
	status, timed, err := runCombinedTimed(1, m.ResourceLimit(), func() (lbool.LBool, error) {
		for !m.ResourceLimit().CancelFlag() {
			runtime.Gosched()
		}
		return lbool.Undef, errors.New("canceled by timeout")
	})
	if status != lbool.Undef || !timed || err == nil {
		t.Fatal("timeout result", status, timed, err)
	}
	if m.ResourceLimit().CancelFlag() {
		t.Fatal("timeout cancellation escaped its scope")
	}
}

type countedSolverFactory struct {
	Factory
	created, closed int
}

func (f *countedSolverFactory) New(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, logic string) (Solver, error) {
	f.created++
	return f.Factory.New(m, p, proofs, models, cores, logic)
}
func (f *countedSolverFactory) Close() error {
	f.closed++
	return f.Factory.Close()
}

func TestCombinedSolverFactoryOwnsFactoriesAndBuildsPair(t *testing.T) {
	m := solverTestManager(t)
	pctx := solverTestParams(t, m)
	globals := combinedSolverGlobals(t, pctx)
	f1base, err := NewTacticSolverFactory(tactic.NewSkipTactic(), TacticSolverConfig{Params: pctx})
	solverMust(t, err)
	f2base, err := NewTacticSolverFactory(tactic.NewSkipTactic(), TacticSolverConfig{Params: pctx})
	solverMust(t, err)
	f1 := &countedSolverFactory{Factory: f1base}
	f2 := &countedSolverFactory{Factory: f2base}
	factory, err := NewCombinedSolverFactory(f1, f2, CombinedSolverConfig{Params: pctx, Globals: globals})
	solverMust(t, err)
	created, err := factory.New(m, nil, false, false, false, "")
	solverMust(t, err)
	if f1.created != 1 || f2.created != 1 {
		t.Fatal("factory creation", f1.created, f2.created)
	}
	status, err := created.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.True {
		t.Fatal("combined factory solver", status)
	}
	solverMust(t, created.Close())
	solverMust(t, factory.Close())
	if f1.closed != 1 || f2.closed != 1 {
		t.Fatal("factory ownership", f1.closed, f2.closed)
	}
}

func TestCombinedSolverTranslationPreservesState(t *testing.T) {
	from := solverTestManager(t)
	to := solverTestManager(t)
	fromParams := solverTestParams(t, from)
	toParams := solverTestParams(t, to)
	fromGlobals := combinedSolverGlobals(t, fromParams)
	toGlobals := combinedSolverGlobals(t, toParams)
	childConfig := TacticSolverConfig{Params: fromParams}
	childConfig.TargetConfig = func(got *ast.ASTManager) (TacticSolverConfig, error) {
		if got != to {
			t.Fatal("unexpected child target")
		}
		return TacticSolverConfig{Params: toParams}, nil
	}
	first, err := NewTacticSolver(from, tactic.NewSkipTactic(), nil, false, false, false, "", childConfig)
	solverMust(t, err)
	second, err := NewTacticSolver(from, tactic.NewSkipTactic(), nil, false, false, false, "", childConfig)
	solverMust(t, err)
	config := CombinedSolverConfig{Params: fromParams, Globals: fromGlobals}
	config.TargetConfig = func(got *ast.ASTManager) (CombinedSolverConfig, error) {
		if got != to {
			t.Fatal("unexpected combined target")
		}
		return CombinedSolverConfig{Params: toParams, Globals: toGlobals}, nil
	}
	c, err := NewCombinedSolver(first, second, nil, config)
	solverMust(t, err)
	defer func() { solverMust(t, c.Close()) }()
	solverMust(t, c.Assert(solverTacticBool(t, from, "translated-combined")))
	solverMust(t, c.switchIncremental())
	translatedValue, err := c.Translate(to, nil)
	solverMust(t, err)
	translated := translatedValue.(*CombinedSolver)
	defer func() { solverMust(t, translated.Close()) }()
	if level, e := translated.ScopeLevel(); e != nil || level != 0 {
		t.Fatal("translated scope", level, e)
	}
	h, err := translated.Assertion(0)
	solverMust(t, err)
	if _, err = to.GetKind(h); err != nil {
		t.Fatal("translated assertion owner", err)
	}
	if !translated.incMode || !translated.secondStarted {
		t.Fatal("translated routing state")
	}
}

func TestCombinedSolverAssumptionsAndTrackedAssertionsForceSecond(t *testing.T) {
	m := solverTestManager(t)
	pctx := solverTestParams(t, m)
	globals := combinedSolverGlobals(t, pctx)
	first, second := combinedTestPair(t, m, pctx, lbool.True, lbool.False)
	c, err := NewCombinedSolver(first, second, nil, CombinedSolverConfig{Params: pctx, Globals: globals})
	solverMust(t, err)
	defer func() { solverMust(t, c.Close()) }()
	formula := solverTacticBool(t, m, "tracked-formula")
	track := solverTacticBool(t, m, "tracked-label")
	solverMust(t, c.AssertTracked(formula, track))
	// Source first copies solver1's encoded tracked assertion, then invokes the
	// tracked assertion operation on solver2.
	if n, _ := second.NumAssertions(); n != 2 {
		t.Fatal("tracked assertion not mirrored", n)
	}
	status, err := c.CheckSat([]ast.ASTHandle{track})
	solverMust(t, err)
	if status != lbool.False || first.checks != 0 || second.checks != 1 {
		t.Fatal("assumption routing", status, first.checks, second.checks)
	}
}

func TestCombinedSolverUnknownPolicyAndTimedFallback(t *testing.T) {
	m := solverTestManager(t)
	pctx := solverTestParams(t, m)
	globals := combinedSolverGlobals(t, pctx)
	first, second := combinedTestPair(t, m, pctx, lbool.True, lbool.Undef)
	c, err := NewCombinedSolver(first, second, nil, CombinedSolverConfig{Params: pctx, Globals: globals})
	solverMust(t, err)
	defer func() { solverMust(t, c.Close()) }()
	solverMust(t, c.Push())
	status, err := c.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.True || first.checks != 1 || second.checks != 1 || !c.useFirst {
		t.Fatal("quantifier-free unknown fallback", status, first.checks, second.checks, c.useFirst)
	}

	local, err := pctx.NewValue()
	solverMust(t, err)
	defer func() { solverMust(t, pctx.Del(&local)) }()
	key, err := m.Symbols.Intern("solver2_unknown")
	solverMust(t, err)
	solverMust(t, pctx.SetUInt(&local, key, 0))
	solverMust(t, c.UpdateParams(&local))
	status, err = c.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.Undef || first.checks != 1 || second.checks != 2 || c.useFirst {
		t.Fatal("return-unknown policy", status, first.checks, second.checks, c.useFirst)
	}

	key, err = m.Symbols.Intern("solver2_timeout")
	solverMust(t, err)
	solverMust(t, pctx.SetUInt(&local, key, 1))
	solverMust(t, pctx.SetUInt(&local, mustSolverSymbol(t, m, "solver2_unknown"), 2))
	solverMust(t, c.UpdateParams(&local))
	c.config.RunTimed = func(ms uint32, limit *control.Reslimit, fn func() (lbool.LBool, error)) (lbool.LBool, bool, error) {
		if ms != 1 || limit != m.ResourceLimit() {
			t.Fatal("timed runner arguments")
		}
		_, _ = fn()
		return lbool.Undef, true, errors.New("timeout cancellation")
	}
	status, err = c.CheckSat(nil)
	solverMust(t, err)
	if status != lbool.True || first.checks != 2 || second.checks != 3 || !c.useFirst {
		t.Fatal("timed fallback", status, first.checks, second.checks, c.useFirst)
	}
}

func mustSolverSymbol(t *testing.T, m *ast.ASTManager, name string) usymbol.Symbol {
	t.Helper()
	result, err := m.Symbols.Intern(name)
	solverMust(t, err)
	return result
}
