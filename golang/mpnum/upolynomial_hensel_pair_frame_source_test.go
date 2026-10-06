// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/scoped_numeral_vector.h:29-40.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestUPolynomialHenselActualTmpStopProtectsOlderPairFields(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	r, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	h := r.frame
	h.rm, e = NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	h.constructed = 1
	five := NewInt(5)
	serviceMust(t, h.rm.SetZP(&five))
	for i := uint8(2); i <= 7; i++ {
		m, v := h.vector(i)
		h.constructed = i
		serviceMust(t, m.set(v, []Int{NewInt(1)}))
		a, e := m.read(m.stateView(v), 0)
		serviceMust(t, e)
		serviceMust(t, q.z.Copy(a, &q.z.two64))
	}
	var large Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&large, words))
	defer func() { serviceMust(t, q.z.Del(&large)) }()
	m, v := h.vector(7)
	tmp, e := m.read(m.stateView(v), 0)
	serviceMust(t, e)
	older := h.T
	q.budget.FailAfter(0)
	body := q.z.Copy(tmp, &large)
	if !numericPhysicalReservationRefusal(body) || tmp.cell != q.z.failedRef {
		t.Fatal("actual failed Copy", body, *tmp)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &large)
	h.executing = false
	stop := s.finishHenselPairSource(&r, body)
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) || h.sourceRemaining != 7 || h.T != older || h.rm.sourceFinalizing {
		t.Fatal("tmp must stop before T and related manager", stop, h)
	}
	q.budget.FailAfter(-1)
	serviceMust(t, s.cleanupHenselPairInvocation(&r, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&r, true))
	if !errors.Is(stop, body) || s.henselPairInvocationSourceCause(&r) == nil {
		t.Fatal("owning erased immutable source history")
	}
}
