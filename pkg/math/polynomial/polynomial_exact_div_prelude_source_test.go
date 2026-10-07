// Source: src/math/polynomial/polynomial.cpp:5114-5156,2067-2092.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialExactDivPreludeActualRowsBeforeNUMDeclaration(t *testing.T) {
	saw := false
	for prefix := int64(0); prefix < 40; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		seven, one := util.NewInt(7), util.NewInt(1)
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		q, e := c.MkConstantTakeCopy(&one)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(q))
		// The actual first source checkpoint isolates this prelude from later loop
		// allocation. It is a literal resource frontier, not a fabricated failpoint.
		c.config.Checkpoint = func() bool { return false }
		var r polynomialExactDivInvocationReceipt
		c.integer.Budget().FailAfter(prefix)
		_, e = c.exactDivWithReceipt(p, q, &r)
		if !errors.Is(e, ErrUPolynomialCanceled) && !numericPhysicalReservationRefusal(e) {
			t.Fatal("source prelude/cancel category", prefix, e)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
		if r.prelude.invocation == 0 {
			t.Fatal("actual SOM prelude absent before NUM/ref ctor", prefix, e)
		}
		if r.invocation == 0 {
			copied := r
			frame := *r.prelude.frame
			copied.prelude.frame = &frame
			serviceMust(t, c.cleanupExactDivInvocation(&copied, true))
			if !c.matchesArithmeticInvocation(&r.prelude) {
				t.Fatal("copied frame consumed actual prelude")
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
		}
		serviceMust(t, c.cleanupExactDivInvocation(&r, false))
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupExactDivInvocation(&r, true))
		n, x := c.som.ms.size(c)
		serviceMust(t, x)
		a, x := c.som.as.size(c)
		serviceMust(t, x)
		position, x := c.som.positions.Get(c.monomials.unit)
		serviceMust(t, x)
		if n != 0 || a != 0 || position != ^uint32(0) {
			t.Fatal("matched prelude abort retained rows/position", prefix, n, a, position)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
		// An actual later invocation owns its newly acquired rows. The old
		// receipt may retain history but cannot consume that invocation.
		var later polynomialExactDivInvocationReceipt
		_, laterCause := c.exactDivWithReceipt(p, q, &later)
		if !errors.Is(laterCause, ErrUPolynomialCanceled) {
			t.Fatal("same-owner retry", laterCause)
		}
		serviceMust(t, c.cleanupExactDivInvocation(&r, true))
		rows, x := c.som.ms.size(c)
		serviceMust(t, x)
		if rows != 1 || !c.matchesArithmeticInvocation(&later.prelude) {
			t.Fatal("stale prelude consumed later invocation", rows)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
		serviceMust(t, c.cleanupExactDivInvocation(&later, false))
		serviceMust(t, c.cleanupExactDivInvocation(&later, true))
		if !errors.Is(e, ErrUPolynomialCanceled) && !numericPhysicalReservationRefusal(e) {
			t.Fatal("old cause changed")
		}
		if r.invocation == 0 {
			saw = true
		}
		if errors.Is(e, ErrUPolynomialCanceled) {
			if !saw {
				t.Fatal("no failed pre-NUM source prefix")
			}
			serviceMust(t, c.DecRef(p))
			serviceMust(t, c.DecRef(q))
			return
		}
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.DecRef(q))
	}
	t.Fatal("real source prelude horizon not reached")
}

func TestPolynomialExactDivCompletedSourceResetPermitsLiteralRetry(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	c.config.Checkpoint = func() bool { return false }
	_, body := c.ExactDiv(p, q)
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("actual source checkpoint", body)
	}
	polynomialExactDivisionPhysical(t, c, []Polynomial{p, q})
	old := c.exactDivision.directReceipt
	c.config.Checkpoint = func() bool { return true }
	result, e := c.ExactDiv(p, q)
	if e != nil {
		t.Fatal("literal R.reset before next actual invocation must consume completed rows", e)
	}
	used := c.integer.Budget().Used()
	serviceMust(t, c.cleanupExactDivInvocation(&old, false))
	serviceMust(t, c.cleanupExactDivInvocation(&old, true))
	if used != c.integer.Budget().Used() || !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("older exact receipt changed replacement or original exception")
	}
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
	serviceMust(t, c.CleanupValue(&result))
}
