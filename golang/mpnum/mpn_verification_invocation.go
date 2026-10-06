// Source: src/util/mpn.cpp:206-217; src/util/buffer.h.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// The reached lexical constructor has its own identity. An unfinished
// construction owns exact raw/pending leaves but has no source destructor.
type naturalPNVerificationScope struct {
	henselCallerContext                          integerAllocationInvocationContext
	henselPair                                   upolyHenselPairInvocationReceipt
	divides                                      polynomialDividesInvocationReceipt
	boolDivides                                  integerDividesNumeralInvocationReceipt
	scalarExact                                  polynomialScalarExactDivInvocationReceipt
	exactDiv                                     polynomialExactDivInvocationReceipt
	raw                                          integerRawInvocationReceipt
	invocation                                   uint64
	parent                                       *integerPNormalizeScope
	scalar                                       polynomialScalarDivInvocationReceipt
	content                                      polynomialContentInvocationReceipt
	active, executing, constructed, ended, owned bool
	buffer                                       naturalPNBufferReceipt
	sourceCause                                  error
}
type naturalPNVerificationReceipt struct {
	owner      *NaturalManager
	frame      *naturalPNVerificationScope
	invocation uint64
}

func (m *NaturalManager) matchesPNVerification(r *naturalPNVerificationReceipt) bool {
	return r != nil && r.owner == m && r.frame == &m.verificationInvocation && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *NaturalManager) finishPNVerificationSource(s *naturalPNVerificationScope) error {
	if s != &m.verificationInvocation {
		return nil
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active || !s.constructed || s.ended || s.owned {
		return nil
	}
	if e := m.releasePNVerificationRaw(s); e != nil {
		s.sourceCause = e
		return e
	}
	s.ended = true
	m.verificationActive = false
	s.active = false
	return nil
}
func (m *NaturalManager) releasePNVerificationRaw(s *naturalPNVerificationScope) error {
	r := &s.buffer
	if r.published != (memory.Block{}) {
		if r.buffer != &m.verification || m.verification.raw != r.published {
			return ErrPolynomialPending
		}
		old := r.published
		e := m.runtime.Deallocate(&m.verification.raw)
		if m.verification.raw == (memory.Block{}) && r.published == old {
			r.published = memory.Block{}
		}
		if e != nil {
			return e
		}
	}
	m.verification.cap, m.verification.pos, m.verification.initialized = 0, 0, 0
	return nil
}
func (m *NaturalManager) cleanupPNVerification(r *naturalPNVerificationReceipt, abort bool) error {
	if !m.matchesPNVerification(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if s.constructed && !s.ended && !s.owned {
		if e := m.releasePNVerificationRaw(s); e != nil {
			return e
		}
		s.owned = true
	}
	if !abort && !s.constructed {
		return nil
	}
	if abort {
		if e := s.buffer.cleanup(); e != nil {
			return e
		}
		if e := m.releasePNVerificationRaw(s); e != nil {
			return e
		}
	}
	if s.buffer.pending == (memory.Block{}) && s.buffer.published == (memory.Block{}) {
		m.verificationActive = false
		*s = naturalPNVerificationScope{invocation: s.invocation, sourceCause: s.sourceCause, constructed: s.constructed, ended: s.ended, owned: s.owned}
	}
	return nil
}
func (m *NaturalManager) pnVerificationSourceCause(r *naturalPNVerificationReceipt) error {
	if r == nil || r.owner != m || r.frame != &m.verificationInvocation || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.sourceCause
}
func (m *IntegerModularManager) pNormalizeInvocationSourceCause(r *integerPNormalizeInvocationReceipt) error {
	if r == nil || r.manager != m || r.frame != &m.normalization || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.sourceCause
}
