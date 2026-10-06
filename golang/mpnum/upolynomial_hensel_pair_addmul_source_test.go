// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/scoped_numeral_vector.h:29-40.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// Public signed-host TLS operations establish the reached postphysical NUM
// release guard. This is an owner-primitive guard profile, not ordinary Hensel
// mathematical failure reachability; the vector's actual field owns the NUM.
func TestUPolynomialHenselAddMulEndStopProtectsScopedAmAndPair(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, u.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual owner refunds", b.Used())
		}
	}()
	r, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	h := r.frame
	h.rm, e = NewRelatedUPolynomialManager(u)
	serviceMust(t, e)
	h.constructed = 1
	five := NewInt(5)
	serviceMust(t, h.rm.SetZP(&five))
	for i := uint8(2); i <= 7; i++ {
		m, v := h.vector(i)
		h.constructed = i
		serviceMust(t, m.set(v, []Int{NewInt(1)}))
	}
	h.amConstructed = true
	serviceMust(t, q.z.Copy(&h.am, &q.z.two64))
	olderAm, olderT := h.am, h.T
	var a, c, warm Int
	digits := make([]uint32, 64)
	digits[0], digits[63] = 17, 1
	serviceMust(t, q.z.SetLimbs(&c, digits))
	product := make([]uint32, 64)
	product[0], product[63] = 34, 2
	serviceMust(t, q.z.SetLimbs(&a, product))
	serviceMust(t, q.z.Neg(&a))
	prime := NewInt(251)
	serviceMust(t, q.z.Rem(&c, &prime, &warm))
	d, two := NewInt(9), NewInt(2)
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	e = rt.Deallocate(&external)
	if e == nil || external != (memory.Block{}) {
		t.Fatal("actual signed release frontier", e)
	}
	context := u.modular.henselPairAllocationContext(&r)
	body := q.z.addMulWithAllocationContext(&a, &two, &c, &d, &h.addMul, context)
	firstCause := q.z.addMulInvocationSourceCause(&h.addMul)
	if firstCause == nil || h.addMul.frame.stage != 3 || d != NewInt(0) {
		t.Fatal("actual successful body/end DEL stop", body, firstCause, d)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &a, &c, &warm, &d)
	h.executing = false
	stop := s.finishHenselAmSource(&r, body)
	if !errors.Is(stop, firstCause) || h.am != olderAm || h.amEnded || h.amSourceCause != firstCause {
		t.Fatal("younger source stop drained scoped am", stop)
	}
	stop = s.finishHenselPairSource(&r, stop)
	if !errors.Is(stop, firstCause) || h.sourceRemaining != 7 || h.T != olderT || h.am != olderAm {
		t.Fatal("older pair source fields changed", stop)
	}
	primeTrialPhysicalAuditExtra(t, q, s, &a, &c, &warm, &d)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	defer func() {
		e := rt.Deallocate(&recovery)
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal(e)
		}
	}()
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, true) })
	for _, n := range []*Int{&a, &c, &warm, &d} {
		contextTLSFinish(t, rt, func() error { return q.z.Del(n) })
	}
	if s.henselPairInvocationSourceCause(&r) != firstCause || !errors.Is(stop, arena.ErrBounds) {
		t.Fatal("owning erased immutable source history")
	}
	primeTrialPhysicalAuditExternal(t, q, s, []memory.Block{recovery})
	next, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	next.frame.executing = false
	serviceMust(t, s.finishHenselPairSource(&next, nil))
	serviceMust(t, s.cleanupHenselPairInvocation(&next, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&next, true))
	if !errors.Is(stop, arena.ErrBounds) {
		t.Fatal("new invocation changed old returned diagnostic")
	}
}
