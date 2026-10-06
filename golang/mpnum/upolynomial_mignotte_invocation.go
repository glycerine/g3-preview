// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/scoped_numeral.h:31-33.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

type upolyMignotteInvocationReceipt struct {
	service    *UPolynomialFactorizationService
	parent     *upolyFactorSquareFreeScope
	frame      *upolyMignotteScope
	invocation uint64
}

// These are the actual lexical norm, bound and tmp NUM values, charged once
// within the enclosing square-free scope header, with no source malloc ticket.
type upolyMignotteScope struct {
	receipt                                    upolyMignotteInvocationReceipt
	self                                       *upolyMignotteScope
	callerPrevious                             *upolyMignotteScope
	callDepth                                  uint32
	service                                    *UPolynomialFactorizationService
	invocation                                 uint64
	active, executing                          bool
	constructed, sourceRemaining, ownRemaining uint8
	sourceStarted, ownStarted                  bool
	norm, bound, tmp                           Int
	numeric                                    polynomialNumericRoots
	root                                       integerRootInvocationReceipt
	addMul                                     integerAddMulInvocationReceipt
	normalize                                  integerPNormalizeInvocationReceipt
	bodyCause, sourceCause                     error
}

func (s *UPolynomialFactorizationService) matchesMignotteHistory(r *upolyMignotteInvocationReceipt) bool {
	return s != nil && s.self == s && s.manager != nil && s.manager.self == s.manager && r != nil && r.service == s && r.parent != nil && r.frame == &r.parent.mignotte && r.frame.self == r.frame && r.frame.service == s && r.invocation != 0 && r.frame.invocation == r.invocation
}
func (s *UPolynomialFactorizationService) matchesMignotteInvocation(r *upolyMignotteInvocationReceipt) bool {
	return s.matchesMignotteHistory(r) && s.scope == r.parent && r.frame.active
}
func (s *UPolynomialFactorizationService) beginMignotteInvocation() (upolyMignotteInvocationReceipt, error) {
	if e := s.checkOwner(); e != nil {
		return upolyMignotteInvocationReceipt{}, e
	}
	if s.scope == nil {
		return upolyMignotteInvocationReceipt{}, ErrUPolynomialCleanupPending
	}
	q := &s.scope.mignotte
	if q.active || q.invocation == ^uint64(0) {
		return upolyMignotteInvocationReceipt{}, ErrUPolynomialCleanupPending
	}
	*q = upolyMignotteScope{service: s, invocation: q.invocation + 1, active: true, executing: true}
	q.self = q
	q.receipt = upolyMignotteInvocationReceipt{s, s.scope, q, q.invocation}
	return q.receipt, nil
}
func (q *upolyMignotteScope) numeral(i uint8) *Int {
	switch i {
	case 1:
		return &q.norm
	case 2:
		return &q.bound
	case 3:
		return &q.tmp
	}
	return nil
}
func (s *UPolynomialFactorizationService) mignotteInvocationSourceCause(r *upolyMignotteInvocationReceipt) error {
	if !s.matchesMignotteHistory(r) {
		return nil
	}
	return r.frame.sourceCause
}
func (s *UPolynomialFactorizationService) finishMignotteSource(r *upolyMignotteInvocationReceipt, body error) error {
	if !s.matchesMignotteHistory(r) {
		return ErrUPolynomialOwner
	}
	q := r.frame
	q.enterCall(s.manager)
	defer q.leaveCall(s.manager)
	z := s.manager.config.Integer
	if q.bodyCause == nil {
		q.bodyCause = body
	}
	if !q.sourceStarted {
		q.sourceRemaining = q.constructed
		q.sourceStarted = true
	}
	if q.sourceCause == nil {
		q.sourceCause = s.manager.modular.pNormalizeInvocationSourceCause(&q.normalize)
	}
	if q.sourceCause == nil {
		q.sourceCause = z.rootInvocationSourceCause(&q.root)
	}
	if q.sourceCause == nil {
		q.sourceCause = z.addMulInvocationSourceCause(&q.addMul)
	}
	if q.sourceCause != nil {
		return errors.Join(q.bodyCause, q.sourceCause)
	}
	for q.sourceRemaining != 0 {
		if _, e := upolynomialSourceNumeralDel(z, q.numeral(q.sourceRemaining)); e != nil {
			q.sourceCause = e
			return errors.Join(q.bodyCause, e)
		}
		q.sourceRemaining--
	}
	if !polynomialNegNumericPending(&q.numeric) && !z.matchesRootInvocation(&q.root) && !z.matchesAddMulInvocation(&q.addMul) && !s.manager.modular.matchesPNormalizeInvocation(&q.normalize) {
		q.active = false
	}
	return q.bodyCause
}
func (s *UPolynomialFactorizationService) cleanupMignotteInvocation(r *upolyMignotteInvocationReceipt, abort bool) error {
	if !s.matchesMignotteInvocation(r) {
		return nil
	}
	q := r.frame
	z := s.manager.config.Integer
	if q.executing || q.callDepth != 0 {
		return ErrUPolynomialCleanupPending
	}
	q.enterCall(s.manager)
	defer q.leaveCall(s.manager)
	if e := s.manager.modular.cleanupPNormalizeInvocation(&q.normalize, false); e != nil {
		return e
	}
	if e := z.cleanupRootInvocation(&q.root, false); e != nil {
		return e
	}
	if e := z.cleanupAddMulInvocation(&q.addMul, false); e != nil {
		return e
	}
	if !q.ownStarted {
		q.ownRemaining = q.constructed
		q.ownStarted = true
	}
	for q.ownRemaining != 0 {
		if e := z.cleanupRawNumeral(q.numeral(q.ownRemaining)); e != nil {
			return e
		}
		q.ownRemaining--
	}
	if abort {
		if e := s.manager.modular.cleanupPNormalizeInvocation(&q.normalize, true); e != nil {
			return e
		}
		if e := z.cleanupRootInvocation(&q.root, true); e != nil {
			return e
		}
		if e := z.cleanupAddMulInvocation(&q.addMul, true); e != nil {
			return e
		}
		if e := q.cleanupNumeric(z); e != nil {
			return e
		}
		q.active = false
	}
	return nil
}

// Pending identity reads only acknowledge consumption or refuse unsafe cleanup.
// No returned inventory is copied into the old invocation's acquired authority.
func (q *upolyMignotteScope) cleanupNumeric(z *IntegerManager) error {
	return cleanupUPolynomialNumericRoots(z, &q.numeric)
}
func cleanupUPolynomialNumericRoots(z *IntegerManager, r *polynomialNumericRoots) error {
	if r.cell.object != (memory.ObjectRef{}) {
		old := r.cell
		if z.pendingCell == old {
			e := z.cleanupPendingAllocation()
			if z.pendingCell != old {
				r.cell = sourceCellBacking{}
			}
			if e != nil {
				return e
			}
		} else {
			r.cell = sourceCellBacking{}
		}
	}
	now, e := z.allocator.PendingRoots()
	if e != nil {
		return e
	}
	owned, foreign := false, false
	for i, x := range now {
		if x != (memory.Block{}) {
			if x == r.allocator[i] {
				owned = true
			} else {
				foreign = true
			}
		}
	}
	if owned && foreign {
		return ErrUPolynomialCleanupPending
	}
	if owned {
		e = z.allocator.CleanupPending()
		now, x := z.allocator.PendingRoots()
		if x != nil {
			return errors.Join(e, x)
		}
		for i, old := range r.allocator {
			if now[i] != old {
				r.allocator[i] = memory.Block{}
			}
		}
		if e != nil {
			return e
		}
	} else {
		for i, old := range r.allocator {
			if now[i] != old {
				r.allocator[i] = memory.Block{}
			}
		}
	}
	return nil
}

// This chain protects actual synchronous scalar and destructor calls only.
// It is never queried for allocation ownership or retained-scope admission.
func (q *upolyMignotteScope) enterCall(m *UPolynomialManager) {
	if q.callDepth == 0 {
		q.callerPrevious = m.mignotteCaller
		m.mignotteCaller = q
	}
	q.callDepth++
}
func (q *upolyMignotteScope) leaveCall(m *UPolynomialManager) {
	q.callDepth--
	if q.callDepth == 0 {
		m.mignotteCaller = q.callerPrevious
		q.callerPrevious = nil
	}
}
func (m *UPolynomialManager) mignotteCallPending() bool {
	if m.displayExecuting() {
		return true
	}

	if m.preparationCallPending() || m.primePreparationCallPending() {
		return true
	}
	if m.recombinationCallPending() {
		return true
	}
	for q := m.trialCaller; q != nil; {
		if q.matches() && q.callDepth != 0 && (q.owner.manager == m || q.factors.manager == m) {
			return true
		}
		if q.owner != nil && q.owner.manager == m {
			q = q.callerPrevious[0]
		} else {
			q = q.callerPrevious[1]
		}
	}
	if c := m.henselCallerGuard; c != nil && c.matches() && c.callDepth != 0 && m.henselCallerInvocation == c.invocation && c.ownsNumericManager(m) {
		return true
	}
	for h := m.henselPairCaller; h != nil; {
		if h.self == h && h.service != nil && h.service.self == h.service && h.callDepth != 0 && h.ownsNumericManager(m) {
			return true
		}
		if h.service != nil && h.service.manager == m {
			h = h.rootPrevious
		} else if h.check.brCore == m {
			h = h.check.brPrevious
		} else if h.check.bCore == m {
			h = h.check.bPrevious
		} else {
			h = h.rmPrevious
		}
	}
	for q := m.primeTrialCaller; q != nil; {
		if q.self == q && q.service != nil && q.service.self == q.service && q.callDepth != 0 && (q.zp == m || q.service.manager == m) {
			return true
		}
		if q.service != nil && q.service.manager == m {
			q = q.rootPrevious
		} else {
			q = q.zpPrevious
		}
	}
	for c := m.squareFreeCaller; c != nil; c = c.callerPrevious {
		if c.self == c && c.service != nil && c.service.self == c.service && c.manager == m && c.service.manager == m && c.callDepth != 0 {
			return true
		}
	}
	for q := m.mignotteCaller; q != nil; q = q.callerPrevious {
		if q.self == q && q.service != nil && q.service.self == q.service && q.service.manager == m && q.callDepth != 0 {
			return true
		}
	}
	if s := m.factorFrontendConstantCaller; s != nil && s.manager == m && s.constantCallerExecuting() {
		return true
	}
	for q := m.factorFrontendCaller; q != nil; q = q.previous {
		if q.self == q && q.service != nil && q.service.self == q.service && q.service.manager == m && (q.executing || q.sourceExecuting || q.ownExecuting) {
			return true
		}
	}
	return false
}
