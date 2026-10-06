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

// Actual Copy destroys the old cell before its refused replacement. This is
// the guarded owner-primitive destructor domain, not ordinary Hensel inputs.
func TestUPolynomialHenselReachedVectorSourceStopPreservesBacking(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, m.set(&c.fpp, []Int{q.z.two64}))
	var large Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&large, words))
	defer func() { serviceMust(t, q.z.Del(&large)) }()
	a, e := m.read(m.stateView(&c.fpp), 0)
	serviceMust(t, e)
	old := c.fpp
	q.budget.FailAfter(0)
	body := q.z.Copy(a, &large)
	if !numericPhysicalReservationRefusal(body) || a.cell != q.z.failedRef {
		t.Fatal("real destroyed old/refused replacement", body, *a)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &large)
	cursor := uint32(0)
	stop := upolynomialScopedVectorSourceDel(m, &c.fpp, &cursor)
	if !errors.Is(stop, arena.ErrInvalidSlot) || cursor != 0 || c.fpp != old {
		t.Fatal("source NUM guard must retain unfinished vector and backing", old, c.fpp, body)
	}
	q.budget.FailAfter(-1)
	serviceMust(t, s.CleanupRetained())
}
