// Source: upolynomial.cpp:2520-2580; public sequence construction locals.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type upolySequenceSturmScope struct {
	copy                           unumeral.IntegerCopyAllocationReceipt
	normalize                      unumeral.IntegerNormalizeInvocationReceipt
	integerBinding, naturalBinding unumeral.NumericCallerBinding
	owner                          *UPolynomialSequence
	invocation                     uint64
	previousCallbackDepth          uint32
	remainder, prime               upolyVectorState
	live                           [2]bool // remainder, prime/product
	sourceCursor, ownCursor        [2]uint32
	active, executing, finishing   bool
	bodyCause, sourceStop          error
}

func (q *upolySequenceSturmScope) valid() bool {
	return q != nil && q.owner != nil && q.owner.self == q.owner && q == &q.owner.sturm && q.owner.manager != nil && q.owner.manager.self == q.owner.manager
}
func (q *upolySequenceSturmScope) vector(which uint8) *upolyVectorState {
	if which == 0 {
		return &q.remainder
	}
	return &q.prime
}
func (q *upolySequenceSturmScope) finishVector(which uint8, cause error) error {
	if q.bodyCause == nil {
		q.bodyCause = cause
	}
	if q.sourceStop != nil {
		return errors.Join(cause, q.sourceStop)
	}
	if !q.live[which] {
		return cause
	}
	q.finishing = true
	defer func() { q.finishing = false }()
	m := q.owner.manager
	state := q.vector(which)
	values := m.values(state)
	for q.sourceCursor[which] < state.size {
		if e := unumeral.SourceIntegerDel(m.config.Integer, &values[q.sourceCursor[which]]); e != nil {
			q.sourceStop = e
			return errors.Join(cause, e)
		}
		q.sourceCursor[which]++
	}
	state.size = 0
	m.freeBacking(state)
	q.live[which] = false
	return cause
}
func (q *upolySequenceSturmScope) core() (cause error) {
	m, s := q.owner.manager, &q.owner.state
	q.live[0] = true
	defer func() { cause = q.finishVector(0, cause) }()
	for {
		n := uint32(len(s.sizes))
		a, e := m.sequenceRow(s, n-2)
		if e != nil {
			return e
		}
		b, e := m.sequenceRow(s, n-1)
		if e != nil {
			return e
		}
		if e = m.signedRemState(a, b, &q.remainder); e != nil {
			return e
		}
		if q.remainder.size == 0 {
			return nil
		}
		if e = m.normalize(m.stateView(&q.remainder)); e != nil {
			return e
		}
		if e = m.sequencePush(s, m.stateView(&q.remainder), true); e != nil {
			return e
		}
	}
}
func (s *UPolynomialSequence) buildSturm(a, b UPolynomialView, kind uint8) (cause error) {
	m := s.manager
	if e := m.sequenceMutationOutput(s); e != nil {
		return e
	}
	q := &s.sturm
	if q.invocation == ^uint64(0) {
		return ErrUPolynomialCleanupPending
	}
	*q = upolySequenceSturmScope{owner: s, invocation: q.invocation + 1, active: true, executing: true}
	if e := q.bindNumericOwners(); e != nil {
		q.executing = false
		q.active = false
		return e
	}
	q.previousCallbackDepth = m.callbackDepth
	m.callbackDepth = 0
	defer func() { m.callbackDepth = q.previousCallbackDepth; cause = errors.Join(cause, q.unbindNumericOwners()) }()
	defer func() {
		cause = q.finishVector(1, cause)
		q.executing = false
		if !q.pending() {
			q.active = false
		}
	}()
	if e := m.sequenceReset(&s.state); e != nil {
		return e
	}
	if kind != 1 {
		q.live[1] = true
	} // declaration follows reset, before initial push
	if e := q.pushCopy(a); e != nil {
		return e
	}
	if kind == 1 {
		if e := q.pushCopy(b); e != nil {
			return e
		}
		return q.core()
	}
	if kind == 3 {
		if a.count == 0 {
			return nil
		}
		for i := uint32(0); i < a.count-1; i++ {
			v, e := m.sequenceRow(&s.state, uint32(len(s.state.sizes))-1)
			if e != nil {
				return e
			}
			if e = m.derivativeState(v, &q.prime); e != nil {
				return e
			}
			if e = m.normalize(m.stateView(&q.prime)); e != nil {
				return e
			}
			if e = m.sequencePush(&s.state, m.stateView(&q.prime), true); e != nil {
				return e
			}
		}
		return nil
	}
	if e := m.derivativeState(a, &q.prime); e != nil {
		return e
	}
	if kind == 2 {
		if e := m.mul(m.stateView(&q.prime), b, &q.prime); e != nil {
			return e
		}
	}
	if e := m.sequencePush(&s.state, m.stateView(&q.prime), true); e != nil {
		return e
	}
	return q.core()
}

// CleanupSturm owns only the persistent source vectors of this public sequence.
// Source cursors and the first destructor stop remain unchanged on continuation.
func (s *UPolynomialSequence) CleanupSturm() (cause error) {
	if e := s.check(); e != nil {
		return e
	}
	q := &s.sturm
	if !q.active {
		return nil
	}
	if !q.valid() {
		return ErrUPolynomialOwner
	}
	if q.executing || q.finishing || s.evaluating || s.displaying {
		return ErrUPolynomialCleanupPending
	}
	q.finishing = true
	defer func() { q.finishing = false }()
	if e := q.bindNumericOwners(); e != nil {
		return e
	}
	defer func() { cause = errors.Join(cause, q.unbindNumericOwners()) }()
	m := s.manager
	if e := m.modular.CleanupPNormalizeInvocation(&q.normalize, true); e != nil {
		return e
	}
	if e := m.config.Integer.CleanupCopyAllocationInvocation(&q.copy); e != nil {
		return e
	}
	for which := uint8(0); which < 2; which++ {
		if !q.live[which] {
			continue
		}
		state := q.vector(which)
		values := m.values(state)
		for q.ownCursor[which] < state.size {
			if e := m.config.Integer.CleanupRawNumeral(&values[q.ownCursor[which]]); e != nil {
				return e
			}
			q.ownCursor[which]++
		}
		state.size = 0
		m.freeBacking(state)
		q.live[which] = false
	}
	q.active = false
	return nil
}

// These bindings protect borrowed managers only during actual execution or
// owning continuation. They confer no ownership over numeric failure roots.
func (q *upolySequenceSturmScope) NumericExecuting() bool {
	return q.valid() && q.active && (q.executing || q.finishing)
}
func (q *upolySequenceSturmScope) bindNumericOwners() error {
	if !q.NumericExecuting() {
		return ErrUPolynomialOwner
	}
	z := q.owner.manager.config.Integer
	if e := z.BindCaller(&q.integerBinding, q, 17); e != nil {
		return e
	}
	if e := z.Naturals().BindCaller(&q.naturalBinding, q, 17); e != nil {
		return errors.Join(e, q.integerBinding.Unbind())
	}
	return nil
}
func (q *upolySequenceSturmScope) unbindNumericOwners() error {
	n := q.naturalBinding.Unbind()
	z := q.integerBinding.Unbind()
	return errors.Join(n, z)
}

func (q *upolySequenceSturmScope) pending() bool {
	m := q.owner.manager
	return q.live[0] || q.live[1] || m.config.Integer.MatchesCopyAllocationInvocation(&q.copy) || m.modular.MatchesPNormalizeInvocation(&q.normalize)
}

// Source seq.push(m(),sz,p): preserve metadata/default construction before
// each SET, and capture only the integer copy actually accepted by that SET.
func (q *upolySequenceSturmScope) pushCopy(v UPolynomialView) error {
	if !q.valid() || !q.executing || q.finishing || q.owner.manager.callbackDepth != 0 || q.owner.manager.config.Integer.RuntimeOwner().FailureTransportActive() || !q.currentNumericOwner() {
		return ErrUPolynomialOwner
	}
	m, s := q.owner.manager, &q.owner.state
	if e := upolyUnsignedPush(&s.begins, s.coefficients.size); e != nil {
		return e
	}
	if e := upolyUnsignedPush(&s.sizes, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		dst, e := m.pushDefault(&s.coefficients)
		if e != nil {
			return e
		}
		src, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.CopyFromOwnerTracked(dst, m.config.Integer, src, &q.copy, &q.normalize); e != nil {
			return e
		}
	}
	return nil
}

func (q *upolySequenceSturmScope) currentNumericOwner() bool {
	if !q.valid() {
		return false
	}
	for b := q.owner.manager.config.Integer.CallerBindings(); b != nil; b = b.Previous() {
		if b.Kind() == 17 {
			return b.Guard() == q
		}
	}
	return false
}
