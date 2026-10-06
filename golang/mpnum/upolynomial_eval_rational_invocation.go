// Source: src/math/polynomial/upolynomial.cpp:1767-1797.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type upolyRationalEvaluationScope struct {
	constructed                              uint8
	target                                   UPolynomialView
	falseAcknowledged                        bool
	owner                                    *UPolynomialManager
	invocation                               uint64
	values                                   [3]Int
	numeric                                  polynomialNumericRoots
	normalize                                integerPNormalizeInvocationReceipt
	addMul                                   integerAddMulInvocationReceipt
	sourceRemaining, ownRemaining            uint8
	active, executing, finishing, ownStarted bool
	bodyCause, sourceCause, ownCause         error
	history                                  *upolyRationalEvaluationHistory
}

// Diagnostics contain issued child tuples, never NUM payload/owning authority.
type upolyRationalEvaluationHistory struct {
	constructed                      uint8
	owner                            *UPolynomialManager
	frame                            *upolyRationalEvaluationScope
	next                             *upolyRationalEvaluationHistory
	invocation                       uint64
	normalize                        integerPNormalizeInvocationReceipt
	addMul                           integerAddMulInvocationReceipt
	bodyCause, sourceCause, ownCause error
	sourceRemaining                  uint8
}

func (m *UPolynomialManager) rationalEvaluationExecuting() bool {
	return m != nil && m.self == m && m.rationalEvaluation.owner == m && m.rationalEvaluation.active && (m.rationalEvaluation.executing || m.rationalEvaluation.finishing)
}
func (z *IntegerManager) rationalEvaluationCallerExecuting() bool {
	return z != nil && z.self == z && z.rationalEvaluationCaller != nil && z.rationalEvaluationCaller.config.Integer == z && z.rationalEvaluationCaller.rationalEvaluationExecuting()
}
func (c integerAllocationInvocationContext) rationalEvaluationFrame(z *IntegerManager) *upolyRationalEvaluationScope {
	q := c.rationalEvaluation
	if z == nil || z.self != z || q == nil || q.owner == nil || q.owner.self != q.owner || q != &q.owner.rationalEvaluation || q.owner.config.Integer != z || z.rationalEvaluationCaller != q.owner || q.owner.callbackDepth != 0 || !q.active || !q.executing || c.rationalEvaluationInvocation == 0 || c.rationalEvaluationInvocation != q.invocation {
		return nil
	}
	if q.history != nil && (q.history.owner != q.owner || q.history.frame != q || q.history.invocation != q.invocation) {
		return nil
	}
	return q
}
func (q *upolyRationalEvaluationScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{rationalEvaluation: q, rationalEvaluationInvocation: q.invocation}
}
func (q *upolyRationalEvaluationScope) prepareHistory() error {
	if q.history != nil {
		return nil
	}
	n := uint64(reflect.TypeOf(upolyRationalEvaluationHistory{}).Size())
	if e := q.owner.config.Integer.budget.Reserve(n); e != nil {
		return e
	}
	q.history = &upolyRationalEvaluationHistory{owner: q.owner, frame: q, next: q.owner.rationalEvaluationHistory, invocation: q.invocation}
	q.owner.rationalEvaluationHistory = q.history
	q.owner.rationalEvaluationHistoryBytes += n
	q.updateHistory()
	return nil
}
func (q *upolyRationalEvaluationScope) updateHistory() {
	if q.history != nil {
		h := q.history
		h.constructed = q.constructed
		h.normalize = q.normalize
		h.addMul = q.addMul
		h.bodyCause = q.bodyCause
		h.sourceCause = q.sourceCause
		h.ownCause = q.ownCause
		h.sourceRemaining = q.sourceRemaining
	}
}
func (q *upolyRationalEvaluationScope) pending() bool {
	z := q.owner.config.Integer
	mm := q.owner.modular
	return q.sourceRemaining != 0 || polynomialNegNumericPending(&q.numeric) || mm.matchesPNormalizeInvocation(&q.normalize) || z.matchesAddMulInvocation(&q.addMul)
}
func (q *upolyRationalEvaluationScope) finish(body error) error {
	if q.owner == nil || q.owner.self != q.owner || q != &q.owner.rationalEvaluation || q.executing || q.finishing {
		return ErrUPolynomialCleanupPending
	}
	q.finishing = true
	defer func() { q.finishing = false; q.updateHistory() }()
	if q.bodyCause == nil {
		q.bodyCause = body
	}
	z := q.owner.config.Integer
	if q.sourceCause == nil {
		q.sourceCause = q.owner.modular.pNormalizeInvocationSourceCause(&q.normalize)
	}
	if q.sourceCause == nil {
		q.sourceCause = z.addMulInvocationSourceCause(&q.addMul)
	}
	if q.sourceCause == nil {
		for q.sourceRemaining != 0 {
			if _, e := upolynomialSourceNumeralDel(z, &q.values[q.sourceRemaining-1]); e != nil {
				q.sourceCause = e
				break
			}
			q.sourceRemaining--
		}
	}
	if !q.pending() {
		q.active = false
	}
	return errors.Join(body, q.sourceCause)
}

// CleanupRationalEvaluation owns only the actual current public Horner scope.
// Completed younger children precede ak/r/d_i, and unjoined children follow them.
func (m *UPolynomialManager) CleanupRationalEvaluation() error {
	if e := m.cleanupRationalEvaluation(false); e != nil {
		return e
	}
	return m.cleanupRationalEvaluation(true)
}

type upolyRationalEvaluationInvocationReceipt struct {
	owner      *UPolynomialManager
	frame      *upolyRationalEvaluationScope
	invocation uint64
	history    *upolyRationalEvaluationHistory
}

func (m *UPolynomialManager) rationalEvaluationReceipt() upolyRationalEvaluationInvocationReceipt {
	q := &m.rationalEvaluation
	return upolyRationalEvaluationInvocationReceipt{m, q, q.invocation, q.history}
}
func (m *UPolynomialManager) cleanupRationalEvaluation(abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	r := m.rationalEvaluationReceipt()
	return m.cleanupRationalEvaluationWithReceipt(&r, abort)
}
func (m *UPolynomialManager) cleanupRationalEvaluationWithReceipt(r *upolyRationalEvaluationInvocationReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	q := &m.rationalEvaluation
	if r == nil || r.owner != m || r.frame != q || r.invocation == 0 || r.invocation != q.invocation || r.history != q.history {
		return nil
	}
	if q.history != nil && (q.history.owner != m || q.history.frame != q) {
		return ErrUPolynomialOwner
	}
	if !q.active {
		return nil
	}
	if q.owner != m || q.invocation == 0 || q.executing || q.finishing {
		return ErrUPolynomialCleanupPending
	}
	q.finishing = true
	z := m.config.Integer
	old := z.rationalEvaluationCaller
	z.rationalEvaluationCaller = m
	defer func() { z.rationalEvaluationCaller = old; q.finishing = false; q.updateHistory() }()
	stop := func(e error) error {
		if e != nil && q.ownCause == nil {
			q.ownCause = e
		}
		return e
	}
	if e := m.modular.cleanupPNormalizeInvocation(&q.normalize, false); e != nil {
		return stop(e)
	}
	if e := z.cleanupAddMulInvocation(&q.addMul, false); e != nil {
		return stop(e)
	}
	if !q.ownStarted {
		q.ownRemaining = q.constructed
		q.ownStarted = true
	}
	for q.ownRemaining != 0 {
		if e := z.cleanupRawNumeral(&q.values[q.ownRemaining-1]); e != nil {
			return stop(e)
		}
		q.ownRemaining--
	}
	q.falseAcknowledged = true
	if !abort {
		return nil
	}
	if e := m.modular.cleanupPNormalizeInvocation(&q.normalize, true); e != nil {
		return stop(e)
	}
	if e := z.cleanupAddMulInvocation(&q.addMul, true); e != nil {
		return stop(e)
	}
	if e := cleanupUPolynomialNumericRoots(z, &q.numeric); e != nil {
		return stop(e)
	}
	q.active = false
	return nil
}
func (m *UPolynomialManager) beginRationalEvaluation() (*upolyRationalEvaluationScope, error) {
	q := &m.rationalEvaluation
	if q.active {
		return nil, ErrUPolynomialCleanupPending
	}
	if q.invocation == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	*q = upolyRationalEvaluationScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, sourceRemaining: 1, constructed: 1}
	return q, nil
}
