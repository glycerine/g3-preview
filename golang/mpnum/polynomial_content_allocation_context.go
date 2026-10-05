// Source: src/math/polynomial/polynomial.cpp:3405-3430.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type polynomialContentInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialContentScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesContentInvocation(r *polynomialContentInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.contentLocal && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *IntegerManager) contentAllocationRoots(r *polynomialContentInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.manager == nil || r.manager.integer != m || !r.manager.matchesContentInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerModularManager) contentAllocationContext(r *polynomialContentInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.contentAllocationRoots(r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{content: r, contentInvocation: r.invocation}
}

func (c integerAllocationInvocationContext) contentReceipt(m *IntegerManager) *polynomialContentInvocationReceipt {
	if c.content != nil && c.contentInvocation != 0 && c.contentInvocation == c.content.invocation && m.contentAllocationRoots(c.content) != nil {
		return c.content
	}
	return nil
}
func (c integerAllocationInvocationContext) recordContentRaw(m *IntegerManager, r *integerRawInvocationReceipt) {
	p := c.contentReceipt(m)
	if p == nil || r == nil || !m.matchesRawInvocation(r) {
		return
	}
	if r.frame == &m.gcdRaw {
		p.frame.gcd = *r
	}
	r.frame.content = *p
	if r.frame == &m.divisionRaw {
		p.frame.division = *r
	}
	if r.frame == &m.extendedRaw {
		p.frame.extended = *r
	}
}

func (m *MultivariatePolynomialCore) cleanupContentInvocation(r *polynomialContentInvocationReceipt, abort bool) error {
	if !m.matchesContentInvocation(r) {
		return nil
	}
	return m.finishIntegerContentPhase(true, abort)
}
func (m *IntegerModularManager) gcdManyIndexedWithContentReceipt(as integerArrayView, g *Int, r *polynomialContentInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.gcdManyIndexedWithAllocationContext(as, g, m.contentAllocationContext(r)); e != nil {
		return e
	}
	return m.assertNormalized(g)
}

func (m *IntegerModularManager) copyWithContentReceipt(out, value *Int, r *polynomialContentInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.contentAllocationContext(r)
	if e := m.manager.copyFromOwnerWithAllocationContext(out, m.manager, value, nil, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}

func (m *MultivariatePolynomialCore) contentInvocationSourceCause(r *polynomialContentInvocationReceipt) error {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame != &m.contentLocal || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.cleanupCause
}
