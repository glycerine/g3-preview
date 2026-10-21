package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestArithmeticPolynomialShapeRecognizers(t *testing.T) {
	m, c, g, local := boolRewriterConfigFixture(t)
	key := "mul_to_power"
	astTestMust(t, c.SetBoolName(local, &key, true))
	r, err := NewArithRewriter(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	sort, err := m.MkFamilySort(r.family, 1, nil)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkVar(1, sort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	one := arithRewriterNumeral(t, m, "1", true)
	two := arithRewriterNumeral(t, m, "2", true)
	three := arithRewriterNumeral(t, m, "3", true)
	minusOne := arithRewriterNumeral(t, m, "-1", true)
	negative, err := r.app(arithOpMul, minusOne, x)
	astTestMust(t, err)
	astTestPin(t, m, negative)

	term, matched, err := r.IsTimesMinusOne(negative)
	astTestMust(t, err)
	if !matched || term != x {
		t.Fatal("arithmetic times-minus-one", term, matched)
	}
	if _, matched, err = r.IsTimesMinusOne(x); err != nil || matched {
		t.Fatal("arithmetic non-product", matched, err)
	}
	powerTwo, err := r.app(arithOpPower, x, two)
	astTestMust(t, err)
	astTestPin(t, m, powerTwo)
	powerThree, err := r.app(arithOpPower, x, three)
	astTestMust(t, err)
	astTestPin(t, m, powerThree)
	status, folded := arithRewriterCore(t, r, arithOpMul, powerTwo, powerThree, x)
	if status != RewriteDone {
		t.Fatal("adjacent powers status", status)
	}
	_, powerArgs, power, err := r.operation(folded, arithOpPower)
	astTestMust(t, err)
	if !power || len(powerArgs) != 2 || powerArgs[0] != x {
		t.Fatal("adjacent powers", power, len(powerArgs))
	}
	arithRewriterWantNumeral(t, m, powerArgs[1], "6", true)

	inner, err := r.app(arithOpAdd, negative, two)
	astTestMust(t, err)
	astTestPin(t, m, inner)
	expression, err := r.app(arithOpAdd, one, inner)
	astTestMust(t, err)
	astTestPin(t, m, expression)
	groundOut := astTestPin(t, m, expression)
	inverse, variable, matched, err := r.IsVarPlusGround(expression, groundOut)
	astTestMust(t, err)
	if !matched || !inverse || variable != x {
		t.Fatal("arithmetic variable plus ground", inverse, variable, matched)
	}
	ground, err := m.RefGet(groundOut)
	astTestMust(t, err)
	if ground != three {
		t.Fatal("arithmetic ground sum", ground, three)
	}

	twoVariables, err := r.app(arithOpAdd, x, y)
	astTestMust(t, err)
	astTestPin(t, m, twoVariables)
	if _, _, matched, err = r.IsVarPlusGround(twoVariables, groundOut); err != nil || matched {
		t.Fatal("arithmetic two-variable refusal", matched, err)
	}
	allGround, err := r.app(arithOpAdd, one, two)
	astTestMust(t, err)
	astTestPin(t, m, allGround)
	if _, _, matched, err = r.IsVarPlusGround(allGround, groundOut); err != nil || matched {
		t.Fatal("arithmetic ground refusal", matched, err)
	}
}

func TestBitVectorPolynomialShapeRecognizers(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, err := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, p.Close()) })
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkVar(1, sort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	one, err := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, err)
	astTestPin(t, m, one)
	two, err := m.BitVectorNumeralUnsigned(2, 8)
	astTestMust(t, err)
	astTestPin(t, m, two)
	three, err := m.BitVectorNumeralUnsigned(3, 8)
	astTestMust(t, err)
	astTestPin(t, m, three)
	minusOne, err := m.BitVectorNumeralUnsigned(255, 8)
	astTestMust(t, err)
	astTestPin(t, m, minusOne)
	negative, err := m.MkBitVectorPolynomialMul(sort, ast.SliceArguments([]ast.ASTHandle{minusOne, x}))
	astTestMust(t, err)
	astTestPin(t, m, negative)

	term, matched, err := p.IsTimesMinusOne(negative)
	astTestMust(t, err)
	if !matched || term != x {
		t.Fatal("bit-vector times-minus-one", term, matched)
	}
	if _, matched, err = p.IsTimesMinusOne(x); err != nil || matched {
		t.Fatal("bit-vector non-product", matched, err)
	}

	inner, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{negative, two}))
	astTestMust(t, err)
	astTestPin(t, m, inner)
	expression, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{one, inner}))
	astTestMust(t, err)
	astTestPin(t, m, expression)
	groundOut := astTestPin(t, m, expression)
	inverse, variable, matched, err := p.IsVarPlusGround(expression, groundOut)
	astTestMust(t, err)
	if !matched || !inverse || variable != x {
		t.Fatal("bit-vector variable plus ground", inverse, variable, matched)
	}
	ground, err := m.RefGet(groundOut)
	astTestMust(t, err)
	if ground != three {
		t.Fatal("bit-vector ground sum", ground, three)
	}

	twoVariables, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{x, y}))
	astTestMust(t, err)
	astTestPin(t, m, twoVariables)
	if _, _, matched, err = p.IsVarPlusGround(twoVariables, groundOut); err != nil || matched {
		t.Fatal("bit-vector two-variable refusal", matched, err)
	}
}
