// Source: src/math/polynomial/upolynomial_factorization.cpp:873-977.
// Licensed under the MIT license. Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"testing"
)

func TestUPolynomialHenselInitialSetZPFailureRetainsActualBodyCause(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	zp, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	zpe, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	var p Int
	serviceMust(t, q.z.SetLimbs(&p, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	serviceMust(t, zp.SetZP(&p))
	pf, e := zp.NewFactors()
	serviceMust(t, e)
	lf, e := zpe.NewFactors()
	serviceMust(t, e)
	var f, a upolyVectorState
	serviceMust(t, m.set(&f, []Int{NewInt(-1), NewInt(0), NewInt(1)}))
	serviceMust(t, m.set(&a, []Int{NewInt(-1), NewInt(1)}))
	serviceMust(t, pf.PushCopy(m.stateView(&a), 1))
	q.budget.FailAfter(0)
	body := s.henselFactorsWithCaller(m.stateView(&f), pf, 2, lf, c)
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf}, &p)
	if !numericPhysicalReservationRefusal(body) || c.bodyCause != body || c.executing || c.outerConstructed != 0 || c.finalLCInvConstructed || lf.count != 0 {
		t.Fatal("initial reached SET cause/declaration frontier lost", body, c.bodyCause, c.outerConstructed)
	}
	q.budget.FailAfter(-1)
	serviceMust(t, c.own(false))
	serviceMust(t, s.CleanupRetained())
	for _, v := range []*upolyVectorState{&f, &a} {
		serviceMust(t, m.reset(v))
		m.freeBacking(v)
	}
	serviceMust(t, pf.Close())
	serviceMust(t, zpe.SetZ())
	serviceMust(t, lf.Close())
	serviceMust(t, zp.Close())
	serviceMust(t, zpe.Close())
	serviceMust(t, q.z.Del(&p))
}
func TestUPolynomialHenselPublicFactorsEveryActualReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := henselDirectFixtureWithConfig(t, func() bool { return true }, true)
		m := s.manager
		zp, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		zpe, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		five := NewInt(5)
		serviceMust(t, zp.SetZP(&five))
		pf, e := zp.NewFactors()
		serviceMust(t, e)
		lf, e := zpe.NewFactors()
		serviceMust(t, e)
		var f, a upolyVectorState
		serviceMust(t, m.set(&f, []Int{NewInt(-6), NewInt(0), NewInt(1)}))
		for _, v := range [][]Int{{NewInt(-1), NewInt(1)}, {NewInt(1), NewInt(1)}} {
			serviceMust(t, m.set(&a, v))
			serviceMust(t, pf.PushCopy(m.stateView(&a), 1))
		}
		q.budget.FailAfter(ordinal)
		body := s.HenselLiftFactors(m.stateView(&f), pf, 2, lf)
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf})
		if body != nil && s.scope == nil && s.henselPairs != nil && s.henselPairs.active {
			t.Fatal("public caller failed with no exact enclosing owner", ordinal, body)
		}
		if body == nil {
			if s.scope != nil || lf.count != 2 {
				t.Fatal("public completed source header/publication")
			}
			for i, want := range []Int{NewInt(9), NewInt(-9)} {
				if lf.factors[i].size != 2 || m.values(&lf.factors[i])[0] != want || m.values(&lf.factors[i])[1] != NewInt(1) {
					t.Fatal("independent public factors", i)
				}
			}
		}

		// Capture only diagnostic identity/value observations, never ownership.
		var published [2]upolyVectorState
		var values [2][2]Int
		publishedCount := lf.count
		for i := uint32(0); i < publishedCount; i++ {
			published[i] = lf.factors[i]
			if lf.factors[i].size > 2 {
				t.Fatal("source partial output exceeds independent degree bound")
			}
			for j := uint32(0); j < lf.factors[i].size; j++ {
				values[i][j] = m.values(&lf.factors[i])[j]
			}
		}
		owner := s.scope
		var originalBody, originalSource error
		if owner != nil {
			originalBody, originalSource = owner.hensel.bodyCause, owner.hensel.sourceCause
			if originalSource != nil {
				before := owner.hensel
				if owner.hensel.finish() != originalSource || owner.hensel != before {
					t.Fatal("source retry drained older public caller suffix")
				}
			}
		}
		q.budget.FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		if lf.count != publishedCount {
			t.Fatal("matched public abort consumed published output prefix")
		}
		for i := uint32(0); i < publishedCount; i++ {
			if lf.factors[i] != published[i] {
				t.Fatal("own cleanup changed published factor backing", i)
			}
			for j := uint32(0); j < published[i].size; j++ {
				if m.values(&lf.factors[i])[j] != values[i][j] {
					t.Fatal("own cleanup changed published factor value", i, j)
				}
			}
		}
		if owner != nil {
			if owner.hensel.bodyCause != originalBody || owner.hensel.sourceCause != originalSource || !errors.Is(body, originalBody) {
				t.Fatal("own acknowledgment changed immutable public returned/body/source history")
			}
		}
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf})
		if pf.count != 2 {
			t.Fatal("public owning abort consumed borrowed prime factors")
		}
		for _, v := range []*upolyVectorState{&f, &a} {
			serviceMust(t, m.reset(v))
			m.freeBacking(v)
		}
		serviceMust(t, pf.Close())
		serviceMust(t, lf.Close())
		serviceMust(t, zp.Close())
		serviceMust(t, zpe.Close())
		if body == nil {
			t.Log("first public caller completion", ordinal)
			return
		}
	}
	t.Fatal("public caller never completes")
}
