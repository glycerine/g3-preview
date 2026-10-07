// Source: src/util/mpzzp.h:58-68,111-114; src/util/mpz.cpp:quot_rem_core.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

// Direct transport of actual integer-cell and source-buffer acquisition leaves.
// Whole-parent and exhaustive prefix validation remain open.
type integerPNormalizeScope struct {
	factorHistoryContext integerAllocationInvocationContext
	owner                *IntegerModularManager
	invocation           uint64
	Stage                uint8
	active, executing    bool
	Roots                NumericAllocationRoots
	buffers              [4]naturalPNBufferReceipt
	Verification         naturalPNVerificationReceipt
	sourceCause          error
}
type integerPNormalizeInvocationReceipt struct {
	Manager    *IntegerModularManager
	Frame      *integerPNormalizeScope
	Invocation uint64
}

func (m *IntegerModularManager) MatchesPNormalizeInvocation(r *integerPNormalizeInvocationReceipt) bool {
	return r != nil && r.Manager == m && r.Frame == &m.Normalization && r.Invocation != 0 && r.Invocation == r.Frame.invocation && r.Frame.active
}
func (m *IntegerModularManager) CleanupPNormalizeInvocation(r *integerPNormalizeInvocationReceipt, abort bool) error {
	if !m.MatchesPNormalizeInvocation(r) {
		return nil
	}
	s := r.Frame
	if s.executing {
		return ErrNumericPending
	}
	if e := m.manager.Natural.CleanupPNVerification(&s.Verification, abort); e != nil {
		return e
	}
	if !abort {
		return nil
	}
	for i := len(s.buffers) - 1; i >= 0; i-- {
		if e := s.buffers[i].cleanup(); e != nil {
			return e
		}
	}
	if e := s.Roots.Cleanup(m.manager); e != nil {
		return e
	}
	*s = integerPNormalizeScope{owner: m, invocation: s.invocation, Verification: s.Verification, sourceCause: s.sourceCause}
	return nil
}

// A receipt exists only for a real pending growth of the named source buffer.
// Successful publication returns authority to the persistent Natural member.
type naturalPNBufferReceipt struct {
	CallerSelf *naturalPNBufferReceipt
	Caller     *NumericInvocationHooks

	rationalSetContext integerAllocationInvocationContext

	roundContext  integerAllocationInvocationContext
	formatContext integerAllocationInvocationContext

	BoolDivides integerDividesNumeralInvocationReceipt

	raw integerRawInvocationReceipt

	Owner        *NaturalManager
	context      *integerPNormalizeScope
	Invocation   uint64
	Buffer       *SourceLimbBuffer
	PendingField Block
	stage        uint8
	private      bool
	Published    Block
}

func (m *NaturalManager) pnBufferReceipt(pn *integerPNormalizeScope, i int) *naturalPNBufferReceipt {
	if pn == nil || pn.owner == nil || pn.owner.manager.Natural != m || pn != &pn.owner.Normalization || !pn.active || !pn.executing {
		return nil
	}
	r := &pn.buffers[i]
	if r.context == nil {
		*r = naturalPNBufferReceipt{Owner: m, context: pn, Invocation: pn.invocation, Buffer: &m.Division[i], stage: pn.Stage}
	}
	return r
}
func (r *naturalPNBufferReceipt) acquire(s *SourceLimbBuffer, block Block) {
	if r.Buffer == s && r.Bound(true) {
		r.PendingField = block
		if c := r.rationalSetContext; c.mpfRationalSet != nil {
			q := c.rationalSetFrame(c.mpfRationalSet.manager.integer, true)
			if q != nil && q.history != nil {
				if r.private {
					q.history.naturalOriginal[4] = block
				} else {
					for i := range q.buffers {
						if r == &q.buffers[i] {
							q.history.naturalOriginal[i] = block
						}
					}
				}
			}
		}

		if c := r.roundContext; c.mpfRound != nil {
			q := c.roundFrame(c.mpfRound.Manager.integer, true)
			if q != nil && q.history != nil {
				if r.private {
					q.history.naturalOriginal[4] = block
				} else {
					for i := range q.buffers {
						if r == &q.buffers[i] {
							q.history.naturalOriginal[i] = block
						}
					}
				}
			}
		}

	}
}
func (r *naturalPNBufferReceipt) publish(s *SourceLimbBuffer, block Block) {
	if r.Buffer == s && r.PendingField == block {
		if r.private {
			r.Published = block
		}
		r.PendingField = Block{}
	}
}
func (r *naturalPNBufferReceipt) cleanup() error {
	if r.PendingField == (Block{}) {
		return nil
	}
	if !r.Bound(false) || r.Buffer.Pending != r.PendingField {
		return nil
	}
	e := r.Owner.runtime.Deallocate(&r.Buffer.Pending)
	if r.Buffer.Pending == (Block{}) {
		r.PendingField = Block{}
	}
	return e
}
func (s *integerPNormalizeScope) pending() bool {
	if s.owner.manager.Natural.MatchesPNVerification(&s.Verification) {
		return true
	}
	if NumericAllocationPending(&s.Roots) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].PendingField != (Block{}) {
			return true
		}
	}
	return false
}

func (r *naturalPNBufferReceipt) Bound(executing bool) bool {
	if r.Caller != nil {
		return r.CallerSelf == r && r.Owner != nil && r.Owner.self == r.Owner && r.Invocation != 0 && r.Buffer != nil && r.Caller.callBound(executing)
	}

	if c := r.rationalSetContext; c.mpfRationalSet != nil {
		s := c.mpfRationalSet
		if r.Owner == nil || s.manager == nil || r.Owner != s.manager.integer.Natural || c.rationalSetFrame(s.manager.integer, executing) == nil || r.Invocation != s.invocation {
			return false
		}
		if r.private {
			v := &r.Owner.VerificationInvocation
			return r == &v.Buffer && r.Buffer == &r.Owner.Verification && v.rationalSetContext.mpfRationalSet == s && v.rationalSetContext.mpfRationalSetInvocation == s.invocation && r.Owner.MatchesPNVerification(&s.verification)
		}
		for i := range s.buffers {
			if r == &s.buffers[i] {
				return r.Buffer == &r.Owner.Division[i]
			}
		}
		return false
	}

	if c := r.roundContext; c.mpfRound != nil {
		s := c.mpfRound
		if r.Owner == nil || s.Manager == nil || r.Owner != s.Manager.integer.Natural || c.roundFrame(s.Manager.integer, executing) == nil || r.Invocation != s.Invocation {
			return false
		}
		if r.private {
			v := &r.Owner.VerificationInvocation
			return r == &v.Buffer && r.Buffer == &r.Owner.Verification && v.roundContext.mpfRound == s && v.roundContext.mpfRoundInvocation == s.Invocation && r.Owner.MatchesPNVerification(&s.verification)
		}
		for i := range s.buffers {
			if r == &s.buffers[i] {
				return r.Buffer == &r.Owner.Division[i]
			}
		}
		return false
	}

	if c := r.formatContext; c.mpfFormat != nil {
		s := c.mpfFormat
		if r.Owner == nil || s.manager == nil || r.Owner != s.manager.integer.Natural || c.formatFrame(s.manager.integer, executing) == nil || r.Invocation != s.invocation {
			return false
		}
		buffers := &s.buffers
		verification := &s.verification
		if c.formatDecimal != nil {
			buffers = &c.formatDecimal.buffers
			verification = &c.formatDecimal.verification
		}
		if r.private {
			v := &r.Owner.VerificationInvocation
			return r == &v.Buffer && r.Buffer == &r.Owner.Verification && v.formatContext.mpfFormat == s && v.formatContext.mpfFormatInvocation == s.invocation && v.formatContext.formatDecimal == c.formatDecimal && r.Owner.MatchesPNVerification(verification)
		}
		for i := range buffers {
			if r == &buffers[i] {
				return r.Buffer == &r.Owner.Division[i]
			}
		}
		return false
	}

	if s := r.BoolDivides; s.Manager != nil {
		if !s.Manager.MatchesDividesNumeralInvocation(&s) || r.Owner != s.Manager.Natural || r.Invocation != s.Invocation || (executing && s.Manager.dividesNumeralContextRoots(integerAllocationInvocationContext{BoolDivides: &s, BoolDividesInvocation: s.Invocation}) == nil) {
			return false
		}
		if r.private {
			v := &r.Owner.VerificationInvocation
			return r == &v.Buffer && r.Buffer == &r.Owner.Verification && v.boolDivides == s && r.Owner.MatchesPNVerification(&s.Frame.verification)
		}
		for i := range s.Frame.Buffers {
			if r == &s.Frame.Buffers[i] {
				return r.Buffer == &r.Owner.Division[i]
			}
		}
		return false
	}

	if r.raw.Manager != nil {
		s := r.raw
		if !s.Manager.MatchesRawInvocation(&s) || r.Owner != s.Manager.Natural || r.Invocation != s.Invocation || (executing && s.Manager.RawAllocationRoots(&s) == nil) {
			return false
		}
		if r.private {
			v := &r.Owner.VerificationInvocation
			return r == &v.Buffer && r.Buffer == &r.Owner.Verification && v.raw == s && r.Owner.MatchesPNVerification(&s.Frame.verification)
		}
		for i := range s.Frame.buffers {
			if r == &s.Frame.buffers[i] {
				return r.Buffer == &r.Owner.Division[i]
			}
		}
		return false
	}

	return r.context != nil && r.context.active && r.Invocation == r.context.invocation && (!executing || r.context.executing)
}
func (m *NaturalManager) DivisionBufferReceipt(context integerAllocationInvocationContext, i int) *naturalPNBufferReceipt {
	if !context.hasLocalInvocation() && context.Caller != nil && context.Caller.hasBuffer() {
		return context.Caller.callBuffer(m, i)
	}
	if context.mpfRound != nil {
		s := context.roundFrame(context.mpfRound.Manager.integer, true)
		if s == nil || s.Manager.integer.Natural != m {
			return nil
		}
		b := &s.buffers[i]
		if b.Owner == nil {
			*b = naturalPNBufferReceipt{Owner: m, roundContext: context, Invocation: s.Invocation, Buffer: &m.Division[i]}
		}
		return b
	}

	if r := context.BoolDivides; r != nil {
		if r.Manager == nil || r.Manager.Natural != m || r.Manager.dividesNumeralContextRoots(context) == nil {
			return nil
		}
		b := &r.Frame.Buffers[i]
		if b.Owner == nil {
			*b = naturalPNBufferReceipt{Owner: m, BoolDivides: *r, Invocation: r.Invocation, Buffer: &m.Division[i]}
		}
		return b
	}

	if r := context.raw; r != nil && r.Manager != nil && r.Manager.Natural == m && context.rawReceipt(r.Manager) != nil {
		b := &r.Frame.buffers[i]
		if b.Owner == nil {
			*b = naturalPNBufferReceipt{Owner: m, raw: *r, Invocation: r.Invocation, Buffer: &m.Division[i]}
		}
		return b
	}

	if context.raw != nil {
		return nil
	}
	if context.mpfRationalSet != nil {
		s := context.rationalSetFrame(context.mpfRationalSet.manager.integer, true)
		if s == nil || s.manager.integer.Natural != m {
			return nil
		}
		b := &s.buffers[i]
		if b.Owner == nil {
			*b = naturalPNBufferReceipt{Owner: m, rationalSetContext: context, Invocation: s.invocation, Buffer: &m.Division[i]}
		}
		return b
	}
	if context.mpfFormat != nil {
		q := context.formatFrame(context.mpfFormat.manager.integer, true)
		if q == nil || q.manager.integer.Natural != m {
			return nil
		}
		buffers := &q.buffers
		if context.formatDecimal != nil {
			buffers = &context.formatDecimal.buffers
		}
		b := &buffers[i]
		if b.Owner == nil {
			*b = naturalPNBufferReceipt{Owner: m, formatContext: context, Invocation: q.invocation, Buffer: &m.Division[i]}
		}
		return b
	}

	return m.pnBufferReceipt(context.pn, i)
}
