// Source: src/math/polynomial/upolynomial.cpp:1734-1764; src/util/mpz.h:393-407.
// Source: src/util/mpz.h:468-485; src/util/mpz.cpp:730-817,915-985.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// Real retained RAW declarations in the charged Integer owner. Count marks
// reached declarations; values beyond count carry no source constructor credit.
type integerRawScope struct {
	formatContext                                  integerAllocationInvocationContext
	conversionContext                              integerAllocationInvocationContext
	somNormalize                                   polynomialSOMNormalizeInvocationReceipt
	normalization                                  polynomialNormalizeInvocationReceipt
	henselCallerContext                            integerAllocationInvocationContext
	henselPair                                     upolyHenselPairInvocationReceipt
	henselManager                                  *UPolynomialManager
	divides                                        polynomialDividesInvocationReceipt
	content                                        polynomialContentInvocationReceipt
	addMul                                         integerAddMulInvocationReceipt
	scalarExact                                    polynomialScalarExactDivInvocationReceipt
	exactDiv                                       polynomialExactDivInvocationReceipt
	owner                                          *IntegerManager
	receipt                                        integerRawInvocationReceipt
	invocation                                     uint64
	active, executing, bodySucceeded, ended, owned bool
	values                                         [6]Int
	count, cursor                                  uint8
	numeric                                        polynomialNumericRoots
	buffers                                        [4]naturalPNBufferReceipt
	verification                                   naturalPNVerificationReceipt
	bodyCause, sourceCause                         error
}
type integerRawInvocationReceipt struct {
	manager    *IntegerManager
	frame      *integerRawScope
	invocation uint64
}

// Executing RAW and scoped Divides NUM calls retain their real declarations.
// Public owner destruction cannot run from a returning error transport callback.
func (m *IntegerManager) rawInvocationExecuting() bool {
	return m.rationalEvaluationCallerExecuting() || m.binaryEvaluationCallerExecuting() || m.algebraicNegCallerExecuting() || m.conversionCallerExecuting() || (m.mpfParent != nil && m.mpfParent.formatInvocationExecuting()) || m.mpfBorrowedSourceCallbackActive() || m.mpfLifecycleCallbackActive() || (m.divisionRaw.active && m.divisionRaw.executing) ||
		(m.extendedRaw.active && m.extendedRaw.executing) ||
		(m.gcdRaw.active && m.gcdRaw.executing) ||
		(m.addMulInvocation.active && m.addMulInvocation.executing) ||
		(m.dividesNumeral.active && m.dividesNumeral.executing)
}

func (m *IntegerManager) rawFrameAddress(s *integerRawScope) bool {
	return m != nil && m.self == m && s != nil && (s == &m.divisionRaw || s == &m.extendedRaw || s == &m.gcdRaw)
}
func (m *IntegerManager) matchesRawInvocation(r *integerRawInvocationReceipt) bool {
	return r != nil && r.manager == m && m.rawFrameAddress(r.frame) && r.frame.owner == m && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *IntegerManager) rawAllocationRoots(r *integerRawInvocationReceipt) *polynomialNumericRoots {

	if !m.matchesRawInvocation(r) || !r.frame.executing || m.closed || m.sourceFinalizing || m.constructionFailed {
		return nil
	}
	if c := r.frame.conversionContext; c.mpfConversion != nil && c.conversionFrame(m, true) == nil {
		return nil
	}
	if c := r.frame.formatContext; c.mpfFormat != nil && c.formatFrame(m, true) == nil {
		return nil
	}
	if c := r.frame.henselCallerContext; c.henselCaller != nil && c.henselCallerFrame(m) == nil {
		return nil
	}
	if p := &r.frame.henselPair; p.service != nil && (integerAllocationInvocationContext{henselPair: p, henselPairInvocation: p.invocation, henselManager: r.frame.henselManager}).henselPairReceipt(m) == nil {
		return nil
	}
	if p := &r.frame.divides; p.core != nil && m.dividesAllocationRoots(p) == nil {
		return nil
	}
	if p := &r.frame.scalarExact; p.core != nil && m.scalarExactAllocationRoots(p) == nil {
		return nil
	}
	if p := &r.frame.exactDiv; p.core != nil && m.exactDivAllocationRoots(p) == nil {
		return nil
	}
	if p := &r.frame.somNormalize; p.manager != nil && m.somNormalizeAllocationRoots(p) == nil {
		return nil
	}
	if p := &r.frame.normalization; p.manager != nil && m.normalizationAllocationRoots(p) == nil {
		return nil
	}
	if p := &r.frame.content; p.manager != nil && m.contentAllocationRoots(p) == nil {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) beginRawInvocation(s *integerRawScope, count uint8) (*integerRawInvocationReceipt, error) {
	if !m.rawFrameAddress(s) || m.closed || m.sourceFinalizing || m.constructionFailed {
		return nil, ErrPolynomialOwner
	}
	if s.active || s.invocation == ^uint64(0) {
		return nil, ErrPolynomialPending
	}
	*s = integerRawScope{owner: m, invocation: s.invocation + 1, active: true, executing: true, count: count}
	s.receipt = integerRawInvocationReceipt{m, s, s.invocation}
	return &s.receipt, nil
}
func (m *IntegerManager) finishRawSource(s *integerRawScope) error {
	if !m.rawFrameAddress(s) || s.owner != m {
		return ErrPolynomialOwner
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if e := m.addMulInvocationSourceCause(&s.addMul); e != nil {
		s.sourceCause = e
		return e
	}
	if e := m.natural.pnVerificationSourceCause(&s.verification); e != nil {
		s.sourceCause = e
		return e
	}
	if !s.active || !s.bodySucceeded || s.ended || s.owned {
		return nil
	}
	for s.cursor < s.count {
		_, e := upolynomialSourceNumeralDel(m, &s.values[s.cursor])
		if e != nil {
			s.sourceCause = e
			return e
		}
		s.cursor++
	}
	s.ended = true
	if !s.pending(m) {
		s.active = false
	}
	return nil
}
func (m *IntegerManager) rawInvocationSourceCause(r *integerRawInvocationReceipt) error {
	if r != nil && r.manager == m && m.rawFrameAddress(r.frame) && r.invocation != 0 && r.invocation == r.frame.invocation {
		return r.frame.sourceCause
	}
	return nil
}
func (s *integerRawScope) pending(m *IntegerManager) bool {
	if m.matchesAddMulInvocation(&s.addMul) || polynomialNegNumericPending(&s.numeric) || m.natural.matchesPNVerification(&s.verification) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (naturalPNBufferReceipt{}).pending {
			return true
		}
	}
	return false
}
func (m *IntegerManager) cleanupRawInvocation(r *integerRawInvocationReceipt, abort bool) error {
	if !m.matchesRawInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.cleanupAddMulInvocation(&s.addMul, false); e != nil {
		return e
	}
	if e := m.natural.cleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if (s.bodySucceeded || abort) && !s.owned && !s.ended {
		for s.cursor < s.count {
			if e := m.cleanupRawNumeral(&s.values[s.cursor]); e != nil {
				return e
			}
			s.cursor++
		}
		s.owned = true
	}
	if !abort {
		return nil
	}
	if e := m.cleanupAddMulInvocation(&s.addMul, true); e != nil {
		return e
	}
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
	*s = integerRawScope{owner: m, invocation: s.invocation, bodySucceeded: s.bodySucceeded, ended: s.ended, owned: s.owned, count: s.count, cursor: s.cursor, verification: s.verification, bodyCause: s.bodyCause, sourceCause: s.sourceCause}
	return nil
}

// Explicit owning acknowledgment; neither method replays a source end DEL.
func (m *IntegerManager) CleanupDivision() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.divisionRaw
	r := integerRawInvocationReceipt{m, s, s.invocation}
	return m.cleanupRawInvocation(&r, true)
}
func (m *IntegerManager) CleanupExtendedGCD() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.extendedRaw
	r := integerRawInvocationReceipt{m, s, s.invocation}
	return m.cleanupRawInvocation(&r, true)
}

// Owning continuation of exactly named RAW fields after source manager destruction.
func (m *IntegerManager) cleanupRawNumeral(p *Int) error {
	if m.sourceFinalized || m.constructorUnwound {
		if p.cell != 0 && !m.isFailureRef(p.cell) {
			if e := m.freeAfterSourceFinalize(p.cell); e != nil {
				return e
			}
		}
		p.cell = 0
		return nil
	}
	if m.sourceFinalizing {
		if p.cell != 0 && !m.isFailureRef(p.cell) {
			if e := m.free(p.cell); e != nil {
				return e
			}
		}
		p.cell = 0
		return nil
	}
	return m.Del(p)
}

// CleanupGCDRaw acknowledges the actual retained Lehmer RAW fields; it does
// not repair source scratch holes or replay source explicit DEL.
func (m *IntegerManager) CleanupGCDRaw() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.gcdRaw
	r := integerRawInvocationReceipt{m, s, s.invocation}
	return m.cleanupRawInvocation(&r, true)
}
