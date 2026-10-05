// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpzzp.h:167-185,195-208; src/util/mpz.h:124-149.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) scalarDivAllocationRoots(r *polynomialScalarDivInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.self != r.manager || r.manager.integer != m || !r.manager.matchesScalarDivInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerModularManager) scalarDivAllocationContext(r *polynomialScalarDivInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.scalarDivAllocationRoots(r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{scalarDiv: r}
}
func (m *IntegerModularManager) copyWithScalarDivReceipt(out, value *Int, r *polynomialScalarDivInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.scalarDivAllocationContext(r)
	if e := m.manager.copyFromOwnerWithAllocationContext(out, m.manager, value, nil, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}

func (m *IntegerModularManager) divWithScalarDivReceipt(a, b, out *Int, r *polynomialScalarDivInvocationReceipt) error {
	return m.divWithAllocationContext(a, b, out, m.scalarDivAllocationContext(r))
}
