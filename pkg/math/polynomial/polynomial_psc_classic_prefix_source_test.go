// Source: src/math/polynomial/polynomial.cpp:5817-5895; src/util/ref_vector.h:96-100.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func pscClassicNamedAudit(t *testing.T, full *MultivariatePolynomialManager, out *PolynomialRefVector, p, q Polynomial) {
	t.Helper()
	ps := []Polynomial{p, q}
	for i := range full.core.pscClassic.refs {
		r := &full.core.pscClassic.refs[i].root
		ps = append(ps, r.value, r.incoming)
	}
	subresultantNamedAudit(t, full, out, ps, nil)
}
func TestPolynomialPSCClassicActualEveryReservationPrefix(t *testing.T) {
	for route := 0; route < 2; route++ {
		success := false
		ctor12, ctor13 := false, false
		for prefix := int64(0); prefix < 1024; prefix++ {
			full := polynomialPSCFullFixture(t)
			c := full.core
			var p, q Polynomial
			want := []int32{8, -2}
			if route == 0 {
				p = polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 4}}, {{0, 1}}, nil})
				q = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			} else {
				p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil})
				q = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 3}}, nil})
				want = []int32{2, 1}
			}
			out, e := NewPolynomialRefVector(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				full.originalBudget.FailAfter(-1)
				serviceMust(t, c.CleanupPSCClassic())
				serviceMust(t, out.Close())
			})
			b := full.originalBudget
			b.FailAfter(prefix)
			var receipt polynomialPSCClassicInvocationReceipt
			returned := c.pscChainClassicWithReceipt(p, q, 0, out, &receipt)
			pscClassicNamedAudit(t, full, out, p, q)
			s := &c.pscClassic
			if returned != nil && s.constructed == 12 {
				ctor12 = true
			}
			if returned != nil && s.constructed == 13 {
				ctor13 = true
			}
			copied := *s
			wrong := receipt
			wrong.frame = &copied
			state := c.memory.State()
			serviceMust(t, c.cleanupPSCClassicInvocation(&wrong, true))
			if c.memory.State() != state {
				t.Fatal("copied frame consumed real roots")
			}
			body, source := s.bodyCause, s.sourceCause
			own := c.cleanupPSCClassicInvocation(&receipt, false)
			if own == nil {
				own = c.cleanupPSCClassicInvocation(&receipt, true)
			}
			pscClassicNamedAudit(t, full, out, p, q)
			if s.bodyCause != body || s.sourceCause != source {
				t.Fatal("own ack erased source/body history", prefix)
			}
			b.FailAfter(-1)
			serviceMust(t, c.cleanupPSCClassicInvocation(&receipt, false))
			serviceMust(t, c.cleanupPSCClassicInvocation(&receipt, true))
			if own != nil && s.cleanupCause != own {
				t.Fatal("first own cause erased", prefix)
			}
			if returned != nil {
				message := returned.Error()
				old := receipt
				serviceMust(t, c.pscChainClassicWithReceipt(p, q, 0, out, &receipt))
				state = c.memory.State()
				serviceMust(t, c.cleanupPSCClassicInvocation(&old, true))
				if c.memory.State() != state || receipt.invocation == old.invocation || returned.Error() != message {
					t.Fatal("old history/receipt retargeted later invocation", prefix)
				}
				pscClassicNamedAudit(t, full, out, p, q)
			}
			n, e := out.Size()
			serviceMust(t, e)
			if n != 2 {
				t.Fatal("compact source range", prefix, n)
			}
			for i, w := range want {
				v, e := out.Get(uint32(i))
				serviceMust(t, e)
				constant, e := c.IsConstant(v)
				serviceMust(t, e)
				a, e := c.Coefficient(v, 0)
				serviceMust(t, e)
				if !constant || a != util.NewInt(w) {
					t.Fatal("independent retry math", prefix, i, a, w)
				}
			}
			serviceMust(t, out.Close())
			if returned == nil {
				success = true
				t.Log("actual first success", route, prefix)
				break
			}
		}
		if !success || !ctor12 || !ctor13 {
			t.Fatal("actual ctor/leaf horizon", success, ctor12, ctor13)
		}
	}
}
