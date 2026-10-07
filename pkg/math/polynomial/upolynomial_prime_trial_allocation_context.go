// Source: src/math/polynomial/upolynomial_factorization.cpp:1097-1145;
// src/math/polynomial/upolynomial_factorization_int.h:42-52.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj), 2011-11-15.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func (c integerAllocationInvocationContext) primeTrialReceipt(m *util.IntegerManager) *upolyPrimeTrialInvocationReceipt {
	r := c.primeTrial
	if m == nil || !m.ValidOwner() || r == nil || r.service == nil || !r.service.matchesPrimeTrialInvocation(r) || r.frame.zp.config.Integer != m || (!r.frame.executing && !r.frame.sourceExecuting) || r.frame.constructed == 0 || (r.frame.zp.callbackDepth != 0 || r.service.manager.callbackDepth != 0) || c.primeTrialInvocation == 0 || c.primeTrialInvocation != r.invocation {
		return nil
	}
	return r
}
func numericIntegerModularManager_primeTrialAllocationContext(m *util.IntegerModularManager, r *upolyPrimeTrialInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.Integers() == nil || r == nil || r.frame == nil || r.frame.zp == nil || (r.frame.zp.modular != m && (r.service == nil || r.service.manager == nil || r.service.manager.modular != m)) {
		return integerAllocationInvocationContext{}
	}
	c := integerAllocationInvocationContext{primeTrial: r, primeTrialInvocation: r.invocation}
	if c.primeTrialReceipt(m.Integers()) == nil {
		return integerAllocationInvocationContext{}
	}
	return c
}
func (m *UPolynomialManager) toZPCopyWithPrimeReceipt(p UPolynomialView, out *upolyVectorState, r *upolyPrimeTrialInvocationReceipt) error {
	if r == nil || r.service == nil || !r.service.matchesPrimeTrialInvocation(r) || r.frame.zp != m || out != &r.frame.fzp || (!r.frame.executing && !r.frame.sourceExecuting) || r.frame.transportBound {
		return ErrUPolynomialOwner
	}
	r.frame.enterCall()
	defer r.frame.leaveCall()
	return m.toZPCopyEngine(p, out, r, nil)
}
