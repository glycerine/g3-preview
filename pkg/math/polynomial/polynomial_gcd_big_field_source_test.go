// Source: src/math/polynomial/polynomial.cpp:3867-4230.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"math/big"
	"testing"
)

func TestPolynomialGCDRecursiveBigFieldEveryReservationPrefix(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			z := c.integer
			var prime unumeral.Int
			pv := new(big.Int).Lsh(big.NewInt(1), 127)
			pv.Sub(pv, big.NewInt(1))
			serviceMust(t, z.SetString(&prime, pv.String()))
			t.Cleanup(func() { serviceMust(t, z.Del(&prime)) })
			as := []unumeral.Int{{}, unumeral.NewInt(1), unumeral.NewInt(1)}
			serviceMust(t, z.SetLimbs(&as[0], []uint32{19, 0, 1}))
			t.Cleanup(func() { serviceMust(t, z.Del(&as[0])) })
			powers := [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil}
			ms := make([]Monomial, 3)
			for i := range ms {
				a, e := c.monomials.MkMonomial(powers[i])
				serviceMust(t, e)
				ms[i] = a
			}
			common, e := c.MkTerms(as, ms)
			serviceMust(t, e)
			polynomialCaller(t, c, common)
			a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
			b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
			p, e := c.Mul(common, a)
			serviceMust(t, e)
			polynomialCaller(t, c, p)
			q, e := c.Mul(common, b)
			serviceMust(t, e)
			polynomialCaller(t, c, q)
			serviceMust(t, c.modular.SetZP(&prime))
			serviceMust(t, full.newton.SetOwner(c))
			serviceMust(t, full.skeletons.Push(nil))
			serviceMust(t, full.skeletons.Push(nil))
			var vars []uint32

			vars = append(vars, 0)
			vars = append(vars, 1)
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				vars = nil
				serviceMust(t, r.Close())
			})
			z.Budget().FailAfter(ordinal)
			cause := full.modGCDRec(p, q, 11, 0, vars, &r.root)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, []*unumeral.Int{&prime, &as[0]}, r.header)
			z.Budget().FailAfter(-1)
			if cause != nil {
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual wide field dependency", ordinal, cause)
				}
				failures++
				serviceMust(t, full.CleanupGCD())
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, []*unumeral.Int{&prime, &as[0]}, r.header)
				return
			}
			success = true
			same, e := c.Equal(common, r.root.value)
			serviceMust(t, e)
			if !same {
				t.Fatal("independent wide field shared factor")
			}
		})
	}
	if !success || failures == 0 {
		t.Fatal("wide field refusal-to-success horizon", failures, success)
	}
}
