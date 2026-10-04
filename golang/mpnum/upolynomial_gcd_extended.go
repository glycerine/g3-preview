// Licensed under the MIT license.
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
	var v1, v3, q, r, t, v1q upolyVectorState
	defer m.dropLocalVector(&v1)
	defer m.dropLocalVector(&v3)
	defer m.dropLocalVector(&q)
	defer m.dropLocalVector(&r)
	defer m.dropLocalVector(&t)
	defer m.dropLocalVector(&v1q)
	if e := m.reset(u); e != nil {
		return e
	}
	one, e := m.pushDefault(u)
	if e != nil {
		return e
	}
	if e = m.modular.SetInt32(one, 1); e != nil {
		return e
	}
	if e = m.setView(d, a); e != nil {
		return e
	}
	if e = m.monicLocal(m.stateView(d).WithCount(a.count)); e != nil {
		return e
	}
	if e = m.reset(&v1); e != nil {
		return e
	}
	if e = m.setView(&v3, b); e != nil {
		return e
	}
	for {
		if v3.size == 0 {
			au, minus := &v1, &v3
			if e = m.mul(a, m.stateView(u), au); e != nil {
				return e
			}
			if e = m.addSubState(m.stateView(d), m.stateView(au), minus, true); e != nil {
				return e
			}
			if e = m.divState(m.stateView(minus), b, v); e != nil {
				return e
			}
			if modularSourceDebug {
				if e = m.extendedGCDDiagnostic(a, b, u, v, d, au); e != nil {
					return e
				}
			}
			// Declaration order lc_inv then lc => destruct lc before lc_inv.
			return m.extendedGCDFinish(u, v, d)
		}
		var power uint32
		if e = m.divRemCore(m.stateView(d), m.stateView(&v3), &power, &m.div2, &m.div1); e != nil {
			return e
		}
		r, m.div1 = m.div1, r
		q, m.div2 = m.div2, q
		if e = m.mul(m.stateView(&v1), m.stateView(&q), &v1q); e != nil {
			return e
		}
		if e = m.addSubState(m.stateView(u), m.stateView(&v1q), &t, true); e != nil {
			return e
		}
		*u, v1 = v1, *u
		*d, v3 = v3, *d
		v1, t = t, v1
		v3, r = r, v3
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
