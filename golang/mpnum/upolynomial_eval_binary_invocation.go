// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type upolyBinaryEvaluationScope struct {
	target                                   UPolynomialView
	falseAcknowledged                        bool
	owner                                    *UPolynomialManager
	invocation                               uint64
	values                                   [2]Int
	numeric                                  polynomialNumericRoots
	normalize                                integerPNormalizeInvocationReceipt
	addMul                                   integerAddMulInvocationReceipt
	sourceRemaining, ownRemaining            uint8
	active, executing, finishing, ownStarted bool
	bodyCause, sourceCause, ownCause         error
	history                                  *upolyBinaryEvaluationHistory
}

// Diagnostics contain issued child tuples, never NUM payload/owning authority.
type upolyBinaryEvaluationHistory struct {
	owner                            *UPolynomialManager
	frame                            *upolyBinaryEvaluationScope
	next                             *upolyBinaryEvaluationHistory
	invocation                       uint64
	normalize                        integerPNormalizeInvocationReceipt
	addMul                           integerAddMulInvocationReceipt
	bodyCause, sourceCause, ownCause error
	sourceRemaining                  uint8
}

func (m *UPolynomialManager) binaryEvaluationExecuting() bool {
	return m != nil && m.self == m && m.binaryEvaluation.owner == m && m.binaryEvaluation.active && (m.binaryEvaluation.executing || m.binaryEvaluation.finishing)
}
func (z *IntegerManager) binaryEvaluationCallerExecuting() bool {
	return z != nil && z.self == z && z.binaryEvaluationCaller != nil && z.binaryEvaluationCaller.config.Integer == z && z.binaryEvaluationCaller.binaryEvaluationExecuting()
}
func (c integerAllocationInvocationContext) binaryEvaluationFrame(z *IntegerManager) *upolyBinaryEvaluationScope {
	q := c.binaryEvaluation
	if z == nil || z.self != z || q == nil || q.owner == nil || q.owner.self != q.owner || q != &q.owner.binaryEvaluation || q.owner.config.Integer != z || z.binaryEvaluationCaller != q.owner || q.owner.callbackDepth != 0 || !q.active || !q.executing || c.binaryEvaluationInvocation == 0 || c.binaryEvaluationInvocation != q.invocation {
		return nil
	}
	if q.history != nil && (q.history.owner != q.owner || q.history.frame != q || q.history.invocation != q.invocation) {
		return nil
	}
	return q
}
func (q *upolyBinaryEvaluationScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{binaryEvaluation: q, binaryEvaluationInvocation: q.invocation}
}
func (q *upolyBinaryEvaluationScope) prepareHistory() error {
	if q.history != nil {
		return nil
	}
	n := uint64(reflect.TypeOf(upolyBinaryEvaluationHistory{}).Size())
	if e := q.owner.config.Integer.budget.Reserve(n); e != nil {
		return e
	}
	q.history = &upolyBinaryEvaluationHistory{owner: q.owner, frame: q, next: q.owner.binaryEvaluationHistory, invocation: q.invocation}
	q.owner.binaryEvaluationHistory = q.history
	q.owner.binaryEvaluationHistoryBytes += n
	q.updateHistory()
	return nil
}
func (q *upolyBinaryEvaluationScope) updateHistory() {
	if q.history != nil {
		h := q.history
		h.normalize = q.normalize
		h.addMul = q.addMul
		h.bodyCause = q.bodyCause
		h.sourceCause = q.sourceCause
		h.ownCause = q.ownCause
		h.sourceRemaining = q.sourceRemaining
	}
}
func (q *upolyBinaryEvaluationScope) pending() bool {
	z := q.owner.config.Integer
	mm := q.owner.modular
	return q.sourceRemaining != 0 || polynomialNegNumericPending(&q.numeric) || mm.matchesPNormalizeInvocation(&q.normalize) || z.matchesAddMulInvocation(&q.addMul)
}
func (q *upolyBinaryEvaluationScope) finish(body error) error {
	if q.owner == nil || q.owner.self != q.owner || q != &q.owner.binaryEvaluation || q.executing || q.finishing {
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

// CleanupBinaryEvaluation owns only the actual current public Horner scope.
// Completed younger children precede ak/r, and unjoined children follow them.
func (m *UPolynomialManager) CleanupBinaryEvaluation() error {
	if e := m.cleanupBinaryEvaluation(false); e != nil {
		return e
	}
	return m.cleanupBinaryEvaluation(true)
}

type upolyBinaryEvaluationInvocationReceipt struct {
	owner      *UPolynomialManager
	frame      *upolyBinaryEvaluationScope
	invocation uint64
	history    *upolyBinaryEvaluationHistory
}

func (m *UPolynomialManager) binaryEvaluationReceipt() upolyBinaryEvaluationInvocationReceipt {
	q := &m.binaryEvaluation
	return upolyBinaryEvaluationInvocationReceipt{m, q, q.invocation, q.history}
}
func (m *UPolynomialManager) cleanupBinaryEvaluation(abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	r := m.binaryEvaluationReceipt()
	return m.cleanupBinaryEvaluationWithReceipt(&r, abort)
}
func (m *UPolynomialManager) cleanupBinaryEvaluationWithReceipt(r *upolyBinaryEvaluationInvocationReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	q := &m.binaryEvaluation
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
	old := z.binaryEvaluationCaller
	z.binaryEvaluationCaller = m
	defer func() { z.binaryEvaluationCaller = old; q.finishing = false; q.updateHistory() }()
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
		q.ownRemaining = 2
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
func (m *UPolynomialManager) beginBinaryEvaluation() (*upolyBinaryEvaluationScope, error) {
	q := &m.binaryEvaluation
	if q.active {
		return nil, ErrUPolynomialCleanupPending
	}
	if q.invocation == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	*q = upolyBinaryEvaluationScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, sourceRemaining: 2}
	return q, nil
}
