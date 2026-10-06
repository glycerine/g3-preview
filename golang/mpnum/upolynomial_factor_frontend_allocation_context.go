// Source: src/math/polynomial/upolynomial.cpp:213-232,3042-3118;
// src/util/mpz.cpp:560-843; src/util/mpzzp.h:229-235.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) factorFrontendFrame(z *IntegerManager) *upolyFactorFrontendScope {
	q := c.factorFrontend
	if c.perfectSquare != nil && (z == nil || !z.matchesPerfectSquareInvocation(c.perfectSquare) || c.perfectSquareInvocation != c.perfectSquare.invocation || !c.perfectSquare.frame.executing) {
		return nil
	}
	if z == nil || z.self != z || q == nil || q.self != q || q.service == nil || q.service.self != q.service || q != &q.service.frontend || !q.active || (!q.executing && !q.sourceExecuting && !q.ownExecuting) || q.constructed < 2 || c.factorFrontendInvocation == 0 || c.factorFrontendInvocation != q.invocation {
		return nil
	}
	m := q.service.manager
	if m == nil || m.self != m || m.config.Integer != z || m.callbackDepth != 0 || q.service.closed {
		return nil
	}
	return q
}
func (m *IntegerModularManager) factorFrontendAllocationContext(q *upolyFactorFrontendScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || q.service == nil || q.service.manager == nil || q.service.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	c := integerAllocationInvocationContext{factorFrontend: q, factorFrontendInvocation: q.invocation}
	if c.factorFrontendFrame(m.manager) == nil {
		return integerAllocationInvocationContext{}
	}
	return c
}
func (c integerAllocationInvocationContext) recordFactorFrontendRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.factorFrontendFrame(z)
	if q == nil || !z.matchesRawInvocation(r) {
		return
	}
	switch r.frame {
	case &z.gcdRaw:
		q.gcdRaw = *r
	case &z.divisionRaw:
		q.divisionRaw = *r
	case &z.extendedRaw:
		q.extendedRaw = *r
	}
}

// History validates exact current transport identity independently of whether
// an older caller NUM has already been consumed by explicit owning cleanup.
func (c integerAllocationInvocationContext) factorFrontendHistoryFrame(z *IntegerManager) *upolyFactorFrontendScope {
	if c.perfectSquare != nil && (z == nil || !z.matchesPerfectSquareHistory(c.perfectSquare) || c.perfectSquareInvocation != c.perfectSquare.invocation) {
		return nil
	}
	q := c.factorFrontend
	if z == nil || z.self != z || q == nil || !q.actual() || !q.active || q.service.manager.config.Integer != z || c.factorFrontendInvocation == 0 || c.factorFrontendInvocation != q.invocation || !q.service.matchesFactorFrontendHistory(&q.receipt) {
		return nil
	}
	return q
}
