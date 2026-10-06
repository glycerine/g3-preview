// Source: src/math/polynomial/upolynomial_factorization.cpp:1058-1094;
// src/util/mpz.cpp:560-843; src/util/mpzzp.h:211-216.
// Concrete call adapter: p, saved prime and gcd_tmp remain actual parent NUMs.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

type upolyFactorPrimePreparationInvocationReceipt struct {
	owner      *upolyFactorSquareFreeScope
	frame      *upolyFactorPrimePreparationScope
	invocation uint64
}
type upolyFactorPrimePreparationScope struct {
	receipt                                          upolyFactorPrimePreparationInvocationReceipt
	callerPrevious                                   [2]*upolyFactorPrimePreparationScope
	callDepth                                        uint32
	self                                             *upolyFactorPrimePreparationScope
	owner                                            *upolyFactorSquareFreeScope
	factors                                          *UPolynomialFactors
	related                                          *UPolynomialManager
	invocation                                       uint64
	active, executing, sourceExecuting, ownExecuting bool
	sourceEnded, ownEnded                            bool
	numeric                                          polynomialNumericRoots
	normalize                                        [2]integerPNormalizeInvocationReceipt
	divisionRaw, extendedRaw, gcdRaw                 integerRawInvocationReceipt
	buffers                                          [4]naturalPNBufferReceipt
	verification                                     naturalPNVerificationReceipt
	bodyCause, sourceCause, ownCause                 error
}

func (q *upolyFactorPrimePreparationScope) matches() bool {
	return q != nil && q.self == q && q.owner != nil && q.owner.self == q.owner && q == &q.owner.primePreparation && q.owner.service != nil && q.owner.service.self == q.owner.service && q.owner.service.manager == q.owner.manager && q.owner.manager != nil && q.owner.manager.self == q.owner.manager && q.invocation != 0
}
func (s *UPolynomialFactorizationService) beginPrimePreparation(fs *UPolynomialFactors) (*upolyFactorPrimePreparationScope, error) {
	if e := s.checkOwner(); e != nil {
		return nil, e
	}
	if s.scope == nil || s.scope.self != s.scope || fs == nil || fs.manager != s.manager {
		return nil, ErrUPolynomialOwner
	}
	q := &s.scope.primePreparation
	if q.active || q.invocation == ^uint64(0) {
		return nil, ErrUPolynomialCleanupPending
	}
	*q = upolyFactorPrimePreparationScope{owner: s.scope, factors: fs, related: s.scope.zp, invocation: q.invocation + 1, active: true, executing: true}
	q.self = q
	q.receipt = upolyFactorPrimePreparationInvocationReceipt{q.owner, q, q.invocation}
	return q, nil
}
func (q *upolyFactorPrimePreparationScope) managers() [2]*UPolynomialManager {
	return [2]*UPolynomialManager{q.owner.manager, q.related}
}
func (q *upolyFactorPrimePreparationScope) childCause() error {
	z := q.owner.manager.config.Integer
	for i, m := range q.managers() {
		if m == nil {
			continue
		}
		if e := m.modular.pNormalizeInvocationSourceCause(&q.normalize[i]); e != nil {
			return e
		}
	}
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw, &q.gcdRaw} {
		if e := z.rawInvocationSourceCause(r); e != nil {
			return e
		}
	}
	return z.natural.pnVerificationSourceCause(&q.verification)
}
func (q *upolyFactorPrimePreparationScope) matchesReceipt(r *upolyFactorPrimePreparationInvocationReceipt) bool {
	return q.matches() && r != nil && r.owner == q.owner && r.frame == q && r.invocation != 0 && r.invocation == q.invocation
}
func (q *upolyFactorPrimePreparationScope) finish(r *upolyFactorPrimePreparationInvocationReceipt, body error) error {
	if !q.matchesReceipt(r) {
		return ErrUPolynomialOwner
	}
	if !q.active {
		return errors.Join(q.bodyCause, q.sourceCause)
	}
	if q.owner.service.scope != q.owner {
		return ErrUPolynomialOwner
	}
	if q.executing || q.sourceExecuting || q.ownExecuting {
		return ErrUPolynomialCleanupPending
	}
	q.enterCall()
	defer q.leaveCall()
	q.sourceExecuting = true
	defer func() { q.sourceExecuting = false }()
	if q.bodyCause == nil {
		q.bodyCause = body
	}
	if q.sourceCause == nil {
		q.sourceCause = q.childCause()
	}
	if q.sourceCause != nil {
		return errors.Join(q.bodyCause, q.sourceCause)
	}
	q.sourceEnded = true

	if !q.pending() {
		q.active = false
	}
	return q.bodyCause
}
func (q *upolyFactorPrimePreparationScope) pending() bool {
	z := q.owner.manager.config.Integer
	if polynomialNegNumericPending(&q.numeric) {
		return true
	}
	if z.natural.matchesPNVerification(&q.verification) {
		return true
	}
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw, &q.gcdRaw} {
		if z.matchesRawInvocation(r) {
			return true
		}
	}
	for i := range q.buffers {
		if q.buffers[i].pending != (memory.Block{}) {
			return true
		}
	}
	for i, m := range q.managers() {
		if m == nil {
			continue
		}
		if m.modular.matchesPNormalizeInvocation(&q.normalize[i]) {
			return true
		}
	}
	return false
}
func (q *upolyFactorPrimePreparationScope) own(r *upolyFactorPrimePreparationInvocationReceipt, abort bool) (cause error) {
	if !q.matchesReceipt(r) {
		return ErrUPolynomialOwner
	}
	if !q.active {
		return nil
	}
	if q.owner.service.scope != q.owner {
		return ErrUPolynomialOwner
	}
	if q.executing || q.sourceExecuting || q.ownExecuting {
		return ErrUPolynomialCleanupPending
	}
	defer func() {
		if cause != nil && q.ownCause == nil {
			q.ownCause = cause
		}
	}()
	q.ownExecuting = true
	defer func() { q.ownExecuting = false }()
	q.enterCall()
	defer q.leaveCall()
	z := q.owner.manager.config.Integer
	for i, m := range q.managers() {
		if m == nil {
			continue
		}
		if e := m.modular.cleanupPNormalizeInvocation(&q.normalize[i], false); e != nil {
			return e
		}
	}
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw, &q.gcdRaw} {
		if e := z.cleanupRawInvocation(r, false); e != nil {
			return e
		}
	}
	if e := z.natural.cleanupPNVerification(&q.verification, false); e != nil {
		return e
	}
	q.ownEnded = true
	if abort {
		for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw, &q.gcdRaw} {
			if e := z.cleanupRawInvocation(r, true); e != nil {
				return e
			}
		}
		if e := z.natural.cleanupPNVerification(&q.verification, true); e != nil {
			return e
		}
		for i := range q.buffers {
			if e := q.buffers[i].cleanup(); e != nil {
				return e
			}
		}
		for i, m := range q.managers() {
			if m == nil {
				continue
			}
			if e := m.modular.cleanupPNormalizeInvocation(&q.normalize[i], true); e != nil {
				return e
			}
		}
		if e := cleanupUPolynomialNumericRoots(z, &q.numeric); e != nil {
			return e
		}
		q.active = false
	}
	return nil
}

func (q *upolyFactorPrimePreparationScope) enterCall() {
	if q.callDepth == 0 {
		for i, m := range q.managers() {
			if m == nil {
				continue
			}
			if i == 1 && m == q.owner.manager {
				continue
			}
			q.callerPrevious[i] = m.primePreparationCaller
			m.primePreparationCaller = q
		}
	}
	q.callDepth++
	q.owner.enterCall()
}
func (q *upolyFactorPrimePreparationScope) leaveCall() {
	q.owner.leaveCall()
	q.callDepth--
	if q.callDepth == 0 {
		for i, m := range q.managers() {
			if m == nil {
				continue
			}
			if i == 1 && m == q.owner.manager {
				continue
			}
			m.primePreparationCaller = q.callerPrevious[i]
			q.callerPrevious[i] = nil
		}
	}
}

func (m *UPolynomialManager) primePreparationCallPending() bool {
	for q := m.primePreparationCaller; q != nil; {
		if q.matches() && q.callDepth != 0 && (q.owner.manager == m || q.related == m) {
			return true
		}
		if q.owner != nil && q.owner.manager == m {
			q = q.callerPrevious[0]
		} else {
			q = q.callerPrevious[1]
		}
	}
	return false
}

// The related caller chain is bound only after the actual constructor returns.
func (q *upolyFactorPrimePreparationScope) bindRelated(m *UPolynomialManager) error {
	if !q.matches() || !q.active || !q.executing || q.owner.zp != m || m == nil || m.self != m || m.config.Integer != q.owner.manager.config.Integer || q.related != nil {
		return ErrUPolynomialOwner
	}
	q.related = m
	if q.callDepth != 0 {
		q.callerPrevious[1] = m.primePreparationCaller
		m.primePreparationCaller = q
	}
	return nil
}

// Selected lexical NUM DT retains its first source stop. Real parent NUMs
// are never adopted into the adapter or given a second source destructor.
func (c *upolyFactorSquareFreeScope) sourcePrimeNumeral(n *Int) error {
	q := &c.primePreparation
	if q.matches() {
		q.enterCall()
		defer q.leaveCall()
		q.sourceExecuting = true
		defer func() { q.sourceExecuting = false }()
	}
	_, e := upolynomialSourceNumeralDel(c.manager.config.Integer, n)
	if e != nil && q.matches() {
		if q.sourceCause == nil {
			q.sourceCause = e
		}
		q.sourceEnded = false
		q.active = true
	}
	return e
}

func (m *UPolynomialManager) primePreparationDestructorAuthorized(r *upolyFactorPrimePreparationInvocationReceipt, own bool) bool {
	if r == nil || r.frame == nil {
		return false
	}
	q := r.frame
	if !q.matchesReceipt(r) || q.owner.service.scope != q.owner || q.owner.constructed != 3 || q.owner.zp != m || q.related != m || q.callDepth == 0 || q.executing {
		return false
	}
	if own {
		return q.ownExecuting || (q.sourceExecuting && m.sourceFinalized)
	}
	return q.sourceExecuting && !q.ownExecuting
}
func (c *upolyFactorSquareFreeScope) closePrimeRelated(source bool) (cause error) {
	q := &c.primePreparation
	if !q.matches() || c.service.scope != c || c.constructed != 3 || c.zp != q.related {
		return ErrUPolynomialOwner
	}
	if q.executing || q.sourceExecuting || q.ownExecuting {
		return ErrUPolynomialCleanupPending
	}
	q.enterCall()
	defer q.leaveCall()
	if source {
		q.sourceExecuting = true
		defer func() { q.sourceExecuting = false }()
	} else {
		q.ownExecuting = true
		defer func() { q.ownExecuting = false }()
	}
	defer func() {
		if cause != nil {
			q.active = true
			if source {
				if q.sourceCause == nil {
					q.sourceCause = cause
				}
				q.sourceEnded = false
			} else if q.ownCause == nil {
				q.ownCause = cause
			}
		}
	}()
	r := q.receipt
	if source {
		if e := c.zp.sourceFinalizeWithCallerOwners(nil, nil, &r); e != nil {
			return e
		}
	}
	// Source private members are complete before the Go cold class storage
	// can be refunded. Close cannot discover or consume caller descriptors.
	return c.zp.closeWithCallerOwners(nil, nil, &r)
}
