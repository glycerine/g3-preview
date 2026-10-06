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
func TestUPolynomialHenselActualConditionalChildStopsOlderPair(t *testing.T) {
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
	c := &h.check
	c.br = NewInt(1)
	c.constructed = 1
	c.brCore, e = NewRelatedUPolynomialManager(u)
	serviceMust(t, e)
	c.constructed = 2
	serviceMust(t, c.brCore.SetZP(&five))
	serviceMust(t, u.set(&c.test1, []Int{NewInt(1)}))
	c.constructed = 3
	c.bCore, e = NewRelatedUPolynomialManager(u)
	serviceMust(t, e)
	c.constructed = 4
	serviceMust(t, c.bCore.SetZP(&five))
	for i := uint8(5); i <= 8; i++ {
		serviceMust(t, u.set(h.checkVector(i), []Int{NewInt(1)}))
		c.constructed = i
	}
	tmp, e := u.read(u.stateView(&c.test3b), 0)
	serviceMust(t, e)
	serviceMust(t, q.z.Copy(tmp, &q.z.two64))
	serviceMust(t, q.z.ensureOwnedCapacity(tmp, 40000))
	older := h.T
	olderPairTmp := h.tmp
	olderChild := c.test3a
	oldRelatedStage := h.rm.sourceStage
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	h.enterCall()
	childStop := h.finishHenselCheckSource(nil)
	h.leaveCall()
	h.executing = false
	stop := s.finishHenselPairSource(&r, childStop)
	primeTrialPhysicalAuditExtra(t, q, s)
	if !errors.Is(stop, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || h.sourceRemaining != 7 || h.sourceIndex != 0 || h.T != older || h.rm.sourceStage != oldRelatedStage || h.tmp != olderPairTmp || c.test3a != olderChild || c.sourceRemaining != 8 || c.sourceIndex != 0 {
		t.Fatal("actual consumed helper child must stop before test3a and pair tmp", stop, h.sourceRemaining, h.sourceIndex, rt.State(), before)
	}
	firstCause := h.sourceCause
	if !errors.Is(s.finishHenselPairSource(&r, nil), firstCause) || h.sourceCause != firstCause || h.T != older {
		t.Fatal("source retry lost first stop or drained T")
	}
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, true) })
	if s.henselPairInvocationSourceCause(&r) != firstCause || !errors.Is(stop, arena.ErrBounds) {
		t.Fatal("owning erased immutable source history")
	}
	primeTrialPhysicalAuditExtra(t, q, s)
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
