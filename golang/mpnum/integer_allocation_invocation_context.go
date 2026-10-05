// Source: src/util/mpz.h:allocate; src/util/mpz.cpp scratch/publication callers.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// Explicit passed invocation identity. It carries no allocation snapshot and
// creates no source numeral, reservation, or ambient callback owner hook.
type integerAllocationInvocationContext struct {
	pn           *integerPNormalizeScope
	pnInvocation uint64
	arithmetic   *polynomialArithmeticInvocationReceipt
	addMul       *integerAddMulInvocationReceipt
	derivative   *polynomialDerivativeInvocationReceipt
	scalarDiv    *polynomialScalarDivInvocationReceipt
}

func pnAllocationContext(pn *integerPNormalizeScope) integerAllocationInvocationContext {
	c := integerAllocationInvocationContext{pn: pn}
	if pn != nil {
		c.pnInvocation = pn.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) roots(m *IntegerManager) *polynomialNumericRoots {
	if s := c.pn; s != nil && s.owner != nil && s.owner.manager == m && s == &s.owner.normalization && s.active && s.executing && c.pnInvocation != 0 && c.pnInvocation == s.invocation {
		return &s.roots
	}
	if c.scalarDiv != nil {
		return m.scalarDivAllocationRoots(c.scalarDiv)
	}
	if c.derivative != nil {
		return m.derivativeAllocationRoots(c.derivative)
	}
	if c.addMul != nil && m.matchesAddMulInvocation(c.addMul) && c.addMul.frame.executing {
		s := c.addMul.frame
		if s.parent.manager != nil && m.arithmeticAllocationRoots(&s.parent) == nil {
			return nil
		}
		return &s.numeric
	}
	return m.arithmeticAllocationRoots(c.arithmetic)
}

func (m *IntegerManager) copyFromOwnerWithArithmeticReceipt(out *Int, owner *IntegerManager, value *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.copyFromOwnerWithAllocationContext(out, owner, value, nil, integerAllocationInvocationContext{arithmetic: receipt})
}
func (m *IntegerModularManager) arithmeticAddMulReceipt(r *polynomialArithmeticInvocationReceipt) *integerAddMulInvocationReceipt {
	if m.manager.arithmeticAllocationRoots(r) == nil {
		return nil
	}
	return &r.frame.addMul
}
func (m *IntegerModularManager) normalizeWithArithmeticReceipt(out *Int, r *polynomialArithmeticInvocationReceipt) error {
	if m.manager.arithmeticAllocationRoots(r) == nil {
		return m.PNormalize(out)
	}
	e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
	if r.frame.sourceCause == nil {
		r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
	}
	return e
}
