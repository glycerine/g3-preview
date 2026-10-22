// Source: src/model/model_evaluator.cpp:149-193.
package model_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestModelEvaluatorOriginalTheoryReducerDispatch(t *testing.T) {
	m, _, evaluator, context, _, _ := evaluatorFixture(t)
	out := astTestPin(t, m, ast.ASTHandle{})
	evaluate := func(term ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		astTestPin(t, m, term)
		astTestMust(t, evaluator.Reset(nil))
		astTestMust(t, evaluator.Evaluate(term, out))
		result, err := m.RefGet(out)
		astTestMust(t, err)
		return result
	}

	numeral, err := context.NumeralsField.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, context.NumeralsField.Del(&numeral)) }()
	integer := func(value int64) ast.ASTHandle {
		t.Helper()
		astTestMust(t, context.NumeralsField.SetInt64(&numeral, value))
		h, err := m.ArithmeticNumeral(&numeral, false)
		astTestMust(t, err)
		return h
	}
	one, two, three := integer(1), integer(2), integer(3)
	arithmeticFamily, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	addition, err := m.MkFamilyApp(arithmeticFamily, 6, nil, []ast.ASTHandle{one, two})
	astTestMust(t, err)
	if got := evaluate(addition); got != three {
		t.Fatal("arithmetic reducer", got, three)
	}

	bitVectorFamily, err := m.BitVectorUtilityFamilyID()
	astTestMust(t, err)
	bvOne, err := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, err)
	bvTwo, err := m.BitVectorNumeralUnsigned(2, 8)
	astTestMust(t, err)
	bvThree, err := m.BitVectorNumeralUnsigned(3, 8)
	astTestMust(t, err)
	bvAddition, err := m.MkFamilyApp(bitVectorFamily, 4, nil, []ast.ASTHandle{bvOne, bvTwo})
	astTestMust(t, err)
	if got := evaluate(bvAddition); got != bvThree {
		t.Fatal("bit-vector reducer", got, bvThree)
	}

	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsehood, err := m.MkFalse()
	astTestMust(t, err)
	pseudoBoolean, err := m.MkPseudoBooleanAtLeast([]ast.ASTHandle{truth, falsehood}, 1)
	astTestMust(t, err)
	if got := evaluate(pseudoBoolean); !m.IsTrue(got) {
		t.Fatal("pseudo-Boolean reducer", got)
	}

	left, err := m.MkString("a")
	astTestMust(t, err)
	right, err := m.MkString("b")
	astTestMust(t, err)
	concatenation, err := m.MkSequenceConcat(left, right)
	astTestMust(t, err)
	got := evaluate(concatenation)
	text, matched, err := m.SequenceStringValue(got)
	astTestMust(t, err)
	if !matched || text != "ab" {
		t.Fatal("sequence reducer", matched, text)
	}

	datatypeFamily, err := m.DatatypeFamilyID()
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	name, err := m.NewParameterSymbol(astTestSymbol(t, m, "EvalPair"))
	astTestMust(t, err)
	datatypeSort, err := m.MkFamilySort(datatypeFamily, ast.ASTDatatypeSort, []ast.ASTParameter{name})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&name))
	constructorName, err := m.NewParameterSymbol(astTestSymbol(t, m, "eval-pair"))
	astTestMust(t, err)
	constructor, err := m.MkFamilyFuncDecl(datatypeFamily, ast.ASTDatatypeConstructor, []ast.ASTParameter{constructorName}, []ast.ASTHandle{boolSort, boolSort}, datatypeSort)
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&constructorName))
	constructorParameter, err := m.NewParameterAST(constructor)
	astTestMust(t, err)
	recognizer, err := m.MkFamilyFuncDecl(datatypeFamily, ast.ASTDatatypeIs, []ast.ASTParameter{constructorParameter}, []ast.ASTHandle{datatypeSort}, boolSort)
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&constructorParameter))
	pair, err := m.MkApp(constructor, []ast.ASTHandle{truth, falsehood})
	astTestMust(t, err)
	selection, err := m.MkApp(recognizer, []ast.ASTHandle{pair})
	astTestMust(t, err)
	if got := evaluate(selection); !m.IsTrue(got) {
		t.Fatal("datatype reducer", got)
	}
}
