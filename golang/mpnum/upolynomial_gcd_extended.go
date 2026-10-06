// Source: src/math/polynomial/upolynomial.cpp:1045-1105; src/math/polynomial/upolynomial.h:361-364.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *UPolynomialManager) dropLocalVector(s *upolyVectorState) { m.reset(s); m.freeBacking(s) }
func (m *UPolynomialManager) extendedGCDDiagnostic(a, b UPolynomialView, u, v, d, au *upolyVectorState) error {
	var bv, expected upolyVectorState
	defer m.dropLocalVector(&bv)
	defer m.dropLocalVector(&expected)
	if e := m.mul(b, m.stateView(v), &bv); e != nil {
		return e
	}
	if e := m.addSubState(m.stateView(au), m.stateView(&bv), &expected, false); e != nil {
		return e
	}
	ok, e := m.EqualViews(m.stateView(&expected), m.stateView(d))
	if e != nil {
		return e
	}
	if !ok {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (m *UPolynomialManager) extendedGCD(a, b UPolynomialView, u, v, d *upolyVectorState) error {
	return m.extendedGCDWithCaller(a, b, u, v, d, nil)
}
func (m *UPolynomialManager) extendedGCDWithCaller(a, b UPolynomialView, u, v, d *upolyVectorState, caller *upolyHenselCallerScope) (failure error) {
	context := integerAllocationInvocationContext{}
	if caller != nil {
		context = m.modular.henselCallerAllocationContext(caller)
		if context.henselCallerFrame(m.config.Integer) == nil {
			return ErrUPolynomialOwner
		}
	}
	if modularSourceDebug {
		if m.alias(a, u) || m.alias(a, v) || m.alias(a, d) || m.alias(b, u) || m.alias(b, v) || m.alias(b, d) {
			return ErrUPolynomialAssertion
		}
		field, e := m.modular.Field()
		if e != nil {
			return e
		}
		if !field {
			return ErrUPolynomialAssertion
		}
	}
	var locals [6]upolyVectorState
	v1, v3, q, r, t, v1q := &locals[0], &locals[1], &locals[2], &locals[3], &locals[4], &locals[5]
	if caller == nil {
		defer m.dropLocalVector(v1)
		defer m.dropLocalVector(v3)
		defer m.dropLocalVector(q)
		defer m.dropLocalVector(r)
		defer m.dropLocalVector(t)
		defer m.dropLocalVector(v1q)
	} else {
		caller.enterCall()
		defer caller.leaveCall()
		if caller.extendedActive {
			return ErrUPolynomialCleanupPending
		}
		// A new reached invocation may replace only a completed source scope.
		caller.extended = [6]upolyVectorState{}
		caller.extendedDebug = [2]upolyVectorState{}
		caller.extendedScoped = [2]Int{}
		caller.monicRaw = [2]Int{}
		caller.extendedDebugConstructed = 0
		caller.extendedDebugSource = 0
		caller.extendedDebugOwn = 0
		caller.extendedScopedConstructed = 0
		caller.extendedScopedSource = 0
		caller.extendedScopedOwn = 0
		caller.extendedRawConstructed = false
		caller.extendedRawEndEntered = false
		caller.extendedRawEnded = false
		caller.extendedRawSource = 0
		caller.extendedRawOwn = 0
		caller.extendedActive = true
		caller.extendedConstructed = 6
		caller.extendedSource = 6
		caller.extendedOwn = 6
		v1, v3, q, r, t, v1q = &caller.extended[0], &caller.extended[1], &caller.extended[2], &caller.extended[3], &caller.extended[4], &caller.extended[5]
		defer func() {
			failure = joinHenselCaller(failure, caller.finishExtendedSource())
			if failure == nil {
				caller.extendedActive = false
				// Every scoped field and the reached RAW end sequence actually completed.
				caller.extendedOwn = 0
				caller.extendedDebugOwn = 0
				caller.extendedScopedOwn = 0
				caller.extendedRawOwn = 2
			}
		}()
	}
	if e := m.reset(u); e != nil {
		return e
	}
	one, e := m.pushDefault(u)
	if e != nil {
		return e
	}
	if e = m.modular.setInt64WithSourceContext(one, 1, context); e != nil {
		return e
	}
	if e = m.setViewWithSourceContext(d, a, context); e != nil {
		return e
	}
	if e = m.monicLocalWithCaller(m.stateView(d).WithCount(a.count), caller, context); e != nil {
		return e
	}
	if e = m.reset(v1); e != nil {
		return e
	}
	if e = m.setViewWithSourceContext(v3, b, context); e != nil {
		return e
	}
	for {
		if v3.size == 0 {
			au, minus := v1, v3
			if e = m.mulWithSourceContext(a, m.stateView(u), au, henselNumericAddMulChild(nil, caller), context); e != nil {
				return e
			}
			if e = m.addSubStateWithSourceContext(m.stateView(d), m.stateView(au), minus, true, context); e != nil {
				return e
			}
			if e = m.divStateWithCaller(m.stateView(minus), b, v, caller, context); e != nil {
				return e
			}
			if modularSourceDebug {
				if e = m.extendedGCDDiagnosticWithCaller(a, b, u, v, d, au, caller, context); e != nil {
					return e
				}
			}
			// Declaration order lc_inv then lc => destruct lc before lc_inv.
			return m.extendedGCDFinishWithCaller(u, v, d, caller, context)
		}
		var power uint32
		if e = m.divRemCoreSpanWithSourceContext(m.stateView(d), &upolyDivisorSpan{view: m.stateView(v3)}, &power, &m.div2, &m.div1, nil, caller, context); e != nil {
			return e
		}
		*r, m.div1 = m.div1, *r
		*q, m.div2 = m.div2, *q
		if e = m.mulWithSourceContext(m.stateView(v1), m.stateView(q), v1q, henselNumericAddMulChild(nil, caller), context); e != nil {
			return e
		}
		if e = m.addSubStateWithSourceContext(m.stateView(u), m.stateView(v1q), t, true, context); e != nil {
			return e
		}
		*u, *v1 = *v1, *u
		*d, *v3 = *v3, *d
		*v1, *t = *t, *v1
		*v3, *r = *r, *v3
	}
}
func (m *UPolynomialManager) extendedGCDFinish(u, v, d *upolyVectorState) error {
	var inv, lc Int
	defer m.modular.Del(&inv)
	defer m.modular.Del(&lc)
	if e := m.monic(m.stateView(d), &lc, &inv); e != nil {
		return e
	}
	if e := m.mulScalarState(u, &inv); e != nil {
		return e
	}
	return m.mulScalarState(v, &inv)
}
func (m *UPolynomialManager) ExtendedGCDViews(a, b UPolynomialView, u, v, d UPolynomial) error {
	U, e := m.descriptor(u)
	if e != nil {
		return e
	}
	V, e := m.descriptor(v)
	if e != nil {
		return e
	}
	D, e := m.descriptor(d)
	if e != nil {
		return e
	}
	return m.extendedGCD(a, b, &U.state, &V.state, &D.state)
}
func (m *UPolynomialManager) ExtendedGCD(a, b, u, v, d UPolynomial) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.ExtendedGCDViews(av, bv, u, v, d)
}
func (m *UPolynomialManager) squareFree(v UPolynomialView, out *upolyVectorState) error {
	if modularSourceDebug && m.alias(v, out) {
		return ErrUPolynomialAssertion
	}
	if v.count <= 1 {
		return m.setView(out, v)
	}
	derivative, g := &m.sqf1, &m.sqf2
	if e := m.derivativeState(v, derivative); e != nil {
		return e
	}
	if e := m.gcd(v, m.stateView(derivative), g); e != nil {
		return e
	}
	if g.size <= 1 {
		return m.setView(out, v)
	}
	if e := m.divState(v, m.stateView(g), out); e != nil {
		return e
	}
	return m.normalize(m.stateView(out))
}
func (m *UPolynomialManager) SquareFreeView(v UPolynomialView, out UPolynomial) error {
	o, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.squareFree(v, &o.state)
}
func (m *UPolynomialManager) SquareFree(p, out UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.SquareFreeView(v, out)
}
func (m *UPolynomialManager) IsSquareFreeView(v UPolynomialView) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if v.count <= 1 {
		return true, nil
	}
	derivative, g := &m.sqf1, &m.sqf2
	if e := m.derivativeState(v, derivative); e != nil {
		return false, e
	}
	if e := m.gcd(v, m.stateView(derivative), g); e != nil {
		return false, e
	}
	return g.size <= 1, nil
}
func (m *UPolynomialManager) IsSquareFree(p UPolynomial) (bool, error) {
	v, e := m.wholeView(p)
	if e != nil {
		return false, e
	}
	return m.IsSquareFreeView(v)
}
