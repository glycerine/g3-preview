// Source: src/math/polynomial/polynomial.cpp:5940-5951;
// src/util/mpzzp.h:58-68,131; src/util/mpz.h:124-149.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) negAllocationRoots(r *polynomialNegInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.self != r.manager || r.manager.integer != m || !r.manager.matchesNegInvocation(r) || !r.frame.negativeExecuting || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerModularManager) negAllocationContext(r *polynomialNegInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.negAllocationRoots(r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{neg: r, negInvocation: r.invocation}
}
func (m *IntegerModularManager) copyWithNegReceipt(out, value *Int, r *polynomialNegInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.negAllocationContext(r)
	if e := m.manager.copyFromOwnerWithAllocationContext(out, m.manager, value, nil, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}
func (m *IntegerModularManager) negWithNegReceipt(out *Int, r *polynomialNegInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Neg(out); e != nil {
		return e
	}
	// PN context is validated only at this reached source call, after NEG.
	return m.normalizeWithSourceContext(out, m.negAllocationContext(r))
}
