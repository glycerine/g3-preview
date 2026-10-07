// Source: src/math/polynomial/polynomial.cpp:3432-3534,4352-4506,5904-5940;
// src/util/obj_ref.h; src/util/scoped_numeral.h.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialICCPInvocationReceiptRejectsReusedFrame(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 129}, {1, 1}}, {{1, 1}}})
	var i util.Int
	var content, pp polynomialReference
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupContentPrimitiveParts()) })
	var old, current polynomialICCPInvocationReceipt
	serviceMust(t, m.iccpPartsWithReceipt(p, 0, &i, &content, &pp, &old))
	serviceMust(t, content.close(c))
	serviceMust(t, pp.close(c))
	serviceMust(t, c.integer.Del(&i))
	if old.frame == nil || old.invocation == 0 {
		t.Fatal("actual guarded invocation missing")
	}
	c.integer.Budget().FailAfter(0)
	e := m.iccpPartsWithReceipt(p, 0, &i, &content, &pp, &current)
	if !numericPhysicalReservationRefusal(e) || current.frame != old.frame || current.invocation == old.invocation {
		t.Fatal("actual second guarded invocation", e, old, current)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&i}, 0)
	state := *current.frame
	used := c.integer.Budget().Used()
	serviceMust(t, m.cleanupICCPInvocation(&old, false))
	serviceMust(t, m.cleanupICCPInvocation(&old, true))
	foreign := current
	foreign.manager = &MultivariatePolynomialManager{}
	serviceMust(t, m.cleanupICCPInvocation(&foreign, true))
	if *current.frame != state || c.integer.Budget().Used() != used {
		t.Fatal("stale or foreign receipt consumed replacement")
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.cleanupICCPInvocation(&current, false))
	serviceMust(t, m.cleanupICCPInvocation(&current, true))
	serviceMust(t, content.close(c))
	serviceMust(t, pp.close(c))
	serviceMust(t, c.integer.Del(&i))
}

func TestPolynomialGCDInvocationReceiptRejectsReusedFrame(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	powers := make([]MonomialPower, 40)
	for j := range powers {
		powers[j] = MonomialPower{uint32(j), 1}
	}
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{powers, nil})
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupGCD()) })
	var r polynomialReference
	var old, current polynomialGCDInvocationReceipt
	serviceMust(t, m.gcdIntoWithReceipt(p, p, &r, &old))
	serviceMust(t, r.close(c))
	v := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	c.integer.Budget().FailAfter(0)
	e := m.gcdIntoWithReceipt(p, v, &r, &current)
	// The constant-content route constructs source numerals without a buffer grow;
	// use the actual nonconstant distinct input to reach variable-degree backing.
	if e == nil {
		serviceMust(t, r.close(c))
		c.integer.Budget().FailAfter(-1)
		t.Fatal("expected actual reservation refusal")
	}
	if !numericPhysicalReservationRefusal(e) || current.frame != old.frame || current.invocation == old.invocation {
		t.Fatal("actual reused GCD frame", e, old, current)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.value, r.incoming}, nil, 0)
	depth, used := current.frame.depth, c.integer.Budget().Used()
	serviceMust(t, m.cleanupGCDInvocation(&old, false))
	serviceMust(t, m.cleanupGCDInvocation(&old, true))
	if current.frame.depth != depth || c.integer.Budget().Used() != used {
		t.Fatal("old GCD receipt consumed replacement")
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.cleanupGCDInvocation(&current, false))
	serviceMust(t, m.cleanupGCDInvocation(&current, true))
	serviceMust(t, r.close(c))
}

func TestPolynomialNormalizeInvocationReceiptRejectsReusedFrame(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var large util.Int
	serviceMust(t, c.integer.SetLimbs(&large, []uint32{0, 0, 7}))
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&large)) })
	x, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{large, util.NewInt(7)}, []Monomial{x, c.monomials.unit})
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupNormalization()) })
	var old, current polynomialNormalizeInvocationReceipt
	result, e := m.normalizeWithReceipt(p, &old)
	serviceMust(t, e)
	var r polynomialReference
	serviceMust(t, r.assign(c, result))
	serviceMust(t, r.close(c))
	result, e = m.normalizeWithReceipt(p, &current)
	serviceMust(t, e)
	serviceMust(t, r.assign(c, result))
	serviceMust(t, r.close(c))
	if current.frame != old.frame || current.invocation == old.invocation {
		t.Fatal("same-address invocation was not unique", old, current)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&large}, 0)
	state, used := current.frame.cursor, c.integer.Budget().Used()
	serviceMust(t, m.cleanupNormalizeInvocation(&old, false))
	serviceMust(t, m.cleanupNormalizeInvocation(&old, true))
	if current.frame.cursor != state || c.integer.Budget().Used() != used {
		t.Fatal("old Normalize receipt consumed replacement")
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.cleanupNormalizeInvocation(&current, false))
	serviceMust(t, m.cleanupNormalizeInvocation(&current, true))
}

func TestPolynomialICCPExpandedFilterEveryActualReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 129}, {1, 1}}, {{1, 1}}})
			var i util.Int
			var content, pp polynomialReference
			var receipt polynomialICCPInvocationReceipt
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, m.CleanupContentPrimitiveParts())
				serviceMust(t, content.close(c))
				serviceMust(t, pp.close(c))
				serviceMust(t, c.integer.Del(&i))
			})
			c.integer.Budget().FailAfter(ordinal)
			e := m.iccpPartsWithReceipt(p, 0, &i, &content, &pp, &receipt)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation failure", e)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, content.value, content.incoming, pp.value, pp.incoming}, []*util.Int{&i}, 0)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.cleanupICCPInvocation(&receipt, false))
			serviceMust(t, m.cleanupICCPInvocation(&receipt, true))
			if e == nil {
				success = true
				if i != util.NewInt(2) {
					t.Fatal("independent integer content", i)
				}
				polynomialQueryExpect(t, c, content.value, map[[3]uint32]int64{{0, 1, 0}: 1})
				polynomialQueryExpect(t, c, pp.value, map[[3]uint32]int64{{129, 0, 0}: 1, {}: 2})
			}
		})
	}
	if !success {
		t.Fatal("bounded source filter never completed")
	}
}

func TestPolynomialICCPRetainsActualPendingGCDInvocation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			// With respect to y, coefficients x^2+x+1 and (x+1)^2 have gcd 1.
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1, 2, 1}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}, {1, 1}}, {{1, 1}}, {{0, 2}}, {{0, 1}}, nil})
			var i util.Int
			var content, pp polynomialReference
			var receipt polynomialICCPInvocationReceipt
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, m.CleanupGCD())
				serviceMust(t, m.CleanupContentPrimitiveParts())
				serviceMust(t, content.close(c))
				serviceMust(t, pp.close(c))
				serviceMust(t, c.integer.Del(&i))
			})
			c.integer.Budget().FailAfter(ordinal)
			e := m.iccpPartsWithReceipt(p, 1, &i, &content, &pp, &receipt)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual callee refusal", e)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, content.value, content.incoming, pp.value, pp.incoming}, []*util.Int{&i}, 0)
			// This is the explicit first actual gcd(c,ci,c) frame of this fixture.
			// Observation of its source state does not supply owning cleanup authority.
			if len(m.gcdFrames) != 0 && m.gcdFrames[0].active && !m.matchesICCPInvocation(&receipt) {
				t.Errorf("caller erased exact still-pending source GCD receipt")
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.cleanupICCPInvocation(&receipt, false))
			serviceMust(t, m.cleanupICCPInvocation(&receipt, true))
			if e == nil {
				success = true
				if i != util.NewInt(1) {
					t.Fatal("independent integer content", i)
				}
				one, e := c.Equal(content.value, c.unit)
				serviceMust(t, e)
				if !one {
					t.Fatal("independent coprime coefficients")
				}
			}
		})
	}
	if !success {
		t.Fatal("bounded actual ICCP/GCD never completed")
	}
}
