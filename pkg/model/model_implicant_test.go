// Source: src/model/model_implicant.cpp:46-466,617-917.
package model_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func modelImplicantFixture(t *testing.T) (*ast.ASTManager, *model.Model, *model.ModelImplicant) {
	t.Helper()
	m, md, _, _, _, config := evaluatorFixture(t)
	i, err := model.NewModelImplicant(m, config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, i.Close()) })
	return m, md, i
}

func TestModelImplicantBooleanLiteralAndConeSelection(t *testing.T) {
	m, md, implicant := modelImplicantFixture(t)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	a := goalTestAtom(t, m, "implicant-a")
	b := goalTestAtom(t, m, "implicant-b")
	aDecl, err := m.GetDecl(a)
	astTestMust(t, err)
	bDecl, err := m.GetDecl(b)
	astTestMust(t, err)
	astTestMust(t, md.Core().RegisterConstant(aDecl, yes))
	astTestMust(t, md.Core().RegisterConstant(bDecl, no))
	or, err := m.MkOr(b, a)
	astTestMust(t, err)
	formula, err := m.MkAnd(a, or)
	astTestMust(t, err)
	astTestPin(t, m, formula)

	literals, err := implicant.MinimizeLiterals([]ast.ASTHandle{formula}, md)
	astTestMust(t, err)
	defer func() { astTestMust(t, model.CloseImplicantResults(m, &literals)) }()
	if len(literals) != 1 {
		t.Fatal("source OR witness and duplicate pruning", len(literals))
	}
	got, err := m.RefGet(literals[0])
	astTestMust(t, err)
	if got != a {
		t.Fatal("selected literal", got, a)
	}

	cone, err := implicant.MinimizeModel([]ast.ASTHandle{formula}, md)
	astTestMust(t, err)
	defer func() { astTestMust(t, model.CloseImplicantResults(m, &cone)) }()
	if len(cone) != 1 {
		t.Fatal("cone should retain only a", len(cone))
	}
	eq, err := m.RefGet(cone[0])
	astTestMust(t, err)
	left, err := m.GetArg(eq, 0)
	astTestMust(t, err)
	right, err := m.GetArg(eq, 1)
	astTestMust(t, err)
	if left != a || right != yes {
		t.Fatal("cone equality", left, right)
	}
}

func TestModelImplicantArithmeticDisequalitySplit(t *testing.T) {
	m, md, implicant := modelImplicantFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	two, err := m.ArithmeticIntegerNumeralText("2")
	astTestMust(t, err)
	intSort, err := m.GetSort(one)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "implicant-x"), intSort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "implicant-y"), intSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	astTestPin(t, m, y)
	xDecl, err := m.GetDecl(x)
	astTestMust(t, err)
	yDecl, err := m.GetDecl(y)
	astTestMust(t, err)
	astTestMust(t, md.Core().RegisterConstant(xDecl, one))
	astTestMust(t, md.Core().RegisterConstant(yDecl, two))
	eq, err := m.MkEq(x, y)
	astTestMust(t, err)
	formula, err := m.MkNot(eq)
	astTestMust(t, err)
	astTestPin(t, m, formula)

	literals, err := implicant.MinimizeLiterals([]ast.ASTHandle{formula}, md)
	astTestMust(t, err)
	defer func() { astTestMust(t, model.CloseImplicantResults(m, &literals)) }()
	if len(literals) != 1 {
		t.Fatal(len(literals))
	}
	lt, err := m.RefGet(literals[0])
	astTestMust(t, err)
	family, err := m.GetFamilyID(lt)
	astTestMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	op, err := m.GetDeclKind(lt)
	astTestMust(t, err)
	left, err := m.GetArg(lt, 0)
	astTestMust(t, err)
	right, err := m.GetArg(lt, 1)
	astTestMust(t, err)
	if family != arithmetic || op != 4 || left != x || right != y {
		t.Fatal("arithmetic disequality split", family, op, left, right)
	}
}

func TestModelImplicantEvalExpressionUsesCompletion(t *testing.T) {
	m, md, implicant := modelImplicantFixture(t)
	a := goalTestAtom(t, m, "implicant-eval")
	result, err := implicant.EvalExpr(md, a)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&result)) }()
	got, err := m.RefGet(result)
	astTestMust(t, err)
	if !m.IsFalse(got) {
		t.Fatal("model completion should choose false for missing Boolean", got)
	}
}
