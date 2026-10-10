// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/mpz.cpp:2054-2069; src/util/scoped_numeral.h:31-33.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialDividesBIGActualEveryReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 512 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialArithmeticDirectNumericFixture(t)
			c := full.core
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			var big unumeral.Int
			serviceMust(t, c.integer.SetLimbs(&big, []uint32{0, 0, 0, 0, 1}))
			t.Cleanup(func() { serviceMust(t, c.integer.Del(&big)) })
			term, e := c.monomials.MkVariable(0, 5)
			serviceMust(t, e)
			p, e := c.MkTerms([]unumeral.Int{big}, []Monomial{term})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}}})
			var spare [16]unumeral.Int
			named := []*unumeral.Int{&big}
			used := 0
			for c.integer.Cells.Stats().Slots > c.integer.Cells.Stats().Live+c.integer.Cells.Stats().Retired || c.integer.Cells.Stats().Slots < c.integer.Cells.Stats().Pages*2 {
				if used == len(spare) {
					t.Fatal("actual descriptor setup exceeded bounded caller roots")
				}
				serviceMust(t, c.integer.Copy(&spare[used], &big))
				named = append(named, &spare[used])
				used++
			}
			t.Cleanup(func() {
				for i := 0; i < used; i++ {
					serviceMust(t, c.integer.Del(&spare[i]))
				}
			})
			var receipt polynomialDividesInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			got, cause := c.dividesWithReceipt(q, p, &receipt)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, named, 0)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual BIG source refusal", ordinal, cause)
			}
			// The complete immediate source graph precedes fault lift. ID recycling
			// in owning monomial destruction can itself reserve, unlike NUM cleanup.
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
			serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, named, 0)
			if c.dividesLocal.active {
				t.Fatal("matched BIG invocation retained owning authority", ordinal)
			}
			coefficient, e := c.Coefficient(p, 0)
			serviceMust(t, e)
			equal, e := c.modular.Equal(&coefficient, &big)
			serviceMust(t, e)
			degree, e := c.Degree(p, 0)
			serviceMust(t, e)
			if !equal || degree != 5 {
				t.Fatal("source changed independent nonaliased 2^128*x^5 input")
			}
			if cause == nil {
				if !got {
					t.Fatal("independent x^2 divides 2^128*x^5")
				}
				success = true
				t.Log("actual BIG first complete prefix", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("actual BIG source campaign did not reach success")
	}
}
