package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtDatalogSolver(t *testing.T, models bool) (*ast.ASTManager, ast.ASTHandle, *SMTSolver) {
	t.Helper()
	m, modelConfig := smtModelGeneratorFixture(t)
	sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "smt-dl-domain"), 4)
	astTestMust(t, err)
	config := SMTSolverConfig{Params: modelConfig.Context}
	if models {
		config.ModelEvaluator = modelConfig
	}
	solver, err := NewSMTSolver(m, nil, "QF_FD", config)
	astTestMust(t, err)
	if models {
		astTestMust(t, solver.SetProduceModels(true))
	}
	t.Cleanup(func() {
		astTestMust(t, solver.Close())
	})
	return m, sort, solver
}

func TestSMTDatalogStrictOrderReduction(t *testing.T) {
	m, sort, solver := smtDatalogSolver(t, false)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-dl-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "smt-dl-y"), sort)
	astTestMust(t, err)
	xy, err := m.MkDatalogLT(x, y)
	astTestMust(t, err)
	yx, err := m.MkDatalogLT(y, x)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(xy))
	astTestMust(t, solver.Assert(yx))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("cyclic finite-domain order accepted", status)
	}
}

func TestSMTDatalogNumeralOrderAndModel(t *testing.T) {
	m, sort, solver := smtDatalogSolver(t, true)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-dl-model-x"), sort)
	astTestMust(t, err)
	two, err := m.MkDatalogNumeral(2, sort)
	astTestMust(t, err)
	three, err := m.MkDatalogNumeral(3, sort)
	astTestMust(t, err)
	equality, err := m.MkEq(x, two)
	astTestMust(t, err)
	less, err := m.MkDatalogLT(x, three)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(equality))
	astTestMust(t, solver.Assert(less))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		reason, _ := solver.ReasonUnknown()
		t.Fatal("finite-domain numeral constraints", status, reason)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	if generated == nil {
		t.Fatal("finite-domain solver omitted model")
	}
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	declaration, err := m.GetDecl(x)
	astTestMust(t, err)
	value, err := generated.ConstantInterpretation(declaration)
	astTestMust(t, err)
	n, matched, err := m.DatalogNumeralValue(value)
	astTestMust(t, err)
	if !matched || n != 2 {
		t.Fatal("finite-domain model value", value, n, matched)
	}
}
