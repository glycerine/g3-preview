// Source: src/math/polynomial/upolynomial.cpp:1045-1105; src/math/polynomial/upolynomial_factorization.cpp:703-725.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialHenselActualExtendedGCDCallerIndependentIdentity(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	c.related, e = NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	five := util.NewInt(5)
	serviceMust(t, c.related.SetZP(&five))
	var inputs [2]upolyVectorState
	serviceMust(t, s.manager.set(&inputs[0], []util.Int{util.NewInt(-1), util.NewInt(1)}))
	serviceMust(t, s.manager.set(&inputs[1], []util.Int{util.NewInt(1), util.NewInt(1)}))
	c.quadraticConstructed = 3
	e = c.related.extendedGCDWithCaller(s.manager.stateView(&inputs[0]), s.manager.stateView(&inputs[1]), &c.quadratic[0], &c.quadratic[1], &c.quadratic[2], c)
	serviceMust(t, e)
	for i, want := range []util.Int{util.NewInt(2), util.NewInt(-2), util.NewInt(1)} {
		if c.quadratic[i].size != 1 || s.manager.values(&c.quadratic[i])[0] != want {
			t.Fatal("independent Bezout U*A+V*B=1", i, c.quadratic[i])
		}
	}
	if c.extendedSource != 0 || c.extendedScopedSource != 0 || !c.extendedRawEnded {
		t.Fatal("actual extended scope source suffix incomplete")
	}
	c.executing = false
	serviceMust(t, c.finish())
	serviceMust(t, c.own(false))
	serviceMust(t, s.CleanupRetained())
	for i := range inputs {
		serviceMust(t, s.manager.reset(&inputs[i]))
		s.manager.freeBacking(&inputs[i])
	}
	primeTrialPhysicalAuditExtra(t, q, s)
}

func TestUPolynomialHenselActualExtendedGCDEveryBIGReservation(t *testing.T) {
	sawActualNumericLeaf := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := mignotteDirectFixture(t)
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		serviceMust(t, s.beginHenselCaller(parent))
		c := &parent.hensel
		c.related, e = NewRelatedUPolynomialManager(s.manager)
		serviceMust(t, e)
		var p, n, minus util.Int
		serviceMust(t, q.Integers().SetLimbs(&p, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		serviceMust(t, q.Integers().SetLimbs(&n, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x3fffffff}))
		serviceMust(t, q.Integers().Copy(&minus, &n))
		serviceMust(t, q.Integers().Neg(&minus))
		serviceMust(t, c.related.SetZP(&p))
		var inputs [2]upolyVectorState
		serviceMust(t, s.manager.set(&inputs[0], []util.Int{minus, util.NewInt(1)}))
		serviceMust(t, s.manager.set(&inputs[1], []util.Int{n, util.NewInt(1)}))
		c.quadraticConstructed = 3
		q.Budget().FailAfter(ordinal)
		body := c.related.extendedGCDWithCaller(s.manager.stateView(&inputs[0]), s.manager.stateView(&inputs[1]), &c.quadratic[0], &c.quadratic[1], &c.quadratic[2], c)
		if body == nil {
			// -2*N == 1 (mod p), hence U=1,V=-1 is the independent identity.
			for i, want := range []util.Int{util.NewInt(1), util.NewInt(-1), util.NewInt(1)} {
				if c.quadratic[i].size != 1 || s.manager.values(&c.quadratic[i])[0] != want {
					t.Fatal("BIG independent Bezout identity", ordinal, i, c.quadratic[i])
				}
			}
		}
		roots := []*util.Int{&p, &n, &minus}
		for i := range inputs {
			for j := uint32(0); j < inputs[i].initialized; j++ {
				roots = append(roots, &s.manager.values(&inputs[i])[j])
			}
		}
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		pending := polynomialNegNumericPending(&c.numeric)
		if pending {
			sawActualNumericLeaf = true
		}
		c.executing = false
		// This owns only completed child suffixes and older scoped declarations.
		// Unpublished lower allocations and unjoined header RAWs remain for true.
		serviceMust(t, c.own(false))
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		if pending {
			// Older caller declarations have completed the false phase; the exact
			// unpublished numeric backing is now rescued while failure stays armed.
			serviceMust(t, c.own(true))
			primeTrialPhysicalAuditExtra(t, q, s, roots...)
		}
		q.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		for i := range inputs {
			serviceMust(t, s.manager.reset(&inputs[i]))
			s.manager.freeBacking(&inputs[i])
		}
		for _, a := range []*util.Int{&p, &n, &minus} {
			serviceMust(t, q.Integers().Del(a))
		}
		primeTrialPhysicalAuditExtra(t, q, s)
		if body == nil {
			if !sawActualNumericLeaf {
				t.Fatal("campaign reached no actual direct numeric allocation receipt")
			}
			t.Log("first actual BIG ext_gcd source completion", ordinal)
			return
		}
	}
	t.Fatal("actual BIG ext_gcd never completed")
}

func TestUPolynomialHenselActualQuadraticEveryReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		serviceMust(t, s.beginHenselCaller(parent))
		c := &parent.hensel
		lift, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.lift = lift
		five := util.NewInt(5)
		serviceMust(t, lift.SetZP(&five))
		var inputs [3]upolyVectorState
		for i, v := range [][]util.Int{{util.NewInt(-1), util.NewInt(1)}, {util.NewInt(1), util.NewInt(1)}, {util.NewInt(-6), util.NewInt(0), util.NewInt(1)}} {
			serviceMust(t, m.set(&inputs[i], v))
		}
		q.Budget().FailAfter(ordinal)
		body := s.henselQuadraticWithCaller(m.stateView(&inputs[2]), lift, &inputs[0], &inputs[1], 2, c)
		c.executing = false
		if body == nil {
			for i, want := range []util.Int{util.NewInt(9), util.NewInt(-9)} {
				if inputs[i].size != 2 || m.values(&inputs[i])[0] != want || m.values(&inputs[i])[1] != util.NewInt(1) {
					t.Fatal("independent quadratic p² lift", ordinal, i, inputs[i])
				}
			}
		}
		var roots []*util.Int
		for i := range inputs {
			for j := uint32(0); j < inputs[i].initialized; j++ {
				roots = append(roots, &m.values(&inputs[i])[j])
			}
		}
		// lift is a caller-owned related manager: name its actual private fields.
		roots = append(roots, &(*lift.modular.BorrowModulus()), &lift.modular.Lower, &lift.modular.Upper, &lift.modular.InvTmp1, &lift.modular.InvTmp2, &lift.modular.InvTmp3, &lift.modular.DivTmp)
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		c.executing = false
		q.Budget().FailAfter(-1)
		serviceMust(t, c.own(false))
		serviceMust(t, s.CleanupRetained())
		for i := range inputs {
			serviceMust(t, m.reset(&inputs[i]))
			m.freeBacking(&inputs[i])
		}
		serviceMust(t, lift.Close())
		if body == nil {
			t.Log("first actual quadratic source completion", ordinal)
			return
		}
	}
	t.Fatal("no actual quadratic completion")
}

func TestUPolynomialHenselRepeatedActualQuadraticDeclarations(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	lift, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	c.lift = lift
	var inputs [3]upolyVectorState
	defer func() {
		c.executing = false
		serviceMust(t, s.CleanupRetained())
		for i := range inputs {
			serviceMust(t, m.reset(&inputs[i]))
			m.freeBacking(&inputs[i])
		}
		serviceMust(t, lift.Close())
	}()
	five := util.NewInt(5)
	for invocation := 0; invocation < 2; invocation++ {
		serviceMust(t, lift.SetZP(&five))
		for i, v := range [][]util.Int{{util.NewInt(-1), util.NewInt(1)}, {util.NewInt(1), util.NewInt(1)}, {util.NewInt(-6), util.NewInt(0), util.NewInt(1)}} {
			serviceMust(t, m.set(&inputs[i], v))
		}
		serviceMust(t, s.henselQuadraticWithCaller(m.stateView(&inputs[2]), lift, &inputs[0], &inputs[1], 2, c))
		for i, v := range c.quadratic {
			if v.backing != 0 || v.size != 0 {
				t.Fatal("actual quadratic source destructor omitted on repeated declaration", invocation, i, v)
			}
		}
		if c.related != nil || s.matchesHenselPairInvocation(&c.pair) {
			t.Fatal("completed callee storage still active", invocation)
		}
		for i, want := range []util.Int{util.NewInt(9), util.NewInt(-9)} {
			if m.values(&inputs[i])[0] != want {
				t.Fatal("repeated independent lift", invocation, i)
			}
		}
	}
	primeTrialPhysicalAuditExtra(t, q, s)
}

func TestUPolynomialHenselActualTaggedQuadraticEveryReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := henselDirectFixtureWithConfig(t, func() bool { return true }, true)
		m := s.manager
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		serviceMust(t, s.beginHenselCaller(parent))
		c := &parent.hensel
		lift, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.lift = lift
		five := util.NewInt(5)
		serviceMust(t, lift.SetZP(&five))
		var inputs [3]upolyVectorState
		for i, v := range [][]util.Int{{util.NewInt(-1), util.NewInt(1)}, {util.NewInt(1), util.NewInt(1)}, {util.NewInt(-6), util.NewInt(0), util.NewInt(1)}} {
			serviceMust(t, m.set(&inputs[i], v))
		}
		q.Budget().FailAfter(ordinal)
		body := s.henselQuadraticWithCaller(m.stateView(&inputs[2]), lift, &inputs[0], &inputs[1], 2, c)
		c.executing = false
		if body == nil {
			for i, want := range []util.Int{util.NewInt(9), util.NewInt(-9)} {
				if inputs[i].size != 2 || m.values(&inputs[i])[0] != want || m.values(&inputs[i])[1] != util.NewInt(1) {
					t.Fatal("independent quadratic p² lift", ordinal, i, inputs[i])
				}
			}
		}
		var roots []*util.Int
		for i := range inputs {
			for j := uint32(0); j < inputs[i].initialized; j++ {
				roots = append(roots, &m.values(&inputs[i])[j])
			}
		}
		// lift is a caller-owned related manager: name its actual private fields.
		roots = append(roots, &(*lift.modular.BorrowModulus()), &lift.modular.Lower, &lift.modular.Upper, &lift.modular.InvTmp1, &lift.modular.InvTmp2, &lift.modular.InvTmp3, &lift.modular.DivTmp)
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		c.executing = false
		q.Budget().FailAfter(-1)
		serviceMust(t, c.own(false))
		serviceMust(t, s.CleanupRetained())
		for i := range inputs {
			serviceMust(t, m.reset(&inputs[i]))
			m.freeBacking(&inputs[i])
		}
		serviceMust(t, lift.Close())
		if body == nil {
			t.Log("first actual quadratic source completion", ordinal)
			return
		}
	}
	t.Fatal("no actual quadratic completion")
}
