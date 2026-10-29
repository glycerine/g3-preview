package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::project, integer coefficient/divisor normalization.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestLegacyArithmeticProjectionNormalizesIntegerCoefficients(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-integer-x", f.integer)
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	two := f.own(m.ArithmeticIntegerNumeralText("2"))
	three := f.own(m.ArithmeticIntegerNumeralText("3"))
	four := f.own(m.ArithmeticIntegerNumeralText("4"))
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	twoX := f.own(m.MkArithmeticMul(two, x))
	dividend := f.own(m.MkArithmeticAdd(twoX, one))
	modulus := f.own(m.MkFamilyApp(f.r.arithmetic, 13, nil, []ast.ASTHandle{dividend, four}))
	divides := f.own(m.MkEq(modulus, zero))
	threeX := f.own(m.MkArithmeticMul(three, x))
	negativeThreeX := f.own(m.MkArithmeticNeg(threeX))
	boundTerm := f.own(m.MkArithmeticAdd(negativeThreeX, two))
	bound := f.own(m.MkArithmeticLE(boundTerm, zero))
	divLiteral, err := p.linearLiteral(x, divides)
	fmMust(t, err)
	boundLiteral, err := p.linearLiteral(x, bound)
	fmMust(t, err)
	if divLiteral == nil || boundLiteral == nil {
		t.Fatal("unclassified integer normalization input")
	}
	defer func() { fmMust(t, closeLegacyLinearLiterals(divLiteral, boundLiteral)) }()
	lcmCoefficients, lcmDivisors, err := p.normalizeIntegers([]*legacyLinearLiteral{divLiteral, boundLiteral})
	fmMust(t, err)
	if lcmCoefficients.Cmp(big.NewRat(6, 1)) != 0 || lcmDivisors.Cmp(big.NewRat(12, 1)) != 0 {
		t.Fatal("integer normalization LCMs", lcmCoefficients, lcmDivisors)
	}
	if divLiteral.coefficient.Cmp(big.NewRat(6, 1)) != 0 || divLiteral.divisor.Cmp(big.NewRat(12, 1)) != 0 {
		t.Fatal("normalized divisibility literal", divLiteral.coefficient, divLiteral.divisor)
	}
	if boundLiteral.coefficient.Cmp(big.NewRat(-6, 1)) != 0 || boundLiteral.divisor.Sign() != 0 {
		t.Fatal("normalized inequality literal", boundLiteral.coefficient, boundLiteral.divisor)
	}
	replacements, err := p.substituteIntegerLiterals(
		[]*legacyLinearLiteral{divLiteral, boundLiteral},
		[]ast.ASTHandle{divides, bound},
		four,
		1,
	)
	fmMust(t, err)
	if len(replacements.replacements) != 2 {
		t.Fatal("integer substitution replacement count", len(replacements.replacements))
	}
	divReplacement, err := m.RefGet(replacements.replacements[0].replacement)
	fmMust(t, err)
	contains, err := p.contains(divReplacement, x)
	fmMust(t, err)
	if contains {
		t.Fatal("integer divisibility replacement retains projected variable")
	}
	selectedReplacement, err := m.RefGet(replacements.replacements[1].replacement)
	fmMust(t, err)
	if !m.IsTrue(selectedReplacement) {
		t.Fatal("selected integer literal was not replaced by true")
	}
	fmMust(t, replacements.Close())
}
