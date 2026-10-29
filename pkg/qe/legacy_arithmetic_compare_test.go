package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::mk_lt, mk_le and mk_eq.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestLegacyArithmeticProjectionBuildsBoundComparisons(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-compare-x", f.real)
	y := f.constant("legacy-compare-y", f.real)
	z := f.constant("legacy-compare-z", f.real)
	zero := f.own(m.ArithmeticRealNumeralText("0"))
	two := f.own(m.ArithmeticRealNumeralText("2"))
	negX := f.own(m.MkArithmeticNeg(x))
	twoX := f.own(m.MkArithmeticMul(two, x))
	positiveTerm := f.own(m.MkArithmeticAdd(x, y))
	negativeTerm := f.own(m.MkArithmeticAdd(negX, z))
	strictPositiveTerm := f.own(m.MkArithmeticAdd(twoX, z))
	positive := f.own(m.MkArithmeticLE(positiveTerm, zero))
	negative := f.own(m.MkArithmeticLE(negativeTerm, zero))
	strictPositive := f.own(m.MkFamilyApp(f.r.arithmetic, 4, nil, []ast.ASTHandle{strictPositiveTerm, zero}))
	equalityExpression := f.own(m.MkEq(positiveTerm, zero))
	classify := func(expression ast.ASTHandle) *legacyLinearLiteral {
		literal, e := p.linearLiteral(x, expression)
		fmMust(t, e)
		if literal == nil {
			t.Fatal("unclassified comparison input")
		}
		return literal
	}
	positiveLiteral := classify(positive)
	negativeLiteral := classify(negative)
	strictPositiveLiteral := classify(strictPositive)
	equalityLiteral := classify(equalityExpression)
	defer func() {
		fmMust(t, closeLegacyLinearLiterals(positiveLiteral, negativeLiteral, strictPositiveLiteral, equalityLiteral))
	}()

	opposite, err := p.oppositeBound(positiveLiteral, negativeLiteral)
	fmMust(t, err)
	defer func() { fmMust(t, m.DelRef(&opposite)) }()
	oppositeHandle, err := m.RefGet(opposite)
	fmMust(t, err)
	contains, err := p.contains(oppositeHandle, x)
	fmMust(t, err)
	if contains {
		t.Fatal("opposite comparison retains projected variable")
	}
	op, err := m.GetDeclKind(oppositeHandle)
	fmMust(t, err)
	if op != 2 {
		t.Fatal("opposite non-strict comparison", op)
	}

	same, err := p.sameBound(strictPositiveLiteral, positiveLiteral)
	fmMust(t, err)
	defer func() { fmMust(t, m.DelRef(&same)) }()
	sameHandle, err := m.RefGet(same)
	fmMust(t, err)
	contains, err = p.contains(sameHandle, x)
	fmMust(t, err)
	if contains {
		t.Fatal("same-side comparison retains projected variable")
	}
	negated, inner, err := m.IsNot(sameHandle)
	fmMust(t, err)
	if !negated {
		t.Fatal("same-side strict comparison is not strict")
	}
	op, err = m.GetDeclKind(inner)
	fmMust(t, err)
	if op != 2 {
		t.Fatal("same-side rewritten strict comparison", op)
	}

	equalAtBound, err := p.equalityAtBound(equalityLiteral, negativeLiteral)
	fmMust(t, err)
	defer func() { fmMust(t, m.DelRef(&equalAtBound)) }()
	equalHandle, err := m.RefGet(equalAtBound)
	fmMust(t, err)
	contains, err = p.contains(equalHandle, x)
	fmMust(t, err)
	if contains {
		t.Fatal("equality comparison retains projected variable")
	}
	op, err = m.GetDeclKind(equalHandle)
	fmMust(t, err)
	if op != ast.ASTOpEq {
		t.Fatal("equality comparison", op)
	}
}
