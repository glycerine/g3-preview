// Source: src/util/mpz.h:allocate; src/util/mpz.cpp scratch/publication callers.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// Explicit passed invocation identity. It carries no allocation snapshot and
// creates no source numeral, reservation, or ambient callback owner hook.
type integerAllocationInvocationContext struct {
	raw              *integerRawInvocationReceipt
	rawInvocation    uint64
	pn               *integerPNormalizeScope
	pnInvocation     uint64
	arithmetic       *polynomialArithmeticInvocationReceipt
	addMul           *integerAddMulInvocationReceipt
	addMulInvocation uint64
	derivative       *polynomialDerivativeInvocationReceipt
	scalarDiv        *polynomialScalarDivInvocationReceipt
}

func (c integerAllocationInvocationContext) withAddMulInvocation(r *integerAddMulInvocationReceipt) integerAllocationInvocationContext {
	c.addMul = r
	c.addMulInvocation = 0
	if r != nil {
		c.addMulInvocation = r.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) withRawInvocation(r *integerRawInvocationReceipt) integerAllocationInvocationContext {
	c.raw = r
	c.rawInvocation = 0
	if r != nil {
		c.rawInvocation = r.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) rawReceipt(m *IntegerManager) *integerRawInvocationReceipt {
	if c.raw != nil && c.rawInvocation != 0 && c.rawInvocation == c.raw.invocation && m.rawAllocationRoots(c.raw) != nil {
		return c.raw
	}
	return nil
}

func pnAllocationContext(pn *integerPNormalizeScope) integerAllocationInvocationContext {
	c := integerAllocationInvocationContext{pn: pn}
	if pn != nil {
		c.pnInvocation = pn.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) roots(m *IntegerManager) *polynomialNumericRoots {
	if c.raw != nil {
		if r := c.rawReceipt(m); r != nil {
			return m.rawAllocationRoots(r)
		}
		return nil
	}
	if s := c.pn; s != nil && s.owner != nil && s.owner.manager == m && s == &s.owner.normalization && s.active && s.executing && c.pnInvocation != 0 && c.pnInvocation == s.invocation {
		return &s.roots
	}
	if c.scalarDiv != nil {
		return m.scalarDivAllocationRoots(c.scalarDiv)
	}
	if c.derivative != nil {
		return m.derivativeAllocationRoots(c.derivative)
	}
	if c.addMul != nil && c.addMulInvocation != 0 && c.addMulInvocation == c.addMul.invocation && m.matchesAddMulInvocation(c.addMul) && c.addMul.frame.executing {
		s := c.addMul.frame
		if s.parentContext != (integerAllocationInvocationContext{}) && s.parentContext.roots(m) == nil {
			return nil
		}
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
