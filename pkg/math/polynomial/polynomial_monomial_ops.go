// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
)

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
		return in.manager.rawPowerCore(in.value, i)
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
func (in monomialInput) degree(i uint32) (uint32, error) {
	if in.manager != nil {
		return in.manager.powerFieldCore(in.value, i, 4)
	}
	if uint64(i) >= uint64(len(in.powers)) {
		return 0, arena.ErrBounds
	}
	return in.powers[i].Degree, nil
}

// Whole raw-array copies fetch both members only when the source copy is reached.
func (m *MonomialManager) rawPowerCore(a Monomial, i uint32) (MonomialPower, error) {
	v, e := m.powerFieldCore(a, i, 0)
	if e != nil {
		return MonomialPower{}, e
	}
	d, e := m.powerFieldCore(a, i, 4)
	return MonomialPower{v, d}, e
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
	return m.mulCreated(a, b, nil)
}
func (m *MonomialManager) mulCreated(a, b Monomial, outcome *monomialFactoryOutcome) (Monomial, error) {
	if outcome != nil {
		*outcome = monomialFactoryOutcome{}
	}
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
	return m.mulInputsCreated(monomialInput{manager: m, value: a, count: n}, monomialInput{manager: m, value: b, count: k}, outcome)
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
	return m.mulInputsCreated(left, right, nil)
}
func (m *MonomialManager) mulInputsCreated(left, right monomialInput, outcome *monomialFactoryOutcome) (result Monomial, cause error) {
	if outcome != nil {
		*outcome = monomialFactoryOutcome{}
	}
	defer func() {
		if cause == nil {
			m.creation = monomialCreation{}
		}
	}()
	x, y := MonomialInfo{Size: left.count}, MonomialInfo{Size: right.count}
	var e error
	if polynomialSourceDebug {
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
		v1, e := left.variable(i)
		if e != nil {
			return Monomial{}, e
		}
		v2, e := right.variable(j)
		if e != nil {
			return Monomial{}, e
		}
		var p MonomialPower
		if v1 == v2 {
			d1, e := left.degree(i)
			if e != nil {
				return Monomial{}, e
			}
			d2, e := right.degree(j)
			if e != nil {
				return Monomial{}, e
			}
			p = MonomialPower{v1, d1 + d2}
			i++
			j++
		} else if v1 < v2 {
			p, e = left.read(i)
			if e != nil {
				return Monomial{}, e
			}
			i++
		} else {
			if polynomialSourceDebug && v1 <= v2 {
				return Monomial{}, ErrMonomialAssertion
			}
			p, e = right.read(j)
			if e != nil {
				return Monomial{}, e
			}
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
	r, e := m.internTempCreated(1, outcome)
	if e != nil {
		return r, e
	}
	if polynomialSourceDebug {
		ok, e := m.IsValid(r)
		if e != nil {
			return r, e
		}
		if !ok {
			return r, ErrMonomialAssertion
		}
		total, e := m.degreeCore(r)
		if e != nil {
			return r, e
		}
		dx, e := left.degreeSum()
		if e != nil {
			return r, e
		}
		dy, e := right.degreeSum()
		if e != nil {
			return r, e
		}
		if total != dx+dy {
			return r, ErrMonomialAssertion
		}
	}
	return r, nil
}
func (m *MonomialManager) divCore(a, b Monomial, store bool) (bool, error) {
	n, e := m.sizeCore(a)
	if e != nil {
		return false, e
	}
	k2, e := m.sizeCore(b)
	if e != nil {
		return false, e
	}
	t := &m.temps[1]
	if store {
		if e = m.tempReserve(t, n); e != nil {
			return false, e
		}
	}
	i, j, k := uint32(0), uint32(0), uint32(0)
	if n < k2 {
		return false, nil
	}
	for {
		if j == k2 {
			if store {
				for ; i < n; i++ {
					p, e := m.rawPowerCore(a, i)
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
		if i == n {
			return false, nil
		}
		v1, e := m.powerFieldCore(a, i, 0)
		if e != nil {
			return false, e
		}
		v2, e := m.powerFieldCore(b, j, 0)
		if e != nil {
			return false, e
		}
		if v1 == v2 {
			d1, e := m.powerFieldCore(a, i, 4)
			if e != nil {
				return false, e
			}
			d2, e := m.powerFieldCore(b, j, 4)
			if e != nil {
				return false, e
			}
			if d1 < d2 {
				return false, nil
			}
			if store && d1 > d2 {
				if e = m.tempSetPower(t, k, MonomialPower{v1, d1 - d2}); e != nil {
					return false, e
				}
				k++
			}
			i++
			j++
		} else if v1 < v2 {
			if store {
				p, e := m.rawPowerCore(a, i)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(t, k, p); e != nil {
					return false, e
				}
				k++
			}
			i++
		} else {
			if polynomialSourceDebug && v1 <= v2 {
				return false, ErrMonomialAssertion
			}
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
	return m.divideCreated(a, b, nil)
}
func (m *MonomialManager) divideCreated(a, b Monomial, outcome *monomialFactoryOutcome) (Monomial, bool, error) {
	if outcome != nil {
		*outcome = monomialFactoryOutcome{}
	}
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
	r, e := m.internTempCreated(1, outcome)
	if e == nil {
		m.creation = monomialCreation{}
	}
	return r, true, e
}
func (m *MonomialManager) gcdCore(a, b Monomial) (bool, error) {
	n1, e := m.sizeCore(a)
	if e != nil {
		return false, e
	}
	n2, e := m.sizeCore(b)
	if e != nil {
		return false, e
	}
	g, r1, r2 := &m.temps[1], &m.temps[2], &m.temps[3]
	n := n1
	if n2 < n {
		n = n2
	}
	if e = m.tempReserve(g, n); e != nil {
		return false, e
	}
	// Preserve the source's opposite remainder reserve arguments.
	if e = m.tempReserve(r1, n2); e != nil {
		return false, e
	}
	if e = m.tempReserve(r2, n1); e != nil {
		return false, e
	}
	found := false
	i, j, u, v, w := uint32(0), uint32(0), uint32(0), uint32(0), uint32(0)
	for {
		if i == n1 {
			if !found {
				return false, nil
			}
			for ; j < n2; j++ {
				p, e := m.rawPowerCore(b, j)
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
		if j == n2 {
			if !found {
				return false, nil
			}
			for ; i < n1; i++ {
				p, e := m.rawPowerCore(a, i)
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
		v1, e := m.powerFieldCore(a, i, 0)
		if e != nil {
			return false, e
		}
		v2, e := m.powerFieldCore(b, j, 0)
		if e != nil {
			return false, e
		}
		if v1 == v2 {
			found = true
			d1, e := m.powerFieldCore(a, i, 4)
			if e != nil {
				return false, e
			}
			d2, e := m.powerFieldCore(b, j, 4)
			if e != nil {
				return false, e
			}
			if d1 > d2 {
				if e = m.tempSetPower(r1, u, MonomialPower{v1, d1 - d2}); e != nil {
					return false, e
				}
				p, e := m.rawPowerCore(b, j)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(g, w, p); e != nil {
					return false, e
				}
				u++
				w++
			} else if d2 > d1 {
				if e = m.tempSetPower(r2, v, MonomialPower{v2, d2 - d1}); e != nil {
					return false, e
				}
				p, e := m.rawPowerCore(a, i)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(g, w, p); e != nil {
					return false, e
				}
				v++
				w++
			} else {
				if polynomialSourceDebug && d1 != d2 {
					return false, ErrMonomialAssertion
				}
				p, e := m.rawPowerCore(a, i)
				if e != nil {
					return false, e
				}
				if e = m.tempSetPower(g, w, p); e != nil {
					return false, e
				}
				w++
			}
			i++
			j++
		} else if v1 < v2 {
			p, e := m.rawPowerCore(a, i)
			if e != nil {
				return false, e
			}
			if e = m.tempSetPower(r1, u, p); e != nil {
				return false, e
			}
			u++
			i++
		} else {
			if polynomialSourceDebug && v1 <= v2 {
				return false, ErrMonomialAssertion
			}
			p, e := m.rawPowerCore(b, j)
			if e != nil {
				return false, e
			}
			if e = m.tempSetPower(r2, v, p); e != nil {
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
	return m.sqrtCreated(a, nil)
}
func (m *MonomialManager) sqrtCreated(a Monomial, outcome *monomialFactoryOutcome) (result Monomial, ok bool, cause error) {

	if outcome != nil {
		*outcome = monomialFactoryOutcome{}
	}
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
		d, e := m.degreeAtCore(a, i)
		if e != nil {
			return Monomial{}, false, e
		}
		if d%2 == 1 {
			return Monomial{}, false, nil
		}
		v, e := m.varCore(a, i)
		if e != nil {
			return Monomial{}, false, e
		}
		d, e = m.degreeAtCore(a, i)
		if e != nil {
			return Monomial{}, false, e
		}
		p := MonomialPower{v, d / 2}
		if e = m.tempSetPower(t, i, p); e != nil {
			return Monomial{}, false, e
		}
	}
	if e = m.tempSetSize(t, x.Size); e != nil {
		return Monomial{}, false, e
	}
	r, e := m.internTempCreated(1, outcome)
	if e == nil {
		m.creation = monomialCreation{}
	}
	return r, true, e
}
func (m *MonomialManager) variableTransform(a Monomial, v, k uint32, operation uint8) (Monomial, error) {
	return m.variableTransformCreated(a, v, k, operation, nil)
}

func (m *MonomialManager) variableTransformCreated(a Monomial, v, k uint32, operation uint8, outcome *monomialFactoryOutcome) (result Monomial, cause error) {
	defer func() {
		if cause == nil {
			m.creation = monomialCreation{}
		}
	}()
	if outcome != nil {
		*outcome = monomialFactoryOutcome{}
	}
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if polynomialSourceDebug && v >= m.nextVar {
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
		y, e := m.varCore(a, i)
		if e != nil {
			return Monomial{}, e
		}
		if y == v && operation == 1 {
			continue
		}
		var p MonomialPower
		if y == v {
			if operation == 0 {
				if polynomialSourceDebug {
					d, e := m.powerFieldCore(a, i, 4)
					if e != nil {
						return Monomial{}, e
					}
					if k > d {
						return Monomial{}, ErrMonomialAssertion
					}
				}
				d, e := m.powerFieldCore(a, i, 4)
				if e != nil {
					return Monomial{}, e
				}
				if k >= d {
					continue
				}
				p = MonomialPower{y, d - k}
			} else {
				d, e := m.powerFieldCore(a, i, 4)
				if e != nil {
					return Monomial{}, e
				}
				if d <= 1 {
					continue
				}
				p = MonomialPower{y, d - 1}
			}

		} else {
			p, e = m.rawPowerCore(a, i)
			if e != nil {
				return Monomial{}, e
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
	return m.internTempCreated(1, outcome)
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

func (m *MonomialManager) removeVariableCreated(a Monomial, v uint32, outcome *monomialFactoryOutcome) (Monomial, error) {
	return m.variableTransformCreated(a, v, 0, 1, outcome)
}
