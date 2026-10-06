// Source: src/math/polynomial/upolynomial.cpp:815-843,3042-3118;
// src/math/polynomial/upolynomial.h:361-364.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"reflect"
	"testing"
)

func TestUPolynomialFactorFrontendActualMonicEndOnly(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []Int{NewInt(1), NewInt(1), NewInt(0), NewInt(1)}))
	defer func() {
		z.budget.FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	q := &s.frontend.monic
	if q.invocation == 0 || !q.endEntered || q.sourceIndex != 2 || q.active {
		t.Fatal("actual field Euclid did not capture plain RAW monic successful ascending end DEL", q.invocation, q.endEntered, q.sourceIndex, q.active)
	}
	var caller []*Int
	for i := range m.values(&input) {
		caller = append(caller, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
}

func TestUPolynomialFactorFrontendCopiedSubscopeOwnerRefused(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []Int{NewInt(1), NewInt(1), NewInt(0), NewInt(1)}))
	defer func() {
		z.budget.FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	original := s.frontend
	before := z.budget.Used()
	copied := original
	copied.self = &copied
	copied.image.owner = &copied
	copied.modularGCD.owner = &copied
	for _, action := range []func() error{func() error { return copied.image.finish(nil) }, copied.image.own, func() error { return copied.modularGCD.finish(nil) }, copied.modularGCD.own} {
		if e := action(); e != ErrUPolynomialOwner {
			t.Fatal("copied complete frontend obtained child source/own authority", e)
		}
	}
	if !reflect.DeepEqual(original, s.frontend) || before != z.budget.Used() {
		t.Fatal("copied child changed actual owner or budget")
	}
}
