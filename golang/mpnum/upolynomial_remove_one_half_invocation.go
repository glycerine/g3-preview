// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
)

type upolyRemoveOneHalfRootScope struct {
	constructed                                           uint8
	target                                                UPolynomialView
	output                                                UPolynomial
	falseAcknowledged                                     bool
	owner                                                 *UPolynomialManager
	invocation                                            uint64
	values                                                [1]Int
	numeric                                               polynomialNumericRoots
	failedRowStore                                        *UPolynomialCoefficientStore
	failedRowBacking, failedRowGeneration, failedRowIndex uint32
	failedRowCell                                         numericCellRef
	failedRowAcknowledged                                 bool
	normalize                                             integerPNormalizeInvocationReceipt
	assertion                                             upolyOneHalfRootInvocationReceipt
	addMul                                                integerAddMulInvocationReceipt
	divisionRaw, extendedRaw                              integerRawInvocationReceipt
	buffers                                               [4]naturalPNBufferReceipt
	verification                                          naturalPNVerificationReceipt
	sourceRemaining, ownRemaining                         uint8
	active, executing, finishing, ownStarted              bool
	bodyCause, sourceCause, ownCause                      error
	history                                               *upolyRemoveOneHalfRootHistory
}

// Diagnostics contain issued child tuples, never NUM payload/owning authority.
type upolyRemoveOneHalfRootHistory struct {
	constructed                                           uint8
	failedRowStore                                        *UPolynomialCoefficientStore
	failedRowBacking, failedRowGeneration, failedRowIndex uint32
	failedRowCell                                         numericCellRef
	output                                                UPolynomial
	owner                                                 *UPolynomialManager
	frame                                                 *upolyRemoveOneHalfRootScope
	next                                                  *upolyRemoveOneHalfRootHistory
	invocation                                            uint64
	normalize                                             integerPNormalizeInvocationReceipt
	assertion                                             upolyOneHalfRootInvocationReceipt
	addMul                                                integerAddMulInvocationReceipt
	divisionRaw, extendedRaw                              integerRawInvocationReceipt
	verification                                          naturalPNVerificationReceipt
	buffers                                               [4]upolyRemoveOneHalfBufferHistory
	bodyCause, sourceCause, ownCause                      error
	sourceRemaining                                       uint8
}

func (m *UPolynomialManager) removeOneHalfRootExecuting() bool {
	return m != nil && m.self == m && m.removeOneHalfRoot.owner == m && m.removeOneHalfRoot.active && (m.removeOneHalfRoot.executing || m.removeOneHalfRoot.finishing)
}
func (z *IntegerManager) removeOneHalfRootCallerExecuting() bool {
	return z != nil && z.self == z && z.removeOneHalfRootCaller != nil && z.removeOneHalfRootCaller.config.Integer == z && z.removeOneHalfRootCaller.removeOneHalfRootExecuting()
}
func (c integerAllocationInvocationContext) removeOneHalfRootFrame(z *IntegerManager) *upolyRemoveOneHalfRootScope {
	q := c.removeOneHalfRoot
	if z == nil || z.self != z || q == nil || q.owner == nil || q.owner.self != q.owner || q != &q.owner.removeOneHalfRoot || q.owner.config.Integer != z || z.removeOneHalfRootCaller != q.owner || q.owner.callbackDepth != 0 || !q.active || !q.executing || c.removeOneHalfRootInvocation == 0 || c.removeOneHalfRootInvocation != q.invocation {
		return nil
	}
	if q.history != nil && (q.history.owner != q.owner || q.history.frame != q || q.history.invocation != q.invocation) {
		return nil
	}
	return q
}
func (q *upolyRemoveOneHalfRootScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{removeOneHalfRoot: q, removeOneHalfRootInvocation: q.invocation}
}
func (q *upolyRemoveOneHalfRootScope) prepareHistory() error {
	if q.history != nil {
		return nil
	}
	n := uint64(reflect.TypeOf(upolyRemoveOneHalfRootHistory{}).Size())
	if e := q.owner.config.Integer.budget.Reserve(n); e != nil {
		return e
	}
	q.history = &upolyRemoveOneHalfRootHistory{owner: q.owner, frame: q, next: q.owner.removeOneHalfRootHistory, invocation: q.invocation}
	q.owner.removeOneHalfRootHistory = q.history
	q.owner.removeOneHalfRootHistoryBytes += n
	q.updateHistory()
	return nil
}
func (q *upolyRemoveOneHalfRootScope) updateHistory() {
	if q.history != nil {
		h := q.history
		h.constructed = q.constructed
		h.output = q.output
		h.failedRowStore = q.failedRowStore
		h.failedRowBacking = q.failedRowBacking
		h.failedRowGeneration = q.failedRowGeneration
		h.failedRowIndex = q.failedRowIndex
		h.failedRowCell = q.failedRowCell
		h.assertion = q.assertion
		h.addMul = q.addMul
		h.normalize = q.normalize
		h.divisionRaw = q.divisionRaw
		h.extendedRaw = q.extendedRaw
		h.verification = q.verification
		for i, r := range q.buffers {
			h.buffers[i] = upolyRemoveOneHalfBufferHistory{r.owner, r.buffer, r.invocation, r.private}
		}
		h.bodyCause = q.bodyCause
		h.sourceCause = q.sourceCause
		h.ownCause = q.ownCause
		h.sourceRemaining = q.sourceRemaining
	}
}
func (q *upolyRemoveOneHalfRootScope) pending() bool {
	z := q.owner.config.Integer
	mm := q.owner.modular
	return (q.failedRowStore != nil && !q.failedRowAcknowledged) || (q.assertion.frame != nil && q.assertion.frame.invocation == q.assertion.invocation && q.assertion.frame.active) || z.matchesAddMulInvocation(&q.addMul) || q.sourceRemaining != 0 || polynomialNegNumericPending(&q.numeric) || mm.matchesPNormalizeInvocation(&q.normalize) || z.matchesRawInvocation(&q.divisionRaw) || z.matchesRawInvocation(&q.extendedRaw) || z.natural.matchesPNVerification(&q.verification) || q.pendingBuffers()
}
func (q *upolyRemoveOneHalfRootScope) finish(body error) error {
	if q.owner == nil || q.owner.self != q.owner || q != &q.owner.removeOneHalfRoot || q.executing || q.finishing {
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
	if q.sourceCause == nil && q.assertion.frame != nil && q.assertion.frame.invocation == q.assertion.invocation {
		q.sourceCause = q.assertion.frame.sourceCause
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

// CleanupRemoveOneHalfRoot owns the exact assertion and div_rem_core caller.
// Completed younger children precede the reached NUM suffix; unjoined children follow it.
func (m *UPolynomialManager) CleanupRemoveOneHalfRoot() error {
	if e := m.cleanupRemoveOneHalfRoot(false); e != nil {
		return e
	}
	return m.cleanupRemoveOneHalfRoot(true)
}

type upolyRemoveOneHalfRootInvocationReceipt struct {
	owner      *UPolynomialManager
	frame      *upolyRemoveOneHalfRootScope
	invocation uint64
	history    *upolyRemoveOneHalfRootHistory
}

func (m *UPolynomialManager) removeOneHalfRootReceipt() upolyRemoveOneHalfRootInvocationReceipt {
	q := &m.removeOneHalfRoot
	return upolyRemoveOneHalfRootInvocationReceipt{m, q, q.invocation, q.history}
}
func (m *UPolynomialManager) cleanupRemoveOneHalfRoot(abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	r := m.removeOneHalfRootReceipt()
	return m.cleanupRemoveOneHalfRootWithReceipt(&r, abort)
}
func (m *UPolynomialManager) cleanupRemoveOneHalfRootWithReceipt(r *upolyRemoveOneHalfRootInvocationReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	q := &m.removeOneHalfRoot
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
	old := z.removeOneHalfRootCaller
	z.removeOneHalfRootCaller = m
	oldNatural := z.natural.removeOneHalfRootCaller
	z.natural.removeOneHalfRootCaller = m
	defer func() {
		z.natural.removeOneHalfRootCaller = oldNatural
		z.removeOneHalfRootCaller = old
		q.finishing = false
		q.updateHistory()
	}()
	stop := func(e error) error {
		if e != nil && q.ownCause == nil {
			q.ownCause = e
		}
		return e
	}
	if q.assertion.frame != nil {
		if e := m.cleanupOneHalfRootWithReceipt(&q.assertion, false); e != nil {
			return stop(e)
		}
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
	if q.assertion.frame != nil {
		if e := m.cleanupOneHalfRootWithReceipt(&q.assertion, true); e != nil {
			return stop(e)
		}
	}
	if e := m.modular.cleanupPNormalizeInvocation(&q.normalize, true); e != nil {
		return stop(e)
	}
	if e := q.cleanupNumericChildren(true); e != nil {
		return stop(e)
	}
	if e := q.acknowledgeFailedRow(); e != nil {
		return stop(e)
	}
	if e := cleanupUPolynomialNumericRoots(z, &q.numeric); e != nil {
		return stop(e)
	}
	q.active = false
	return nil
}
func (m *UPolynomialManager) beginRemoveOneHalfRoot() (*upolyRemoveOneHalfRootScope, error) {
	q := &m.removeOneHalfRoot
	if q.active {
		return nil, ErrUPolynomialCleanupPending
	}
	if q.invocation == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	*q = upolyRemoveOneHalfRootScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, sourceRemaining: 0, constructed: 0}
	// First issued diagnostic storage is already charged in the owner header.
	// Later allocating calls acquire stable records only at actual leaf branches.
	if m.removeOneHalfRootFirstHistory.invocation == 0 {
		h := &m.removeOneHalfRootFirstHistory
		*h = upolyRemoveOneHalfRootHistory{owner: m, frame: q, invocation: q.invocation, next: m.removeOneHalfRootHistory}
		m.removeOneHalfRootHistory = h
		q.history = h
		q.updateHistory()
	}
	return q, nil
}

type upolyRemoveOneHalfBufferHistory struct {
	owner      *NaturalManager
	buffer     *sourceLimbBuffer
	invocation uint64
	private    bool
}

func (q *upolyRemoveOneHalfRootScope) matches() bool {
	return q != nil && q.owner != nil && q.owner.self == q.owner && q == &q.owner.removeOneHalfRoot && q.invocation != 0 && (q.history == nil || (q.history.owner == q.owner && q.history.frame == q && q.history.invocation == q.invocation))
}
func (q *upolyRemoveOneHalfRootScope) pendingBuffers() bool {
	for _, r := range q.buffers {
		if r.pending != (memory.Block{}) {
			return true
		}
	}
	return false
}
func (q *upolyRemoveOneHalfRootScope) numericChildCause() error {
	z := q.owner.config.Integer
	if e := z.addMulInvocationSourceCause(&q.addMul); e != nil {
		return e
	}
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
		if e := z.rawInvocationSourceCause(r); e != nil {
			return e
		}
	}
	return z.natural.pnVerificationSourceCause(&q.verification)
}
func (q *upolyRemoveOneHalfRootScope) cleanupNumericChildren(abort bool) error {
	z := q.owner.config.Integer
	if e := z.cleanupAddMulInvocation(&q.addMul, abort); e != nil {
		return e
	}
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
func (c integerAllocationInvocationContext) recordRemoveOneHalfRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.removeOneHalfRootFrame(z)
	if q == nil || !z.matchesRawInvocation(r) {
		return
	}
	r.frame.removeOneHalfRoot = q
	r.frame.removeOneHalfInvocation = q.invocation
	switch r.frame {
	case &z.divisionRaw:
		q.divisionRaw = *r
	case &z.extendedRaw:
		q.extendedRaw = *r
	}
}

func (n *NaturalManager) removeOneHalfRootCallerExecuting() bool {
	return n != nil && n.self == n && n.removeOneHalfRootCaller != nil && n.removeOneHalfRootCaller.self == n.removeOneHalfRootCaller && n.removeOneHalfRootCaller.config.Integer != nil && n.removeOneHalfRootCaller.config.Integer.natural == n && n.removeOneHalfRootCaller.removeOneHalfRootExecuting()
}

// The lexical am source suffix runs at the actual div_rem_core return, before
// the outer div reset/swap. Source destruction remains distinct from own abort.
func (q *upolyRemoveOneHalfRootScope) finishAmSource(body error) error {
	if !q.matches() || !q.executing || q.finishing {
		return ErrUPolynomialCleanupPending
	}
	q.finishing = true
	defer func() { q.finishing = false; q.updateHistory() }()
	if q.sourceCause == nil {
		q.sourceCause = q.owner.modular.pNormalizeInvocationSourceCause(&q.normalize)
	}
	if q.sourceCause == nil {
		q.sourceCause = q.numericChildCause()
	}
	if q.sourceCause == nil && q.sourceRemaining != 0 {
		if _, e := upolynomialSourceNumeralDel(q.owner.config.Integer, &q.values[0]); e != nil {
			q.sourceCause = e
		} else {
			q.sourceRemaining = 0
		}
	}
	return errors.Join(body, q.sourceCause)
}
func removeOrHenselAddMulChild(context integerAllocationInvocationContext, r *upolyHenselPairInvocationReceipt, c *upolyHenselCallerScope) *integerAddMulInvocationReceipt {
	if q := context.removeOneHalfRoot; q != nil && q.owner != nil && context.removeOneHalfRootFrame(q.owner.config.Integer) != nil {
		return &q.addMul
	}
	return henselNumericAddMulChild(r, c)
}

// A failed source SET can publish this owner's dead allocation token into a
// constructed buffer NUM before SIZE. Owning acknowledgment clears only that
// captured exact poison cell; it does not delete a foreign replacement NUM.
func (q *upolyRemoveOneHalfRootScope) recordFailedRow(s *upolyVectorState, i uint32) {
	if !q.matches() || !q.executing || s != &q.owner.div1 || s.backing == 0 || i >= s.initialized {
		return
	}
	row := &q.owner.values(s)[i]
	if !q.owner.config.Integer.isFailureRef(row.cell) {
		return
	}
	q.failedRowStore = q.owner.store
	q.failedRowBacking = s.backing
	q.failedRowGeneration = s.generation
	q.failedRowIndex = i
	q.failedRowCell = row.cell
}
func (q *upolyRemoveOneHalfRootScope) acknowledgeFailedRow() error {
	if q.failedRowStore == nil || q.failedRowAcknowledged {
		return nil
	}
	s := q.failedRowStore
	if s.self != s || q.failedRowBacking == 0 || uint64(q.failedRowBacking) > uint64(len(s.backings)) {
		return ErrUPolynomialOwner
	}
	b := &s.backings[q.failedRowBacking-1]
	if b.live && b.generation == q.failedRowGeneration && q.failedRowIndex < b.initialized {
		row := &b.values[q.failedRowIndex]
		if row.cell == q.failedRowCell && q.owner.config.Integer.isFailureRef(row.cell) {
			row.cell = 0
		}
	}
	q.failedRowAcknowledged = true
	return nil
}
