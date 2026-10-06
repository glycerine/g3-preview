// Source leaves: src/util/mpz.cpp:allocate; src/util/buffer.h:expand.
// Adapter diagnostic records are materialized only at reached allocations.
package mpnum

import "reflect"

func (c integerAllocationInvocationContext) prepareFactorHistory(z *IntegerManager) error {
	if c.pn != nil && c.pn.owner != nil && c.pn.owner.self == c.pn.owner && c.pn == &c.pn.owner.normalization && c.pn.invocation == c.pnInvocation && c.pn.active && c.pn.executing && c.pn.owner.manager == z {
		return c.pn.factorHistoryContext.prepareFactorHistory(z)
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
