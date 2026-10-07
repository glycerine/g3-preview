// Source: src/math/polynomial/polynomial.cpp:3769-4506,5356-5480;
// src/util/rlimit.h; src/util/cooperate.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/clibrand"
)

func TestPolynomialGCDAndResultantEveryReachedCancellation(t *testing.T) {
	for _, gcd := range []bool{false, true} {
		t.Run(map[bool]string{false: "resultant", true: "default-gcd"}[gcd], func(t *testing.T) {
			success := false
			canceled := uint32(0)
			for cancel := uint32(1); cancel < 1024 && !success; cancel++ {
				t.Run(fmt.Sprint(cancel), func(t *testing.T) {
					full := polynomialPSCFullFixture(t)
					c := full.core
					p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
					q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
					var common Polynomial
					if gcd {
						common = polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
						a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
						b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
						var e error
						p, e = c.Mul(common, a)
						serviceMust(t, e)
						polynomialCaller(t, c, p)
						q, e = c.Mul(common, b)
						serviceMust(t, e)
						polynomialCaller(t, c, q)
					}
					r, e := NewPolynomialRef(c)
					serviceMust(t, e)
					t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
					var visits, cooperate uint32
					c.config.Checkpoint = func() bool { visits++; return visits != cancel }
					c.config.Cooperation = func() error { cooperate++; return nil }
					// Test control resets the actual shared source-compatible process stream;
					// production GCD never seeds it per manager or invocation.
					clibrand.Srand(1)
					var cause error
					if gcd {
						cause = full.GCD(p, q, r)
					} else {
						cause = full.Resultant(p, q, 1, r)
					}
					polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, common, r.root.value}, nil, r.header)
					if cause != nil {
						if !errors.Is(cause, ErrUPolynomialCanceled) || visits != cancel || cooperate+1 != visits {
							t.Fatal("exact resource-before-cooperation source frontier", cancel, cause, visits, cooperate)
						}
						canceled++
						serviceMust(t, full.CleanupGCD())
						polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, common, r.root.value}, nil, r.header)
						return
					}
					success = true
					if visits >= cancel || cooperate != visits {
						t.Fatal("actual complete source visit horizon", cancel, visits, cooperate)
					}
					if gcd {
						same, e := c.Equal(common, r.root.value)
						serviceMust(t, e)
						if !same {
							t.Fatal("independent GCD success identity")
						}
					} else {
						polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{2, 0, 0}: -1, {1, 0, 0}: 6})
					}
				})
			}
			if !success || canceled == 0 {
				t.Fatal("incomplete actual checkpoint horizon", canceled, success)
			}
		})
	}
}
