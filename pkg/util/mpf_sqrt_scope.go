// Source: src/util/mpf.cpp:1023-1042 (a, actual Integer root and Round lifetimes).
package util

type arbitraryFloatSqrtScope struct {
	manager                        *ArbitraryFloatManager
	invocation                     uint64
	hooks                          NumericInvocationHooks
	integerBinding, naturalBinding NumericCallerBinding
	numeric                        NumericAllocationRoots
	buffers                        [4]NaturalBufferReceipt
	verification                   naturalPNVerificationReceipt
	copyA                          ArbitraryFloatValueCopyReceipt
	rootCopy                       integerCopyAllocationReceipt
	root                           integerRootInvocationReceipt
	roundFrame                     *ArbitraryFloatRoundScope
	roundInvocation                uint64
	roundReceipt                   ArbitraryFloatRoundReceipt
	a                              ArbitraryFloat
	aLive                          bool
	active, executing, finishing   bool
	bodyCause, sourceCause         error
}

func (s *arbitraryFloatSqrtScope) valid() bool {
	return s != nil && s.manager != nil && s.manager.self == s.manager && s == &s.manager.sqrt && s.invocation != 0
}
func (s *arbitraryFloatSqrtScope) NumericExecuting() bool {
	return s.valid() && (s.executing || s.finishing)
}
func (s *arbitraryFloatSqrtScope) NumericBound(executing bool) bool {
	return s.valid() && (!executing || s.active && s.executing && !s.finishing && !s.manager.memory.FailureTransportActive())
}
func (s *arbitraryFloatSqrtScope) NumericRoots(z *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
	if !s.NumericBound(true) || z != s.manager.integer {
		return nil
	}
	if local != nil {
		return local
	}
	return &s.numeric
}
func (s *arbitraryFloatSqrtScope) NumericBuffer(n *NaturalManager, index int) *NaturalBufferReceipt {
	if !s.NumericBound(true) || n != s.manager.integer.Natural || index < 0 || index >= len(s.buffers) {
		return nil
	}
	if e := n.BindCallerBuffer(&s.buffers[index], index, s.invocation, &s.hooks); e != nil {
		return nil
	}
	return &s.buffers[index]
}
func (s *arbitraryFloatSqrtScope) NumericVerification(n *NaturalManager) (*NaturalVerificationScope, error) {
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
func (s *arbitraryFloatSqrtScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{Caller: &s.hooks}
}
func (s *arbitraryFloatSqrtScope) power(n uint32) (Int, error) {
	w, e := s.manager.powerTwoWrapper(n, false)
	if e != nil {
		return Int{}, e
	}
	return w.Capture()
}
func (s *arbitraryFloatSqrtScope) bind() error {
	z := s.manager.integer
	if e := z.BindCaller(&s.integerBinding, s, 19); e != nil {
		return e
	}
	if e := z.Natural.BindCaller(&s.naturalBinding, s, 19); e != nil {
		return JoinNumericErrors(e, s.integerBinding.Unbind())
	}
	return nil
}
func (s *arbitraryFloatSqrtScope) unbind() error {
	return JoinNumericErrors(s.naturalBinding.Unbind(), s.integerBinding.Unbind())
}
func (m *ArbitraryFloatManager) finishSqrtSource() error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.sqrt
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
	if e := m.integer.RootInvocationSourceCause(&s.root); e != nil {
		s.sourceCause = e
		return e
	}
	if e := m.integer.Natural.PnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.a.significand, &s.aLive}} {
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

func (m *ArbitraryFloatManager) CleanupSqrt() (cause error) {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.sqrt
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrNumericPending
	}
	if !s.active {
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
	if e := z.CleanupRootInvocation(&s.root, false); e != nil {
		return e
	}
	if e := z.Natural.CleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyA, false); e != nil {
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.a.significand, &s.aLive}} {
		if *local.live {
			if e := z.CleanupRawNumeral(local.value); e != nil {
				return e
			}
			*local.live = false
		}
	}
	if e := z.CleanupRootInvocation(&s.root, true); e != nil {
		return e
	}
	if e := z.CleanupCopyAllocationInvocation(&s.rootCopy); e != nil {
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
func (s *arbitraryFloatSqrtScope) enterArbitraryFloatRound(m *ArbitraryFloatManager, frame *ArbitraryFloatRoundScope) error {
	if !s.NumericBound(true) || m != s.manager || frame != &m.RoundScope || !frame.active || !frame.executing {
		return ErrRuntimeOwner
	}
	s.roundFrame = frame
	s.roundInvocation = frame.Invocation
	return nil
}
