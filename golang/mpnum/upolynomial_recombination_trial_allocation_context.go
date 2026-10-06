// Source: src/math/polynomial/upolynomial_factorization.cpp:1213-1240;
// src/math/polynomial/upolynomial_factorization_int.h:345-413;
// src/util/mpz.h:124-149; src/util/mpzzp.h:150-163.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) trialFrame(z *IntegerManager) *upolyRecombinationTrialScope {
	q := c.trial
	if z == nil || z.self != z || q == nil || !q.matches() || !q.active || !q.executing || !q.constructed || c.trialInvocation == 0 || c.trialInvocation != q.invocation || q.owner.service.scope != q.owner || q.owner.manager.config.Integer != z {
		return nil
	}
	for _, m := range q.managers() {
		if m == nil || m.self != m || m.config.Integer != z || m.callbackDepth != 0 {
			return nil
		}
	}
	m := c.trialManager
	if m != q.owner.manager && m != q.factors.manager {
		return nil
	}
	return q
}
func (m *IntegerModularManager) trialAllocationContext(q *upolyRecombinationTrialScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || !q.matches() {
		return integerAllocationInvocationContext{}
	}
	for _, owner := range q.managers() {
		if owner != nil && owner.modular == m {
			c := integerAllocationInvocationContext{trial: q, trialInvocation: q.invocation, trialManager: owner}
			if c.trialFrame(m.manager) != nil {
				return c
			}
		}
	}
	return integerAllocationInvocationContext{}
}
func (q *upolyRecombinationTrialScope) normalizeReceipt(m *IntegerModularManager) *integerPNormalizeInvocationReceipt {
	for i, owner := range q.managers() {
		if owner.modular == m {
			return &q.normalize[i]
		}
	}
	return nil
}
