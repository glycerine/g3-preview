// Source: src/math/polynomial/polynomial.cpp:852-870,1840,2504,6260-6301.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialBorrowedTemporaryActualFactoryIdentity(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 3; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	var tmp monomialTemp
	serviceMust(t, m.tempConstruct(&tmp))
	t.Cleanup(func() { serviceMust(t, m.tempCleanup(&tmp)) })
	serviceMust(t, m.tempSetSize(&tmp, 2))
	serviceMust(t, m.tempSetPower(&tmp, 0, MonomialPower{0, 3}))
	serviceMust(t, m.tempSetPower(&tmp, 1, MonomialPower{2, 5}))
	// Unrelated internal scratch deliberately differs: source must use the
	// actual externally supplied pointer, not an internal scratch index.
	serviceMust(t, m.tempSetSize(&m.temps[1], 1))
	serviceMust(t, m.tempSetPower(&m.temps[1], 0, MonomialPower{1, 7}))
	outcome := monomialFactoryOutcome{}
	a, e := m.internBorrowedTempCreated(&tmp, &outcome)
	serviceMust(t, e)
	if !outcome.created || !outcome.started {
		t.Fatal(outcome)
	}
	serviceMust(t, m.IncRef(a))
	t.Cleanup(func() { serviceMust(t, m.DecRef(a)) })
	for i, want := range []MonomialPower{{0, 3}, {2, 5}} {
		got, e := m.ReadPower(a, uint32(i))
		serviceMust(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	outcome = monomialFactoryOutcome{}
	other, e := m.internBorrowedTempCreated(&tmp, &outcome)
	serviceMust(t, e)
	if other != a || outcome.created {
		t.Fatal("source intern existing branch", other, outcome)
	}
	monomialPhysicalAudit(t, m, tmp.raw)
}
func TestPolynomialBorrowedTemporaryStoppedInsertionLifetime(t *testing.T) {
	success := false
	for fault := int64(0); fault < 32 && !success; fault++ {
		m := monomialFixture(t)
		_, e := m.MkVar()
		serviceMust(t, e)
		var tmp monomialTemp
		serviceMust(t, m.tempConstruct(&tmp))
		serviceMust(t, m.tempSetSize(&tmp, 1))
		serviceMust(t, m.tempSetPower(&tmp, 0, MonomialPower{0, 27}))
		m.config.Budget.FailAfter(fault)
		outcome := monomialFactoryOutcome{}
		a, e := m.internBorrowedTempCreated(&tmp, &outcome)
		monomialPhysicalAudit(t, m, tmp.raw)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fault, e)
			}
			if m.borrowedTemp != &tmp {
				t.Fatal("actual stopped source pointer must remain borrowed", fault)
			}
			var other monomialTemp
			if _, e = m.internBorrowedTempCreated(&other, nil); e != ErrMonomialPending {
				t.Fatal("no replace while reached binding pending", e)
			}
		} else {
			success = true
			serviceMust(t, m.IncRef(a))
			if m.borrowedTemp != nil {
				t.Fatal("successful temporary no longer table key")
			}
		}
		m.config.Budget.FailAfter(-1)
		if success {
			serviceMust(t, m.DecRef(a))
		}
		serviceMust(t, m.CleanupCreation())
		if m.borrowedTemp != nil {
			t.Fatal("explicit callee abort releases borrowed identity")
		}
		serviceMust(t, m.tempCleanup(&tmp))
		if tmp.raw != (util.Block{}) {
			t.Fatal("temporary source owner cleanup")
		}
	}
	if !success {
		t.Fatal("no success")
	}
}
