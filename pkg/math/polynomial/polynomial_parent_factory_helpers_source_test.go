// Source: src/math/polynomial/polynomial.cpp:2757-2768,5940-5951.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialParentFactoryHelpersEveryActualPrefix(t *testing.T) {
	for _, negate := range []bool{false, true} {
		success := false
		for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
			t.Run(fmt.Sprint(negate, "/", ordinal), func(t *testing.T) {
				m := polynomialPSCFullFixture(t)
				c := m.core
				p := polynomialQueryTerms(t, c, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
				var appendReceipt polynomialCheapAppendInvocationReceipt
				var factoryReceipt polynomialFactoryInvocationReceipt
				var negReceipt polynomialNegInvocationReceipt
				var divReceipt polynomialScalarDivInvocationReceipt
				d := unumeral.NewInt(2)
				c.integer.Budget().FailAfter(ordinal)
				var result Polynomial
				var e error
				if negate {
					result, e = c.negWithParentReceipts(p, &negReceipt, &appendReceipt, &factoryReceipt)
				} else {
					result, e = c.scalarDivWithParentReceipts(p, &d, &divReceipt, &appendReceipt, &factoryReceipt)
				}
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.factoryInvocation.result, c.negativeResult, c.scalarDivision.returned}, nil, 0)
				if e == nil && (c.factoryInvocation.result != result || !c.matchesFactoryInvocation(&factoryReceipt)) {
					t.Fatal("actual factory did not retain its guarded parent authority")
				}
				if c.matchesFactoryInvocation(&factoryReceipt) && (c.negativeResult != (Polynomial{}) || c.scalarDivision.returned != (Polynomial{})) {
					t.Fatal("local NUM frame duplicated parent factory authority")
				}
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, c.cleanupNegInvocation(&negReceipt, false))
				serviceMust(t, c.cleanupScalarDivInvocation(&divReceipt, false))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.factoryInvocation.result}, nil, 0)
				serviceMust(t, c.cleanupFactoryInvocation(&factoryReceipt, true))
				serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
				serviceMust(t, c.cleanupNegInvocation(&negReceipt, true))
				serviceMust(t, c.cleanupScalarDivInvocation(&divReceipt, true))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
				if e == nil {
					success = true
					t.Log("first actual helper success", ordinal)
				}
			})
		}
		if !success {
			t.Fatal("actual helper horizon not reached")
		}
	}
}
