package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::find_max.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArithmeticProjectionSelectsModelBounds(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	x := f.constant("legacy-bound-x", f.real)
	one := f.own(m.ArithmeticRealNumeralText("1"))
	three := f.own(m.ArithmeticRealNumeralText("3"))
	six := f.own(m.ArithmeticRealNumeralText("6"))
	eight := f.own(m.ArithmeticRealNumeralText("8"))
	two := f.own(m.ArithmeticRealNumeralText("2"))
	zero := f.own(m.ArithmeticRealNumeralText("0"))
	twoX := f.own(m.MkArithmeticMul(two, x))
	upperThree := f.own(m.MkArithmeticLE(x, three))
	upperFour := f.own(m.MkArithmeticLE(twoX, eight))
	falseUpper := f.own(m.MkArithmeticLE(x, zero))
	strictThree := f.own(m.MkFamilyApp(f.r.arithmetic, 4, nil, []ast.ASTHandle{twoX, six}))
	negX := f.own(m.MkArithmeticNeg(x))
	lowerZero := f.own(m.MkArithmeticLE(negX, zero))
	negTwoX := f.own(m.MkArithmeticNeg(twoX))
	negOne := f.own(m.ArithmeticRealNumeralText("-1"))
	lowerHalf := f.own(m.MkArithmeticLE(negTwoX, negOne))

	classify := func(expression ast.ASTHandle) *legacyLinearLiteral {
		literal, err := p.linearLiteral(x, expression)
		fmMust(t, err)
		if literal == nil {
			t.Fatal("unclassified bound")
		}
		return literal
	}
	upperThreeLiteral := classify(upperThree)
	upperFourLiteral := classify(upperFour)
	falseUpperLiteral := classify(falseUpper)
	strictThreeLiteral := classify(strictThree)
	lowerZeroLiteral := classify(lowerZero)
	lowerHalfLiteral := classify(lowerHalf)
	defer func() {
		fmMust(t, closeLegacyLinearLiterals(upperThreeLiteral, upperFourLiteral, falseUpperLiteral, strictThreeLiteral, lowerZeroLiteral, lowerHalfLiteral))
		fmMust(t, p.Close())
	}()

	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	xDecl, err := m.GetDecl(x)
	fmMust(t, err)
	fmMust(t, md.Core().RegisterConstant(xDecl, one))

	selected, err := p.findMax(md.Core(), x, []*legacyLinearLiteral{upperFourLiteral, falseUpperLiteral, upperThreeLiteral}, true)
	fmMust(t, err)
	if selected != 2 {
		t.Fatal("positive model bound", selected)
	}
	selected, err = p.findMax(md.Core(), x, []*legacyLinearLiteral{upperThreeLiteral, strictThreeLiteral}, true)
	fmMust(t, err)
	if selected != 1 {
		t.Fatal("strict tie bound", selected)
	}
	selected, err = p.findMax(md.Core(), x, []*legacyLinearLiteral{lowerZeroLiteral, lowerHalfLiteral}, false)
	fmMust(t, err)
	if selected != 1 {
		t.Fatal("negative model bound", selected)
	}
}
