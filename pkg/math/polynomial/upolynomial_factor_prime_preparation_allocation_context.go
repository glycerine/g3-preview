// Source: src/math/polynomial/upolynomial_factorization.cpp:1058-1094,1153-1175;
// src/util/mpz.h:680-707; src/util/mpz.cpp:560-843;
// src/util/mpzzp.h:122-123,211-216,229-235.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func (c integerAllocationInvocationContext) primePreparationFrame(z *util.IntegerManager) *upolyFactorPrimePreparationScope {
	q := c.primePreparation
	if z == nil || !z.ValidOwner() || q == nil || !q.matches() || !q.active || (!q.executing && !q.sourceExecuting) || q.owner.constructed < 2 || c.primePreparationInvocation == 0 || c.primePreparationInvocation != q.invocation || q.owner.service.scope != q.owner || q.owner.manager.config.Integer != z {
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
	if m != q.owner.manager && m != q.related && m != q.lift {
		return nil
	}
	return q
}
func numericIntegerModularManager_primePreparationAllocationContext(m *util.IntegerModularManager, q *upolyFactorPrimePreparationScope) integerAllocationInvocationContext {
	if m == nil || !m.ValidOwner() || m.Integers() == nil || q == nil || !q.matches() {
		return integerAllocationInvocationContext{}
	}
	for _, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			c := integerAllocationInvocationContext{primePreparation: q, primePreparationInvocation: q.invocation, primePreparationManager: owner}
			if c.primePreparationFrame(m.Integers()) != nil {
				return c
			}
		}
	}
	return integerAllocationInvocationContext{}
}
func (q *upolyFactorPrimePreparationScope) normalizeReceipt(m *util.IntegerModularManager) *util.IntegerNormalizeInvocationReceipt {
	for i, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			return &q.normalize[i]
		}
	}
	return nil
}

func (c integerAllocationInvocationContext) recordPrimePreparationRaw(z *util.IntegerManager, r *util.IntegerRawInvocationReceipt) {
	q := c.primePreparationFrame(z)
	if q == nil || !z.MatchesRawInvocation(r) {
		return
	}
	switch r.Kind() {
	case util.IntegerRawDivision:
		q.divisionRaw = *r
	case util.IntegerRawExtendedGCD:
		q.extendedRaw = *r
	case util.IntegerRawGCD:
		q.gcdRaw = *r
	}
}
