// Source: algebraic_numbers.cpp:1876-1878,1887-1889,2945-2947,2959-2961.
package polynomial

// Equal follows source eq(numeral, numeral), including any refinement performed
// by comparison. It does not substitute encoded-cell identity for value equality.
func (m *AlgebraicRootCore) Equal(a, b *Algebraic) (bool, error) {
	order, err := m.Compare(a, b)
	return order == 0 && err == nil, err
}

// Less follows source lt(numeral, numeral).
func (m *AlgebraicRootCore) Less(a, b *Algebraic) (bool, error) {
	order, err := m.Compare(a, b)
	return order < 0 && err == nil, err
}
