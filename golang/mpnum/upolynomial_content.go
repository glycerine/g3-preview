// Licensed under the MIT license.
package mpnum

// gcdSpan translates raw IntegerManager.GCDMany's source traversal at actual
// array reads. Only the final result receives mpzzp's normalized assertion;
// per-pair modular GCD would introduce different checks/actions.
func (m *UPolynomialManager) gcdSpan(v UPolynomialView, out *Int) error {
	if e := m.modular.check(); e != nil {
		return e
	}
	z := m.modular.manager
	if e := z.check(out); e != nil {
		return e
	}
	switch v.count {
	case 0:
		if e := z.Reset(out); e != nil {
			return e
		}
	case 1:
		first, e := m.read(v, 0)
		if e != nil {
			return e
		}
		if e = z.Copy(out, first); e != nil {
			return e
		}
		if e = z.Abs(out); e != nil {
			return e
		}
	default:
		first, e := m.read(v, 0)
		if e != nil {
			return e
		}
		second, e := m.read(v, 1)
		if e != nil {
			return e
		}
		if e = z.GCD(first, second, out); e != nil {
			return e
		}
		for i := uint32(2); i < v.count; i++ {
			if isOneInteger(out) {
				break
			}
			next, e := m.read(v, i)
			if e != nil {
				return e
			}
			if e = z.GCD(out, next, out); e != nil {
				return e
			}
		}
	}
	return m.modular.assertNormalized(out)
}
func (m *UPolynomialManager) primitiveState(v UPolynomialView, pp *upolyVectorState, cont *Int) error {
	if modularSourceDebug && v.count == 0 {
		return ErrUPolynomialAssertion
	}
	if e := m.gcdSpan(v, cont); e != nil {
		return e
	}
	if modularSourceDebug {
		positive, e := m.modular.IsPos(cont)
		if e != nil {
			return e
		}
		if !positive {
			return ErrUPolynomialAssertion
		}
	}
	one, e := m.modular.IsOne(cont)
	if e != nil {
		return e
	}
	if one {
		return m.setView(pp, v)
	}
	if e = m.reserve(pp, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(x)
		if e != nil {
			return e
		}
		dst := &m.values(pp)[i]
		if zero {
			e = m.modular.SetInt32(dst, 0)
		} else {
			e = m.modular.Div(x, cont, dst)
		}
		if e != nil {
			return e
		}
	}
	return m.setSize(pp, v.count)
}
func (m *UPolynomialManager) PrimitiveAndContentViews(v UPolynomialView, pp UPolynomial, cont *Int) error {
	out, e := m.descriptor(pp)
	if e != nil {
		return e
	}
	return m.primitiveState(v, &out.state, cont)
}
func (m *UPolynomialManager) PrimitiveAndContent(p, pp UPolynomial, cont *Int) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.PrimitiveAndContentViews(v, pp, cont)
}
func (m *UPolynomialManager) primitive(v UPolynomialView, pp *upolyVectorState) error {
	var cont Int
	defer m.modular.Del(&cont)
	return m.primitiveState(v, pp, &cont)
}
func (m *UPolynomialManager) PrimitiveView(v UPolynomialView, pp UPolynomial) error {
	out, e := m.descriptor(pp)
	if e != nil {
		return e
	}
	return m.primitive(v, &out.state)
}
func (m *UPolynomialManager) Primitive(p, pp UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.PrimitiveView(v, pp)
}
func (m *UPolynomialManager) normalize(v UPolynomialView) error {
	if v.count == 0 {
		return nil
	}
	if v.count == 1 {
		x, e := m.read(v, 0)
		if e != nil {
			return e
		}
		positive, e := m.modular.IsPos(x)
		if e != nil {
			return e
		}
		if positive {
			return m.modular.SetInt32(x, 1)
		}
		return m.modular.SetInt32(x, -1)
	}
	var g Int
	defer m.modular.Del(&g)
	if e := m.gcdSpan(v, &g); e != nil {
		return e
	}
	one, e := m.modular.IsOne(&g)
	if e != nil || one {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		if e = m.normalizeCoefficient(v, i, &g); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) normalizeCoefficient(v UPolynomialView, i uint32, g *Int) error {
	var old, tmp Int
	// Source Z3DEBUG lexical temporaries are destroyed tmp then old.
	if modularSourceDebug {
		defer m.modular.Del(&old)
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&old, x); e != nil {
			return e
		}
	}
	x, e := m.read(v, i)
	if e != nil {
		return e
	}
	if e = m.modular.Div(x, g, x); e != nil {
		return e
	}
	if modularSourceDebug {
		defer m.modular.Del(&tmp)
		if e = m.modular.Mul(g, x, &tmp); e != nil {
			return e
		}
		ok, e := m.modular.Equal(&tmp, &old)
		if e != nil {
			return e
		}
		if !ok {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (m *UPolynomialManager) NormalizeView(v UPolynomialView) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.normalize(v)
}
func (m *UPolynomialManager) Normalize(p UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.NormalizeView(v)
}

// WithCount is the explicit source (pointer,unsigned size) boundary: it
// changes only the scalar count. It neither borrows/dereferences an owner nor
// prevalidates physical extent. Algorithms guard at each actual source read;
// an early return may therefore skip an unavailable tail exactly as C++ does.
func (v UPolynomialView) WithCount(count uint32) UPolynomialView {
	v.count = count
	return v
}
