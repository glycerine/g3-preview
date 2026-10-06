// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674,873-977.
// Licensed under the MIT license. Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"

	"testing"
)

// Actual signed-accounting release guard is a helper domain, not ordinary OOM.
func TestUPolynomialHenselActualOwningRefusalRetainsFirstHistory(t *testing.T) {
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
	tmp, e := h.rm.read(h.rm.stateView(&h.tmp), 0)
	serviceMust(t, e)
	serviceMust(t, q.z.Copy(tmp, &q.z.two64))
	serviceMust(t, q.z.ensureOwnedCapacity(tmp, 40000))
	h.executing = false
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	first := s.cleanupHenselPairInvocation(&r, false)
	if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || h.ownRemaining != 7 || h.ownIndex != 0 || h.sourceStarted || h.sourceCause != nil || h.bodyCause != nil || r.frame != h || r.invocation != h.invocation {
		t.Fatal("actual matched owning refusal frontier", first)
	}
	primeTrialPhysicalAuditExtra(t, q, s)
	if h.ownCause != first || s.henselPairInvocationOwnCause(&r) != first {
		t.Fatal("actual first owning cleanup cause was discarded", first)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupHenselPairInvocation(&r, true) })
	if h.ownCause != first || s.henselPairInvocationOwnCause(&r) != first {
		t.Fatal("retry erased original own cleanup cause")
	}
	primeTrialPhysicalAuditExternal(t, q, s, []memory.Block{recovery})
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal(e)
	}
}

func TestUPolynomialHenselActualCallerOwningRefusalRetainsFirstHistory(t *testing.T) {
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

	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	c.outerConstructed = 4
	for i := range c.outer {
		serviceMust(t, u.set(&c.outer[i], []Int{NewInt(1)}))
	}
	tmp, e := u.read(u.stateView(&c.outer[3]), 0)
	serviceMust(t, e)
	serviceMust(t, q.z.Copy(tmp, &q.z.two64))
	serviceMust(t, q.z.ensureOwnedCapacity(tmp, 40000))
	c.executing = false
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	first := c.own(false)
	if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || c.ownCause != first || c.outerOwn != 4 || c.ownIndex != 0 || c.sourceCause != nil || c.bodyCause != nil || c.outerSourceStarted {
		t.Fatal("real caller owning refusal first cause", first, c.ownCause)
	}
	primeTrialPhysicalAuditExtra(t, q, s)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.own(false) })
	contextTLSFinish(t, rt, s.CleanupRetained)
	if c.ownCause != first || c.sourceCause != nil || c.bodyCause != nil {
		t.Fatal("caller retirement erased or replaced first own history")
	}
	primeTrialPhysicalAuditExternal(t, q, s, []memory.Block{recovery})
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal(e)
	}
}
