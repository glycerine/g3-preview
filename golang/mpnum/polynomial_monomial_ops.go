// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// A source raw array or actual monomial pointer is a borrowed, ephemeral input;
// this descriptor is never persisted in hot storage or allocated as an owner.
type monomialInput struct {
	manager *MonomialManager
	value   Monomial
	powers  []MonomialPower
	count   uint32
}

func (in monomialInput) read(i uint32) (MonomialPower, error) {
	if in.manager != nil {
		return in.manager.powerCore(in.value, i)
	}
	if uint64(i) >= uint64(len(in.powers)) {
		return MonomialPower{}, arena.ErrBounds
	}
	return in.powers[i], nil
}
func (in monomialInput) variable(i uint32) (uint32, error) {
	if in.manager != nil {
		return in.manager.powerFieldCore(in.value, i, 0)
	}
	if uint64(i) >= uint64(len(in.powers)) {
		return 0, arena.ErrBounds
	}
	return in.powers[i].Var, nil
}
func (in monomialInput) valid() (bool, error) {
	for i := uint32(1); i < in.count; i++ {
		previous, e := in.variable(i - 1)
		if e != nil {
			return false, e
		}
		current, e := in.variable(i)
		if e != nil {
			return false, e
		}
		if previous >= current {
			return false, nil
		}
	}
	return true, nil
}
func (in monomialInput) degreeSum() (uint32, error) {
	if in.manager != nil {
		return in.manager.degreeSumCore(in.value, in.count)
	}
	var n uint32
	for i := uint32(0); i < in.count; i++ {
		if uint64(i) >= uint64(len(in.powers)) {
			return 0, arena.ErrBounds
		}
		n += in.powers[i].Degree
	}
	return n, nil
}
func (m *MonomialManager) Mul(a, b Monomial) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if a == m.unit {
		return b, nil
	}
	if b == m.unit {
		return a, nil
	}
	n, e := m.sizeCore(a)
	if e != nil {
		return Monomial{}, e
	}
	k, e := m.sizeCore(b)
	if e != nil {
		return Monomial{}, e
	}
	return m.mulInputs(monomialInput{manager: m, value: a, count: n}, monomialInput{manager: m, value: b, count: k})
}
func (m *MonomialManager) MulPowers(a, b []MonomialPower) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if uint64(len(a)) > uint64(^uint32(0)) || uint64(len(b)) > uint64(^uint32(0)) {
		return Monomial{}, arena.ErrBounds
	}
	return m.mulInputs(monomialInput{powers: a, count: uint32(len(a))}, monomialInput{powers: b, count: uint32(len(b))})
}
func (m *MonomialManager) mulInputs(left, right monomialInput) (Monomial, error) {
	x, y := MonomialInfo{Size: left.count}, MonomialInfo{Size: right.count}
	var e error
	if modularSourceDebug {
		for _, in := range [2]monomialInput{left, right} {
			valid, e := in.valid()
			if e != nil {
				return Monomial{}, e
			}
			if !valid {
				return Monomial{}, ErrMonomialAssertion
			}
		}
	}
	t := &m.temps[1]
	if e = m.tempReserve(t, x.Size+y.Size); e != nil {
		return Monomial{}, e
	}
	i, j, k := uint32(0), uint32(0), uint32(0)
	for {
		if i == x.Size {
			for ; j < y.Size; j++ {
				p, e := right.read(j)
				if e != nil {
					return Monomial{}, e
				}
				if e = m.tempSetPower(t, k, p); e != nil {
					return Monomial{}, e
				}
				k++
			}
			break
		}
		if j == y.Size {
			for ; i < x.Size; i++ {
				p, e := left.read(i)
				if e != nil {
					return Monomial{}, e
				}
				if e = m.tempSetPower(t, k, p); e != nil {
					return Monomial{}, e
				}
				k++
			}
			break
		}
		p, e := left.read(i)
		if e != nil {
			return Monomial{}, e
		}
		q, e := right.read(j)
		if e != nil {
			return Monomial{}, e
		}
		if p.Var == q.Var {
			p.Degree += q.Degree
			i++
			j++
		} else if p.Var < q.Var {
			i++
		} else {
			p = q
			j++
		}
		if e = m.tempSetPower(t, k, p); e != nil {
			return Monomial{}, e
		}
		k++
	}
	if e = m.tempSetSize(t, k); e != nil {
		return Monomial{}, e
	}
	r, e := m.internTemp(1)
	if e != nil {
		return r, e
	}
	if modularSourceDebug {
		z, e := m.headerCore(r)
		if e != nil {
			return r, e
		}
		ok, e := m.IsValid(r)
		if e != nil {
			return r, e
		}
		if !ok {
			return r, ErrMonomialAssertion
		}
		dx, e := left.degreeSum()
		if e != nil {
			return r, e
		}
		dy, e := right.degreeSum()
		if e != nil {
			return r, e
		}
		if z.TotalDegree != dx+dy {
			return r, ErrMonomialAssertion
		}
	}
	return r, nil
}
func (m *MonomialManager) divCore(a, b Monomial, store bool) (bool, error) {
	x, e := m.sizeInfoCore(a)
	if e != nil {
		return false, e
	}
	y, e := m.sizeInfoCore(b)
	if e != nil {
		return false, e
	}
	t := &m.temps[1]
	if store {
		if e = m.tempReserve(t, x.Size); e != nil {
			return false, e
		}
	}
	i, j, k := uint32(0), uint32(0), uint32(0)
	if x.Size < y.Size {
		return false, nil
	}
	for {
		if j == y.Size {
			if store {
				for ; i < x.Size; i++ {
					p, e := m.powerCore(a, i)
					if e != nil {
						return false, e
					}
					if e = m.tempSetPower(t, k, p); e != nil {
						return false, e
					}
					k++
				}
				if e = m.tempSetSize(t, k); e != nil {
					return false, e
				}
			}
			return true, nil
		}
		if i == x.Size {
			return false, nil
		}
		p, e := m.powerCore(a, i)
		if e != nil {
			return false, e
		}
		q, e := m.powerCore(b, j)
		if e != nil {
			return false, e
		}
		if p.Var == q.Var {
			if p.Degree < q.Degree {
				return false, nil
			}
			if store && p.Degree > q.Degree {
				p.Degree -= q.Degree
				if e = m.tempSetPower(t, k, p); e != nil {
					return false, e
				}
				k++
			}
			i++
			j++
		} else if p.Var < q.Var {
			if store {
				if e = m.tempSetPower(t, k, p); e != nil {
					return false, e
				}
				k++
			}
			i++
		} else {
			return false, nil
		}
	}
}
func (m *MonomialManager) Divides(a, b Monomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	x, e := m.degreeInfoCore(a)
	if e != nil {
		return false, e
	}
	y, e := m.degreeInfoCore(b)
	if e != nil {
		return false, e
	}
	if x.TotalDegree < y.TotalDegree {
		return false, nil
	}
	if a == b {
		return true, nil
	}
	return m.divCore(a, b, false)
}
func (m *MonomialManager) Divide(a, b Monomial) (Monomial, bool, error) {
	if e := m.check(); e != nil {
		return Monomial{}, false, e
	}
	x, e := m.degreeInfoCore(a)
	if e != nil {
		return Monomial{}, false, e
	}
	y, e := m.degreeInfoCore(b)
	if e != nil {
		return Monomial{}, false, e
	}
	if x.TotalDegree < y.TotalDegree {
		return Monomial{}, false, nil
	}
	if a == b {
		return m.unit, true, nil
	}
	ok, e := m.divCore(a, b, true)
	if e != nil || !ok {
		return Monomial{}, ok, e
	}
	r, e := m.internTemp(1)
	return r, true, e
}
func (m *MonomialManager) gcdCore(a, b Monomial) (bool, error) {
	x, e := m.sizeInfoCore(a)
	if e != nil {
		return false, e
	}
	y, e := m.sizeInfoCore(b)
	if e != nil {
		return false, e
	}
	g, r1, r2 := &m.temps[1], &m.temps[2], &m.temps[3]
	n := x.Size
	if y.Size < n {
		n = y.Size
	}
	if e = m.tempReserve(g, n); e != nil {
		return false, e
	}
	// These source reserve arguments are deliberately opposite to the eventual
	// quotient lengths. Do not silently correct its endangered wide-size case.
	if e = m.tempReserve(r1, y.Size); e != nil {
		return false, e
	}
	if e = m.tempReserve(r2, x.Size); e != nil {
		return false, e
	}
	found := false
	i, j, u, v, w := uint32(0), uint32(0), uint32(0), uint32(0), uint32(0)
	for {
		if i == x.Size {
			if !found {
				return false, nil
			}
			for ; j < y.Size; j++ {
				p, e := m.powerCore(b, j)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(r2, v, p); e != nil {
					return false, e
				}
				v++
			}
			break
		}
		if j == y.Size {
			if !found {
				return false, nil
			}
			for ; i < x.Size; i++ {
				p, e := m.powerCore(a, i)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(r1, u, p); e != nil {
					return false, e
				}
				u++
			}
			break
		}
		p, e := m.powerCore(a, i)
		if e != nil {
			return false, e
		}
		q, e := m.powerCore(b, j)
		if e != nil {
			return false, e
		}
		if p.Var == q.Var {
			found = true
			if p.Degree > q.Degree {
				p.Degree -= q.Degree
				if e = m.tempSetPower(r1, u, p); e != nil {
					return false, e
				}
				u++
				if e = m.tempSetPower(g, w, q); e != nil {
					return false, e
				}
				w++
			} else if q.Degree > p.Degree {
				q.Degree -= p.Degree
				if e = m.tempSetPower(r2, v, q); e != nil {
					return false, e
				}
				v++
				if e = m.tempSetPower(g, w, p); e != nil {
					return false, e
				}
				w++
			} else {
				if e = m.tempSetPower(g, w, p); e != nil {
					return false, e
				}
				w++
			}
			i++
			j++
		} else if p.Var < q.Var {
			if e = m.tempSetPower(r1, u, p); e != nil {
				return false, e
			}
			u++
			i++
		} else {
			if e = m.tempSetPower(r2, v, q); e != nil {
				return false, e
			}
			v++
			j++
		}
	}
	if e = m.tempSetSize(r1, u); e != nil {
		return false, e
	}
	if e = m.tempSetSize(r2, v); e != nil {
		return false, e
	}
	if e = m.tempSetSize(g, w); e != nil {
		return false, e
	}
	return true, nil
}
func (m *MonomialManager) GCD(a, b Monomial) (Monomial, Monomial, Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, Monomial{}, Monomial{}, e
	}
	found, e := m.gcdCore(a, b)
	if e != nil {
		return Monomial{}, Monomial{}, Monomial{}, e
	}
	if !found {
		return m.unit, b, a, nil
	}
	q1, e := m.internTemp(2)
	if e != nil {
		return Monomial{}, q1, Monomial{}, e
	}
	q2, e := m.internTemp(3)
	if e != nil {
		return Monomial{}, q1, q2, e
	}
	g, e := m.internTemp(1)
	return g, q1, q2, e
}
func (m *MonomialManager) Unify(a, b Monomial) (Monomial, Monomial, bool, error) {
	if e := m.check(); e != nil {
		return Monomial{}, Monomial{}, false, e
	}
	found, e := m.gcdCore(a, b)
	if e != nil || !found {
		return Monomial{}, Monomial{}, found, e
	}
	q1, e := m.internTemp(2)
	if e != nil {
		return q1, Monomial{}, true, e
	}
	q2, e := m.internTemp(3)
	return q1, q2, true, e
}
func (m *MonomialManager) Power(a Monomial, k uint32) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if k == 0 {
		return m.unit, nil
	}
	if k == 1 {
		return a, nil
	}
	x, e := m.sizeInfoCore(a)
	if e != nil {
		return Monomial{}, e
	}
	t := &m.temps[1]
	if e = m.tempReserve(t, x.Size); e != nil {
		return Monomial{}, e
	}
	for i := uint32(0); i < x.Size; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return Monomial{}, e
		}
		p.Degree *= k
		if e = m.tempSetPower(t, i, p); e != nil {
			return Monomial{}, e
		}
	}
	if e = m.tempSetSize(t, x.Size); e != nil {
		return Monomial{}, e
	}
	return m.internTemp(1)
}
func (m *MonomialManager) Sqrt(a Monomial) (Monomial, bool, error) {
	if e := m.check(); e != nil {
		return Monomial{}, false, e
	}
	if a == m.unit {
		return a, true, nil
	}
	x, e := m.sizeInfoCore(a)
	if e != nil {
		return Monomial{}, false, e
	}
	t := &m.temps[1]
	if e = m.tempReserve(t, x.Size); e != nil {
		return Monomial{}, false, e
	}
	for i := uint32(0); i < x.Size; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return Monomial{}, false, e
		}
		if p.Degree%2 == 1 {
			return Monomial{}, false, nil
		}
		p.Degree /= 2
		if e = m.tempSetPower(t, i, p); e != nil {
			return Monomial{}, false, e
		}
	}
	if e = m.tempSetSize(t, x.Size); e != nil {
		return Monomial{}, false, e
	}
	r, e := m.internTemp(1)
	return r, true, e
}
func (m *MonomialManager) variableTransform(a Monomial, v, k uint32, operation uint8) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if modularSourceDebug && v >= m.nextVar {
		return Monomial{}, ErrMonomialAssertion
	}
	info, e := m.sizeInfoCore(a)
	if e != nil {
		return Monomial{}, e
	}
	t := &m.temps[1]
	if e = m.tempReserve(t, info.Size); e != nil {
		return Monomial{}, e
	}
	n := uint32(0)
	for i := uint32(0); i < info.Size; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return Monomial{}, e
		}
		if p.Var == v {
			if operation == 0 {
				if modularSourceDebug && k > p.Degree {
					return Monomial{}, ErrMonomialAssertion
				}
				if k >= p.Degree {
					continue
				}
				p.Degree -= k
			}
			if operation == 1 {
				continue
			}
			if operation == 2 {
				if p.Degree <= 1 {
					continue
				}
				p.Degree--
			}
		}
		if e = m.tempSetPower(t, n, p); e != nil {
			return Monomial{}, e
		}
		n++
	}
	if e = m.tempSetSize(t, n); e != nil {
		return Monomial{}, e
	}
	return m.internTemp(1)
}
func (m *MonomialManager) DivideVariablePower(a Monomial, v, k uint32) (Monomial, error) {
	return m.variableTransform(a, v, k, 0)
}
func (m *MonomialManager) RemoveVariable(a Monomial, v uint32) (Monomial, error) {
	return m.variableTransform(a, v, 0, 1)
}
func (m *MonomialManager) Derivative(a Monomial, v uint32) (Monomial, error) {
	return m.variableTransform(a, v, 0, 2)
}

var _ = arena.ErrBounds
