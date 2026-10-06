// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj).
package mpnum

import (
	"reflect"
	"testing"
)

func TestUPolynomialFactorLiftPreparationActualRelatedCheckpointGuards(t *testing.T) {
	var s *UPolynomialFactorizationService
	probes := 0
	z, service := henselDirectFixtureWithCheckpoint(t, func() bool {
		if s == nil || s.scope == nil {
			return true
		}
		c := s.scope
		q := &c.primePreparation
		if !q.executing || q.lift == nil || q.hensel == nil {
			return true
		}
		probes++
		before := *q
		for _, m := range []*UPolynomialManager{c.zp, c.zpe} {
			for _, action := range []func() error{m.Close, m.SourceFinalize, m.SetZ} {
				if e := action(); e != ErrUPolynomialCleanupPending {
					t.Fatalf("actual related caller guard %v", e)
				}
				if !reflect.DeepEqual(*q, before) {
					t.Fatal("rejected related operation changed exact caller")
				}
			}
		}
		return true
	})
	s = service
	m := s.manager
	out, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { z.budget.FailAfter(-1); serviceMust(t, s.CleanupRetained()); serviceMust(t, out.Close()) }()
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, m.set(&c.fpp, []Int{NewInt(-6), NewInt(1), NewInt(1)}))
	serviceMust(t, s.preparePrimeSearch(out))
	serviceMust(t, m.modular.SetInt32(&c.p, 7))
	serviceMust(t, c.zp.SetZP(&c.p))
	trials := uint32(0)
	route, e := s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
	serviceMust(t, e)
	if route != 1 {
		t.Fatal("actual trial route", route)
	}
	serviceMust(t, s.prepareFactorLift(out))
	if probes == 0 {
		t.Fatal("no actual reached lift caller checkpoint")
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
}

// Invalid copied/stale Go contexts are separately qualified guard fixtures;
// actual frame begin supplies the nonwrapping identity and real member owners.
func TestUPolynomialFactorLiftPreparationCopiedStaleFactorContexts(t *testing.T) {
	z, s := mignotteDirectFixture(t)
	m := s.manager
	out, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, out.Close()) }()
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, m.set(&c.fpp, []Int{NewInt(-6), NewInt(1), NewInt(1)}))
	serviceMust(t, s.preparePrimeSearch(out))
	serviceMust(t, m.modular.SetInt32(&c.p, 7))
	serviceMust(t, c.zp.SetZP(&c.p))
	trials := uint32(0)
	_, e = s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
	serviceMust(t, e)
	serviceMust(t, s.prepareFactorLift(out))
	old := c.primePreparation.receipt
	q, e := s.beginPrimePreparation(out)
	serviceMust(t, e)
	context := c.zpe.modular.primePreparationAllocationContext(q)
	if context.primePreparationFrame(z.z) != q {
		t.Fatal("actual lift context did not bind")
	}
	before := *q
	used := z.budget.Used()
	copied := *q
	bad := context
	bad.primePreparation = &copied
	if bad.primePreparationFrame(z.z) != nil {
		t.Fatal("copied frame acquired authority")
	}
	bad = context
	bad.primePreparationInvocation = old.invocation
	if bad.primePreparationFrame(z.z) != nil {
		t.Fatal("old context retargeted new invocation")
	}
	foreign, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	if foreign.modular.primePreparationAllocationContext(q).primePreparationFrame(z.z) != nil {
		t.Fatal("foreign receiver acquired lift authority")
	}
	serviceMust(t, foreign.Close())
	if e := q.own(&old, true); e == nil || !reflect.DeepEqual(*q, before) || z.budget.Used() != used {
		t.Fatal("stale acknowledgment mutated new invocation", e)
	}
	q.executing = false
	serviceMust(t, q.finish(&q.receipt, nil))
	fsCopy := *c.lifted
	beforeFactors := *c.lifted
	if e := c.closePreparedFactors(&fsCopy, 1, false); e != ErrUPolynomialOwner || !reflect.DeepEqual(*c.lifted, beforeFactors) {
		t.Fatal("copied factors acquired owning authority", e)
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
}
