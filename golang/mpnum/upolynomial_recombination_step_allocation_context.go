// Source: src/math/polynomial/upolynomial_factorization.cpp:1238-1292;
// src/math/polynomial/upolynomial.cpp:56-63,213-232,609-651;
// src/util/mpz.h:124-149; src/util/mpzzp.h:150-165.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) recombinationFrame(z *IntegerManager) *upolyRecombinationStepScope {
	q := c.recombination
	if z == nil || z.self != z || q == nil || !q.matches() || !q.active || !q.executing || c.recombinationInvocation == 0 || c.recombinationInvocation != q.invocation || q.owner.service.scope != q.owner || q.owner.manager.config.Integer != z {
		return nil
	}
	for _, m := range q.managers() {
		if m == nil || m.self != m || m.config.Integer != z || m.callbackDepth != 0 {
			return nil
		}
	}
	m := c.recombinationManager
	if m != q.owner.manager && m != q.related {
		return nil
	}
	return q
}
func (m *IntegerModularManager) recombinationAllocationContext(q *upolyRecombinationStepScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || !q.matches() {
		return integerAllocationInvocationContext{}
	}
	for _, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			c := integerAllocationInvocationContext{recombination: q, recombinationInvocation: q.invocation, recombinationManager: owner}
			if c.recombinationFrame(m.manager) != nil {
				return c
			}
		}
	}
	return integerAllocationInvocationContext{}
}
func (q *upolyRecombinationStepScope) normalizeReceipt(m *IntegerModularManager) *integerPNormalizeInvocationReceipt {
	for i, owner := range q.managers() {
		if owner.modular == m {
			return &q.normalize[i]
		}
	}
	return nil
}

func (c integerAllocationInvocationContext) recordRecombinationRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.recombinationFrame(z)
	if q == nil || !z.matchesRawInvocation(r) {
		return
	}
	switch r.frame {
	case &z.divisionRaw:
		q.divisionRaw = *r
	case &z.extendedRaw:
		q.extendedRaw = *r
	case &z.gcdRaw:
		q.gcdRaw = *r
	}
}

func (m *UPolynomialManager) mulScalarStateWithRecombination(s *upolyVectorState, b *Int, q *upolyRecombinationStepScope) error {
	context := m.modular.recombinationAllocationContext(q)
	if context.recombinationFrame(m.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	zero, e := m.modular.IsZero(b)
	if e != nil {
		return e
	}
	if zero {
		return m.reset(s)
	}
	return m.mulScalarViewWithSourceContext(m.stateView(s), b, context)
}
