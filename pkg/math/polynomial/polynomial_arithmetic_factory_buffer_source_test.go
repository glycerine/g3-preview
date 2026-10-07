// Source: src/math/polynomial/polynomial.cpp:2137-2145,2481-2495.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialRegularFactoryReceiptBindsActualSourceBuffer(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialICCPUnownedTerms(t, c, []int32{7}, [][]MonomialPower{nil})
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	one := util.NewInt(1)
	serviceMust(t, c.som.addMul(&one, c.monomials.unit, p))
	var factory polynomialFactoryInvocationReceipt
	out, e := c.som.mkWithFactoryReceipt(false, &factory)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out}, nil, 0)
	if !c.matchesFactoryInvocation(&factory) || factory.frame.buffer != &c.som || factory.frame.result != out {
		t.Fatal("constructor outcome lost actual regular source-buffer identity")
	}
	serviceMust(t, c.transferFactoryResult(&factory, out))
	serviceMust(t, c.IncRef(out))
	serviceMust(t, c.DecRef(out))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
}
