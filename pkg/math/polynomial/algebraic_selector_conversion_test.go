// Source: algebraic_numbers.cpp:978-983, retained by binary/unary selection.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// (2^64*x-1)*(2^64*x-2), isolating the second root in (3/2^65,9/2^66).
func algebraicSelectorDyadicProductRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	z, p := c.config.Rationals.Integers(), c.config.Polynomials
	coeff := [3]unumeral.Int{unumeral.NewInt(2)}
	serviceMust(t, z.SetString(&coeff[1], "-55340232221128654848"))
	serviceMust(t, z.SetString(&coeff[2], "340282366920938463463374607431768211456"))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, coeff[:]))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	l, u := unumeral.NewBinaryIntPower(3, 65), unumeral.NewBinaryIntPower(9, 66)
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &l, &u, false))
	serviceMust(t, p.DelPolynomial(&poly))
	for i := range coeff {
		serviceMust(t, z.Del(&coeff[i]))
	}
	return a
}

func TestAlgebraicSelectorExactRootConversionReservations(t *testing.T) {
	succeeded, conversion, sourceStop := false, false, false
	for prefix := int64(0); prefix < 4096 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			// Product roots are {3,6,11,22}/2^64. The computed interval
			// (3,9)/2^64 has a root at its lower boundary. After selecting
			// its unique interior root, isolating2refinable finds the midpoint
			// 6/2^64 and enters set_core's scoped to_mpq branch.
			a := algebraicSelectorDyadicProductRoot(t, c)
			b := algebraicCompareRoot(t, c, []int32{33, -14, 1}, 2, 0, 4, 0, false)
			z := c.config.Rationals.Integers()
			var out Algebraic
			defer func() { serviceMust(t, c.Del(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
			z.Budget().FailAfter(prefix)
			body := c.Mul(&a, &b, &out)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("selector left actual reservation path", body)
			}
			pending := c.binary.conversion.pending(c)
			if unumeral.NumericAllocationPending(&c.binary.numericRoots) && (c.binary.owner != c || c.binary.numericHooks.Policy != &c.binary) {
				t.Fatal("selector allocation lost concrete binary caller identity")
			}
			if pending {
				conversion = true
				if body == nil {
					t.Fatal("conversion retained child after successful source call")
				}
				if c.binary.conversion.stop != nil {
					sourceStop = true
					if !c.binary.qConstructed || !c.binary.intervalConstructed {
						t.Fatal("younger conversion source stop destroyed older selector locals")
					}
				}
				if e := c.Neg(&a); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending selector conversion admitted public mutation", e)
				}
			}
			if e := c.CleanupRetained(); e != nil {
				t.Fatalf("cleanup: %v body: %v pending conversion: %v conversion owner: %v stop: %v selector stop: %v q: %v refinable: %v", e, body, pending, c.binary.conversion.owner == c, c.binary.conversion.stop, c.binary.sourceStop, c.binary.qConstructed, c.config.Polynomials.refinableReceiptPending(&c.binary.refinable))
			}
			if c.binary.active || c.binary.conversion.pending(c) {
				t.Fatal("selector failed to acknowledge exact conversion children")
			}
			algebraicExpectNoPendingAllocation(t, z)
			if succeeded {
				if c.binary.conversion.owner != c {
					t.Fatal("actual rational result bypassed tracked selector conversion")
				}
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&q, "3/9223372036854775808"))
				cmp, e := c.CompareRational(&out, &q)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("selected dyadic product differs from 3/2^63", cmp)
				}
				serviceMust(t, c.config.Rationals.Del(&q))
			}
		}) {
			return
		}
	}
	if !succeeded || !conversion {
		t.Fatal("sweep missed actual conversion refusal or success", succeeded, conversion, sourceStop)
	}
}
