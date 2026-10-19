// Source: src/test/polynomial.cpp:702-723,741-821,1814-1825.
package polynomial

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialPSCRegisteredDriverAssertionsWithTraces(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("psc_chain_classic")
	tags.Enable("Lazard")
	tags.Disable("polynomial")
	m.monomials.config.Trace = tags
	assertions := 0
	for i, tc := range polynomialPSCOriginalCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p := pscOriginalPolynomial(t, m, tc.p)
			q := pscOriginalPolynomial(t, m, tc.q)
			first := pscOriginalPolynomial(t, m, tc.first)
			second := pscOriginalPolynomial(t, m, tc.second)
			chain, e := NewPolynomialRefVector(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, chain.Close()) })
			serviceMust(t, full.PSCChain(p, q, tc.x, chain))
			size, e := chain.Size()
			serviceMust(t, e)
			equalUpToSign := func(a, b Polynomial) {
				t.Helper()
				equal, e := m.Equal(a, b)
				serviceMust(t, e)
				if !equal {
					negative, e := m.Neg(b)
					serviceMust(t, e)
					serviceMust(t, m.IncRef(negative))
					equal, e = m.Equal(a, negative)
					serviceMust(t, e)
					serviceMust(t, m.DecRef(negative))
				}
				if !equal {
					t.Fatal("registered PSC ENSURE", i, assertions)
				}
				assertions++
			}
			if size > 0 {
				a, e := chain.Get(0)
				serviceMust(t, e)
				equalUpToSign(a, first)
			}
			if size > 1 {
				a, e := chain.Get(1)
				serviceMust(t, e)
				equalUpToSign(a, second)
			}
			if size > 0 {
				result, e := NewPolynomialRef(m)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, result.Close()) })
				serviceMust(t, full.Resultant(p, q, tc.x, result))
				a, e := chain.Get(0)
				serviceMust(t, e)
				equalUpToSign(a, result.root.value)
			}
		})
	}
	if assertions != 21 {
		t.Fatal("registered source assertion coverage", assertions)
	}
	if tags.ShouldTrace("psc_chain_classic") {
		if n := strings.Count(output.String(), "polynomial.cpp:5752 ---------"); n != 8 {
			t.Fatal("registered optimized PSC entry trace count", n)
		}
		if tags.ShouldTrace("Lazard") && !strings.Contains(output.String(), "[Lazard] Se_Lazard") {
			t.Fatal("registered fixtures omitted Lazard traces")
		}
	} else if output.Len() != 0 {
		t.Fatal("erased registered-driver traces emitted output")
	}
}
