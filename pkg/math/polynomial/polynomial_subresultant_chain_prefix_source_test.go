// Source: src/math/polynomial/polynomial.cpp:5499-5572;
// src/util/ref_vector.h:67-71,128-132.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func subresultantNamedAudit(t *testing.T, m *MultivariatePolynomialManager, out *PolynomialRefVector, polys []Polynomial, numerals []*util.Int) {
	t.Helper()
	s := &m.subresultant
	for i := range s.refs {
		polys = append(polys, s.refs[i].root.value, s.refs[i].root.incoming)
	}
	polys = append(polys, out.state.incoming)
	n, e := out.Size()
	serviceMust(t, e)
	for i := out.state.drain; i < n; i++ {
		p, e := out.Get(i)
		serviceMust(t, e)
		polys = append(polys, p)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, polys, numerals, out.header, nil)
}
func TestPolynomialSubresultantChainDefectiveActualEveryReservationPrefix(t *testing.T) {
	success, body, incomingSeen := false, false, false
	for prefix := int64(0); prefix < 1024; prefix++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		var big util.Int
		serviceMust(t, c.integer.SetLimbs(&big, []uint32{7, 11, 1 << 20}))
		term, e := c.monomials.MkVariable(0, 4)
		serviceMust(t, e)
		linear, e := c.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{util.NewInt(1), big, util.NewInt(1)}, []Monomial{term, linear, c.monomials.unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 2}}, nil})
		out, e := NewPolynomialRefVector(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.CleanupSubresultantChain()); serviceMust(t, out.Close()) })
		b := m.originalBudget
		b.FailAfter(prefix)
		e = m.SubresultantChain(p, q, 0, out)
		subresultantNamedAudit(t, m, out, []Polynomial{p, q}, []*util.Int{&big})
		if e != nil && m.subresultant.constructed > 0 {
			body = true
		}
		cause, bodyCause := m.subresultant.sourceCause, m.subresultant.bodyCause
		for i := range m.subresultant.refs {
			if m.subresultant.refs[i].root.incoming != (Polynomial{}) {
				incomingSeen = true
				if !m.subresultant.active {
					t.Fatal("actual assignment incoming INC lost before owning abort", prefix)
				}
			}
		}
		own := m.CleanupSubresultantChain()
		subresultantNamedAudit(t, m, out, []Polynomial{p, q}, []*util.Int{&big})
		if m.subresultant.bodyCause != bodyCause {
			t.Fatal("owning continuation erased actual body cause", prefix)
		}
		if cause != nil && m.subresultant.sourceCause != cause {
			t.Fatal("owning continuation erased first source cause", prefix)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupSubresultantChain())
		if own != nil && m.subresultant.cleanupCause != own {
			t.Fatal("owning continuation erased first cleanup cause", prefix)
		}
		subresultantNamedAudit(t, m, out, []Polynomial{p, q}, []*util.Int{&big})
		guardedHole := false
		if e != nil {
			message := e.Error()
			invocation := m.subresultant.invocation
			// mpz.h:156-163 frees the old shared TMP before RHS allocation.
			// This exact reached failure leaves a dangling C++ pointer; Go's
			// actual tmp[i]==0 is its guarded representation. Do not repair it
			// or generalize a different retry error to this source domain.
			guardedHole = c.integer.Tmp[0] == 0 || c.integer.Tmp[1] == 0
			retry := m.SubresultantChain(p, q, 0, out)
			if guardedHole {
				if !errors.Is(retry, arena.ErrInvalidSlot) {
					t.Fatal("exact TMP hole retry guard", prefix, retry)
				}
				subresultantNamedAudit(t, m, out, []Polynomial{p, q}, []*util.Int{&big})
				serviceMust(t, m.CleanupSubresultantChain())
			} else {
				serviceMust(t, retry)
			}
			if m.subresultant.invocation == invocation || e.Error() != message {
				t.Fatal("actual new invocation changed old returned error", prefix)
			}
		}
		if guardedHole {
			serviceMust(t, out.Close())
			serviceMust(t, c.integer.Del(&big))
			continue
		}

		// Independent resultant of 2*x^2+1 and x^4+B*x+1 is 25+8*B^2.
		var expected util.Int
		serviceMust(t, c.integer.Mul(&big, &big, &expected))
		eight := util.NewInt(8)
		serviceMust(t, c.integer.Mul(&expected, &eight, &expected))
		twentyFive := util.NewInt(25)
		serviceMust(t, c.integer.Add(&expected, &twentyFive, &expected))
		constant, e2 := out.Get(0)
		serviceMust(t, e2)
		isConstant, e2 := c.IsConstant(constant)
		serviceMust(t, e2)
		if !isConstant {
			t.Fatal("independent constant resultant", prefix)
		}
		value, e2 := c.Coefficient(constant, 0)
		serviceMust(t, e2)
		equal, e2 := c.integer.Equal(&value, &expected)
		serviceMust(t, e2)
		if !equal {
			t.Fatal("independent 25+8*B^2 identity", prefix)
		}
		serviceMust(t, c.integer.Del(&expected))
		if e == nil {
			success = true
			n, e := out.Size()
			serviceMust(t, e)
			if n != 5 {
				t.Fatal("literal chain size", n)
			}
			t.Log("actual first defective BIG chain success", prefix)
		}
		serviceMust(t, out.Close())
		serviceMust(t, c.integer.Del(&big))
		if success {
			break
		}
	}
	if !body || !success || !incomingSeen {
		t.Fatal("actual body/failure/incoming continuation horizon", body, success, incomingSeen)
	}
}
