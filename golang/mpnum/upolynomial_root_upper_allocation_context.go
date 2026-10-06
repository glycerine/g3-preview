// Source: src/math/polynomial/upolynomial.cpp:1906-1951;
// src/util/mpzzp.h:132,150,195-207,235.
package mpnum

func (m *IntegerModularManager) rootUpperContextValid(c integerAllocationInvocationContext) bool {
	return m != nil && m.self == m && m.manager != nil && c.rootUpperBoundFrame(m.manager) != nil && c.rootUpperBound.owner.modular == m
}
func (m *IntegerModularManager) copyWithRootUpperContext(out, value *Int, c integerAllocationInvocationContext) error {
	if !m.rootUpperContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.copyWithSourceContext(out, value, c)
}
func (m *IntegerModularManager) absWithRootUpperContext(out *Int, c integerAllocationInvocationContext) error {
	if !m.rootUpperContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.absWithSourceContext(out, c)
}
func (m *IntegerModularManager) addWithRootUpperContext(a, b, out *Int, c integerAllocationInvocationContext) error {
	if !m.rootUpperContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.addWithSourceContext(a, b, out, c)
}
func (m *IntegerModularManager) divWithRootUpperContext(a, b, out *Int, c integerAllocationInvocationContext) error {
	if !m.rootUpperContextValid(c) {
		return ErrUPolynomialOwner
	}
	return m.divWithAllocationContext(a, b, out, c)
}
