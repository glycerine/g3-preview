// Source: src/math/polynomial/polynomial.cpp:5574-5583,5499-5572;
// src/util/ref_vector.h:128-132. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialPSCChain1ActualCoefficientAndSetEveryReservationPrefix(t *testing.T) {
	success, coefficientFailure := false, false
	for prefix := int64(0); prefix < 1024; prefix++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		var big util.Int
		serviceMust(t, c.integer.SetLimbs(&big, []uint32{11, 7, 1 << 20}))
		term := func(powers []MonomialPower) Monomial {
			t.Helper()
			v, e := c.monomials.MkMonomial(powers)
			serviceMust(t, e)
			return v
		}
		p, e := c.MkTerms([]util.Int{big, big, util.NewInt(1), util.NewInt(2)}, []Monomial{term([]MonomialPower{{0, 1}, {1, 2}}), term([]MonomialPower{{0, 1}, {1, 1}}), term([]MonomialPower{{1, 1}}), c.monomials.unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		q, e := c.MkTerms([]util.Int{big, util.NewInt(3)}, []Monomial{term([]MonomialPower{{0, 1}, {1, 1}}), c.monomials.unit})
		serviceMust(t, e)
		q = polynomialCaller(t, c, q)
		leading, e := c.MkTerms([]util.Int{big}, []Monomial{term([]MonomialPower{{0, 1}})})
		serviceMust(t, e)
		leading = polynomialCaller(t, c, leading)
		out, e := NewPolynomialRefVector(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain1()); serviceMust(t, out.Close()) })
		b := m.originalBudget
		b.FailAfter(prefix)
		returned := m.PSCChain1(p, q, 1, out)
		s := &m.pscChain1
		subresultantNamedAudit(t, m, out, []Polynomial{p, q, leading}, []*util.Int{&big})
		if returned != nil && s.phase == 2 {
			coefficientFailure = true
		}
		source, body := s.sourceCause, s.bodyCause
		own := m.CleanupPSCChain1()
		subresultantNamedAudit(t, m, out, []Polynomial{p, q, leading}, []*util.Int{&big})
		if s.sourceCause != source || s.bodyCause != body {
			t.Fatal("own ACK erased actual causes", prefix)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupPSCChain1())
		if own != nil && s.cleanupCause != own {
			t.Fatal("first owning cause changed", prefix)
		}
		if returned != nil {
			message := returned.Error()
			inv := s.invocation
			oldSub := s.subchain
			// Only the literal freed-old TMP hole qualifies guarded retry; no repair.
			hole := c.integer.Tmp[0] == 0 || c.integer.Tmp[1] == 0
			retry := m.PSCChain1(p, q, 1, out)
			if hole {
				if !errors.Is(retry, arena.ErrInvalidSlot) {
					t.Fatal("exact shared TMP hole", prefix, retry)
				}
				subresultantNamedAudit(t, m, out, []Polynomial{p, q, leading}, []*util.Int{&big})
				serviceMust(t, m.CleanupPSCChain1())
				serviceMust(t, out.Close())
				serviceMust(t, c.integer.Del(&big))
				continue
			}
			serviceMust(t, retry)
			serviceMust(t, m.cleanupSubresultantChainInvocation(&oldSub, false))
			serviceMust(t, m.cleanupSubresultantChainInvocation(&oldSub, true))
			if s.invocation == inv || returned.Error() != message {
				t.Fatal("same-owner retry retargeted old error", prefix)
			}
		}
		n, e := out.Size()
		serviceMust(t, e)
		if n != 3 {
			t.Fatal("literal captured source range", prefix, n)
		}
		got, e := out.Get(1)
		serviceMust(t, e)
		equal, e := c.Equal(got, leading)
		serviceMust(t, e)
		if !equal {
			t.Fatal("ascending coefficient source slot", prefix)
		}
		got, e = out.Get(2)
		serviceMust(t, e)
		if got != c.unit {
			t.Fatal("last source unit SET", prefix)
		}
		subresultantNamedAudit(t, m, out, []Polynomial{p, q, leading}, []*util.Int{&big})
		serviceMust(t, out.Close())
		serviceMust(t, c.integer.Del(&big))
		if returned == nil {
			success = true
			t.Log("actual first success", prefix)
			break
		}
	}
	if !success || !coefficientFailure {
		t.Fatal("actual caller branch coverage", success, coefficientFailure)
	}
}
