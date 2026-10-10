// Source: src/math/polynomial/polynomial.cpp:6898-6922; src/util/mpz.cpp:1186-1212.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialModDBigCoefficientEveryActualRefusal(t *testing.T) {
	success, failures, nullPrefixes := false, 0, 0
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			d, e := NewPolynomialVariableDegrees(m.memory)
			serviceMust(t, e)
			serviceMust(t, d.SetDegree(0, 2))
			big := unumeral.Int{}
			serviceMust(t, m.integer.SetLimbs(&big, []uint32{17, 0, 0, 1}))
			x, e := m.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := m.MkTerms([]unumeral.Int{big}, []Monomial{x})
			serviceMust(t, e)
			serviceMust(t, m.IncRef(p))
			result := Polynomial{}
			t.Cleanup(func() {
				m.integer.Budget().FailAfter(-1)
				if result != (Polynomial{}) {
					serviceMust(t, m.DecRef(result))
				}
				serviceMust(t, m.DecRef(p))
				serviceMust(t, m.integer.Del(&big))
				serviceMust(t, d.CloseChecked())
			})
			m.integer.Budget().FailAfter(ordinal)
			result, e = m.ModD(p, d)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(result))
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation cause", e)
				}
			}
			for i := uint32(0); i < uint32(len(m.cheap.as.values)); i++ {
				a, e := m.cheap.as.coefficient(m, i)
				serviceMust(t, e)
				if (a).Cell == m.integer.FailedNullRef {
					nullPrefixes++
				}
			}
			polynomialModDNewtonPhysicalWithRaw(t, m, nil, nil, nil, nil, []Polynomial{p, result}, nil, nil, []*PolynomialVariableDegrees{d}, &big)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p, result}, nil)
			m.integer.Budget().FailAfter(-1)
			if e == nil {
				equal, e := m.Equal(p, result)
				serviceMust(t, e)
				if !equal {
					t.Fatal("surviving BIG term changed")
				}
			}
		})
	}
	if !success || failures == 0 || nullPrefixes == 0 {
		t.Fatal("every actual refusal/first success/fresh NULL proof", success, failures, nullPrefixes)
	}
	t.Log("actual refusals", failures, "first success", 1, "fresh NULL prefixes", nullPrefixes)
}
