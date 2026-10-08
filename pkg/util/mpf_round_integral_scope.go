// Source: src/util/mpf.cpp:1100-1170 (scoped_mpf and scoped_mpz lifetimes).
package util

type arbitraryFloatIntegralRoundScope struct {
	manager                        *ArbitraryFloatManager
	invocation                     uint64
	hooks                          NumericInvocationHooks
	integerBinding, naturalBinding NumericCallerBinding
	numeric                        NumericAllocationRoots
	buffers                        [4]NaturalBufferReceipt
	verification                   naturalPNVerificationReceipt
	copyA                          ArbitraryFloatValueCopyReceipt
	a                              ArbitraryFloat
	div, rem                       Int
	aLive, divLive, remLive        bool
	active, executing, finishing   bool
	bodyCause, sourceCause         error
}

// This protects exactly the borrowed primitive of the executing MPF caller.
// Other numeric caller policies may intentionally allow an unrelated receiver
// to operate during returning transport; their existing policy still applies.
func (z *IntegerManager) mpfArithmeticTransportActive() bool {
	if z == nil || z.self != z {
		return false
	}
	m := z.MpfParent
	return m != nil && m.self == m && m.integer == z && (m.integralRound.valid() && m.integralRound.NumericExecuting() || m.addSub.valid() && m.addSub.NumericExecuting() || m.mul.valid() && m.mul.NumericExecuting() || m.div.valid() && m.div.NumericExecuting() || m.fma.valid() && m.fma.NumericExecuting() || m.sqrt.valid() && m.sqrt.NumericExecuting()) && m.memory != nil && m.memory.FailureTransportActive()
}

func (s *arbitraryFloatIntegralRoundScope) valid() bool {
	return s != nil && s.manager != nil && s.manager.self == s.manager && s == &s.manager.integralRound && s.invocation != 0
}
func (s *arbitraryFloatIntegralRoundScope) NumericExecuting() bool {
	return s.valid() && (s.executing || s.finishing)
}
func (s *arbitraryFloatIntegralRoundScope) NumericBound(executing bool) bool {
	return s.valid() && (!executing || s.active && s.executing && !s.finishing && !s.manager.memory.FailureTransportActive())
}
func (s *arbitraryFloatIntegralRoundScope) NumericRoots(z *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
	if !s.NumericBound(true) || z != s.manager.integer {
		return nil
	}
	if local != nil {
		return local
	}
	return &s.numeric
}
func (s *arbitraryFloatIntegralRoundScope) NumericBuffer(n *NaturalManager, index int) *NaturalBufferReceipt {
	if !s.NumericBound(true) || n != s.manager.integer.Natural || index < 0 || index >= len(s.buffers) {
		return nil
	}
	if e := n.BindCallerBuffer(&s.buffers[index], index, s.invocation, &s.hooks); e != nil {
		return nil
	}
	return &s.buffers[index]
}
func (s *arbitraryFloatIntegralRoundScope) NumericVerification(n *NaturalManager) (*NaturalVerificationScope, error) {
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
func (s *arbitraryFloatIntegralRoundScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{Caller: &s.hooks}
}
func (s *arbitraryFloatIntegralRoundScope) power(n uint32) (Int, error) {
	w, e := s.manager.powerTwoWrapper(n, false)
	if e != nil {
		return Int{}, e
	}
	return w.Capture()
}
func (s *arbitraryFloatIntegralRoundScope) bind() error {
	z := s.manager.integer
	if e := z.BindCaller(&s.integerBinding, s, 14); e != nil {
		return e
	}
	if e := z.Natural.BindCaller(&s.naturalBinding, s, 14); e != nil {
		return JoinNumericErrors(e, s.integerBinding.Unbind())
	}
	return nil
}
func (s *arbitraryFloatIntegralRoundScope) unbind() error {
	return JoinNumericErrors(s.naturalBinding.Unbind(), s.integerBinding.Unbind())
}
func (m *ArbitraryFloatManager) finishIntegralRoundSource() error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.integralRound
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
	if e := m.integer.Natural.PnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.rem, &s.remLive}, {&s.div, &s.divLive}, {&s.a.significand, &s.aLive}} {
		if *local.live {
			if _, e := SourceNumeralDel(m.integer, local.value); e != nil {
				s.sourceCause = e
				return e
			}
			*local.live = false
		}
	}
	s.active = s.bodyCause != nil || NumericAllocationPending(&s.numeric) || m.integer.Natural.MatchesPNVerification(&s.verification)
	for i := range s.buffers {
		s.active = s.active || s.buffers[i].PendingField != (Block{})
	}
	return nil
}

func (m *ArbitraryFloatManager) CleanupRoundToIntegral() (cause error) {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	s := &m.integralRound
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
	if e := z.Natural.CleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if e := m.CleanupValueCopy(&s.copyA, false); e != nil {
		return e
	}
	for _, local := range [...]struct {
		value *Int
		live  *bool
	}{{&s.rem, &s.remLive}, {&s.div, &s.divLive}, {&s.a.significand, &s.aLive}} {
		if *local.live {
			if e := z.CleanupRawNumeral(local.value); e != nil {
				return e
			}
			*local.live = false
		}
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
