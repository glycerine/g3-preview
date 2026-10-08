// Source: src/util/mpf.cpp:1462-1481 (ST0/ST1 and actual descendant lifetimes).
package util

type arbitraryFloatRemScope struct {
	manager                        *ArbitraryFloatManager
	invocation                     uint64
	hooks                          NumericInvocationHooks
	integerBinding, naturalBinding NumericCallerBinding
	numeric                        NumericAllocationRoots
	buffers                        [4]NaturalBufferReceipt
	verification                   naturalPNVerificationReceipt
	copyA, copyB                   ArbitraryFloatValueCopyReceipt
	roundFrame                     *ArbitraryFloatRoundScope
	roundInvocation                uint64
	roundReceipt                   ArbitraryFloatRoundReceipt
	partialInvocation              uint64
	partialDone                    bool
	formatReceipt                  ArbitraryFloatFormattedSetReceipt
	formatSourceCause              error
	a, b                           ArbitraryFloat
	aLive, bLive                   bool
	active, executing, finishing   bool
	bodyCause, sourceCause         error
}

func (s *arbitraryFloatRemScope) valid() bool {
	return s != nil && s.manager != nil && s.manager.self == s.manager && s == &s.manager.rem && s.invocation != 0
}
func (s *arbitraryFloatRemScope) NumericExecuting() bool {
	return s.valid() && (s.executing || s.finishing)
}
func (s *arbitraryFloatRemScope) NumericBound(executing bool) bool {
	return s.valid() && (!executing || s.active && s.executing && !s.finishing && !s.manager.memory.FailureTransportActive())
}
func (s *arbitraryFloatRemScope) NumericRoots(z *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
	if !s.NumericBound(true) || z != s.manager.integer {
		return nil
	}
	if local != nil {
		return local
	}
	return &s.numeric
}
func (s *arbitraryFloatRemScope) NumericBuffer(n *NaturalManager, index int) *NaturalBufferReceipt {
	if !s.NumericBound(true) || n != s.manager.integer.Natural || index < 0 || index >= len(s.buffers) {
		return nil
	}
	if e := n.BindCallerBuffer(&s.buffers[index], index, s.invocation, &s.hooks); e != nil {
		return nil
	}
	return &s.buffers[index]
}
func (s *arbitraryFloatRemScope) NumericVerification(n *NaturalManager) (*NaturalVerificationScope, error) {
	if !s.NumericBound(true) || n != s.manager.integer.Natural {
		return nil, ErrNumericOwner
	}
	f, e := n.BeginCallerVerification(&s.hooks)
	if e != nil {
		return nil, e
	}
	s.verification = naturalPNVerificationReceipt{n, f, f.invocation}
	return f, nil
}
func (s *arbitraryFloatRemScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{Caller: &s.hooks}
}
func (s *arbitraryFloatRemScope) power(n uint32) (Int, error) {
	w, e := s.manager.powerTwoWrapper(n, false)
	if e != nil {
		return Int{}, e
	}
	return w.Capture()
}
func (s *arbitraryFloatRemScope) bind() error {
	z := s.manager.integer
	if e := z.BindCaller(&s.integerBinding, s, 23); e != nil {
		return e
	}
	if e := z.Natural.BindCaller(&s.naturalBinding, s, 23); e != nil {
		return JoinNumericErrors(e, s.integerBinding.Unbind())
	}
	return nil
}
func (s *arbitraryFloatRemScope) unbind() error {
	return JoinNumericErrors(s.naturalBinding.Unbind(), s.integerBinding.Unbind())
}
func (m *ArbitraryFloatManager) finishRemSource() error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.rem
	if s.executing || s.finishing {
		return ErrNumericPending
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active {
		return nil
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	if s.roundFrame != nil {
		if s.roundFrame != &m.RoundScope || s.roundInvocation != m.RoundScope.Invocation {
			return ErrRuntimeOwner
		}
		if s.roundFrame.sourceCause != nil {
			s.sourceCause = s.roundFrame.sourceCause
			return s.sourceCause
		}
	}
	if s.formatSourceCause != nil {
		s.sourceCause = s.formatSourceCause
		return s.sourceCause
	}
	if s.partialInvocation != 0 && !s.partialDone {
		child := &m.partialRemainder
		if child.invocation != s.partialInvocation {
			return ErrRuntimeOwner
		}
		if child.sourceCause != nil {
			s.sourceCause = child.sourceCause
			return s.sourceCause
		}
	}
	if e := m.integer.Natural.PnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.b.significand, &s.bLive}, {&s.a.significand, &s.aLive}} {
		if *local.live {
			if _, e := SourceNumeralDel(m.integer, local.value); e != nil {
				s.sourceCause = e
				return e
			}
			*local.live = false
		}
	}
	s.active = s.bodyCause != nil || NumericAllocationPending(&s.numeric) || m.integer.Natural.MatchesPNVerification(&s.verification) || s.roundFrame != nil && s.roundFrame.active
	for i := range s.buffers {
		s.active = s.active || s.buffers[i].PendingField != (Block{})
	}
	return nil
}

func (m *ArbitraryFloatManager) CleanupRem() (cause error) {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.rem
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrNumericPending
	}
	if !s.active {
		s.roundFrame = nil
		s.roundInvocation = 0
		s.partialInvocation = 0
		return nil
	}
	if !s.valid() {
		return ErrRuntimeOwner
	}
	if e := s.bind(); e != nil {
		return e
	}
	s.finishing = true
	defer func() { s.finishing = false; cause = JoinNumericErrors(cause, s.unbind()) }()
	z := m.integer
	if s.roundFrame != nil {
		if s.roundFrame != &m.RoundScope || s.roundInvocation != m.RoundScope.Invocation {
			return ErrRuntimeOwner
		}
		if e := m.CleanupRound(); e != nil {
			return e
		}
	}
	if e := m.CleanupFormattedSetWithReceipt(&s.formatReceipt, false); e != nil {
		return e
	}
	if s.partialInvocation != 0 && !s.partialDone {
		if m.partialRemainder.invocation != s.partialInvocation {
			return ErrRuntimeOwner
		}
		if e := m.CleanupPartialRemainder(); e != nil {
			return e
		}
		s.partialDone = true
	}
	if e := z.Natural.CleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyB, false); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyA, false); e != nil {
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.b.significand, &s.bLive}, {&s.a.significand, &s.aLive}} {
		if *local.live {
			if e := z.CleanupRawNumeral(local.value); e != nil {
				return e
			}
			*local.live = false
		}
	}
	if e := m.CleanupFormattedSetWithReceipt(&s.formatReceipt, true); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyB, true); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyA, true); e != nil {
		return e
	}
	if e := z.Natural.CleanupPNVerification(&s.verification, true); e != nil {
		return e
	}
	for i := len(s.buffers) - 1; i >= 0; i-- {
		if e := s.buffers[i].Cleanup(); e != nil {
			return e
		}
	}
	if e := s.numeric.Cleanup(z); e != nil {
		return e
	}
	s.active = false
	return nil
}

// The actual Round entry stamps its new child tuple, including nonallocating
// invocations for which no history receipt is materialized.
func (s *arbitraryFloatRemScope) enterArbitraryFloatRound(m *ArbitraryFloatManager, frame *ArbitraryFloatRoundScope) error {
	if !s.NumericBound(true) || m != s.manager || frame != &m.RoundScope || !frame.active || !frame.executing {
		return ErrRuntimeOwner
	}
	s.roundFrame = frame
	s.roundInvocation = frame.Invocation
	return nil
}
