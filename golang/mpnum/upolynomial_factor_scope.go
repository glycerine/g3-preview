// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1299.
// Source author: Leonardo de Moura.
// Licensed under the MIT license.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrUPolynomialCleanupPending = errors.New("mpnum: retained factorization lexical cleanup")

// Actual cold ownership for factor_square_free's lexical objects. The source
// cannot safely continue a destructor through an invalid numeral. Go retains
// the unfinished reverse destruction cursor instead of losing its owner. This
// additional charged header is a Go exception/lifetime adapter, not C++ malloc.
type upolyFactorSquareFreeScope struct {
	primePreparation  upolyFactorPrimePreparationScope
	iterator          PrimeIterator
	hensel            upolyHenselCallerScope
	trialHelper       upolyRecombinationTrialScope
	recombinationStep upolyRecombinationStepScope
	self              *upolyFactorSquareFreeScope
	service           *UPolynomialFactorizationService
	callerPrevious    *upolyFactorSquareFreeScope
	callDepth         uint32
	mignotte          upolyMignotteScope
	primeTrial        upolyPrimeTrialScope
	manager           *UPolynomialManager
	fpp               upolyVectorState
	p                 Int
	zp                *UPolynomialManager
	best              *UPolynomialFactors
	bestP             Int
	ds                upolyFactorDegreeState
	gcdTmp            Int
	zpe               *UPolynomialManager
	lifted            *UPolynomialFactors
	leading           Int
	it                upolyFactorCombinationState
	trial, quotient   upolyVectorState
	trialContent      Int
	constructed       uint8
	header            uint64
}

func (s *UPolynomialFactorizationService) newSquareFreeScope() (*upolyFactorSquareFreeScope, error) {
	if s.scope != nil {
		return nil, ErrUPolynomialCleanupPending
	}
	n := uint64(reflect.TypeOf(upolyFactorSquareFreeScope{}).Size())
	if e := s.manager.config.Integer.budget.Reserve(n); e != nil {
		return nil, e
	}
	c := &upolyFactorSquareFreeScope{manager: s.manager, service: s, header: n}
	c.self = c
	s.scope = c
	return c, nil
}

// Reverse lexical source order: content, quotient, trial, combination iterator,
// leading coefficient, lifted factors, lifted core, gcd, degree set, saved
// prime, prime factors, prime core, prime, primitive vector. Only constructed
// objects participate; publication/cleanup advances one completed action.
func (c *upolyFactorSquareFreeScope) cleanup() error {
	if q := &c.primePreparation; q.active {
		if q.executing || q.sourceExecuting || q.ownExecuting {
			return ErrUPolynomialCleanupPending
		}
		if e := q.finish(&q.receipt, q.bodyCause); e != nil && q.sourceCause != nil {
			return e
		}
	}
	if q := &c.service.preparation; q.active && q.parent == c {
		if e := q.finish(&q.receipt, q.bodyCause); e != nil && q.sourceCause != nil {
			return e
		}
	}
	if q := &c.recombinationStep; q.active {
		if q.executing || q.sourceExecuting || q.ownExecuting || q.callDepth != 0 {
			return ErrUPolynomialCleanupPending
		}
		q.finish(&q.receipt, q.bodyCause)
		if q.sourceCause != nil {
			return q.sourceCause
		}
	}
	if q := &c.trialHelper; q.active {
		if q.executing || q.sourceExecuting || q.ownExecuting || q.callDepth != 0 {
			return ErrUPolynomialCleanupPending
		}
		q.finish(&q.receipt, q.bodyCause)
		if q.sourceCause != nil {
			return q.sourceCause
		}
	}
	if c.hensel.active {
		if e := c.hensel.finish(); e != nil {
			return e
		}
		return ErrUPolynomialCleanupPending
	}
	if q := &c.primeTrial; q.active {
		if q.executing || q.callDepth != 0 {
			return ErrUPolynomialCleanupPending
		}
		q.service.finishPrimeTrialSource(&q.receipt, q.bodyCause)
		if q.sourceCause != nil {
			return q.sourceCause
		}
	}
	if q := &c.mignotte; q.active && q.service != nil {
		if q.executing || q.callDepth != 0 {
			return ErrUPolynomialCleanupPending
		}
		q.service.finishMignotteSource(&q.receipt, q.bodyCause)
		if q.sourceCause != nil {
			return q.sourceCause
		}
	}
	if e := c.cleanupMembersPhase(true); e != nil {
		return e
	}
	if c.primePreparation.active || c.mignotte.active || c.primeTrial.active || c.trialHelper.active || c.recombinationStep.active || c.service.preparation.active {
		return ErrUPolynomialCleanupPending
	}
	return nil
}
func (c *upolyFactorSquareFreeScope) cleanupMembers() error { return c.cleanupMembersPhase(false) }
func (c *upolyFactorSquareFreeScope) cleanupMembersPhase(source bool) error {
	m := c.manager
	for c.constructed != 0 {
		var e error
		switch c.constructed {
		case 14:
			e = m.modular.Del(&c.trialContent)
		case 13:
			e = m.reset(&c.quotient)
			if e == nil {
				m.freeBacking(&c.quotient)
			}
		case 12:
			e = m.reset(&c.trial)
			if e == nil {
				m.freeBacking(&c.trial)
			}
		case 11:
			c.it.close()
			c.it = upolyFactorCombinationState{}
		case 10:
			e = m.modular.Del(&c.leading)
		case 9:
			e = c.lifted.Close()
			if e == nil {
				c.lifted = nil
			}
		case 8:
			e = c.zpe.Close()
			if e == nil {
				c.zpe = nil
			}
		case 7:
			if source {
				e = c.sourcePrimeNumeral(&c.gcdTmp)
			} else {
				e = m.modular.Del(&c.gcdTmp)
			}
		case 6:
			c.ds.close(m.config.Integer.budget)
			c.ds = upolyFactorDegreeState{}
		case 5:
			if source {
				e = c.sourcePrimeNumeral(&c.bestP)
			} else {
				e = m.modular.Del(&c.bestP)
			}
		case 4:
			e = c.best.Close()
			if e == nil {
				c.best = nil
			}
		case 3:
			if c.primePreparation.matches() {
				e = c.closePrimeRelated(source)
			} else {
				e = c.zp.Close()
			}
			if e == nil {
				c.zp = nil
			}
		case 2:
			if source {
				e = c.sourcePrimeNumeral(&c.p)
			} else {
				e = m.modular.Del(&c.p)
			}
		case 1:
			e = m.reset(&c.fpp)
			if e == nil {
				m.freeBacking(&c.fpp)
			}
		}
		if e != nil {
			return e
		}
		c.constructed--
	}
	return nil
}

// CleanupRetained retries actual unfinished source destruction. It never
// changes modular mode, reconstructs a numeral or rolls back caller outputs.
func (s *UPolynomialFactorizationService) CleanupRetained() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrUPolynomialOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	for _, q := range []*upolyFactorPreparationScope{&s.diagnostic, &s.preparation} {
		if q.active {
			if e := q.own(&q.receipt, false); e != nil {
				return e
			}
		}
	}
	if s.scope != nil {
		c := &s.scope.hensel
		if c.active && (c.executing || c.sourceExecuting || c.ownExecuting || c.callDepth != 0) {
			return ErrUPolynomialCleanupPending
		}
	}
	if s.scope == nil || !s.scope.hensel.active {
		if e := s.cleanupHenselPairs(); e != nil {
			return e
		}
	}
	// Empty-service Close still refunds its own header after a borrowed owner
	// closes; no manager/prime access is needed without a retained scope.
	if s.scope == nil {
		for _, q := range []*upolyFactorPreparationScope{&s.diagnostic, &s.preparation} {
			if q.active {
				if e := q.own(&q.receipt, true); e != nil {
					return e
				}
			}
		}
		return nil
	}
	if e := s.checkOwner(); e != nil {
		return e
	}
	if s.scope.callDepth != 0 {
		return ErrUPolynomialCleanupPending
	}
	caller := s.scope
	if q := &caller.primePreparation; q.active {
		if e := q.own(&q.receipt, false); e != nil {
			return e
		}
	}
	if caller.recombinationStep.active {
		if e := caller.recombinationStep.own(&caller.recombinationStep.receipt, false); e != nil {
			return e
		}
	}
	if caller.trialHelper.active {
		if e := caller.trialHelper.own(&caller.trialHelper.receipt, false); e != nil {
			return e
		}
	}
	caller.enterCall()
	defer caller.leaveCall()
	if s.scope.hensel.active {
		if e := s.scope.hensel.own(false); e != nil {
			return e
		}
	}
	prime := &s.scope.primeTrial
	if prime.active {
		if e := s.cleanupPrimeTrialInvocation(&prime.receipt, false); e != nil {
			return e
		}
	}
	q := &s.scope.mignotte
	if q.executing || q.callDepth != 0 {
		return ErrUPolynomialCleanupPending
	}
	if q.active {
		if e := s.cleanupMignotteInvocation(&q.receipt, false); e != nil {
			return e
		}
	}
	if e := s.scope.cleanupMembers(); e != nil {
		return e
	}
	if q.active {
		if e := s.cleanupMignotteInvocation(&q.receipt, true); e != nil {
			return e
		}
	}
	if prime.active {
		if e := s.cleanupPrimeTrialInvocation(&prime.receipt, true); e != nil {
			return e
		}
	}
	if s.scope.hensel.active {
		if e := s.scope.hensel.own(true); e != nil {
			return e
		}
	}
	if caller.recombinationStep.active {
		if e := caller.recombinationStep.own(&caller.recombinationStep.receipt, true); e != nil {
			return e
		}
	}
	if caller.trialHelper.active {
		if e := caller.trialHelper.own(&caller.trialHelper.receipt, true); e != nil {
			return e
		}
	}
	for _, q := range []*upolyFactorPreparationScope{&s.diagnostic, &s.preparation} {
		if q.active {
			if e := q.own(&q.receipt, true); e != nil {
				return e
			}
		}
	}
	if q := &caller.primePreparation; q.active {
		if e := q.own(&q.receipt, true); e != nil {
			return e
		}
	}
	s.manager.config.Integer.budget.Release(s.scope.header)
	s.scope = nil
	return nil
}

// RetainedCleanupCore borrows a still-owned lexical core. It lets the caller
// explicitly select set_z after auditing a guarded invalid modular bound.
// The borrow expires when CleanupRetained consumes that core.
func (s *UPolynomialFactorizationService) RetainedCleanupCore(index uint32) (*UPolynomialManager, error) {
	if e := s.checkOwner(); e != nil {
		return nil, e
	}
	if s.scope != nil {
		for _, m := range [3]*UPolynomialManager{s.scope.zpe, s.scope.zp, s.scope.hensel.related} {
			if m != nil {
				if index == 0 {
					return m, nil
				}
				index--
			}
		}
	}
	for h := s.henselPairs; h != nil; h = h.next {
		if h.executing || h.sourceExecuting || h.callDepth != 0 {
			return nil, ErrUPolynomialCleanupPending
		}
		for _, m := range []*UPolynomialManager{h.check.bCore, h.check.brCore, h.rm} {
			if m != nil && !m.closed {
				if index == 0 {
					return m, nil
				}
				index--
			}
		}
	}
	return nil, arena.ErrBounds
}
func (s *UPolynomialFactorizationService) RetainedCleanupFactors(index uint32) (*UPolynomialFactors, error) {
	if e := s.checkOwner(); e != nil {
		return nil, e
	}
	if s.scope != nil {
		for _, f := range [3]*UPolynomialFactors{s.scope.primeTrial.current, s.scope.lifted, s.scope.best} {
			if f != nil {
				if f == s.scope.primeTrial.current && (s.scope.primeTrial.executing || s.scope.primeTrial.sourceExecuting || s.scope.primeTrial.callDepth != 0) {
					return nil, ErrUPolynomialCleanupPending
				}
				if index == 0 {
					return f, nil
				}
				index--
			}
		}
	}
	return nil, arena.ErrBounds
}

// The automatic enclosing epilogue is source-only. Explicit CleanupRetained
// separately acknowledges acquired unjoined child roots after older locals.
func (s *UPolynomialFactorizationService) finishSquareFreeScopeSource() error {
	if e := s.checkOwner(); e != nil {
		return e
	}
	if s.manager.primePreparationCallPending() {
		return ErrUPolynomialCleanupPending
	}
	c := s.scope
	if c == nil {
		return nil
	}
	c.enterCall()
	defer c.leaveCall()
	if e := c.cleanup(); e != nil {
		return e
	}
	s.manager.config.Integer.budget.Release(c.header)
	s.scope = nil
	return nil
}

// Actual enclosing destructor calls carry their own concrete caller identity;
// an idle retained scope is never treated as an executing caller.
func (c *upolyFactorSquareFreeScope) enterCall() {
	if c.callDepth == 0 {
		c.callerPrevious = c.manager.squareFreeCaller
		c.manager.squareFreeCaller = c
	}
	c.callDepth++
}
func (c *upolyFactorSquareFreeScope) leaveCall() {
	c.callDepth--
	if c.callDepth == 0 {
		c.manager.squareFreeCaller = c.callerPrevious
		c.callerPrevious = nil
	}
}

func (s *UPolynomialFactorizationService) finishSquareFreeScopeError(body error) error {
	return errors.Join(body, s.finishSquareFreeScopeSource())
}
