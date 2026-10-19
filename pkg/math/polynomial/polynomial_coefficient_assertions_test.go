package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialCoefficientVariableAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, e := m.MkVariable(0, 1)
			serviceMust(t, e)
			polynomialCaller(t, m, p)
			choices := strings.Repeat(action, 64)
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			r, coeffErr := m.Coeff(p, 99, 0)
			var c unumeral.Int
			_, constErr := m.ConstantCoefficient(p, 99, 0, &c)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(coeffErr, ErrPolynomialAssertion) || !errors.Is(constErr, ErrPolynomialAssertion) {
					t.Fatal("variable assertion did not throw", coeffErr, constErr)
				}
			} else {
				serviceMust(t, coeffErr)
				serviceMust(t, constErr)
				polynomialCaller(t, m, r)
			}
		})
	}
}

func TestPolynomialUnivariateCoefficientAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{3, 7}, [][]MonomialPower{{{0, 2}}, {{1, 1}}})
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			got, e := m.UnivariateCoefficient(p, 2)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("univariate assertion did not throw", e)
				}
				return
			}
			serviceMust(t, e)
			value, e := m.integer.Int64(&got)
			serviceMust(t, e)
			if value != 3 {
				t.Fatal("continued total-degree coefficient scan", value)
			}
		})
	}
}

func TestPolynomialCoefficientBufferAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, e := m.MkVariable(0, 1)
			serviceMust(t, e)
			polynomialCaller(t, m, p)
			a := unumeral.NewInt(3)
			unit, e := m.monomials.Unit()
			serviceMust(t, e)
			serviceMust(t, m.cheap.add(&a, unit, false))
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 64))
			r, e := m.Coeff(p, 0, 1)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("nonempty buffer assertion did not throw", e)
				}
				serviceMust(t, m.cheap.reset())
			} else if polynomialSourceDebug {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("Continue did not reach the later dirty-buffer guard", e)
				}
				serviceMust(t, m.cheap.reset())
			} else {
				serviceMust(t, e)
				polynomialCaller(t, m, r)
			}
		})
	}
}
