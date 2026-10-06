// Source: src/math/polynomial/upolynomial_factorization.cpp:703-801,845-933;
// src/util/mpz.h:124-149; src/util/mpzzp.h:46-56,154,230-233.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj), 2011-11-15.
package mpnum

func (q *upolyHenselCallerScope) numericOwners() [4]*UPolynomialManager {
	if q == nil || q.owner == nil {
		return [4]*UPolynomialManager{}
	}
	return [4]*UPolynomialManager{q.owner.manager, q.related, q.prime, q.lift}
}
func (q *upolyHenselCallerScope) ownsNumericManager(m *UPolynomialManager) bool {
	for _, o := range q.numericOwners() {
		if o == m {
			return true
		}
	}
	return false
}
func (c integerAllocationInvocationContext) henselCallerFrame(z *IntegerManager) *upolyHenselCallerScope {
	q := c.henselCaller
	if z == nil || z.self != z || q == nil || !q.matches() || q.invocation == 0 || c.henselCallerInvocation != q.invocation || !q.executing || q.owner.manager.config.Integer != z {
		return nil
	}
	m := c.henselCallerManager
	if m == nil || m.self != m || !q.ownsNumericManager(m) || m.config.Integer != z {
		return nil
	}
	for _, o := range q.numericOwners() {
		if o != nil && o.callbackDepth != 0 {
			return nil
		}
	}
	return q
}
func (m *IntegerModularManager) henselCallerAllocationContext(q *upolyHenselCallerScope) integerAllocationInvocationContext {
	if m == nil || m.self != m || m.manager == nil || q == nil || q.owner == nil {
		return integerAllocationInvocationContext{}
	}
	for _, o := range q.numericOwners() {
		if o != nil && o.modular == m {
			c := integerAllocationInvocationContext{henselCaller: q, henselCallerInvocation: q.invocation, henselCallerManager: o}
			if c.henselCallerFrame(m.manager) != nil {
				return c
			}
		}
	}
	return integerAllocationInvocationContext{}
}
func (q *upolyHenselCallerScope) normalizeReceipt(m *IntegerModularManager) *integerPNormalizeInvocationReceipt {
	for i, o := range q.numericOwners() {
		if o != nil && o.modular == m {
			return &q.normalize[i]
		}
	}
	return nil
}
func (z *IntegerManager) bindHenselCallerRaw(c integerAllocationInvocationContext, r *integerRawInvocationReceipt, division bool) {
	if q := c.henselCallerFrame(z); q != nil {
		if division {
			q.divisionRaw = *r
		} else {
			q.extendedRaw = *r
		}
		r.frame.henselCallerContext = c
	}
}

func (q *upolyHenselCallerScope) numericManagers() [4]*IntegerModularManager {
	var out [4]*IntegerModularManager
	for i, m := range q.numericOwners() {
		if m != nil {
			out[i] = m.modular
		}
	}
	return out
}
