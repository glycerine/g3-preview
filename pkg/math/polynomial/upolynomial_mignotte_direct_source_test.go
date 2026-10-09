// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/scoped_numeral.h:31-33.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// Real Copy destroys the old BIG cell before refusing its replacement. This
// fixture checks the guarded owner-primitive destructor domain, not ordinary
// mathematical reachability of a dangling Mignotte temporary.
func TestUPolynomialMignotteActualYoungerNUMStopsOlderFields(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	primes, e := primegen.NewGenerator(m.originalBudget)
	serviceMust(t, e)
	defer func() { serviceMust(t, primes.Close()) }()
	s, e := NewUPolynomialFactorizationService(m.upm, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	r, e := s.beginMignotteInvocation()
	serviceMust(t, e)
	q := r.frame
	q.constructed = 3
	z := m.core.integer
	serviceMust(t, z.Copy(&q.norm, &z.Two64))
	serviceMust(t, z.Copy(&q.bound, &z.Two64))
	serviceMust(t, z.Copy(&q.tmp, &z.Two64))
	var large util.Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, z.SetLimbs(&large, words))
	defer func() { serviceMust(t, z.Del(&large)) }()
	norm, bound := q.norm, q.bound
	z.Budget().FailAfter(0)
	body := z.Copy(&q.tmp, &large)
	if !numericPhysicalReservationRefusal(body) || (q.tmp).Cell != z.FailedRef {
		t.Fatal("actual destroyed-old/refused replacement", body, q.tmp)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&large, &q.norm, &q.bound, &q.tmp}, s.header+c.header+primes.Stats().TotalBytes)
	q.executing = false
	stop := s.finishMignotteSource(&r, body)
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) || q.sourceRemaining != 3 || q.norm != norm || q.bound != bound {
		t.Fatal("source younger tmp stop must preserve older bound/norm", stop, q)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, s.cleanupMignotteInvocation(&r, false))
	serviceMust(t, s.cleanupMignotteInvocation(&r, true))
	if s.mignotteInvocationSourceCause(&r) == nil || !errors.Is(stop, body) {
		t.Fatal("owning acknowledgment erased original source history")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&large}, s.header+c.header+primes.Stats().TotalBytes)
	serviceMust(t, s.CleanupRetained())
}
