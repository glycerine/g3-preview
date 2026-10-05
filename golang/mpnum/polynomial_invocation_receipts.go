// Source: src/math/polynomial/polynomial.cpp:3432-3534,4352-4506,5904-5940;
// src/util/obj_ref.h; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

// These are owning adapter receipts for an actual guarded callee invocation.
// The shared PSC owner generation is deliberately not invocation identity.
// Stable frame address, manager and monotonically increasing frame nonce must
// all match; rejected entry and quick paths never acquire a new receipt.
type polynomialICCPInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialICCPScope
	invocation uint64
}
type polynomialGCDInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialGCDFunctionScope
	invocation uint64
}
type polynomialNormalizeInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialNormalizeScope
	invocation uint64
}

func (m *MultivariatePolynomialManager) matchesICCPInvocation(r *polynomialICCPInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame != nil && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *MultivariatePolynomialManager) matchesGCDInvocation(r *polynomialGCDInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame != nil && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *MultivariatePolynomialManager) matchesNormalizeInvocation(r *polynomialNormalizeInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.normalization && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *MultivariatePolynomialManager) iccpInvocationSourceCause(r *polynomialICCPInvocationReceipt) error {
	if m.matchesICCPInvocation(r) {
		return r.frame.cleanupCause
	}
	return nil
}
func (m *MultivariatePolynomialManager) gcdInvocationSourceCause(r *polynomialGCDInvocationReceipt) error {
	if m.matchesGCDInvocation(r) {
		return r.frame.cleanupCause
	}
	return nil
}
func (m *MultivariatePolynomialManager) normalizeInvocationSourceCause(r *polynomialNormalizeInvocationReceipt) error {
	if m.matchesNormalizeInvocation(r) {
		return r.frame.cleanupCause
	}
	return nil
}
func (m *MultivariatePolynomialManager) cleanupICCPInvocation(r *polynomialICCPInvocationReceipt, abort bool) error {
	if !m.matchesICCPInvocation(r) {
		return nil
	}
	if m.iccpDepth != 0 {
		return ErrPolynomialPending
	}
	return m.cleanupICCPFrame(r.frame, abort)
}
func (m *MultivariatePolynomialManager) cleanupICCPFrame(s *polynomialICCPScope, abort bool) error {
	if s.child != nil && s.child.active && s.child.invocation == s.childInvocation {
		if e := m.cleanupICCPFrame(s.child, abort); e != nil {
			return e
		}
	}
	one := [1]*polynomialICCPScope{s}
	return m.cleanupICCPScopes(one[:], abort)
}
func (m *MultivariatePolynomialManager) cleanupGCDInvocation(r *polynomialGCDInvocationReceipt, abort bool) error {
	if !m.matchesGCDInvocation(r) {
		return nil
	}
	if m.gcdDepth != 0 {
		return ErrPolynomialPending
	}
	return m.cleanupGCDFrame(r.frame, abort)
}
func (m *MultivariatePolynomialManager) cleanupGCDFrame(s *polynomialGCDFunctionScope, abort bool) error {
	if s.child != nil && s.child.active && s.child.invocation == s.childInvocation {
		if e := m.cleanupGCDFrame(s.child, abort); e != nil {
			return e
		}
	}
	one := [1]*polynomialGCDFunctionScope{s}
	return m.cleanupGCDScopes(one[:], abort)
}
func (m *MultivariatePolynomialManager) cleanupNormalizeInvocation(r *polynomialNormalizeInvocationReceipt, abort bool) error {
	if !m.matchesNormalizeInvocation(r) {
		return nil
	}
	return m.cleanupNormalizationPhase(abort)
}
