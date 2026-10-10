// Source: src/math/polynomial/polynomial.cpp:5659-5692.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialLazardActualInvocationZeroShortcutAndForeignRefusal(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, out.Close()) })
	var receipt polynomialLazardInvocationReceipt
	serviceMust(t, m.seLazardWithReceipt(3, m.unit, p, 0, &out.root, &receipt))
	if !m.matchesLazardInvocation(&receipt) {
		t.Fatal("actual guarded invocation not captured")
	}
	old := receipt
	serviceMust(t, m.seLazardWithReceipt(2, Polynomial{owner: ^uint64(0)}, p, 0, &out.root, &receipt))
	if receipt != old {
		t.Fatal("source n==0 acquired a nonexistent frame")
	}
	copied := *old.frame
	foreign := old
	foreign.frame = &copied
	serviceMust(t, m.cleanupLazardInvocation(&foreign, false))
	serviceMust(t, m.cleanupLazardInvocation(&foreign, true))
}

func TestPolynomialLazardActualBigChildrenEveryReservationPrefix(t *testing.T) {
	success, arithmeticFailure, divisionFailure := false, false, false
	for prefix := int64(0); prefix < 512; prefix++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		var big unumeral.Int
		serviceMust(t, m.integer.SetLimbs(&big, []uint32{0, 0, 1 << 16}))
		linear, e := m.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		p, e := m.MkTerms([]unumeral.Int{big, unumeral.NewInt(1)}, []Monomial{linear, m.monomials.unit})
		serviceMust(t, e)
		p = polynomialCaller(t, m, p)
		y, e := m.MkTerms([]unumeral.Int{big}, []Monomial{m.monomials.unit})
		serviceMust(t, e)
		y = polynomialCaller(t, m, y)
		out, e := NewPolynomialRef(m)
		serviceMust(t, e)
		t.Cleanup(func() {
			full.originalBudget.FailAfter(-1)
			serviceMust(t, m.CleanupLazard())
			serviceMust(t, out.Close())
		})
		var receipt polynomialLazardInvocationReceipt
		full.originalBudget.FailAfter(prefix)
		returned := m.seLazardWithReceipt(4, y, p, 0, &out.root, &receipt)
		s := &m.lazardLocal
		audit := func() {
			polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, y, out.root.value, out.root.incoming, s.x.value, s.x.incoming, s.c.value, s.c.incoming}, []*unumeral.Int{&big}, out.header, nil)
		}
		audit()
		if returned != nil && s.phase == 2 {
			arithmeticFailure = true
		}
		if returned != nil && s.phase == 3 {
			divisionFailure = true
		}
		body, source := s.bodyCause, s.sourceCause
		own := m.cleanupLazardInvocation(&receipt, false)
		if own == nil {
			own = m.cleanupLazardInvocation(&receipt, true)
		}
		audit()
		if s.bodyCause != body || s.sourceCause != source {
			t.Fatal("own acknowledgment erased child history", prefix)
		}
		full.originalBudget.FailAfter(-1)
		serviceMust(t, m.cleanupLazardInvocation(&receipt, false))
		serviceMust(t, m.cleanupLazardInvocation(&receipt, true))
		if own != nil && s.cleanupCause != own {
			t.Fatal("first own cause erased", prefix)
		}
		if returned == nil {
			equal, e := m.Equal(out.root.value, p)
			serviceMust(t, e)
			if !equal {
				t.Fatal("independent cancellation identity", prefix)
			}
		}
		serviceMust(t, out.Close())
		serviceMust(t, m.integer.Del(&big))
		if returned == nil {
			success = true
			t.Log("actual first success", prefix)
			break
		}
	}
	if !success || !arithmeticFailure || !divisionFailure {
		t.Fatal("reached source children coverage", success, arithmeticFailure, divisionFailure)
	}
}

// Only the compiled unsigned assertion domain is tested here: literal 2*a
// wraps at uint32 width, after X construction and before C construction.
func TestPolynomialLazardUnsignedAssertionBeforeCConstruction(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	for _, d := range []uint32{uint32(1)<<31 + 1, 0} {
		full := polynomialPSCFullFixture(t)
		m := full.core
		out, e := NewPolynomialRef(m)
		serviceMust(t, e)
		var receipt polynomialLazardInvocationReceipt
		returned := m.seLazardWithReceipt(d, m.unit, m.unit, 0, &out.root, &receipt)
		s := &m.lazardLocal
		if !errors.Is(returned, ErrPolynomialAssertion) || s.constructed != 1 || s.cursor != 1 || s.c.value != (Polynomial{}) || s.c.incoming != (Polynomial{}) || out.root.value != (Polynomial{}) {
			t.Fatal("literal uint32 assertion/declaration prefix", d, returned, s.constructed, s.cursor)
		}
		body := s.bodyCause
		serviceMust(t, m.cleanupLazardInvocation(&receipt, false))
		serviceMust(t, m.cleanupLazardInvocation(&receipt, true))
		if s.bodyCause != body {
			t.Fatal("unsigned guard history erased")
		}
		polynomialImpPhysicalAuditNamedRaw(t, full, nil, nil, out.header, nil)
		serviceMust(t, out.Close())
	}
}
