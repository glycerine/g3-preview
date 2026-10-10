// Source: src/math/polynomial/polynomial.cpp:5154-5194.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialDividesPersistentMemberReturnDoesNotConsumeRows(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	seven := unumeral.NewInt(7)
	p, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	var receipt polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&receipt))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &receipt))
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(foreign))
	defer func() { serviceMust(t, c.DecRef(foreign)) }()
	serviceMust(t, c.som.addPolynomial(foreign))
	beforeAs, e := c.som.as.size(c)
	serviceMust(t, e)
	beforeMs, e := c.som.ms.size(c)
	serviceMust(t, e)
	unitRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	fm, e := c.monomialCore(foreign, 0)
	serviceMust(t, e)
	foreignRC, e := c.monomialRefCount(fm)
	serviceMust(t, e)
	rawAs, rawMs, used := append([]unumeral.Int(nil), c.som.as.values...), append([]Monomial(nil), c.som.ms...), c.integer.Budget().Used()
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, foreign}, nil, 0)
	copyReceipt := receipt
	copyFrame := *receipt.frame
	copyReceipt.frame = &copyFrame
	serviceMust(t, c.som.transferArithmeticRowsToMember(&copyReceipt))
	if !receipt.frame.active || !c.som.appendScope.active {
		t.Fatal("copied frame acquired return handoff authority")
	}
	serviceMust(t, c.som.transferArithmeticRowsToMember(&receipt))
	if receipt.frame.active || c.som.appendScope.active {
		t.Fatal("normal return retained adapter ownership")
	}
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, true))
	as, e := c.som.as.size(c)
	serviceMust(t, e)
	ms, e := c.som.ms.size(c)
	serviceMust(t, e)
	rc, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	frc, e := c.monomialRefCount(fm)
	serviceMust(t, e)
	if as != beforeAs || ms != beforeMs || rc != unitRC || frc != foreignRC || !polynomialEqualNumerals(c.som.as.values, rawAs) || !polynomialEqualMonomials(c.som.ms, rawMs) || c.integer.Budget().Used() != used {
		t.Fatal("tag handoff mutated actual member or foreign rows")
	}
	v, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	equal, e := c.modular.Equal(&v, &seven)
	serviceMust(t, e)
	if !equal {
		t.Fatal("member coefficient changed")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, foreign}, nil, 0)
	serviceMust(t, c.som.reset())
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, foreign}, nil, 0)
	var replacement polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&replacement))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &replacement))
	serviceMust(t, c.som.transferArithmeticRowsToMember(&receipt))
	if !replacement.frame.active || !c.som.matchesAppendInvocation(&replacement.frame.append) {
		t.Fatal("old receipt handed off later invocation")
	}
	serviceMust(t, c.cleanupArithmeticInvocation(&replacement, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, foreign}, nil, 0)
}
