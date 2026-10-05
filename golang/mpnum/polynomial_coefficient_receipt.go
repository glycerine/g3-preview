// Source: src/math/polynomial/polynomial.cpp:3264-3277,1392-1423.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

type polynomialCoefficientScope struct {
	invocation, calleeGeneration          uint64
	active, executing, foreignTransferred bool
	generatedCleanupStarted               bool
	generated                             Monomial
	allocator                             [3]memory.Block
}
type polynomialCoefficientInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialCoefficientScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesCoefficientInvocation(r *polynomialCoefficientInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.coefficientInvocation && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *MultivariatePolynomialCore) transferCoefficientGenerated(r *polynomialCoefficientInvocationReceipt, term Monomial) {
	s := &m.coefficientInvocation
	if !s.active || s.generated == (Monomial{}) || s.generated != term {
		return
	}
	if m.matchesCoefficientInvocation(r) && s.executing {
		s.generated = Monomial{}
	} else {
		// This real INC belongs to a foreign append, so ownership cannot be
		// recovered merely by finding the same monomial token in that vector.
		s.foreignTransferred = true
	}
}
func (m *MultivariatePolynomialCore) cleanupCoefficientInvocation(r *polynomialCoefficientInvocationReceipt, abort bool) error {
	if !abort || !m.matchesCoefficientInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing || s.foreignTransferred {
		return ErrPolynomialPending
	}
	if s.generated != (Monomial{}) {
		// A caller INC is a refusal guard, never authority for this root.
		if !s.generatedCleanupStarted {
			count, e := m.monomialRefCount(s.generated)
			if e != nil {
				return e
			}
			if count != 0 {
				return ErrPolynomialPending
			}
			s.generatedCleanupStarted = true
		}
		before := s.generated
		e := m.monomials.CleanupValue(&s.generated)
		if s.generated == (Monomial{}) && m.generatedMonomial == before {
			m.generatedMonomial = Monomial{}
			m.generatedActive = false
			m.generatedPSC = 0
			m.generatedBuffer = nil
		}
		if e != nil {
			return e
		}
	}
	if s.calleeGeneration != 0 && m.monomials.creation.generation == s.calleeGeneration && m.monomials.creation.coefficientParent == s && m.monomials.creation.coefficientInvocation == s.invocation {
		if e := m.monomials.CleanupCreation(); e != nil {
			return e
		}
	}
	// Only actual newly acquired allocator leaf tokens belong to this invocation.
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
	}
	*s = polynomialCoefficientScope{invocation: s.invocation}
	return nil
}
