// Source: src/math/polynomial/polynomial.cpp:3318-3338;
// src/util/mpzzp.h:set; src/util/mpz.cpp:big_set.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// This partial receipt covers actual SET allocation. The original finite-field
// PNormalize engine remains unchanged; allocating normalization needs its own
// actual allocation-leaf receipt before the parent can claim that mode complete.
type polynomialConstantCoefficientScope struct {
	invocation        uint64
	active, executing bool
	copy              unumeral.IntegerCopyAllocationReceipt
	normalize         unumeral.IntegerNormalizeInvocationReceipt
	sourceCause       error
}
type polynomialConstantCoefficientInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialConstantCoefficientScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesConstantCoefficientInvocation(r *polynomialConstantCoefficientInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.constantCoefficientInvocation && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *MultivariatePolynomialCore) cleanupConstantCoefficientInvocation(r *polynomialConstantCoefficientInvocationReceipt, abort bool) error {
	if !m.matchesConstantCoefficientInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.modular.CleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		return e
	}
	if !abort {
		return nil
	}
	if e := m.integer.CleanupCopyAllocationInvocation(&s.copy); e != nil {
		return e
	}
	*s = polynomialConstantCoefficientScope{invocation: s.invocation, normalize: s.normalize, sourceCause: s.sourceCause}
	return nil
}

func (m *MultivariatePolynomialCore) constantCoefficientInvocationSourceCause(r *polynomialConstantCoefficientInvocationReceipt) error {
	if r == nil || r.manager != m || r.frame != &m.constantCoefficientInvocation || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.sourceCause
}
