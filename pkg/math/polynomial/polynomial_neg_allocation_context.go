// Source: src/math/polynomial/polynomial.cpp:5940-5951;
// src/util/mpzzp.h:58-68,131; src/util/mpz.h:124-149.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func numericIntegerManager_negAllocationRoots(m *util.IntegerManager, r *polynomialNegInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.self != r.manager || r.manager.integer != m || !r.manager.matchesNegInvocation(r) || !r.frame.negativeExecuting || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func numericIntegerModularManager_negAllocationContext(m *util.IntegerModularManager, r *polynomialNegInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.Integers() == nil || numericIntegerManager_negAllocationRoots(m.Integers(), r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{neg: r, negInvocation: r.invocation}
}
func numericIntegerModularManager_copyWithNegReceipt(m *util.IntegerModularManager, out, value *util.Int, r *polynomialNegInvocationReceipt) error {
	if e := m.Check(); e != nil {
		return e
	}
	context := numericIntegerModularManager_negAllocationContext(m, r)
	if e := m.Integers().WithAllocationContext((context).utility(m.Integers())).CopyFromOwnerTracked(out, m.Integers(), value, nil); e != nil {
		return e
	}
	return numericIntegerModularManager_normalizeWithSourceContext(m, out, context)
}
func numericIntegerModularManager_negWithNegReceipt(m *util.IntegerModularManager, out *util.Int, r *polynomialNegInvocationReceipt) error {
	if e := m.Check(); e != nil {
		return e
	}
	if e := m.Integers().Neg(out); e != nil {
		return e
	}
	// PN context is validated only at this reached source call, after NEG.
	return numericIntegerModularManager_normalizeWithSourceContext(m, out, numericIntegerModularManager_negAllocationContext(m, r))
}
