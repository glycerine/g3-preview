// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017,1023-1465;
// src/util/scoped_numeral.h:31-33. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Actual destroyed-old Copy is a guarded owner-primitive source-DT profile,
// not ordinary mathematical failure reachability in factor_square_free.
func TestUPolynomialMignotteSourceFinalizerPreservesOlderParent(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, s.manager.set(&c.fpp, []Int{NewInt(4), NewInt(0), NewInt(3)}))
	c.constructed = 2
	serviceMust(t, q.z.Copy(&c.p, &q.z.two64))
	r, e := s.beginMignotteInvocation()
	serviceMust(t, e)
	r.frame.constructed = 1
	serviceMust(t, q.z.Copy(&r.frame.norm, &q.z.two64))
	var large Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&large, words))
	defer func() { serviceMust(t, q.z.Del(&large)) }()
	older := c.p
	q.budget.FailAfter(0)
	body := s.mignotteNumericCall(&r, 4, &large, nil, &r.frame.norm, 0)
	r.frame.executing = false
	if !numericPhysicalReservationRefusal(body) || r.frame.norm.cell != q.z.failedRef {
		t.Fatal("actual destroyed-old/refused replacement", body)
	}
	stop := s.finishMignotteSource(&r, body)
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal(stop)
	}
	mignotteScopePhysicalAudit(t, q, s, &large)
	first := s.finishSquareFreeScopeError(body)
	if !errors.Is(first, body) || !errors.Is(first, arena.ErrInvalidSlot) || s.scope != c || c.p != older || c.constructed != 2 {
		t.Fatal("source finalizer acknowledged younger stop/drained older parent", first, c)
	}
	q.budget.FailAfter(-1)
	serviceMust(t, s.CleanupRetained())
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal("owning erased returned first cause")
	}
	next, e := s.newSquareFreeScope()
	serviceMust(t, e)
	nr, e := s.beginMignotteInvocation()
	serviceMust(t, e)
	nr.frame.constructed = 1
	nr.frame.executing = false
	serviceMust(t, s.finishMignotteSource(&nr, nil))
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal("new invocation changed earlier returned error")
	}
	if s.scope != next {
		t.Fatal("actual new scope")
	}
	serviceMust(t, s.CleanupRetained())
}
