// Source: src/util/mpf.cpp:2100-2131 (shared concrete allocation/lifetime engine).
// Source: src/util/mpf.cpp:1963-2098.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// Actual lexical NUM storage is reflected in the MPF owner header. Declaration
// flags change only at literal reached constructors; sticky is branch-scoped.
type arbitraryFloatRoundScope struct {
	receipt                                             *ArbitraryFloatRoundReceipt
	numeric                                             polynomialNumericRoots
	history                                             *arbitraryFloatRoundHistory
	transferred                                         memory.SmallAllocatorPendingAllocation
	cellRetired                                         bool
	buffers                                             [4]naturalPNBufferReceipt
	verification                                        naturalPNVerificationReceipt
	manager                                             *ArbitraryFloatManager
	invocation                                          uint64
	sigma, cap, sticky                                  Int
	sigmaConstructed, capConstructed, stickyConstructed bool
	active, executing, finishing                        bool
	bodyCause, sourceCause, ownCause                    error
}

func (m *ArbitraryFloatManager) finishRoundSource() error {
	return m.finishRoundScopeSource(&m.roundScope)
}
func (m *ArbitraryFloatManager) finishRoundSqrtSource() error {
	return m.finishRoundScopeSource(&m.roundSqrtScope)
}
func (m *ArbitraryFloatManager) finishRoundScopeSource(s *arbitraryFloatRoundScope) error {
	if m == nil || m.self != m || (s != &m.roundScope && s != &m.roundSqrtScope) {
		return memory.ErrRuntimeOwner
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active {
		return nil
	}
	if s.executing || s.finishing {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordRoundHistory() }()
	if e := m.integer.natural.pnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}

	for _, a := range []struct {
		v           *Int
		constructed *bool
	}{{&s.sticky, &s.stickyConstructed}, {&s.cap, &s.capConstructed}, {&s.sigma, &s.sigmaConstructed}} {
		if *a.constructed {
			if _, e := upolynomialSourceNumeralDel(m.integer, a.v); e != nil {
				s.sourceCause = e
				return e
			}
			*a.constructed = false
		}
	}
	s.active = polynomialNegNumericPending(&s.numeric) || s.transferred.Block() != (memory.Block{})
	for i := range s.buffers {
		if s.buffers[i].pending != (memory.Block{}) {
			s.active = true
		}
	}
	if m.integer.natural.matchesPNVerification(&s.verification) {
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
		return memory.ErrRuntimeOwner
	}
	return m.cleanupRoundScope(&m.roundScope, r, abort)
}
func (m *ArbitraryFloatManager) CleanupRoundSqrt() error {
	if e := m.CleanupRoundSqrtWithReceipt(nil, false); e != nil {
		return e
	}
	return m.CleanupRoundSqrtWithReceipt(nil, true)
}
func (m *ArbitraryFloatManager) CleanupRoundSqrtWithReceipt(r *ArbitraryFloatRoundReceipt, abort bool) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	return m.cleanupRoundScope(&m.roundSqrtScope, r, abort)
}
func (m *ArbitraryFloatManager) cleanupRoundScope(s *arbitraryFloatRoundScope, r *ArbitraryFloatRoundReceipt, abort bool) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if s != &m.roundScope && s != &m.roundSqrtScope {
		return memory.ErrRuntimeOwner
	}
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrPolynomialPending
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
		if s.history != h || s.invocation != h.invocation {
			return memory.ErrRuntimeOwner
		}
	}
	if s.manager == nil {
		return nil
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordRoundHistory() }()
	if e := m.integer.natural.cleanupPNVerification(&s.verification, false); e != nil {
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
			if e := m.integer.cleanupRawNumeral(a.v); e != nil {
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
	if e := m.integer.natural.cleanupPNVerification(&s.verification, true); e != nil {
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
	if e := m.integer.cleanupFormatAllocation(&s.numeric, &s.transferred, &s.cellRetired); e != nil {
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
