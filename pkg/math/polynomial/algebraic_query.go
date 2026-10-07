// Licensed under the MIT license.
// Source: src/math/polynomial/algebraic_numbers.cpp:217-240,350-356,2807-2828.
package polynomial

// IsPos uses the source basic rational or actual algebraic isolating interval.
func (m *AlgebraicRootCore) IsPos(a *Algebraic) (bool, error) {
	l, e := m.location(a)
	if e != nil || l == nil {
		return false, e
	}
	r, e := m.load(l)
	if e != nil {
		return false, e
	}
	if l.kind == 0 {
		return m.config.Rationals.IsPos(&r.value)
	}
	return m.config.Intervals.IsPos(&r.interval)
}

// IsNeg uses the source basic rational or actual algebraic isolating interval.
func (m *AlgebraicRootCore) IsNeg(a *Algebraic) (bool, error) {
	l, e := m.location(a)
	if e != nil || l == nil {
		return false, e
	}
	r, e := m.load(l)
	if e != nil {
		return false, e
	}
	if l.kind == 0 {
		return m.config.Rationals.IsNeg(&r.value)
	}
	return m.config.Intervals.IsNeg(&r.interval)
}

// Degree preserves source zero/basic/root dispatch; it does not minimize a
// polynomial or refine an interval.
func (m *AlgebraicRootCore) Degree(a *Algebraic) (uint32, error) {
	l, e := m.location(a)
	if e != nil || l == nil {
		return 0, e
	}
	if l.kind == 0 {
		return 1, nil
	}
	r, e := m.load(l)
	if e != nil {
		return 0, e
	}
	return r.size - 1, nil
}
