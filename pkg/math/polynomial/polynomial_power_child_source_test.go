package polynomial

// Source: src/math/polynomial/polynomial.cpp:4600-4614,4905-4932.

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialPowerActualProductChildPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			c.integer.Budget().FailAfter(prefix)
			body := c.Power(p, 3, r)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, nil, r.header)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("literal Power body", prefix, body)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.CleanupPolynomialPower())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, nil, r.header)
			for _, b := range []*polynomialSOM{&c.som, &c.som2, &c.cheap, &c.cheap2} {
				as, e := b.as.size(c)
				serviceMust(t, e)
				ms, e := b.ms.size(c)
				serviceMust(t, e)
				if as != 0 || ms != 0 || b.hasPendingReference {
					t.Fatalf("Power's actual Mul child not retired prefix=%d as=%d ms=%d", prefix, as, ms)
				}
			}
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{3, 3, 0}: 8, {2, 2, 0}: 12, {1, 1, 0}: 6, {}: 1})
				t.Log("actual firstsuccess", prefix)
			}
		})
	}
	if !success {
		t.Fatal("actual first success absent")
	}
}

func TestPolynomialPowerActualReceiptQuickAndReuse(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	var out polynomialReference
	var first polynomialPowerInvocationReceipt
	serviceMust(t, c.powerIntoWithReceipt(p, 2, &out, &first))
	if first.manager != c || first.frame != &c.powerLocal || first.invocation == 0 {
		t.Fatal("real constructor identity")
	}
	old := first
	serviceMust(t, c.powerIntoWithReceipt(p, 0, &out, &first))
	if first != old {
		t.Fatal("zero shortcut changed incoming receipt")
	}
	serviceMust(t, c.powerIntoWithReceipt(p, 1, &out, &first))
	if first != old {
		t.Fatal("alias shortcut changed incoming receipt")
	}
	var later polynomialPowerInvocationReceipt
	serviceMust(t, c.powerIntoWithReceipt(p, 2, &out, &later))
	if later.invocation == old.invocation {
		t.Fatal("reused actual invocation")
	}
	copyFrame := *later.frame
	forged := later
	forged.frame = &copyFrame
	before := c.powerLocal
	serviceMust(t, c.cleanupPowerInvocation(&forged, true))
	serviceMust(t, c.cleanupPowerInvocation(&old, true))
	if c.powerLocal != before {
		t.Fatal("copied/stale receipt mutated current source history")
	}
	serviceMust(t, out.closeValue(c, true))
	serviceMust(t, out.abort(c))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
}

func TestPolynomialPowerMatchedAbortPreservesForeignRegularRow(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	foreign, e := c.monomials.MkVariable(2, 1)
	serviceMust(t, e)
	var out polynomialReference
	c.integer.Budget().FailAfter(2)
	body := c.powerInto(p, 3, &out)
	if !numericPhysicalReservationRefusal(body) || !c.powerLocal.active {
		t.Fatal("actual reached Power product refusal", body)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out.value, out.incoming}, nil, 0)
	c.integer.Budget().FailAfter(-1)
	a := unumeral.NewInt(7)
	serviceMust(t, c.som.add(&a, foreign, false))
	before, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	serviceMust(t, c.CleanupPolynomialPower())
	as, e := c.som.as.size(c)
	serviceMust(t, e)
	ms, e := c.som.ms.size(c)
	serviceMust(t, e)
	if as != 1 || ms != 1 {
		t.Fatal("matched cleanup erased later foreign row", as, ms)
	}
	term, e := c.som.ms.monomial(c, 0)
	serviceMust(t, e)
	value, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	after, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	if term != foreign || value != a || after != before {
		t.Fatal("foreign payload/ref changed", term, value, before, after)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out.value, out.incoming}, nil, 0)
	serviceMust(t, c.som.reset())
	serviceMust(t, out.closeValue(c, true))
	serviceMust(t, out.abort(c))
}

func TestPolynomialPowerSourceContinuationCopiedOwnerRefuses(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	var out polynomialReference
	c.integer.Budget().FailAfter(2)
	body := c.powerInto(p, 3, &out)
	if !numericPhysicalReservationRefusal(body) || !c.powerLocal.active {
		t.Fatal("real source failure", body)
	}
	copyOwner := *c
	before := copyOwner.powerLocal
	if e := copyOwner.finishPolynomialPowerSource(); e != ErrPolynomialOwner || copyOwner.powerLocal != before {
		t.Fatal("copied owner continued source destruction", e)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out.value, out.incoming}, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.CleanupPolynomialPower())
	serviceMust(t, out.closeValue(c, true))
	serviceMust(t, out.abort(c))
}

func TestPolynomialPowerBodyHistorySurvivesExactOwnACK(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	var output polynomialReference
	var receipt polynomialPowerInvocationReceipt
	c.integer.Budget().FailAfter(2)
	returned := c.powerIntoWithReceipt(p, 3, &output, &receipt)
	if !numericPhysicalReservationRefusal(returned) || !c.matchesPowerInvocation(&receipt) {
		t.Fatal("actual retained body refusal", returned)
	}
	body := c.powerInvocationBodyCause(&receipt)
	child := receipt.frame.arithmetic
	if body == nil || body != returned || child.invocation == 0 {
		t.Fatal("captured actual body/child identity", returned, body, child)
	}
	incoming := receipt
	savedOutput := output
	if e := c.powerIntoWithReceipt(p, 2, &output, &incoming); e != ErrPolynomialPending || incoming != receipt || output != savedOutput || c.powerInvocationBodyCause(&receipt) != body {
		t.Fatal("real guarded reentry changed caller receipt/output/history", e)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, output.value, output.incoming}, nil, 0)
	serviceMust(t, c.cleanupPowerInvocation(&receipt, false))
	if c.powerInvocationBodyCause(&receipt) != body || receipt.frame.arithmetic != child {
		t.Fatal("false ACK erased body/child history")
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cleanupPowerInvocation(&receipt, true))
	if c.powerInvocationBodyCause(&receipt) != body || receipt.frame.arithmetic != child || receipt.frame.active {
		t.Fatal("true ACK erased immutable body/child history")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, output.value, output.incoming}, nil, 0)
	var later polynomialPowerInvocationReceipt
	serviceMust(t, c.powerIntoWithReceipt(p, 2, &output, &later))
	if later.invocation == receipt.invocation || c.powerInvocationBodyCause(&later) != nil || c.powerInvocationBodyCause(&receipt) != nil || !numericPhysicalReservationRefusal(returned) {
		t.Fatal("actual new invocation/history partition")
	}
	serviceMust(t, output.closeValue(c, true))
	serviceMust(t, output.abort(c))
}
