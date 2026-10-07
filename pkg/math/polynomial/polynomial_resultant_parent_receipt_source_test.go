// Source: src/math/polynomial/polynomial.cpp:5356-5480,4507-4528;
// Go exact invoked Resultant/GCD frame boundary (invalid copied metadata).
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialResultantReceiptRejectsCopiedActualFrame(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 128 && !reached; prefix++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		var leading util.Int
		words := make([]uint32, 64)
		words[0], words[63] = 7, 1
		serviceMust(t, c.integer.SetLimbs(&leading, words))
		x2, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
		serviceMust(t, e)
		unit, e := c.monomials.MkMonomial(nil)
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{leading, util.NewInt(1)}, []Monomial{x2, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		q, e := m.Derivative(p, 0)
		serviceMust(t, e)
		q = polynomialCaller(t, c, q)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() {
			m.originalBudget.FailAfter(-1)
			serviceMust(t, m.CleanupGCD())
			serviceMust(t, r.Close())
			serviceMust(t, c.integer.Del(&leading))
		})
		m.originalBudget.FailAfter(prefix)
		var actual polynomialGCDInvocationReceipt
		body := m.resultantWithReceipt(p, q, 0, &r.root, &actual)
		if body != nil && m.matchesGCDInvocation(&actual) {
			reached = true
			copied := *actual.frame
			forged := actual
			forged.frame = &copied
			if m.matchesGCDInvocation(&forged) {
				t.Fatal("copied Resultant/GCD frame acquired real local ownership", prefix)
			}
			nums, refs, depth, used := actual.frame.nums, actual.frame.refs, actual.frame.depth, m.originalBudget.Used()
			pInfo, x := c.Info(p)
			serviceMust(t, x)
			qInfo, x := c.Info(q)
			serviceMust(t, x)
			serviceMust(t, m.cleanupGCDInvocation(&forged, true))
			ownerCopy := *m
			forged = actual
			forged.manager = &ownerCopy
			if ownerCopy.matchesGCDInvocation(&forged) {
				t.Fatal("copied owner acquired Resultant frame")
			}
			serviceMust(t, ownerCopy.cleanupGCDInvocation(&forged, true))
			if actual.frame.nums != nums || actual.frame.refs != refs || actual.frame.depth != depth || used != m.originalBudget.Used() || !actual.frame.active {
				t.Fatal("copied cleanup mutated actual stopped frame")
			}
			pAfter, x := c.Info(p)
			serviceMust(t, x)
			qAfter, x := c.Info(q)
			serviceMust(t, x)
			if pInfo != pAfter || qInfo != qAfter {
				t.Fatal("copied cleanup changed known caller payload/refcounts")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, q, r.root.value, r.root.incoming}, []*util.Int{&leading}, r.header)
		}
		m.originalBudget.FailAfter(-1)
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, r.Close())
		serviceMust(t, c.integer.Del(&leading))
	}
	if !reached {
		t.Fatal("actual stopped invoked Resultant frame absent")
	}
}
