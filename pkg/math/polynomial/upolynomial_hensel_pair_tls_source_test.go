// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/scoped_numeral_vector.h:29-40.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Public signed-host TLS operations establish the reached postphysical NUM
// release guard. This is an owner-primitive guard profile, not ordinary Hensel
// mathematical failure reachability; the vector's actual field owns the NUM.
func TestUPolynomialHenselActualTmpPostphysicalStopsOlderPair(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(b)
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
	five := util.NewInt(5)
	serviceMust(t, h.rm.SetZP(&five))
	for i := uint8(2); i <= 7; i++ {
		m, v := h.vector(i)
		h.constructed = i
		serviceMust(t, m.set(v, []util.Int{util.NewInt(1)}))
	}
	tmp, e := h.rm.read(h.rm.stateView(&h.tmp), 0)
	serviceMust(t, e)
	serviceMust(t, q.Integers().Copy(tmp, &q.Integers().Two64))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(tmp, 40000))
	older := h.T
	oldRelatedStage := h.rm.sourceStage
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	h.executing = false
	stop := s.finishHenselPairSource(&r, nil)
	primeTrialPhysicalAuditExtra(t, q, s)
	if !errors.Is(stop, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || h.sourceRemaining != 7 || h.sourceIndex != 0 || h.T != older || h.rm.sourceStage != oldRelatedStage {
		t.Fatal("actual consumed tmp must stop before older pair fields", stop, h.sourceRemaining, h.sourceIndex, rt.State(), before)
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
