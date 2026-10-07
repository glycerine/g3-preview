// Source: src/math/polynomial/polynomial.cpp:5588-5658; src/util/ref_vector.h:96-100.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func pscChain2NamedAudit(t *testing.T, m *MultivariatePolynomialManager, out *PolynomialRefVector, p, q Polynomial) {
	t.Helper()
	polys := []Polynomial{p, q}
	for i := range m.pscChain2.refs {
		r := &m.pscChain2.refs[i].root
		polys = append(polys, r.value, r.incoming)
	}
	subresultantNamedAudit(t, m, out, polys, nil)
}
func TestPolynomialPSCChain2ActualEveryReservationPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 512; prefix++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 4}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
		out, e := NewPolynomialRefVector(c)
		serviceMust(t, e)
		t.Cleanup(func() {
			m.originalBudget.FailAfter(-1)
			serviceMust(t, m.CleanupPSCChain2())
			serviceMust(t, out.Close())
		})
		b := m.originalBudget
		b.FailAfter(prefix)
		returned := m.PSCChain2(p, q, 0, out)
		pscChain2NamedAudit(t, m, out, p, q)
		source, body := m.pscChain2.sourceCause, m.pscChain2.bodyCause
		own := m.CleanupPSCChain2()
		pscChain2NamedAudit(t, m, out, p, q)
		if m.pscChain2.sourceCause != source || m.pscChain2.bodyCause != body {
			t.Fatal("own acknowledgment erased history", prefix)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupPSCChain2())
		if own != nil && m.pscChain2.cleanupCause != own {
			t.Fatal("own first cause erased", prefix)
		}
		if returned != nil {
			message := returned.Error()
			oldPush := m.pscChain2.push
			inv := m.pscChain2.invocation
			serviceMust(t, m.PSCChain2(p, q, 0, out))
			serviceMust(t, oldPush.cleanup())
			if returned.Error() != message || m.pscChain2.invocation == inv {
				t.Fatal("later invocation changed original error", prefix)
			}
			pscChain2NamedAudit(t, m, out, p, q)
		}
		n, e := out.Size()
		serviceMust(t, e)
		if n != 2 {
			t.Fatal("compact published range", prefix, n)
		}
		for i, want := range []int32{8, -2} {
			v, e := out.Get(uint32(i))
			serviceMust(t, e)
			constant, e := c.IsConstant(v)
			serviceMust(t, e)
			value, e := c.Coefficient(v, 0)
			serviceMust(t, e)
			if !constant || value != util.NewInt(want) {
				t.Fatal("retry mathematics", prefix, i, value)
			}
		}
		serviceMust(t, out.Close())
		if returned == nil {
			success = true
			t.Log("first actual success", prefix)
			break
		}
	}
	if !success {
		t.Fatal("no actual success horizon")
	}
}
