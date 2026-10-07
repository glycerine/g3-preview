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
func TestUPolynomialHenselAddMulEndStopProtectsScopedAmAndPair(t *testing.T) {
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
	h.amConstructed = true
	serviceMust(t, q.Integers().Copy(&h.am, &q.Integers().Two64))
	olderAm, olderT := h.am, h.T
	var a, c, warm util.Int
	digits := make([]uint32, 64)
	digits[0], digits[63] = 17, 1
	serviceMust(t, q.Integers().SetLimbs(&c, digits))
	product := make([]uint32, 64)
	product[0], product[63] = 34, 2
	serviceMust(t, q.Integers().SetLimbs(&a, product))
	serviceMust(t, q.Integers().Neg(&a))
	prime := util.NewInt(251)
	serviceMust(t, q.Integers().Rem(&c, &prime, &warm))
	d, two := util.NewInt(9), util.NewInt(2)
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	e = rt.Deallocate(&external)
	if e == nil || external != (util.Block{}) {
		t.Fatal("actual signed release frontier", e)
	}
	context := numericIntegerModularManager_henselPairAllocationContext(u.modular, &r)
	body := q.Integers().WithAllocationContext((context).utility(q.Integers())).AddMulTracked(&a, &two, &c, &d, &h.addMul)
	firstCause := q.Integers().AddMulInvocationSourceCause(&h.addMul)
	if firstCause == nil || h.addMul.Frame.Stage != 3 || d != util.NewInt(0) {
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
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
			t.Fatal(e)
		}
	}()
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, true) })
	for _, n := range []*util.Int{&a, &c, &warm, &d} {
		contextTLSFinish(t, rt, func() error { return q.Integers().Del(n) })
	}
	if s.henselPairInvocationSourceCause(&r) != firstCause || !errors.Is(stop, arena.ErrBounds) {
		t.Fatal("owning erased immutable source history")
	}
	primeTrialPhysicalAuditExternal(t, q, s, []util.Block{recovery})
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
