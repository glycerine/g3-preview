package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func (m *UPolynomialManager) sequenceOutput(s *UPolynomialSequence) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := s.check(); e != nil {
		return e
	}
	if s.manager != m {
		return ErrUPolynomialOwner
	}
	return nil
}
func (m *UPolynomialManager) sturmSequenceCore(s *upolySequenceState) error {
	var r upolyVectorState
	defer m.dropLocalVector(&r)
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
		if e = m.signedRemState(a, b, &r); e != nil {
			return e
		}
		if r.size == 0 {
			return nil
		}
		if e = m.normalize(m.stateView(&r)); e != nil {
			return e
		}
		if e = m.sequencePush(s, m.stateView(&r), true); e != nil {
			return e
		}
	}
}
func (m *UPolynomialManager) sturmUnary(p UPolynomialView, s *upolySequenceState) error {
	if e := m.sequenceReset(s); e != nil {
		return e
	}
	var prime upolyVectorState
	defer m.dropLocalVector(&prime)
	if e := m.sequencePush(s, p, false); e != nil {
		return e
	}
	if e := m.derivativeState(p, &prime); e != nil {
		return e
	}
	if e := m.sequencePush(s, m.stateView(&prime), true); e != nil {
		return e
	}
	return m.sturmSequenceCore(s)
}
func (m *UPolynomialManager) SturmSequence(p UPolynomialView, s *UPolynomialSequence) error {
	if e := m.sequenceOutput(s); e != nil {
		return e
	}
	return m.sturmUnary(p, &s.state)
}
func (m *UPolynomialManager) SturmSequencePair(a, b UPolynomialView, s *UPolynomialSequence) error {
	if e := m.sequenceOutput(s); e != nil {
		return e
	}
	if e := m.sequenceReset(&s.state); e != nil {
		return e
	}
	if e := m.sequencePush(&s.state, a, false); e != nil {
		return e
	}
	if e := m.sequencePush(&s.state, b, false); e != nil {
		return e
	}
	return m.sturmSequenceCore(&s.state)
}
func (m *UPolynomialManager) SturmTarskiSequence(a, b UPolynomialView, s *UPolynomialSequence) error {
	if e := m.sequenceOutput(s); e != nil {
		return e
	}
	if e := m.sequenceReset(&s.state); e != nil {
		return e
	}
	var product upolyVectorState
	defer m.dropLocalVector(&product)
	if e := m.sequencePush(&s.state, a, false); e != nil {
		return e
	}
	if e := m.derivativeState(a, &product); e != nil {
		return e
	}
	if e := m.mul(m.stateView(&product), b, &product); e != nil {
		return e
	}
	if e := m.sequencePush(&s.state, m.stateView(&product), true); e != nil {
		return e
	}
	return m.sturmSequenceCore(&s.state)
}
func (m *UPolynomialManager) FourierSequence(p UPolynomialView, s *UPolynomialSequence) error {
	if e := m.sequenceOutput(s); e != nil {
		return e
	}
	if e := m.sequenceReset(&s.state); e != nil {
		return e
	}
	var prime upolyVectorState
	defer m.dropLocalVector(&prime)
	if e := m.sequencePush(&s.state, p, false); e != nil {
		return e
	}
	if p.count == 0 {
		return nil
	}
	for i := uint32(0); i < p.count-1; i++ {
		v, e := m.sequenceRow(&s.state, uint32(len(s.state.sizes))-1)
		if e != nil {
			return e
		}
		if e = m.derivativeState(v, &prime); e != nil {
			return e
		}
		if e = m.normalize(m.stateView(&prime)); e != nil {
			return e
		}
		if e = m.sequencePush(&s.state, m.stateView(&prime), true); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) sequenceVariations(s *upolySequenceState, location uint8, b *unumeral.BinaryRational) (uint32, error) {
	return m.sequenceVariationsWithReceipt(s, location, b, nil)
}
func (m *UPolynomialManager) sequenceVariationsWithReceipt(s *upolySequenceState, location uint8, b *unumeral.BinaryRational, receipt *upolyBinaryEvaluationInvocationReceipt) (uint32, error) {
	if uint32(len(s.sizes)) <= 1 {
		return 0, nil
	}
	var result uint32
	previous := 0
	for i := uint32(0); i < uint32(len(s.sizes)); i++ {
		// Source reads size(i) before coeffs(i). Preserve that assertion/read order.
		n, e := upolyUnsignedRead(s.sizes, i)
		if e != nil {
			return 0, e
		}
		begin, e := upolyUnsignedRead(s.begins, i)
		if e != nil {
			return 0, e
		}
		v := m.stateView(&s.coefficients)
		v.start = begin
		v.count = n
		var sign int
		switch location {
		case 0:
			sign, e = m.EvalSignPlusInfinity(v)
		case 1:
			sign, e = m.EvalSignMinusInfinity(v)
		case 2:
			sign, e = m.EvalSignZero(v)
		case 3:
			sign, e = m.evalSignBinaryWithNegScopeAndReceipt(v, b, nil, receipt)
		}
		if e != nil {
			return 0, e
		}
		if sign == 0 {
			continue
		}
		if polynomialSourceDebug && sign != 1 && sign != -1 {
			return 0, ErrUPolynomialAssertion
		}
		if sign != previous && previous != 0 {
			result++
		}
		previous = sign
	}
	return result, nil
}
func (m *UPolynomialManager) SignVariationsMinusInfinity(s *UPolynomialSequence) (uint32, error) {
	if e := m.sequenceOutput(s); e != nil {
		return 0, e
	}
	return m.sequenceVariations(&s.state, 1, nil)
}
func (m *UPolynomialManager) SignVariationsPlusInfinity(s *UPolynomialSequence) (uint32, error) {
	if e := m.sequenceOutput(s); e != nil {
		return 0, e
	}
	return m.sequenceVariations(&s.state, 0, nil)
}
func (m *UPolynomialManager) SignVariationsZero(s *UPolynomialSequence) (uint32, error) {
	if e := m.sequenceOutput(s); e != nil {
		return 0, e
	}
	return m.sequenceVariations(&s.state, 2, nil)
}
func (m *UPolynomialManager) SignVariationsBinary(s *UPolynomialSequence, b *unumeral.BinaryRational) (uint32, error) {
	if e := m.sequenceOutput(s); e != nil {
		return 0, e
	}
	return m.sequenceVariations(&s.state, 3, b)
}
func (m *UPolynomialManager) GetRootID(p UPolynomialView, b *unumeral.BinaryRational) (uint32, error) {
	return m.getRootIDWithReceipt(p, b, nil)
}
func (m *UPolynomialManager) getRootIDWithReceipt(p UPolynomialView, b *unumeral.BinaryRational, receipt *upolyRootIDInvocationReceipt) (value uint32, cause error) {
	q, e := m.beginRootID(p, receipt)
	if e != nil {
		return 0, e
	}
	defer func() { cause = q.finish(cause) }()
	if e = q.sturm(p); e != nil {
		return 0, e
	}
	return q.evaluate(b)
}
func (m *UPolynomialManager) beginRootID(p UPolynomialView, receipt *upolyRootIDInvocationReceipt) (*upolyRootIDScope, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	q := &m.rootID
	if q.active || q.invocation == ^uint64(0) {
		return nil, ErrUPolynomialCleanupPending
	}
	*q = upolyRootIDScope{owner: m, invocation: q.invocation + 1, active: true, executing: true, target: p}
	q.live[2] = true
	if receipt != nil {
		*receipt = upolyRootIDInvocationReceipt{m, q, q.invocation}
	}
	return q, nil
}
func (q *upolyRootIDScope) finish(cause error) error {
	defer func() { q.executing = false }()
	return q.finishVector(2, cause)
}
func (q *upolyRootIDScope) evaluate(b *unumeral.BinaryRational) (uint32, error) {
	m := q.owner
	s0, e := m.sequenceVariations(&q.sequence, 1, nil)
	if e != nil {
		return 0, e
	}
	s1, e := m.sequenceVariationsWithReceipt(&q.sequence, 3, b, &q.evaluation)
	if m.matchesBinaryEvaluationReceipt(&q.evaluation) && q.childStop == nil {
		q.childStop = m.binaryEvaluation.sourceCause
	}
	if e != nil {
		return 0, e
	}
	if e = refinementSourceAssert(s0 >= s1, 2956, "s0 >= s1"); e != nil {
		return 0, e
	}
	return s0 - s1, nil
}
