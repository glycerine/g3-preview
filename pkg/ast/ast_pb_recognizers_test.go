package ast

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTPseudoBooleanOriginalRecognizersCoefficientsAndFreshAux(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	a, e := m.MkTrue()
	astTestMust(t, e)
	b, e := m.MkFalse()
	astTestMust(t, e)
	coefficients := []unumeral.OwnedRational{pbTestRational(t, m, "1"), pbTestRational(t, m, "2")}
	defer func() {
		for i := range coefficients {
			astTestMust(t, m.numerals.Del(&coefficients[i]))
		}
	}()
	bound := pbTestRational(t, m, "3")
	defer func() { astTestMust(t, m.numerals.Del(&bound)) }()
	h, e := m.MkPseudoBooleanGE(coefficients, []ASTHandle{a, b}, &bound)
	astTestMust(t, e)
	astTestPin(t, m, h)
	if yes, e := m.IsPseudoBooleanGE(h); e != nil || !yes {
		t.Fatal("PB >= recognizer", yes, e)
	}
	if yes, e := m.PseudoBooleanHasUnitCoefficients(h); e != nil || yes {
		t.Fatal("PB nonunit coefficients", yes, e)
	}
	gotBound, e := m.PseudoBooleanBound(h)
	astTestMust(t, e)
	text, e := m.numerals.String(&gotBound)
	astTestMust(t, e)
	astTestMust(t, m.numerals.Del(&gotBound))
	if text != "3" {
		t.Fatal("PB bound", text)
	}
	coefficient, e := m.PseudoBooleanCoefficient(h, 1)
	astTestMust(t, e)
	text, e = m.numerals.String(&coefficient)
	astTestMust(t, e)
	astTestMust(t, m.numerals.Del(&coefficient))
	if text != "2" {
		t.Fatal("PB coefficient", text)
	}
	aux, e := m.MkFreshPseudoBooleanAux()
	astTestMust(t, e)
	astTestPin(t, m, aux)
	if yes, e := m.IsPseudoBooleanAux(aux); e != nil || !yes {
		t.Fatal("PB auxiliary recognizer", yes, e)
	}
}
