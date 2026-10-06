// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674;
// src/util/mpz.h:124-149; src/util/mpzzp.h:154,230-233.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import "testing"

func TestUPolynomialHenselPairNumericContextActualFrameAndCapturedNonce(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	r, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	h := r.frame
	h.rm, e = NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	h.constructed = 2
	c := s.manager.modular.henselPairAllocationContext(&r)
	if c.henselPairReceipt(q.z) == nil {
		t.Fatal("actual guarded frame lacks direct allocation authority")
	}
	old := c
	ownerCopy := *s
	badOwner := r
	badOwner.service = &ownerCopy
	if s.manager.modular.henselPairAllocationContext(&badOwner).henselPairReceipt(q.z) != nil {
		t.Fatal("copied actual service acquired authority")
	}
	foreign, e := NewIntegerModularManager(q.z)
	serviceMust(t, e)
	if foreign.henselPairAllocationContext(&r).henselPairReceipt(q.z) != nil {
		t.Fatal("foreign modular receiver acquired pair roots")
	}
	serviceMust(t, foreign.Close())
	copied := *h
	bad := r
	bad.frame = &copied
	if s.manager.modular.henselPairAllocationContext(&bad).henselPairReceipt(q.z) != nil {
		t.Fatal("copied cold frame acquired authority")
	}
	s.manager.callbackDepth++
	if c.henselPairReceipt(q.z) != nil {
		t.Fatal("external checkpoint callback acquired parent authority")
	}
	s.manager.callbackDepth--
	h.executing = false
	serviceMust(t, s.cleanupHenselPairInvocation(&r, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&r, true))
	newer, e := s.beginHenselPairInvocation()
	serviceMust(t, e)
	newer.frame.rm, e = NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	newer.frame.constructed = 2
	r = newer
	if old.henselPairReceipt(q.z) != nil {
		t.Fatal("saved context retargeted through reused receipt pointer")
	}
	newer.frame.executing = false
	serviceMust(t, s.cleanupHenselPairInvocation(&newer, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&newer, true))
	// Invalid Go metadata guard fixture: exhausting the actual stored nonce
	// must refuse before resetting the same cold frame or its history.
	newer.frame.invocation = ^uint64(0)
	used, frame := q.budget.Used(), *newer.frame
	_, e = s.beginHenselPairInvocation()
	if e == nil || q.budget.Used() != used || newer.frame.invocation != frame.invocation || newer.frame.active != frame.active || newer.frame.sourceCause != frame.sourceCause {
		t.Fatal("wrapped begin mutated actual frame", e)
	}
}
