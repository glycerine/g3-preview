// Source: src/util/mpz.cpp:2054-2069; src/util/scoped_numeral.h:31-33.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// The actual Integer method owns its scoped tmp NUM, including the zero
// branch. Its younger Natural verification child is distinct from shared
// successful division buffers. There is no RAW end-only DEL in this method.
type integerDividesNumeralScope struct {
	owner                                        *IntegerManager
	receipt                                      integerDividesNumeralInvocationReceipt
	parentContext                                integerAllocationInvocationContext
	invocation                                   uint64
	active, executing, constructed, ended, owned bool
	numeral                                      Int
	numeric                                      polynomialNumericRoots
	buffers                                      [4]naturalPNBufferReceipt
	verification                                 naturalPNVerificationReceipt
	bodyCause, sourceCause                       error
}
type integerDividesNumeralInvocationReceipt struct {
	manager    *IntegerManager
	frame      *integerDividesNumeralScope
	invocation uint64
}

func (m *IntegerManager) beginDividesNumeralInvocation(parent integerAllocationInvocationContext) (*integerDividesNumeralInvocationReceipt, error) {
	if m == nil || m.self != m || m.closed || m.sourceFinalizing || m.constructionFailed {
		return nil, ErrPolynomialOwner
	}
	if parent != (integerAllocationInvocationContext{}) && parent.roots(m) == nil {
		return nil, ErrPolynomialOwner
	}
	s := &m.dividesNumeral
	if s.active || s.invocation == ^uint64(0) {
		return nil, ErrPolynomialPending
	}
	*s = integerDividesNumeralScope{owner: m, invocation: s.invocation + 1, active: true, executing: true, constructed: true, parentContext: parent}
	s.receipt = integerDividesNumeralInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	if m.dividesContextRoots(parent) != nil {
		parent.divides.frame.boolNumeral = s.receipt
	}
	if m.scalarExactContextRoots(parent) != nil {
		parent.scalarExact.frame.boolNumeral = s.receipt
	}
	if q := parent.recombinationFrame(m); q != nil {
		q.boolNumeral = s.receipt
	}
	if q := parent.trialFrame(m); q != nil {
		q.boolNumeral = s.receipt
	}
	if q := parent.henselCallerFrame(m); q != nil {
		q.boolNumeral = s.receipt
	}
	if r := parent.henselPairReceipt(m); r != nil {
		r.frame.boolNumeral = s.receipt
	}
	return &s.receipt, nil
}

func (m *IntegerManager) CleanupDividesNumeral() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if e := m.cleanupDividesNumeralInvocation(&m.dividesNumeral.receipt, false); e != nil {
		return e
	}
	return m.cleanupDividesNumeralInvocation(&m.dividesNumeral.receipt, true)
}

func (m *IntegerManager) matchesDividesNumeralHistory(r *integerDividesNumeralInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.dividesNumeral && r.frame.owner == m && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (m *IntegerManager) matchesDividesNumeralInvocation(r *integerDividesNumeralInvocationReceipt) bool {
	return m.matchesDividesNumeralHistory(r) && r.frame.active
}
func (m *IntegerManager) dividesNumeralContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	r := c.boolDivides
	if c.boolDividesInvocation == 0 || r == nil || c.boolDividesInvocation != r.invocation || !m.matchesDividesNumeralInvocation(r) || !r.frame.executing || m.closed || m.sourceFinalizing || m.constructionFailed {
		return nil
	}
	if p := r.frame.parentContext; p != (integerAllocationInvocationContext{}) && p.roots(m) == nil {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) dividesNumeralInvocationSourceCause(r *integerDividesNumeralInvocationReceipt) error {
	if m.matchesDividesNumeralHistory(r) {
		return r.frame.sourceCause
	}
	return nil
}
func (m *IntegerManager) finishDividesNumeralSource(s *integerDividesNumeralScope) error {
	if m == nil || m.self != m || s != &m.dividesNumeral || s.owner != m {
		return ErrPolynomialOwner
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if e := m.natural.pnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}
	if !s.active || !s.constructed || s.ended || s.owned {
		return nil
	}
	if _, e := upolynomialSourceNumeralDel(m, &s.numeral); e != nil {
		s.sourceCause = e
		return e
	}
	s.ended = true
	if !s.pending(m) {
		s.active = false
	}
	return nil
}
func (s *integerDividesNumeralScope) pending(m *IntegerManager) bool {
	if polynomialNegNumericPending(&s.numeric) || m.natural.matchesPNVerification(&s.verification) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (naturalPNBufferReceipt{}).pending {
			return true
		}
	}
	return false
}
func (m *IntegerManager) cleanupDividesNumeralInvocation(r *integerDividesNumeralInvocationReceipt, abort bool) error {
	if !m.matchesDividesNumeralInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.natural.cleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if s.constructed && !s.ended && !s.owned {
		if e := m.cleanupRawNumeral(&s.numeral); e != nil {
			return e
		}
		s.owned = true
	}
	if abort {
		if e := m.natural.cleanupPNVerification(&s.verification, true); e != nil {
			return e
		}
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].cleanup(); e != nil {
				return e
			}
		}
		if e := s.numeric.cleanup(m); e != nil {
			return e
		}
	}
	if !s.pending(m) {
		s.active = false
	}
	return nil
}
