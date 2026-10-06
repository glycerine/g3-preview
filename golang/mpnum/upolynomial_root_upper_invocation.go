// Source: src/math/polynomial/upolynomial.cpp:1906-1951.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
)

type upolyRootUpperBoundScope struct {
	constructed                              uint8
	target                                   UPolynomialView
	falseAcknowledged                        bool
	owner                                    *UPolynomialManager
	invocation                               uint64
	values                                   [5]Int
	numeric                                  polynomialNumericRoots
	normalize                                integerPNormalizeInvocationReceipt
	divisionRaw, extendedRaw                 integerRawInvocationReceipt
	buffers                                  [4]naturalPNBufferReceipt
	verification                             naturalPNVerificationReceipt
	sourceRemaining, ownRemaining            uint8
	active, executing, finishing, ownStarted bool
	bodyCause, sourceCause, ownCause         error
	history                                  *upolyRootUpperBoundHistory
}

// Diagnostics contain issued child tuples, never NUM payload/owning authority.
type upolyRootUpperBoundHistory struct {
	constructed                      uint8
	owner                            *UPolynomialManager
	frame                            *upolyRootUpperBoundScope
	next                             *upolyRootUpperBoundHistory
	invocation                       uint64
	normalize                        integerPNormalizeInvocationReceipt
	divisionRaw, extendedRaw         integerRawInvocationReceipt
	verification                     naturalPNVerificationReceipt
	buffers                          [4]upolyRootUpperBufferHistory
	bodyCause, sourceCause, ownCause error
	sourceRemaining                  uint8
}

func (m *UPolynomialManager) rootUpperBoundExecuting() bool {
	return m != nil && m.self == m && m.rootUpperBound.owner == m && m.rootUpperBound.active && (m.rootUpperBound.executing || m.rootUpperBound.finishing)
}
func (z *IntegerManager) rootUpperBoundCallerExecuting() bool {
	return z != nil && z.self == z && z.rootUpperBoundCaller != nil && z.rootUpperBoundCaller.config.Integer == z && z.rootUpperBoundCaller.rootUpperBoundExecuting()
}
func (c integerAllocationInvocationContext) rootUpperBoundFrame(z *IntegerManager) *upolyRootUpperBoundScope {
	q := c.rootUpperBound
	if z == nil || z.self != z || q == nil || q.owner == nil || q.owner.self != q.owner || q != &q.owner.rootUpperBound || q.owner.config.Integer != z || z.rootUpperBoundCaller != q.owner || q.owner.callbackDepth != 0 || !q.active || !q.executing || q.constructed < 4 || c.rootUpperBoundInvocation == 0 || c.rootUpperBoundInvocation != q.invocation {
		return nil
	}
	if q.history != nil && (q.history.owner != q.owner || q.history.frame != q || q.history.invocation != q.invocation) {
		return nil
	}
	return q
}
func (q *upolyRootUpperBoundScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{rootUpperBound: q, rootUpperBoundInvocation: q.invocation}
}
func (q *upolyRootUpperBoundScope) prepareHistory() error {
	if q.history != nil {
		return nil
	}
	n := uint64(reflect.TypeOf(upolyRootUpperBoundHistory{}).Size())
	if e := q.owner.config.Integer.budget.Reserve(n); e != nil {
		return e
	}
	q.history = &upolyRootUpperBoundHistory{owner: q.owner, frame: q, next: q.owner.rootUpperBoundHistory, invocation: q.invocation}
	q.owner.rootUpperBoundHistory = q.history
	q.owner.rootUpperBoundHistoryBytes += n
	q.updateHistory()
	return nil
}
func (q *upolyRootUpperBoundScope) updateHistory() {
	if q.history != nil {
		h := q.history
		h.constructed = q.constructed
		h.normalize = q.normalize
		h.divisionRaw = q.divisionRaw
		h.extendedRaw = q.extendedRaw
		h.verification = q.verification
		for i, r := range q.buffers {
			h.buffers[i] = upolyRootUpperBufferHistory{r.owner, r.buffer, r.invocation, r.private}
		}
		h.bodyCause = q.bodyCause
		h.sourceCause = q.sourceCause
		h.ownCause = q.ownCause
		h.sourceRemaining = q.sourceRemaining
	}
}
func (q *upolyRootUpperBoundScope) pending() bool {
	z := q.owner.config.Integer
	mm := q.owner.modular
	return q.sourceRemaining != 0 || polynomialNegNumericPending(&q.numeric) || mm.matchesPNormalizeInvocation(&q.normalize) || z.matchesRawInvocation(&q.divisionRaw) || z.matchesRawInvocation(&q.extendedRaw) || z.natural.matchesPNVerification(&q.verification) || q.pendingBuffers()
}
func (q *upolyRootUpperBoundScope) finish(body error) error {
	if q.owner == nil || q.owner.self != q.owner || q != &q.owner.rootUpperBound || q.executing || q.finishing {
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
		q.sourceCause = q.numericChildCause()
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

// CleanupRootUpperBound owns only the actual current public Cauchy-bound scope.
// Completed younger children precede the reached NUM suffix; unjoined children follow it.
func (m *UPolynomialManager) CleanupRootUpperBound() error {
	if e := m.cleanupRootUpperBound(false); e != nil {
		return e
	}
	return m.cleanupRootUpperBound(true)
}

type upolyRootUpperBoundInvocationReceipt struct {
	owner      *UPolynomialManager
	frame      *upolyRootUpperBoundScope
	invocation uint64
	history    *upolyRootUpperBoundHistory
}

func (m *UPolynomialManager) rootUpperBoundReceipt() upolyRootUpperBoundInvocationReceipt {
	q := &m.rootUpperBound
	return upolyRootUpperBoundInvocationReceipt{m, q, q.invocation, q.history}
}
func (m *UPolynomialManager) cleanupRootUpperBound(abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	r := m.rootUpperBoundReceipt()
	return m.cleanupRootUpperBoundWithReceipt(&r, abort)
}
func (m *UPolynomialManager) cleanupRootUpperBoundWithReceipt(r *upolyRootUpperBoundInvocationReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	q := &m.rootUpperBound
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
	old := z.rootUpperBoundCaller
	z.rootUpperBoundCaller = m
	oldNatural := z.natural.rootUpperBoundCaller
	z.natural.rootUpperBoundCaller = m
	defer func() {
		z.natural.rootUpperBoundCaller = oldNatural
		z.rootUpperBoundCaller = old
		q.finishing = false
		q.updateHistory()
	}()
	stop := func(e error) error {
		if e != nil && q.ownCause == nil {
			q.ownCause = e
		}
		return e
	}
	if e := m.modular.cleanupPNormalizeInvocation(&q.normalize, false); e != nil {
		return stop(e)
	}
	if e := q.cleanupNumericChildren(false); e != nil {
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
	if e := q.cleanupNumericChildren(true); e != nil {
		return stop(e)
	}
	if e := cleanupUPolynomialNumericRoots(z, &q.numeric); e != nil {
		return stop(e)
	}
	q.active = false
	return nil
}
func (m *UPolynomialManager) beginRootUpperBound() (*upolyRootUpperBoundScope, error) {
	q := &m.rootUpperBound
	if q.active {
		return nil, ErrUPolynomialCleanupPending
	}
	if q.invocation == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	*q = upolyRootUpperBoundScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, sourceRemaining: 4, constructed: 4}
	return q, nil
}

type upolyRootUpperBufferHistory struct {
	owner      *NaturalManager
	buffer     *sourceLimbBuffer
	invocation uint64
	private    bool
}

func (q *upolyRootUpperBoundScope) matches() bool {
	return q != nil && q.owner != nil && q.owner.self == q.owner && q == &q.owner.rootUpperBound && q.invocation != 0 && (q.history == nil || (q.history.owner == q.owner && q.history.frame == q && q.history.invocation == q.invocation))
}
func (q *upolyRootUpperBoundScope) pendingBuffers() bool {
	for _, r := range q.buffers {
		if r.pending != (memory.Block{}) {
			return true
		}
	}
	return false
}
func (q *upolyRootUpperBoundScope) numericChildCause() error {
	z := q.owner.config.Integer
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
		if e := z.rawInvocationSourceCause(r); e != nil {
			return e
		}
	}
	return z.natural.pnVerificationSourceCause(&q.verification)
}
func (q *upolyRootUpperBoundScope) cleanupNumericChildren(abort bool) error {
	z := q.owner.config.Integer
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
		if e := z.cleanupRawInvocation(r, abort); e != nil {
			return e
		}
	}
	if e := z.natural.cleanupPNVerification(&q.verification, abort); e != nil {
		return e
	}
	if abort {
		for i := len(q.buffers) - 1; i >= 0; i-- {
			if e := q.buffers[i].cleanup(); e != nil {
				return e
			}
		}
	}
	return nil
}
func (c integerAllocationInvocationContext) recordRootUpperRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.rootUpperBoundFrame(z)
	if q == nil || !z.matchesRawInvocation(r) {
		return
	}
	r.frame.rootUpperBound = q
	r.frame.rootUpperInvocation = q.invocation
	switch r.frame {
	case &z.divisionRaw:
		q.divisionRaw = *r
	case &z.extendedRaw:
		q.extendedRaw = *r
	}
}

func (n *NaturalManager) rootUpperBoundCallerExecuting() bool {
	return n != nil && n.self == n && n.rootUpperBoundCaller != nil && n.rootUpperBoundCaller.self == n.rootUpperBoundCaller && n.rootUpperBoundCaller.config.Integer != nil && n.rootUpperBoundCaller.config.Integer.natural == n && n.rootUpperBoundCaller.rootUpperBoundExecuting()
}
