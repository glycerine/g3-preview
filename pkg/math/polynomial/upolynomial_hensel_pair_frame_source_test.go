package polynomial

// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/scoped_numeral_vector.h:29-40.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialHenselActualTmpStopProtectsOlderPairFields(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	r, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	h := r.frame
	h.rm, e = NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	h.constructed = 1
	five := unumeral.NewInt(5)
	serviceMust(t, h.rm.SetZP(&five))
	for i := uint8(2); i <= 7; i++ {
		m, v := h.vector(i)
		h.constructed = i
		serviceMust(t, m.set(v, []unumeral.Int{unumeral.NewInt(1)}))
		a, e := m.read(m.stateView(v), 0)
		serviceMust(t, e)
		serviceMust(t, q.Integers().Copy(a, &q.Integers().Two64))
	}
	var large unumeral.Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.Integers().SetLimbs(&large, words))
	defer func() { serviceMust(t, q.Integers().Del(&large)) }()
	m, v := h.vector(7)
	tmp, e := m.read(m.stateView(v), 0)
	serviceMust(t, e)
	older := h.T
	q.Budget().FailAfter(0)
	body := q.Integers().Copy(tmp, &large)
	if !numericPhysicalReservationRefusal(body) || (tmp).Cell != q.Integers().FailedRef {
		t.Fatal("actual failed Copy", body, *tmp)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &large)
	h.executing = false
	stop := s.finishHenselPairSource(&r, body)
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) || h.sourceRemaining != 7 || h.T != older || h.rm.sourceFinalizing {
		t.Fatal("tmp must stop before T and related manager", stop, h)
	}
	q.Budget().FailAfter(-1)
	serviceMust(t, s.cleanupHenselPairInvocation(&r, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&r, true))
	if !errors.Is(stop, body) || s.henselPairInvocationSourceCause(&r) == nil {
		t.Fatal("owning erased immutable source history")
	}
}
