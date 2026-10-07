// Source: src/util/mpf.cpp:2100-2131 (shared concrete allocation/lifetime engine).
// Source: src/util/mpf.cpp:1963-2098.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

// Actual lexical NUM storage is reflected in the MPF owner header. Declaration
// flags change only at literal reached constructors; sticky is branch-scoped.
type ArbitraryFloatRoundScope struct {
	parentFormattedSet                                  integerAllocationInvocationContext
	receipt                                             *ArbitraryFloatRoundReceipt
	Numeric                                             NumericAllocationRoots
	history                                             *arbitraryFloatRoundHistory
	Transferred                                         SmallAllocatorPendingAllocation
	cellRetired                                         bool
	buffers                                             [4]naturalPNBufferReceipt
	verification                                        naturalPNVerificationReceipt
	Manager                                             *ArbitraryFloatManager
	Invocation                                          uint64
	sigma, cap, sticky                                  Int
	sigmaConstructed, capConstructed, stickyConstructed bool
	active, executing, finishing                        bool
	bodyCause, sourceCause, ownCause                    error
}

func (m *ArbitraryFloatManager) finishRoundSource() error {
	return m.finishRoundScopeSource(&m.RoundScope)
}
func (m *ArbitraryFloatManager) finishRoundSqrtSource() error {
	return m.finishRoundScopeSource(&m.RoundSqrtScope)
}
func (m *ArbitraryFloatManager) finishRoundScopeSource(s *ArbitraryFloatRoundScope) error {
	if m == nil || m.self != m || (s != &m.RoundScope && s != &m.RoundSqrtScope) {
		return ErrRuntimeOwner
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active {
		return nil
	}
	if s.executing || s.finishing {
		return ErrNumericPending
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordRoundHistory() }()
	if e := m.integer.Natural.PnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}

	for _, a := range []struct {
		v           *Int
		constructed *bool
	}{{&s.sticky, &s.stickyConstructed}, {&s.cap, &s.capConstructed}, {&s.sigma, &s.sigmaConstructed}} {
		if *a.constructed {
			if _, e := SourceNumeralDel(m.integer, a.v); e != nil {
				s.sourceCause = e
				return e
			}
			*a.constructed = false
		}
	}
	s.active = NumericAllocationPending(&s.Numeric) || s.Transferred.Block() != (Block{})
	for i := range s.buffers {
		if s.buffers[i].PendingField != (Block{}) {
			s.active = true
		}
	}
	if m.integer.Natural.MatchesPNVerification(&s.verification) {
		s.active = true
	}
	if !s.active && s.history != nil {
		s.history.owned = true
	}
	return nil
}
func (m *ArbitraryFloatManager) CleanupRound() error {
	if e := m.CleanupRoundWithReceipt(nil, false); e != nil {
		return e
	}
	return m.CleanupRoundWithReceipt(nil, true)
}
func (m *ArbitraryFloatManager) CleanupRoundWithReceipt(r *ArbitraryFloatRoundReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	return m.cleanupRoundScope(&m.RoundScope, r, abort)
}
func (m *ArbitraryFloatManager) CleanupRoundSqrt() error {
	if e := m.CleanupRoundSqrtWithReceipt(nil, false); e != nil {
		return e
	}
	return m.CleanupRoundSqrtWithReceipt(nil, true)
}
func (m *ArbitraryFloatManager) CleanupRoundSqrtWithReceipt(r *ArbitraryFloatRoundReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	return m.cleanupRoundScope(&m.RoundSqrtScope, r, abort)
}
func (m *ArbitraryFloatManager) cleanupRoundScope(s *ArbitraryFloatRoundScope, r *ArbitraryFloatRoundReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if s != &m.RoundScope && s != &m.RoundSqrtScope {
		return ErrRuntimeOwner
	}
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrNumericPending
	}

	if r != nil {
		if *r == (ArbitraryFloatRoundReceipt{}) {
			return nil
		}
		h, e := m.roundHistory(r)
		if e != nil {
			return e
		}
		if h.owned {
			return nil
		}
		if s.history != h || s.Invocation != h.invocation {
			return ErrRuntimeOwner
		}
	}
	if s.Manager == nil {
		return nil
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordRoundHistory() }()
	if e := m.integer.Natural.CleanupPNVerification(&s.verification, false); e != nil {
		if s.ownCause == nil {
			s.ownCause = e
		}
		return e
	}

	for _, a := range []struct {
		v           *Int
		constructed *bool
	}{{&s.sticky, &s.stickyConstructed}, {&s.cap, &s.capConstructed}, {&s.sigma, &s.sigmaConstructed}} {
		if *a.constructed {
			if e := m.integer.CleanupRawNumeral(a.v); e != nil {
				if s.ownCause == nil {
					s.ownCause = e
				}
				return e
			}
			*a.constructed = false
		}
	}

	if !abort {
		return nil
	}
	if e := m.integer.Natural.CleanupPNVerification(&s.verification, true); e != nil {
		if s.ownCause == nil {
			s.ownCause = e
		}
		return e
	}
	for i := len(s.buffers) - 1; i >= 0; i-- {
		if e := s.buffers[i].cleanup(); e != nil {
			if s.ownCause == nil {
				s.ownCause = e
			}
			return e
		}
	}
	if e := m.integer.cleanupFormatAllocation(&s.Numeric, &s.Transferred, &s.cellRetired); e != nil {
		if s.ownCause == nil {
			s.ownCause = e
		}
		return e
	}
	s.active = false
	if s.history != nil {
		s.history.owned = true
	}
	return nil
}
