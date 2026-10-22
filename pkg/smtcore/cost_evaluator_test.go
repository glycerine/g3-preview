package smtcore

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSMTCostEvaluatorSourceLanguageAndReverseVariables(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	three, err := m.ArithmeticRealNumeralText("3")
	astTestMust(t, err)
	real, err := m.GetSort(three)
	astTestMust(t, err)
	x, err := m.MkVar(0, real)
	astTestMust(t, err)
	y, err := m.MkVar(1, real)
	astTestMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	add, err := m.MkFamilyApp(arithmetic, smtArithOpAdd, nil, []ast.ASTHandle{x, y})
	astTestMust(t, err)
	comparison, err := m.MkFamilyApp(arithmetic, smtArithOpGT, nil, []ast.ASTHandle{add, three})
	astTestMust(t, err)
	truth, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpTrue, nil, nil)
	astTestMust(t, err)
	formula, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{truth, comparison})
	astTestMust(t, err)
	var warnings bytes.Buffer
	evaluator, err := NewSMTCostEvaluator(m, diagnostic.NewWarningService(&warnings))
	astTestMust(t, err)
	value, err := evaluator.Eval(formula, []float32{2, 4})
	astTestMust(t, err)
	if value != 1 || warnings.Len() != 0 {
		t.Fatal("source expression or reversed variable mapping", value, warnings.String())
	}
	value, err = evaluator.Eval(comparison, []float32{1, 2})
	astTestMust(t, err)
	if value != 0 {
		t.Fatal("comparison result", value)
	}
}

func TestSMTCostEvaluatorNumeralDivisionAndFallbackWarnings(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	threeHalves, err := m.ArithmeticRealNumeralText("3/2")
	astTestMust(t, err)
	zero, err := m.ArithmeticRealNumeralText("0")
	astTestMust(t, err)
	division, err := m.MkFamilyApp(arithmetic, smtArithOpDiv, nil, []ast.ASTHandle{threeHalves, zero})
	astTestMust(t, err)
	real, err := m.GetSort(zero)
	astTestMust(t, err)
	outOfRange, err := m.MkVar(2, real)
	astTestMust(t, err)
	var warnings bytes.Buffer
	evaluator, err := NewSMTCostEvaluator(m, diagnostic.NewWarningService(&warnings))
	astTestMust(t, err)
	value, err := evaluator.Eval(threeHalves, nil)
	astTestMust(t, err)
	if value != 1.5 {
		t.Fatal("rational conversion", value)
	}
	value, err = evaluator.Eval(division, nil)
	astTestMust(t, err)
	if value != 1 {
		t.Fatal("division fallback", value)
	}
	value, err = evaluator.Eval(outOfRange, []float32{9})
	astTestMust(t, err)
	if value != 1 {
		t.Fatal("variable fallback", value)
	}
	want := "WARNING: cost function division by zero\nWARNING: cost function evaluation error\n"
	if warnings.String() != want {
		t.Fatal("source warnings", warnings.String())
	}
}
