// Source: src/math/polynomial/upolynomial_factorization.cpp:1058-1094;
// src/util/mpz.h:680-707; src/util/mpz.cpp:560-843;
// src/util/mpzzp.h:122-123,211-216,229-235.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) primePreparationFrame(z *IntegerManager) *upolyFactorPrimePreparationScope {
	q := c.primePreparation
	if z == nil || z.self != z || q == nil || !q.matches() || !q.active || !q.executing || q.owner.constructed < 2 || c.primePreparationInvocation == 0 || c.primePreparationInvocation != q.invocation || q.owner.service.scope != q.owner || q.owner.manager.config.Integer != z {
		return nil
	}
	for _, m := range q.managers() {
		if m == nil {
			continue
		}
		if m.self != m || m.config.Integer != z || m.callbackDepth != 0 {
			return nil
		}
	}
	m := c.primePreparationManager
	if m != q.owner.manager && m != q.related {
		return nil
	}
	return q
}
func (m *IntegerModularManager) primePreparationAllocationContext(q *upolyFactorPrimePreparationScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || !q.matches() {
		return integerAllocationInvocationContext{}
	}
	for _, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			c := integerAllocationInvocationContext{primePreparation: q, primePreparationInvocation: q.invocation, primePreparationManager: owner}
			if c.primePreparationFrame(m.manager) != nil {
				return c
			}
		}
	}
	return integerAllocationInvocationContext{}
}
func (q *upolyFactorPrimePreparationScope) normalizeReceipt(m *IntegerModularManager) *integerPNormalizeInvocationReceipt {
	for i, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			return &q.normalize[i]
		}
	}
	return nil
}

func (c integerAllocationInvocationContext) recordPrimePreparationRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	q := c.primePreparationFrame(z)
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
