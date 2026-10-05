// Source: src/math/polynomial/polynomial.cpp:1392-1423,2481-2495.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// The actual core owns the unjoined placement/result. A receipt names this
// stable frame and the real guarded constructor invocation, not a live value
// discovered by examining the polynomial registry or its reference count.
type polynomialFactoryScope struct {
	invocation uint64
	active     bool
	result     Polynomial
	allocator  [3]memory.Block
}
type polynomialFactoryInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialFactoryScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesFactoryInvocation(r *polynomialFactoryInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.factoryInvocation && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}

// Only the actual guarded constructor can publish this raw owning root. It
// contributes a live cell, but no reference: a real caller INC must refuse.
func (m *MultivariatePolynomialCore) factoryCallerGuard() (uint32, error) {
	s := &m.factoryInvocation
	if !s.active || s.result == (Polynomial{}) {
		return 0, nil
	}
	l, e := m.polynomialLocation(s.result)
	if e != nil {
		return 0, e
	}
	if l.deleteStage == 0 {
		rc, e := m.readWord(s.result, 0)
		if e != nil {
			return 0, e
		}
		if rc != 0 {
			return 0, ErrPolynomialLive
		}
	}
	return 1, nil
}

// Transfer acknowledges the actual returned factory pointer at the caller's
// owning boundary. It never creates authority for a quick/cache result.
func (m *MultivariatePolynomialCore) transferFactoryResult(r *polynomialFactoryInvocationReceipt, p Polynomial) error {
	if !m.matchesFactoryInvocation(r) {
		return nil
	}
	s := r.frame
	if p == (Polynomial{}) || p != s.result {
		return ErrPolynomialOwner
	}
	s.result = Polynomial{}
	if s.allocator == ([3]memory.Block{}) {
		s.active = false
	}
	return nil
}

// Placement has no completed lexical source destructor. Its owning abort is
// deliberately deferred until after the parent's completed lexical locals.
func (m *MultivariatePolynomialCore) cleanupFactoryInvocation(r *polynomialFactoryInvocationReceipt, abort bool) error {
	if !abort || !m.matchesFactoryInvocation(r) {
		return nil
	}
	s := r.frame
	if m.pending.generation == s.invocation {
		if e := m.cleanupPendingCreation(); e != nil {
			return e
		}
	}
	if s.result != (Polynomial{}) {
		if e := m.CleanupValue(&s.result); e != nil {
			return e
		}
	}
	roots, e := m.monomials.allocator.PendingRoots()
	if e != nil {
		return e
	}
	owned, foreign := false, false
	for i := range roots {
		if roots[i] != (memory.Block{}) {
			if roots[i] == s.allocator[i] {
				owned = true
			} else {
				foreign = true
			}
		}
	}
	if owned && foreign {
		return ErrPolynomialPending
	}
	if owned {
		e = m.monomials.allocator.CleanupPending()
		now, x := m.monomials.allocator.PendingRoots()
		if x != nil {
			return polynomialNewtonJoin(e, x)
		}
		s.allocator = now
		if e != nil {
			return e
		}
	} else {
		s.allocator = [3]memory.Block{}
	}
	*s = polynomialFactoryScope{invocation: s.invocation}
	return nil
}
