// Licensed under the MIT license.
package mpnum

// modularImage implements the lexical scoped_set_zp body. The old modulus
// copy goes through mpzzp Copy, including source normalization. In modular
// GCD the entering mode is always Z. Cleanup on Go failure is explicit safe
// ownership adaptation; a restoration failure stays visible.
func (m *UPolynomialManager) modularImage(u, v, q *upolyVectorState, p, lc *Int, du, dv uint32) (bad bool, err error) {
	oldMode, e := m.modular.Modular()
	if e != nil {
		return false, e
	}
	var old Int
	defer m.modular.Del(&old)
	if e = m.modular.Copy(&old, &m.modular.p); e != nil {
		return false, e
	}
	defer func() {
		var restore error
		if oldMode {
			restore = m.modular.SetZP(&old)
		} else {
			restore = m.modular.SetZ()
		}
		// Return the actual typed restoration failure if cleanup cannot restore
		// mode; otherwise retain the body error directly. Error wrappers would
		// allocate an uncharged intermediary on every canceled image.
		if restore != nil {
			err = restore
		}
	}()
	if e = m.modular.SetZP(p); e != nil {
		return false, e
	}
	U, V := &m.mgcd[2], &m.mgcd[3]
	if e = m.setView(U, m.stateView(u)); e != nil {
		return false, e
	}
	if e = m.setView(V, m.stateView(v)); e != nil {
		return false, e
	}
	ud, vd := uint32(0), uint32(0)
	if U.size != 0 {
		ud = U.size - 1
	}
	if V.size != 0 {
		vd = V.size - 1
	}
	if ud < du {
		return true, nil
	}
	if vd < dv {
		return true, nil
	}
	if e = m.euclidGCD(m.stateView(U), m.stateView(V), q); e != nil {
		return false, e
	}
	if e = m.monicLocal(m.stateView(q)); e != nil {
		return false, e
	}
	var c Int
	defer m.modular.Del(&c)
	if e = m.modular.Copy(&c, lc); e != nil {
		return false, e
	}
	return false, m.mulScalarState(q, &c)
}
func (m *UPolynomialManager) combineImages(c1 *upolyVectorState, b1 *Int, c2 *upolyVectorState, b2 *Int) error {
	if modularSourceDebug {
		even, e := m.config.Integer.IsEven(b1)
		if e != nil {
			return e
		}
		if even {
			return ErrUPolynomialAssertion
		}
		even, e = m.config.Integer.IsEven(b2)
		if e != nil {
			return e
		}
		if even {
			return ErrUPolynomialAssertion
		}
	}
	R := &m.cra
	if e := m.reset(R); e != nil {
		return e
	}
	var inv1, inv2, g, a1, a2, bound, lower, upper, newA, tmp1, tmp2, tmp3 Int
	defer func() {
		for _, x := range []*Int{&tmp3, &tmp2, &tmp1, &newA, &upper, &lower, &bound, &a2, &a1, &g, &inv2, &inv1} {
			m.modular.Del(x)
		}
	}()
	if e := m.modular.ExtendedGCD(b1, b2, &inv1, &inv2, &g); e != nil {
		return e
	}
	if modularSourceDebug {
		one, e := m.modular.IsOne(&g)
		if e != nil {
			return e
		}
		if !one {
			return ErrUPolynomialAssertion
		}
	}
	z := m.config.Integer
	if e := z.Mod(&inv1, b2, &inv1); e != nil {
		return e
	}
	if e := z.Mod(&inv2, b1, &inv2); e != nil {
		return e
	}
	if e := m.modular.Mul(b2, &inv2, &a1); e != nil {
		return e
	}
	if e := m.modular.Mul(b1, &inv1, &a2); e != nil {
		return e
	}
	if e := m.modular.Mul(b1, b2, &bound); e != nil {
		return e
	}
	two := NewInt(2)
	if e := m.modular.Div(&bound, &two, &upper); e != nil {
		return e
	}
	if e := m.modular.Copy(&lower, &upper); e != nil {
		return e
	}
	if e := m.modular.Neg(&lower); e != nil {
		return e
	}
	zero := NewInt(0)
	n1, n2 := c1.size, c2.size
	n := n1
	if n2 < n {
		n = n2
	}
	add := func(x, y *Int) error {
		if e := m.modular.Mul(x, &a1, &tmp1); e != nil {
			return e
		}
		if e := m.modular.Mul(y, &a2, &tmp2); e != nil {
			return e
		}
		if e := m.modular.Add(&tmp1, &tmp2, &tmp3); e != nil {
			return e
		}
		if e := z.Mod(&tmp3, &bound, &newA); e != nil {
			return e
		}
		greater, e := m.modular.Greater(&newA, &upper)
		if e != nil {
			return e
		}
		if greater {
			if e = m.modular.Sub(&newA, &bound, &newA); e != nil {
				return e
			}
		}
		dst, e := m.pushDefault(R)
		if e != nil {
			return e
		}
		return m.modular.Copy(dst, &newA)
	}
	i := uint32(0)
	for ; i < n; i++ {
		if e := add(&m.values(c1)[i], &m.values(c2)[i]); e != nil {
			return e
		}
	}
	for ; i < n1; i++ {
		if e := add(&m.values(c1)[i], &zero); e != nil {
			return e
		}
	}
	for ; i < n2; i++ {
		if e := add(&zero, &m.values(c2)[i]); e != nil {
			return e
		}
	}
	if e := m.modular.Copy(b2, &bound); e != nil {
		return e
	}
	*R, *c2 = *c2, *R
	return nil
}
func (m *UPolynomialManager) modularGCD(a, b UPolynomialView, out *upolyVectorState) error {
	if modularSourceDebug {
		if a.count == 0 || b.count == 0 {
			return ErrUPolynomialAssertion
		}
		finite, e := m.modular.Modular()
		if e != nil {
			return e
		}
		if finite {
			return ErrUPolynomialAssertion
		}
	}
	var cu, cv, cg, lcg, p, bound Int
	defer func() {
		m.modular.Del(&bound)
		m.modular.Del(&p)
		m.modular.Del(&lcg)
		m.modular.Del(&cg)
		m.modular.Del(&cv)
		m.modular.Del(&cu)
	}()
	u, v, q, C := &m.mgcd[0], &m.mgcd[1], &m.mgcd[4], &m.mgcd[5]
	if e := m.primitiveState(a, u, &cu); e != nil {
		return e
	}
	if e := m.primitiveState(b, v, &cv); e != nil {
		return e
	}
	if e := m.modular.GCD(&cu, &cv, &cg); e != nil {
		return e
	}
	du, dv := a.count-1, b.count-1
	lu, e := m.read(m.stateView(u), du)
	if e != nil {
		return e
	}
	lv, e := m.read(m.stateView(v), dv)
	if e != nil {
		return e
	}
	if e = m.modular.GCD(lu, lv, &lcg); e != nil {
		return e
	}
	for i, prime := range upolyBigPrimes {
		if e = m.modular.SetUint32(&p, prime); e != nil {
			return e
		}
		bad, e := m.modularImage(u, v, q, &p, &lcg, du, dv)
		if e != nil {
			return e
		}
		if bad {
			continue
		}
		if e = m.trim(q); e != nil {
			return e
		}
		if q.size <= 1 {
			if e = m.reset(out); e != nil {
				return e
			}
			dst, e := m.pushDefault(out)
			if e != nil {
				return e
			}
			return m.modular.Copy(dst, &cg)
		}
		discard := i == 0
		if !discard {
			discard = q.size < C.size
			if !discard {
				discard, e = m.config.Integer.IsEven(&p)
				if e != nil {
					return e
				}
			}
			if !discard {
				discard, e = m.config.Integer.IsEven(&bound)
				if e != nil {
					return e
				}
			}
		}
		if discard {
			if e = m.setView(C, m.stateView(q)); e != nil {
				return e
			}
			if e = m.modular.Copy(&bound, &p); e != nil {
				return e
			}
		} else {
			if e = m.combineImages(q, &p, C, &bound); e != nil {
				return e
			}
		}
		if e = m.primitive(m.stateView(C), q); e != nil {
			return e
		}
		if modularSourceDebug && q.size == 0 {
			return ErrUPolynomialAssertion
		}
		last, e := m.read(m.stateView(q), q.size-1)
		if e != nil {
			return e
		}
		ok, e := m.modular.Divides(last, &lcg)
		if e != nil {
			return e
		}
		if ok {
			ok, e = m.dividesViews(m.stateView(u), m.stateView(q))
			if e != nil {
				return e
			}
		}
		if ok {
			ok, e = m.dividesViews(m.stateView(v), m.stateView(q))
			if e != nil {
				return e
			}
		}
		if ok {
			if e = m.mulScalarState(q, &cg); e != nil {
				return e
			}
			if e = m.flipSign(q); e != nil {
				return e
			}
			*q, *out = *out, *q
			return nil
		}
	}
	return m.euclidGCD(a, b, out)
}
func (m *UPolynomialManager) gcd(a, b UPolynomialView, out *upolyVectorState) error {
	done, e := m.gcdEmpty(a, b, out)
	if done {
		return e
	}
	finite, e := m.modular.Modular()
	if e != nil {
		return e
	}
	if !finite {
		return m.modularGCD(a, b, out)
	}
	return m.euclidGCD(a, b, out)
}
func (m *UPolynomialManager) GCDViews(a, b UPolynomialView, g UPolynomial) error {
	out, e := m.descriptor(g)
	if e != nil {
		return e
	}
	return m.gcd(a, b, &out.state)
}
func (m *UPolynomialManager) GCD(a, b, g UPolynomial) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.GCDViews(av, bv, g)
}
