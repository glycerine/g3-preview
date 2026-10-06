// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Licensed under the MIT license.
package mpnum

func (m *UPolynomialManager) sign(a *Int) (int, error) {
	zero, e := m.modular.IsZero(a)
	if e != nil {
		return 0, e
	}
	if zero {
		return 0, nil
	}
	pos, e := m.modular.IsPos(a)
	if e != nil {
		return 0, e
	}
	if pos {
		return 1, nil
	}
	return -1, nil
}
func (m *UPolynomialManager) signAt(v UPolynomialView, i uint32) (int, error) {
	a, e := m.read(v, i)
	if e != nil {
		return 0, e
	}
	return m.sign(a)
}
func (m *UPolynomialManager) EvalSignInt(v UPolynomialView, b *Int) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	if v.count == 1 {
		return m.signAt(v, 0)
	}
	var r Int
	defer m.modular.Del(&r)
	a, e := m.read(v, v.count-1)
	if e != nil {
		return 0, e
	}
	if e = m.modular.Copy(&r, a); e != nil {
		return 0, e
	}
	for i := v.count - 1; i > 0; {
		i--
		a, e := m.read(v, i)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return 0, e
		}
		if zero {
			e = m.modular.Mul(&r, b, &r)
		} else {
			e = m.modular.AddMul(a, &r, b, &r)
		}
		if e != nil {
			return 0, e
		}
	}
	return m.sign(&r)
}
func (m *UPolynomialManager) EvalSignRational(v UPolynomialView, b *Rational) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	if v.count == 1 {
		return m.signAt(v, 0)
	}
	// Source denominator copy precedes r/ak and the first coefficient access.
	var di Int
	defer m.modular.Del(&di)
	if b == nil {
		return 0, ErrUPolynomialOwner
	}
	if e := m.modular.Copy(&di, &b.den); e != nil {
		return 0, e
	}
	var r, ak Int
	defer m.modular.Del(&r)
	defer m.modular.Del(&ak)
	a, e := m.read(v, v.count-1)
	if e != nil {
		return 0, e
	}
	if e = m.modular.Copy(&r, a); e != nil {
		return 0, e
	}
	for i := v.count - 1; i > 0; {
		i--
		a, e := m.read(v, i)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return 0, e
		}
		if zero {
			e = m.modular.Mul(&r, &b.num, &r)
		} else {
			e = m.modular.Mul(a, &di, &ak)
			if e == nil {
				e = m.modular.AddMul(&ak, &r, &b.num, &r)
			}
		}
		if e != nil {
			return 0, e
		}
		if e = m.modular.Mul(&di, &b.den, &di); e != nil {
			return 0, e
		}
	}
	return m.sign(&r)
}
func (m *UPolynomialManager) EvalSignBinary(v UPolynomialView, b *BinaryRational) (int, error) {
	return m.evalSignBinaryWithNegScope(v, b, nil)
}
func (m *UPolynomialManager) evalSignBinaryWithNegScope(v UPolynomialView, b *BinaryRational, parent *algebraicNegScope) (value int, cause error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	if v.count == 1 {
		return m.signAt(v, 0)
	}
	if b == nil {
		return 0, ErrUPolynomialOwner
	}
	k := b.k
	ki := k
	var values *[2]Int
	context := integerAllocationInvocationContext{}
	var child *integerAddMulInvocationReceipt
	if parent == nil {
		q, e := m.beginBinaryEvaluation()
		if e != nil {
			return 0, e
		}
		q.target = v
		values = &q.values
		context = q.context()
		child = &q.addMul
		z := m.config.Integer
		old := z.binaryEvaluationCaller
		z.binaryEvaluationCaller = m
		defer func() { q.executing = false; cause = q.finish(cause); z.binaryEvaluationCaller = old }()
	} else {
		if parent.context(false).algebraicNegFrame(m.config.Integer) == nil || parent.owner.config.Polynomials != m {
			return 0, ErrAlgebraicOwner
		}
		if parent.eval.active {
			return 0, ErrAlgebraicPending
		}
		evalInvocation := parent.eval.invocation + 1
		if evalInvocation == 0 {
			return 0, ErrAlgebraicPending
		}
		parent.eval = algebraicNegBinaryScope{invocation: evalInvocation, constructed: 2, sourceRemaining: 2, active: true, executing: true}
		values = &parent.eval.values
		context = parent.context(true)
		child = &parent.eval.addMul
		defer func() {
			q := &parent.eval
			q.executing = false
			q.bodyReturned = true
			if q.bodyCause == nil {
				q.bodyCause = cause
			}
			cause = parent.finishBinarySource(cause)
		}()
	}
	r, ak := &values[0], &values[1]
	a, e := m.read(v, v.count-1)
	if e != nil {
		return 0, e
	}
	if e = m.modular.copyWithSourceContext(r, a, context); e != nil {
		return 0, e
	}
	for i := v.count - 1; i > 0; {
		i--
		a, e := m.read(v, i)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return 0, e
		}
		if zero {
			e = m.modular.mulWithSourceContext(r, &b.num, r, context)
		} else {
			e = m.modular.mul2kWithSourceContext(a, ki, ak, context)
			if e == nil {
				e = m.modular.addMulWithSourceContext(ak, r, &b.num, r, child, context)
			}
		}
		if e != nil {
			return 0, e
		}
		ki += k
	}
	return m.sign(r)
}
func (m *UPolynomialManager) EvalSignZero(v UPolynomialView) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	return m.signAt(v, 0)
}
func (m *UPolynomialManager) EvalSignPlusInfinity(v UPolynomialView) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	return m.signAt(v, v.count-1)
}
func (m *UPolynomialManager) EvalSignMinusInfinity(v UPolynomialView) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	sign, e := m.signAt(v, v.count-1)
	if (v.count-1)%2 != 0 {
		sign = -sign
	}
	return sign, e
}
