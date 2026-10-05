// Source: src/math/polynomial/polynomial.cpp:4507-4528;
// src/util/mpz.cpp; src/util/mpzzp.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// The passed receipt names the actual full manager's embedded derivative
// frame. Allocation leaves acquire only under its executing source invocation.
func (m *IntegerManager) derivativeAllocationRoots(r *polynomialDerivativeInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.self != r.manager || r.manager.core == nil || r.manager.core.self != r.manager.core || r.manager.core.integer != m || !r.manager.matchesDerivativeInvocation(r) || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}

func (m *IntegerModularManager) setUint32WithDerivativeReceipt(out *Int, v uint32, r *polynomialDerivativeInvocationReceipt) error {
	return m.setUint32WithSourceContext(out, v, m.derivativeAllocationContext(r))
}
func (m *IntegerModularManager) mulWithDerivativeReceipt(a, b, out *Int, r *polynomialDerivativeInvocationReceipt) error {
	return m.mulWithSourceContext(a, b, out, m.derivativeAllocationContext(r))
}
func (m *IntegerModularManager) derivativeAllocationContext(r *polynomialDerivativeInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.derivativeAllocationRoots(r) == nil || r.manager.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{derivative: r}
}

func (m *IntegerModularManager) normalizeWithSourceContext(out *Int, context integerAllocationInvocationContext) error {
	if r := context.derivative; m.manager.derivativeAllocationRoots(r) != nil && r.manager.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	return m.normalizeWithArithmeticReceipt(out, context.arithmetic)
}
