// Source: src/util/mpn.cpp:206-217; src/util/buffer.h.
package util

// The reached lexical constructor has its own identity. An unfinished
// construction owns exact raw/pending leaves but has no source destructor.
type naturalPNVerificationScope struct {
	caller *NumericInvocationHooks

	rationalSetContext integerAllocationInvocationContext

	roundContext  integerAllocationInvocationContext
	formatContext integerAllocationInvocationContext

	boolDivides integerDividesNumeralInvocationReceipt

	raw        integerRawInvocationReceipt
	invocation uint64
	parent     *integerPNormalizeScope

	active, executing, Constructed, Ended, Owned bool
	Buffer                                       naturalPNBufferReceipt
	SourceCause                                  error
}
type naturalPNVerificationReceipt struct {
	owner      *NaturalManager
	Frame      *naturalPNVerificationScope
	Invocation uint64
}

func (m *NaturalManager) MatchesPNVerification(r *naturalPNVerificationReceipt) bool {
	return r != nil && r.owner == m && r.Frame == &m.VerificationInvocation && r.Invocation != 0 && r.Invocation == r.Frame.invocation && r.Frame.active
}
func (m *NaturalManager) FinishPNVerificationSource(s *naturalPNVerificationScope) error {
	if s != &m.VerificationInvocation {
		return nil
	}
	if s.SourceCause != nil {
		return s.SourceCause
	}
	if !s.active || !s.Constructed || s.Ended || s.Owned {
		return nil
	}
	if e := m.releasePNVerificationRaw(s); e != nil {
		s.SourceCause = e
		return e
	}
	s.Ended = true
	m.VerificationActive = false
	s.active = false
	return nil
}
func (m *NaturalManager) releasePNVerificationRaw(s *naturalPNVerificationScope) error {
	r := &s.Buffer
	if r.Published != (Block{}) {
		if r.Buffer != &m.Verification || m.Verification.Raw != r.Published {
			return ErrNumericPending
		}
		old := r.Published
		e := m.runtime.Deallocate(&m.Verification.Raw)
		if m.Verification.Raw == (Block{}) && r.Published == old {
			r.Published = Block{}
		}
		if e != nil {
			return e
		}
	}
	m.Verification.cap, m.Verification.pos, m.Verification.initialized = 0, 0, 0
	return nil
}
func (m *NaturalManager) CleanupPNVerification(r *naturalPNVerificationReceipt, abort bool) error {
	if !m.MatchesPNVerification(r) {
		return nil
	}
	s := r.Frame
	if s.executing {
		return ErrNumericPending
	}
	if s.Constructed && !s.Ended && !s.Owned {
		if e := m.releasePNVerificationRaw(s); e != nil {
			return e
		}
		s.Owned = true
	}
	if !abort && !s.Constructed {
		return nil
	}
	if abort {
		if e := s.Buffer.cleanup(); e != nil {
			return e
		}
		if e := m.releasePNVerificationRaw(s); e != nil {
			return e
		}
	}
	if s.Buffer.PendingField == (Block{}) && s.Buffer.Published == (Block{}) {
		m.VerificationActive = false
		*s = naturalPNVerificationScope{invocation: s.invocation, SourceCause: s.SourceCause, Constructed: s.Constructed, Ended: s.Ended, Owned: s.Owned}
	}
	return nil
}
func (m *NaturalManager) PnVerificationSourceCause(r *naturalPNVerificationReceipt) error {
	if r == nil || r.owner != m || r.Frame != &m.VerificationInvocation || r.Invocation == 0 || r.Invocation != r.Frame.invocation {
		return nil
	}
	return r.Frame.SourceCause
}
func (m *IntegerModularManager) PNormalizeInvocationSourceCause(r *integerPNormalizeInvocationReceipt) error {
	if r == nil || r.Manager != m || r.Frame != &m.Normalization || r.Invocation == 0 || r.Invocation != r.Frame.invocation {
		return nil
	}
	return r.Frame.sourceCause
}
