package smtcore

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func smtSolverFixture(t *testing.T) (*ast.ASTManager, *SMTSolver) {
	t.Helper()
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	s, err := NewSMTSolver(m, nil, "QF_UF", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, s.Close())
		astTestMust(t, params.Close())
	})
	return m, s
}

func TestSMTSolverKernelFacadeAssertionsScopesAndResults(t *testing.T) {
	m, s := smtSolverFixture(t)
	a := smtRelevancyBool(t, m, "smt-solver-a")
	b := smtRelevancyBool(t, m, "smt-solver-b")

	astTestMust(t, s.Assert(a))
	if count, err := s.NumAssertions(); err != nil || count != 1 {
		t.Fatal("base assertions", count, err)
	}
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("base status", status)
	}
	astTestMust(t, s.Push())
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	astTestMust(t, s.Assert(notA))
	status, err = s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("scoped contradiction", status)
	}
	astTestMust(t, s.Pop(1))
	status, err = s.Status()
	astTestMust(t, err)
	if status != lbool.Undef {
		t.Fatal("pop did not clear result", status)
	}
	astTestMust(t, s.Assert(b))
	status, err = s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("post-pop status", status)
	}
	var out strings.Builder
	astTestMust(t, s.Display(&out, nil))
	if !strings.Contains(out.String(), "assert") {
		t.Fatal("solver display", out.String())
	}
	stats := statistics.New()
	astTestMust(t, s.CollectStatistics(stats))
	if stats.Size() == 0 {
		t.Fatal("missing cumulative kernel statistics")
	}
}

func TestSMTSolverTrackedAssertionsCoreDuplicatesAndRollback(t *testing.T) {
	m, s := smtSolverFixture(t)
	a := smtRelevancyBool(t, m, "smt-solver-tracked-a")
	tag := smtRelevancyBool(t, m, "smt-solver-tracked-tag")
	notA, err := m.MkNot(a)
	astTestMust(t, err)

	astTestMust(t, s.Push())
	astTestMust(t, s.AssertTracked(a, tag))
	if err = s.AssertTracked(a, tag); err == nil {
		t.Fatal("duplicate tracked assertion accepted")
	} else {
		var duplicate z3exception.DefaultException
		if !errors.As(err, &duplicate) {
			t.Fatal("duplicate error", err)
		}
	}
	status, err := s.CheckSat([]ast.ASTHandle{notA})
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("tracked assertion status", status)
	}
	core, err := s.UnsatCore()
	astTestMust(t, err)
	seen := make(map[ast.ASTHandle]bool, len(core))
	for _, expression := range core {
		seen[expression] = true
	}
	if len(core) != 2 || !seen[tag] || !seen[notA] {
		t.Fatal("tracked assertion core", core, tag, notA)
	}
	astTestMust(t, s.Pop(1))
	if count, err := s.NumAssumptions(); err != nil || count != 0 {
		t.Fatal("tracked assumptions after pop", count, err)
	}
	// The source duplicate-name table is scope-sensitive.
	astTestMust(t, s.AssertTracked(a, tag))
}

func TestSMTInternalizerImplicationGateWithoutPreprocessing(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "smt-implies-a")
	b := smtRelevancyBool(t, m, "smt-implies-b")
	implies, err := m.MkImplies(a, b)
	astTestMust(t, err)
	notImplies, err := m.MkNot(implies)
	astTestMust(t, err)
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(notImplies, ast.ASTHandle{}))

	status, err := kernel.CheckAssumptions([]ast.ASTHandle{notA})
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("negated implication did not force antecedent", status)
	}
	status, err = kernel.CheckAssumptions([]ast.ASTHandle{b})
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("negated implication did not refute consequent", status)
	}
}

func TestSMTSolverFactoryAndUnsupportedKernelPrerequisites(t *testing.T) {
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, params.Close()) })
	factory, err := NewSMTSolverFactory(SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, factory.Close()) })
	value, err := factory.New(m, nil, false, true, true, "QF_UF")
	astTestMust(t, err)
	s, ok := value.(*SMTSolver)
	if !ok || !s.produceModels {
		t.Fatal("factory result", value)
	}
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	if _, err = s.Translate(m, nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("kernel translation prerequisite", err)
	}
	if status, _, err := s.GetConsequences(nil, nil); status != lbool.Undef || !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("kernel consequences prerequisite", status, err)
	}
	if status, _, err := s.FindMutexes(nil); status != lbool.Undef || !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("kernel mutex prerequisite", status, err)
	}
	if err = s.Pop(1); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("scope underflow", err)
	}
	var _ solver.Solver = s
}
