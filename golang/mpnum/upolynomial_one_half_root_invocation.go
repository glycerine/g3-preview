// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type upolyOneHalfRootScope struct {
	constructed                              uint8
	removeParent                             *upolyRemoveOneHalfRootScope
	removeParentInvocation                   uint64
	target                                   UPolynomialView
	falseAcknowledged                        bool
	owner                                    *UPolynomialManager
	invocation                               uint64
	values                                   [2]Int
	numeric                                  polynomialNumericRoots
	normalize                                integerPNormalizeInvocationReceipt
	sourceRemaining, ownRemaining            uint8
	active, executing, finishing, ownStarted bool
	bodyCause, sourceCause, ownCause         error
	history                                  *upolyOneHalfRootHistory
}

// Diagnostics contain issued child tuples, never NUM payload/owning authority.
type upolyOneHalfRootHistory struct {
	target                           UPolynomialView
	constructed                      uint8
	owner                            *UPolynomialManager
	frame                            *upolyOneHalfRootScope
	next                             *upolyOneHalfRootHistory
	invocation                       uint64
	normalize                        integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, ownCause error
	sourceRemaining                  uint8
}

func (m *UPolynomialManager) oneHalfRootExecuting() bool {
	return m != nil && m.self == m && m.oneHalfRoot.owner == m && m.oneHalfRoot.active && (m.oneHalfRoot.executing || m.oneHalfRoot.finishing)
}
func (z *IntegerManager) oneHalfRootCallerExecuting() bool {
	return z != nil && z.self == z && z.oneHalfRootCaller != nil && z.oneHalfRootCaller.config.Integer == z && z.oneHalfRootCaller.oneHalfRootExecuting()
}
func (n *NaturalManager) oneHalfRootCallerExecuting() bool {
	return n != nil && n.self == n && n.oneHalfRootCaller != nil && n.oneHalfRootCaller.config.Integer != nil && n.oneHalfRootCaller.config.Integer.natural == n && n.oneHalfRootCaller.oneHalfRootExecuting()
}
func (c integerAllocationInvocationContext) oneHalfRootFrame(z *IntegerManager) *upolyOneHalfRootScope {
	q := c.oneHalfRoot
	if z == nil || z.self != z || q == nil || q.owner == nil || q.owner.self != q.owner || q != &q.owner.oneHalfRoot || q.owner.config.Integer != z || z.oneHalfRootCaller != q.owner || q.owner.callbackDepth != 0 || !q.active || !q.executing || q.constructed != 2 || c.oneHalfRootInvocation == 0 || c.oneHalfRootInvocation != q.invocation {
		return nil
	}
	if q.history != nil && (q.history.owner != q.owner || q.history.frame != q || q.history.invocation != q.invocation) {
		return nil
	}
	return q
}
func (q *upolyOneHalfRootScope) context() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{oneHalfRoot: q, oneHalfRootInvocation: q.invocation}
}
func (q *upolyOneHalfRootScope) prepareHistory() error {
	if p := q.removeParent; p != nil {
		if !p.matches() || p.owner != q.owner || p.invocation != q.removeParentInvocation || !p.active || !p.executing {
			return ErrUPolynomialOwner
		}
		if e := p.prepareHistory(); e != nil {
			return e
		}
	}

	if q.history != nil {
		return nil
	}
	n := uint64(reflect.TypeOf(upolyOneHalfRootHistory{}).Size())
	if e := q.owner.config.Integer.budget.Reserve(n); e != nil {
		return e
	}
	q.history = &upolyOneHalfRootHistory{owner: q.owner, frame: q, next: q.owner.oneHalfRootHistory, invocation: q.invocation}
	q.owner.oneHalfRootHistory = q.history
	q.owner.oneHalfRootHistoryBytes += n
	q.updateHistory()
	return nil
}
func (q *upolyOneHalfRootScope) updateHistory() {
	if q.history != nil {
		h := q.history
		h.target = q.target
		h.constructed = q.constructed
		h.normalize = q.normalize
		h.bodyCause = q.bodyCause
		h.sourceCause = q.sourceCause
		h.ownCause = q.ownCause
		h.sourceRemaining = q.sourceRemaining
	}
}
func (q *upolyOneHalfRootScope) pending() bool {
	mm := q.owner.modular
	return q.sourceRemaining != 0 || polynomialNegNumericPending(&q.numeric) || mm.matchesPNormalizeInvocation(&q.normalize)
}
func (q *upolyOneHalfRootScope) finish(body error) error {
	if q.owner == nil || q.owner.self != q.owner || q != &q.owner.oneHalfRoot || q.executing || q.finishing {
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

// CleanupOneHalfRoot owns only the actual current public Horner scope.
// Completed PN children precede ak then r, and unjoined children follow them.
func (m *UPolynomialManager) CleanupOneHalfRoot() error {
	if e := m.cleanupOneHalfRoot(false); e != nil {
		return e
	}
	return m.cleanupOneHalfRoot(true)
}

type upolyOneHalfRootInvocationReceipt struct {
	owner      *UPolynomialManager
	frame      *upolyOneHalfRootScope
	invocation uint64
	history    *upolyOneHalfRootHistory
}

func (m *UPolynomialManager) oneHalfRootReceipt() upolyOneHalfRootInvocationReceipt {
	q := &m.oneHalfRoot
	return upolyOneHalfRootInvocationReceipt{m, q, q.invocation, q.history}
}
func (m *UPolynomialManager) cleanupOneHalfRoot(abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	r := m.oneHalfRootReceipt()
	return m.cleanupOneHalfRootWithReceipt(&r, abort)
}
func (m *UPolynomialManager) cleanupOneHalfRootWithReceipt(r *upolyOneHalfRootInvocationReceipt, abort bool) error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	q := &m.oneHalfRoot
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
	old, oldN := z.oneHalfRootCaller, z.natural.oneHalfRootCaller
	z.oneHalfRootCaller = m
	z.natural.oneHalfRootCaller = m
	defer func() {
		z.oneHalfRootCaller = old
		z.natural.oneHalfRootCaller = oldN
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
	if e := cleanupUPolynomialNumericRoots(z, &q.numeric); e != nil {
		return stop(e)
	}
	q.active = false
	return nil
}
func (m *UPolynomialManager) beginOneHalfRoot() (*upolyOneHalfRootScope, error) {
	q := &m.oneHalfRoot
	if q.active {
		return nil, ErrUPolynomialCleanupPending
	}
	if q.invocation == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	*q = upolyOneHalfRootScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, sourceRemaining: 2, constructed: 2}
	return q, nil
}

// The private typed seam rejects a copied/stale frame or a different real
// modular receiver before the literal numeric source action.
func (m *IntegerModularManager) oneHalfRootContextValid(c integerAllocationInvocationContext) bool {
	return m != nil && m.self == m && m.manager != nil && c.oneHalfRootFrame(m.manager) != nil && c.oneHalfRoot.owner.modular == m
}
func (m *IntegerModularManager) copyWithOneHalfRootContext(out, value *Int, c integerAllocationInvocationContext) error {
	if !m.oneHalfRootContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.copyWithSourceContext(out, value, c)
}
func (m *IntegerModularManager) mul2kWithOneHalfRootContext(value *Int, k uint32, out *Int, c integerAllocationInvocationContext) error {
	if !m.oneHalfRootContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.mul2kWithSourceContext(value, k, out, c)
}
func (m *IntegerModularManager) addWithOneHalfRootContext(a, b, out *Int, c integerAllocationInvocationContext) error {
	if !m.oneHalfRootContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.addWithSourceContext(a, b, out, c)
}
