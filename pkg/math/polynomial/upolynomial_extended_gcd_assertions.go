package polynomial

// Source: upolynomial.cpp:1047-1050. Six independent alias assertions precede
// the field assertion, local-vector declarations and U initialization.
func (m *UPolynomialManager) extendedGCDEntryAssertions(a, b UPolynomialView, u, v, d *upolyVectorState) error {
	for i, input := range [2]UPolynomialView{a, b} {
		for j, out := range [3]*upolyVectorState{u, v, d} {
			enabled, e := refinementSourceAssertionsEnabled()
			if e != nil {
				return e
			}
			if enabled && m.alias(input, out) {
				expression := [2][3]string{{"!is_alias(A, U)", "!is_alias(A, V)", "!is_alias(A, D)"}, {"!is_alias(B, U)", "!is_alias(B, V)", "!is_alias(B, D)"}}[i][j]
				if e := m.scalarDivisionAssertionFailure(int32(1047+i), expression); e != nil {
					return e
				}
			}
		}
	}
	enabled, e := refinementSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	field, e := m.modular.Field()
	if e != nil {
		return e
	}
	if !field {
		return m.scalarDivisionAssertionFailure(1050, "field()")
	}
	return nil
}
