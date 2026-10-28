package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSolverUnivariateNRAAPI(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.SetProduceModels(true))
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	real, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "nra-api-x"), real)
	astTestMust(t, err)
	numeral := func(n int64) ast.ASTHandle {
		h, e := m.ArithmeticNumeralRat(big.NewRat(n, 1), false)
		astTestMust(t, e)
		return h
	}
	// The API supports polynomial powers; QF_NRA intentionally excludes '^'
	// from its SMT-LIB operator names, matching the source declaration plugin.
	cube, err := m.MkFamilyApp(family, 18, nil, []ast.ASTHandle{x, numeral(3)})
	astTestMust(t, err)
	equal, err := m.MkEq(cube, numeral(2))
	astTestMust(t, err)
	equalRef, err := m.NewRef(equal)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&equalRef)) }()
	astTestMust(t, s.Assert(equal))
	for i := 0; i < 2; i++ {
		status, e := s.CheckSat(nil)
		astTestMust(t, e)
		if status != lbool.True {
			t.Fatal(status)
		}
		md, e := s.AcquireModel()
		astTestMust(t, e)
		evaluator, e := model.NewModelEvaluator(md, s.kernel.modelConfig)
		astTestMust(t, e)
		ref, e := evaluator.EvaluateValue(equal)
		astTestMust(t, e)
		value, e := m.RefGet(ref)
		astTestMust(t, e)
		if !m.IsTrue(value) {
			t.Fatal("algebraic API model violates cube equality")
		}
		astTestMust(t, m.DelRef(&ref))
		astTestMust(t, evaluator.Close())
		astTestMust(t, md.DecRef())
	}
	astTestMust(t, s.Push())
	negative, err := m.MkFamilyApp(family, smtArithOpLT, nil, []ast.ASTHandle{x, numeral(0)})
	astTestMust(t, err)
	astTestMust(t, s.Assert(negative))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("negative cube root of two", status)
	}
	astTestMust(t, s.Pop(1))
	status, err = s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("pop failed to restore original input", status)
	}
}

func TestSMTSolverSeparableMultivariableNRAAPI(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.SetProduceModels(true))
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	real, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "nra-api-separable-x"), real)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "nra-api-separable-y"), real)
	astTestMust(t, err)
	numeral := func(n int64) ast.ASTHandle {
		h, e := m.ArithmeticNumeralRat(big.NewRat(n, 1), false)
		astTestMust(t, e)
		return h
	}
	square := func(v ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(family, smtArithOpMul, nil, []ast.ASTHandle{v, v})
		astTestMust(t, e)
		return h
	}
	xEquation, err := m.MkEq(square(x), numeral(2))
	astTestMust(t, err)
	yEquation, err := m.MkEq(square(y), numeral(3))
	astTestMust(t, err)
	xPositive, err := m.MkFamilyApp(family, smtArithOpGT, nil, []ast.ASTHandle{x, numeral(0)})
	astTestMust(t, err)
	yNegative, err := m.MkFamilyApp(family, smtArithOpLT, nil, []ast.ASTHandle{y, numeral(0)})
	astTestMust(t, err)
	formula, err := m.MkAndArgs([]ast.ASTHandle{xEquation, yEquation, xPositive, yNegative})
	astTestMust(t, err)
	formulaRef, err := m.NewRef(formula)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&formulaRef)) }()
	astTestMust(t, s.Assert(formula))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	md, err := s.AcquireModel()
	astTestMust(t, err)
	evaluator, err := model.NewModelEvaluator(md, s.kernel.modelConfig)
	astTestMust(t, err)
	valueRef, err := evaluator.EvaluateValue(formula)
	astTestMust(t, err)
	value, err := m.RefGet(valueRef)
	astTestMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("separable multivariable algebraic model violates its formula")
	}
	astTestMust(t, m.DelRef(&valueRef))
	astTestMust(t, evaluator.Close())
	astTestMust(t, md.DecRef())
}
