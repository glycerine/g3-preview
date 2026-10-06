// Source: src/math/polynomial/upolynomial_factorization.cpp:1030-1050;
// src/util/mpz.h:124-149; src/util/mpzzp.h:131-132,229-235.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) preparationFrame(z *IntegerManager) *upolyFactorPreparationScope {
	q := c.preparation
	if z == nil || z.self != z || q == nil || !q.matches(&q.receipt) || !q.active || !q.executing || c.preparationInvocation == 0 || c.preparationInvocation != q.invocation {
		return nil
	}
	m := q.service.manager
	if m.config.Integer != z || m.callbackDepth != 0 || q.service.closed {
		return nil
	}
	if q.parent != nil && q.service.scope != q.parent {
		return nil
	}
	return q
}
func (m *IntegerModularManager) preparationAllocationContext(q *upolyFactorPreparationScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || !q.matches(&q.receipt) || q.service.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	c := integerAllocationInvocationContext{preparation: q, preparationInvocation: q.invocation}
	if c.preparationFrame(m.manager) == nil {
		return integerAllocationInvocationContext{}
	}
	return c
}
func (c integerAllocationInvocationContext) recordPreparationRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.preparationFrame(z)
	if q != nil && z.matchesRawInvocation(r) && r.frame == &z.gcdRaw {
		q.gcdRaw = *r
	}
}
