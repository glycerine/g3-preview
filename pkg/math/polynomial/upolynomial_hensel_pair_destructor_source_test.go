// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/scoped_numeral_vector.h:29-40.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Actual Copy destroys the old cell before its refused replacement. This is
// the guarded owner-primitive destructor domain, not ordinary Hensel inputs.
func TestUPolynomialHenselReachedVectorSourceStopPreservesBacking(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, m.set(&c.fpp, []unumeral.Int{q.Integers().Two64}))
	var large unumeral.Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.Integers().SetLimbs(&large, words))
	defer func() { serviceMust(t, q.Integers().Del(&large)) }()
	a, e := m.read(m.stateView(&c.fpp), 0)
	serviceMust(t, e)
	old := c.fpp
	q.Budget().FailAfter(0)
	body := q.Integers().Copy(a, &large)
	if !numericPhysicalReservationRefusal(body) || (a).Cell != q.Integers().FailedRef {
		t.Fatal("real destroyed old/refused replacement", body, *a)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &large)
	cursor := uint32(0)
	stop := upolynomialScopedVectorSourceDel(m, &c.fpp, &cursor)
	if !errors.Is(stop, arena.ErrInvalidSlot) || cursor != 0 || c.fpp != old {
		t.Fatal("source NUM guard must retain unfinished vector and backing", old, c.fpp, body)
	}
	q.Budget().FailAfter(-1)
	serviceMust(t, s.CleanupRetained())
}
