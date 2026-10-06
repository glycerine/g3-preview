// Source: src/math/polynomial/polynomial.cpp:5473-5500;
// Go exact invoked Resultant/GCD frame boundary (invalid copied metadata).
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialDiscriminantResultantReceiptRejectsCopiedActualFrame(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 128 && !reached; prefix++ {
		m := polynomialDiscriminantFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		var leading Int
		words := make([]uint32, 64)
		words[0], words[63] = 7, 1
		serviceMust(t, c.integer.SetLimbs(&leading, words))
		x2, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
		serviceMust(t, e)
		unit, e := c.monomials.MkMonomial(nil)
		serviceMust(t, e)
		p, e := c.MkTerms([]Int{leading, NewInt(1)}, []Monomial{x2, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() {
			m.originalBudget.FailAfter(-1)
			serviceMust(t, m.CleanupDiscriminant())
			serviceMust(t, r.Close())
			serviceMust(t, c.integer.Del(&leading))
		})
		m.originalBudget.FailAfter(prefix)
		body := m.Discriminant(p, 0, r)
		actual := m.discriminant.resultant
		if body != nil && m.matchesGCDInvocation(&actual) {
			reached = true
			copied := *actual.frame
			forged := actual
			forged.frame = &copied
			if m.matchesGCDInvocation(&forged) {
				t.Fatal("copied Resultant/GCD frame acquired real local ownership", prefix)
			}
			nums, depth, used := actual.frame.nums, actual.frame.depth, m.originalBudget.Used()
			serviceMust(t, m.cleanupGCDInvocation(&forged, true))
			ownerCopy := *m
			forged = actual
			forged.manager = &ownerCopy
			if ownerCopy.matchesGCDInvocation(&forged) {
				t.Fatal("copied owner acquired Resultant frame")
			}
			serviceMust(t, ownerCopy.cleanupGCDInvocation(&forged, true))
			if actual.frame.nums != nums || actual.frame.depth != depth || used != m.originalBudget.Used() || !actual.frame.active {
				t.Fatal("copied cleanup mutated actual stopped frame")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.root.value, r.root.incoming, m.discriminant.refs[0].value}, []*Int{&leading, &m.discriminant.lc}, r.header)
		}
		m.originalBudget.FailAfter(-1)
		serviceMust(t, m.CleanupDiscriminant())
		serviceMust(t, r.Close())
		serviceMust(t, c.integer.Del(&leading))
	}
	if !reached {
		t.Fatal("actual stopped invoked Resultant frame absent")
	}
}
