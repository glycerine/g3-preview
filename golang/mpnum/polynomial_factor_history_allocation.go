// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577; src/util/mpz.h:124-149.
// Source: src/util/mpz.h:124-149; src/util/buffer.h:39-46;
// src/math/polynomial/upolynomial.h:401-450.
// Adapter diagnostic records are materialized only at reached allocations.
package mpnum

import "reflect"

func (c integerAllocationInvocationContext) prepareFactorHistory(z *IntegerManager) error {
	if q := c.binaryEvaluationFrame(z); q != nil {
		return q.prepareHistory()
	}
	if c.pn != nil && c.pn.owner != nil && c.pn.owner.self == c.pn.owner && c.pn == &c.pn.owner.normalization && c.pn.invocation == c.pnInvocation && c.pn.active && c.pn.executing && c.pn.owner.manager == z {
		return c.pn.factorHistoryContext.prepareFactorHistory(z)
	}
	if q := c.algebraicNegFrame(z); q != nil {
		return q.prepareHistory()
	}
	if c.addMul != nil && c.addMul.manager == z && z.matchesAddMulInvocation(c.addMul) {
		return c.addMul.frame.parentContext.prepareFactorHistory(z)
	}
	if r := c.factorNumeralVector; r != nil && z.factorNumeralVectorRoots(c) != nil {
		return r.manager.archiveFactorNumeralVector()
	}
	if r := c.normalizationReceipt(z); r != nil {
		return prepareConverterParentHistory(z, &r.frame.converterHistoryParent)
	}
	if r := c.rawReceipt(z); r != nil && r.frame.normalization.manager != nil && z.normalizationAllocationRoots(&r.frame.normalization) != nil {
		return prepareConverterParentHistory(z, &r.frame.normalization.frame.converterHistoryParent)
	}
	if r := c.somNormalizeReceipt(z); r != nil {
		return r.manager.prepareNormalizationConverterHistoryAtAllocation(nil, nil, r)
	}
	if r := c.rawReceipt(z); r != nil && r.frame.somNormalize.manager != nil && z.somNormalizeAllocationRoots(&r.frame.somNormalize) != nil {
		return r.frame.somNormalize.manager.prepareNormalizationConverterHistoryAtAllocation(nil, nil, &r.frame.somNormalize)
	}
	if r := c.factorConstant; r != nil && z.factorConstantContextRoots(c) != nil {
		return r.manager.archiveFactorConstantHistory()
	}
	if r := c.factorMemberSet; r != nil && z.factorMemberSetRoots(c) != nil {
		f, s := r.owner, r.frame
		if s.historyRecorded {
			return nil
		}
		if len(f.memberSetHistory) == cap(f.memberSetHistory) {
			old := cap(f.memberSetHistory)
			next := old + old/2 + 1
			stride := uint64(reflect.TypeOf(polynomialFactorMemberSetHistory{}).Size())
			if e := z.budget.Reserve(uint64(next) * stride); e != nil {
				return e
			}
			rows := make([]polynomialFactorMemberSetHistory, len(f.memberSetHistory), next)
			copy(rows, f.memberSetHistory)
			f.memberSetHistory = rows
			z.budget.Release(uint64(old) * stride)
		}
		s.historyIndex, s.historyRecorded = len(f.memberSetHistory), true
		f.memberSetHistory = append(f.memberSetHistory, polynomialFactorMemberSetHistory{s.invocation, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause})
	}
	return nil
}
func (f *PolynomialFactors) updateFactorConstantHistory() {
	s := &f.constantInvocation
	if s.historyRecorded {
		f.constantHistory[s.historyIndex] = polynomialFactorConstantHistory{s.invocation, s.kind, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause}
	}
}
func (f *PolynomialFactors) updateFactorMemberSetHistory() {
	s := &f.memberSet
	if s.historyRecorded {
		f.memberSetHistory[s.historyIndex] = polynomialFactorMemberSetHistory{s.invocation, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause}
	}
}

func prepareConverterParentHistory(z *IntegerManager, r *polynomialFactorNumeralVectorReceipt) error {
	if r.manager == nil {
		return nil
	}
	c := integerAllocationInvocationContext{factorNumeralVector: r, factorNumeralVectorInvocation: r.invocation}
	if z.factorNumeralVectorRoots(c) == nil {
		return ErrPolynomialPending
	}
	return r.manager.archiveFactorNumeralVector()
}

// Diagnostic-only allocation records follow concrete reached normalization
// child tuples. They never acquire NUM, reference, vector or factory authority.
func (c *MultivariatePolynomialCore) prepareNormalizationConverterHistoryAtAllocation(v *polynomialVector, f *polynomialFactoryInvocationReceipt, som *polynomialSOMNormalizeInvocationReceipt) error {
	if c == nil || c.self != c || c.callbackDepth != 0 {
		return nil
	}
	r := c.normalizationOwner
	if r.manager == nil || r.manager.core != c || !r.manager.hasNormalizeInvocationHistory(&r) || !r.frame.active || !r.frame.executing {
		return nil
	}
	s := r.frame
	reached := false
	if v != nil && (v == &c.cheap.as || v == &c.cheap.ms) && c.cheap.matchesAppendInvocation(&s.append) {
		reached = true
	}
	if f != nil && c.matchesFactoryInvocation(f) && *f == s.factory {
		reached = true
	}
	if som != nil && c.hasSOMNormalizeInvocationHistory(som) && *som == s.som {
		reached = true
	}
	if !reached {
		return nil
	}
	return prepareConverterParentHistory(c.integer, &s.converterHistoryParent)
}
