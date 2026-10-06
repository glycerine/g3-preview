// Source: src/math/polynomial/polynomial.cpp:4507-4528;
// src/util/mpz.cpp; src/util/mpzzp.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// The passed receipt names the actual full manager's embedded derivative
// frame. Allocation leaves acquire only under its executing source invocation.
func (m *IntegerManager) derivativeAllocationRoots(r *polynomialDerivativeInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.self != r.manager || r.manager.core == nil || r.manager.core.self != r.manager.core || r.manager.core.integer != m || !r.manager.matchesDerivativeInvocation(r) || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}

func (m *IntegerModularManager) setUint32WithDerivativeReceipt(out *Int, v uint32, r *polynomialDerivativeInvocationReceipt) error {
	return m.setUint32WithSourceContext(out, v, m.derivativeAllocationContext(r))
}
func (m *IntegerModularManager) mulWithDerivativeReceipt(a, b, out *Int, r *polynomialDerivativeInvocationReceipt) error {
	return m.mulWithSourceContext(a, b, out, m.derivativeAllocationContext(r))
}
func (m *IntegerModularManager) derivativeAllocationContext(r *polynomialDerivativeInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.derivativeAllocationRoots(r) == nil || r.manager.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{derivative: r}
}

func (m *IntegerModularManager) normalizeWithSourceContext(out *Int, context integerAllocationInvocationContext) error {
	if q := context.upolyDisplayFrame(m.manager); q != nil && q.owner.modular == m {
		e := m.pNormalizeWithReceipt(out, &q.normalize)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(&q.normalize)
		}
		return e
	}

	if r := context.factorNumeralVector; r != nil && m.manager.factorNumeralVectorRoots(context) != nil && r.manager.upm.modular == m {
		e := m.pNormalizeWithFactorHistory(out, &r.frame.normalize, context)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.toPolynomial; r != nil && m.manager.toPolynomialAllocationRoots(context) != nil && r.manager.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.factorMemberSet; m.manager.factorMemberSetRoots(context) != nil && r.owner.manager.core.modular == m {
		e := m.pNormalizeWithFactorHistory(out, &r.frame.normalize, context)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.factorConstant; m.manager.factorConstantContextRoots(context) != nil && r.manager.manager.core.modular == m {
		child := &r.frame.normalize[r.frame.phase]
		e := m.pNormalizeWithFactorHistory(out, child, context)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}
	if b := context.factorFrontendConstantFrame(m.manager); b != nil && b.service.manager.modular == m {
		return m.pNormalizeWithFactorHistory(out, &b.normalize, context)
	}
	if q := context.factorFrontendFrame(m.manager); q != nil && q.service.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &q.normalize)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(&q.normalize)
		}
		return e
	}
	if q := context.preparationFrame(m.manager); q != nil && q.service.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &q.normalize)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(&q.normalize)
		}
		return e
	}
	if q := context.primePreparationFrame(m.manager); q != nil && context.primePreparationManager.modular == m {
		child := q.normalizeReceipt(m)
		e := m.pNormalizeWithReceipt(out, child)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}
	if q := context.recombinationFrame(m.manager); q != nil && context.recombinationManager.modular == m {
		child := q.normalizeReceipt(m)
		e := m.pNormalizeWithReceipt(out, child)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}
	if q := context.trialFrame(m.manager); q != nil && context.trialManager.modular == m {
		child := q.normalizeReceipt(m)
		e := m.pNormalizeWithReceipt(out, child)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}
	if q := context.henselCallerFrame(m.manager); q != nil && context.henselCallerManager.modular == m {
		child := q.normalizeReceipt(m)
		e := m.pNormalizeWithReceipt(out, child)
		if q.sourceCause == nil {
			q.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}

	if r := context.henselPairReceipt(m.manager); r != nil && context.henselManager.modular == m {
		child := r.frame.normalizeReceipt(m)
		e := m.pNormalizeWithReceipt(out, child)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(child)
		}
		return e
	}

	if r := context.iccpConstant; m.manager.iccpConstantContextRoots(context) != nil && r.manager.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.primeTrialReceipt(m.manager); r != nil && (r.frame.zp.modular == m || r.service.manager.modular == m) {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.pseudo; m.manager.pseudoContextRoots(context) != nil && r.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.mignotteReceipt(m.manager); r != nil && r.service.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.divides; m.manager.dividesContextRoots(context) != nil && r.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.neg; r != nil && context.negInvocation != 0 && context.negInvocation == r.invocation && m.manager.negAllocationRoots(r) != nil && r.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.negativeCleanupCause == nil {
			r.frame.negativeCleanupCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.scalarExact; m.manager.scalarExactContextRoots(context) != nil && r.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.exactDiv; m.manager.exactDivContextRoots(context) != nil && r.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}

	if r := context.somNormalizeReceipt(m.manager); r != nil && r.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.normalizationReceipt(m.manager); r != nil && r.manager.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.contentReceipt(m.manager); r != nil && r.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.cleanupCause == nil {
			r.frame.cleanupCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.scalarDiv; m.manager.scalarDivAllocationRoots(r) != nil && r.manager.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	if r := context.derivative; m.manager.derivativeAllocationRoots(r) != nil && r.manager.core.modular == m {
		e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
		}
		return e
	}
	return m.normalizeWithArithmeticReceipt(out, context.arithmeticReceipt(m.manager))
}
