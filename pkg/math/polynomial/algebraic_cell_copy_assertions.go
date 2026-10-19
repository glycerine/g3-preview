// Source: algebraic_numbers.cpp:235-240,427,445-446,509-539.
package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

func (m *AlgebraicRootCore) copySourceBasicValue(source *AlgebraicRootCore, l *algebraicLocation) (*unumeral.Rational, error) {
	if _, _, e := algebraicAssertionSwitch(); e != nil {
		return nil, e
	} // cpp:236
	if l == nil {
		return &m.zero, nil
	}
	if e := algebraicQueryTagAssertionSwitch(); e != nil {
		return nil, e
	} // h:372
	r, e := source.load(l)
	if e != nil {
		return nil, e
	}
	return &r.value, nil
}
func (m *AlgebraicRootCore) assertCellCopyPolynomialEmpty(r *algebraicRecord) error {
	_, enabled, e := algebraicAssertionSwitch()
	if e != nil {
		return e
	}
	if enabled && r.poly != (UPolynomial{}) {
		if e = m.cellAssertionFailure(445, "c->m_p == 0"); e != nil {
			return e
		}
	}
	_, enabled, e = algebraicAssertionSwitch()
	if e != nil {
		return e
	}
	if enabled && r.size != 0 {
		return m.cellAssertionFailure(446, "c->m_p_sz == 0")
	}
	return nil
}
func (m *AlgebraicRootCore) assertCellResetZero(a *Algebraic) error {
	_, enabled, e := algebraicAssertionSwitch()
	if e != nil || !enabled {
		return e
	}
	if a.slot != 0 {
		return m.cellAssertionFailure(427, "is_zero(a)")
	}
	return nil
}
