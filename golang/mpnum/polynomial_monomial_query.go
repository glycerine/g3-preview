// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import "math"

func (m *MonomialManager) IndexOf(a Monomial, x uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return 0, e
	}
	return m.indexCore(a, h.Size, x)
}
func (m *MonomialManager) indexCore(a Monomial, n, x uint32) (uint32, error) {
	if n == 0 {
		return math.MaxUint32, nil
	}
	last := n - 1
	p, e := m.powerCore(a, last)
	if e != nil {
		return 0, e
	}
	if p.Var == x {
		return last, nil
	}
	if n < 8 {
		for i := last; i > 0; {
			i--
			p, e = m.powerCore(a, i)
			if e != nil {
				return 0, e
			}
			if p.Var == x {
				return i, nil
			}
		}
		return math.MaxUint32, nil
	}
	low, high := int32(0), int32(last)
	for {
		mid := low + (high-low)/2
		p, e = m.powerCore(a, uint32(mid))
		if e != nil {
			return 0, e
		}
		if x > p.Var {
			low = mid + 1
		} else if x < p.Var {
			high = mid - 1
		} else {
			return uint32(mid), nil
		}
		if low > high {
			return math.MaxUint32, nil
		}
	}
}
func (m *MonomialManager) DegreeOf(a Monomial, x uint32) (uint32, error) {
	i, e := m.IndexOf(a, x)
	if e != nil {
		return 0, e
	}
	if i == math.MaxUint32 {
		return 0, nil
	}
	p, e := m.powerCore(a, i)
	return p.Degree, e
}
func (m *MonomialManager) MaxVariable(a Monomial) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return 0, e
	}
	if h.Size == 0 {
		return math.MaxUint32, nil
	}
	p, e := m.powerCore(a, h.Size-1)
	return p.Var, e
}
func (m *MonomialManager) MaxVariableDegree(a Monomial) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	h, e := m.headerCore(a)
	if e != nil || h.Size == 0 {
		return 0, e
	}
	p, e := m.powerCore(a, h.Size-1)
	return p.Degree, e
}
func (m *MonomialManager) maxSmallerCore(a Monomial, x uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return 0, e
	}
	n := h.Size
	if n == 0 {
		return math.MaxUint32, nil
	}
	if n < 8 {
		for i := n; i > 0; {
			i--
			p, e := m.powerCore(a, i)
			if e != nil {
				return 0, e
			}
			if p.Var < x {
				return p.Var, nil
			}
		}
		return math.MaxUint32, nil
	}
	low, high := int32(0), int32(n-1)
	lp, e := m.powerCore(a, 0)
	if e != nil {
		return 0, e
	}
	if x <= lp.Var {
		return math.MaxUint32, nil
	}
	hp, e := m.powerCore(a, uint32(high))
	if e != nil {
		return 0, e
	}
	if x > hp.Var {
		return hp.Var, nil
	}
	if x == hp.Var {
		p, e := m.powerCore(a, uint32(high-1))
		return p.Var, e
	}
	for {
		if high == low+1 {
			p, e := m.powerCore(a, uint32(low))
			return p.Var, e
		}
		mid := low + (high-low)/2
		p, e := m.powerCore(a, uint32(mid))
		if e != nil {
			return 0, e
		}
		if x == p.Var {
			p, e = m.powerCore(a, uint32(mid-1))
			return p.Var, e
		}
		if x < p.Var {
			high = mid
		} else {
			low = mid
		}
	}
}
func (m *MonomialManager) IsValid(a Monomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	n, e := m.sizeCore(a)
	if e != nil {
		return false, e
	}
	valid, e := m.validPowerProductCore(a, n)
	if e != nil || !valid {
		return false, e
	}
	n, e = m.sizeCore(a)
	if e != nil {
		return false, e
	}
	total, e := m.degreeSumCore(a, n)
	if e != nil {
		return false, e
	}
	stored, e := m.degreeCore(a)
	return total == stored, e
}
func (m *MonomialManager) IsSquare(a Monomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < h.Size; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return false, e
		}
		if p.Degree%2 != 0 {
			return false, nil
		}
	}
	return true, nil
}
func (m *MonomialManager) IsPowerOfTwo(a Monomial) (bool, error) { return m.IsSquare(a) }

func (m *MonomialManager) sizeCore(a Monomial) (uint32, error) {
	l, e := m.location(a)
	if e != nil {
		return 0, e
	}
	return m.allocator.ReadUint32(l.raw, 12)
}
func (m *MonomialManager) degreeCore(a Monomial) (uint32, error) {
	l, e := m.location(a)
	if e != nil {
		return 0, e
	}
	return m.allocator.ReadUint32(l.raw, 8)
}
func (m *MonomialManager) varCore(a Monomial, i uint32) (uint32, error) {
	l, e := m.location(a)
	if e != nil {
		return 0, e
	}
	if modularSourceDebug {
		n, e := m.sizeCore(a)
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrMonomialAssertion
		}
	}
	return m.allocator.ReadUint32(l.raw, 20+uint64(i)*8)
}
func (m *MonomialManager) sizeInfoCore(a Monomial) (MonomialInfo, error) {
	n, e := m.sizeCore(a)
	return MonomialInfo{Size: n}, e
}

// The free power-product helpers read raw array fields, not get_power's
// debug index assertion or unrelated fields of each power.
func (m *MonomialManager) powerFieldCore(a Monomial, i uint32, field uint64) (uint32, error) {
	l, e := m.location(a)
	if e != nil {
		return 0, e
	}
	return m.allocator.ReadUint32(l.raw, 20+uint64(i)*8+field)
}
func (m *MonomialManager) validPowerProductCore(a Monomial, n uint32) (bool, error) {
	for i := uint32(1); i < n; i++ {
		previous, e := m.powerFieldCore(a, i-1, 0)
		if e != nil {
			return false, e
		}
		current, e := m.powerFieldCore(a, i, 0)
		if e != nil {
			return false, e
		}
		if previous >= current {
			return false, nil
		}
	}
	return true, nil
}
func (m *MonomialManager) degreeSumCore(a Monomial, n uint32) (uint32, error) {
	var sum uint32
	for i := uint32(0); i < n; i++ {
		v, e := m.powerFieldCore(a, i, 4)
		if e != nil {
			return 0, e
		}
		sum += v
	}
	return sum, nil
}
func (m *MonomialManager) degreeInfoCore(a Monomial) (MonomialInfo, error) {
	n, e := m.degreeCore(a)
	return MonomialInfo{TotalDegree: n}, e
}

func (m *MonomialManager) MaxSmallerThan(a Monomial, x uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if modularSourceDebug && x == math.MaxUint32 {
		return 0, ErrMonomialAssertion
	}
	v, e := m.maxSmallerCore(a, x)
	if e != nil {
		return 0, e
	}
	if modularSourceDebug {
		n, e := m.sizeCore(a)
		if e != nil {
			return 0, e
		}
		found := false
		for i := uint32(0); i < n; i++ {
			q, e := m.varCore(a, i)
			if e != nil {
				return 0, e
			}
			if q < x && (v == math.MaxUint32 || q > v) {
				return 0, ErrMonomialAssertion
			}
			if q == v {
				found = true
			}
		}
		if v != math.MaxUint32 && (v >= x || !found) {
			return 0, ErrMonomialAssertion
		}
	}
	return v, nil
}
