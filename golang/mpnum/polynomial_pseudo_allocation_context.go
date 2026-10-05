// Source: src/math/polynomial/polynomial.cpp:4939-4949,4989-5010;
// src/util/mpzzp.h:58-68,131; src/util/mpz.h:124-149.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) pseudoAllocationRoots(r *polynomialPseudoDivisionInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.manager == nil || r.manager.integer != m || !r.manager.matchesPseudoInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 || r.frame.constructed&(uint16(1)<<pseudoNumeral) == 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) pseudoContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	if c.pseudo == nil || c.pseudoInvocation == 0 || c.pseudoInvocation != c.pseudo.invocation {
		return nil
	}
	return m.pseudoAllocationRoots(c.pseudo)
}
func (m *IntegerModularManager) pseudoAllocationContext(r *polynomialPseudoDivisionInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.pseudoAllocationRoots(r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{pseudo: r, pseudoInvocation: r.invocation}
}
func (m *IntegerModularManager) copyWithPseudoReceipt(out, value *Int, r *polynomialPseudoDivisionInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.pseudoAllocationContext(r)
	if e := m.manager.copyFromOwnerWithAllocationContext(out, m.manager, value, nil, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}
func (m *IntegerModularManager) negWithPseudoReceipt(out *Int, r *polynomialPseudoDivisionInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.pseudoAllocationContext(r)
	if e := m.manager.negWithAllocationContext(out, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}
