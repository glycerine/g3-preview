// Source: src/util/mpz.cpp:big_set; src/util/mpz.h:allocate.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type integerCopyAllocationScope struct {
	invocation        uint64
	active, executing bool
	roots             polynomialNumericRoots
}
type integerCopyAllocationReceipt struct {
	manager    *IntegerManager
	frame      *integerCopyAllocationScope
	invocation uint64
}

func (m *IntegerManager) matchesCopyAllocationInvocation(r *integerCopyAllocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.copyAllocation && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *IntegerManager) cleanupCopyAllocationInvocation(r *integerCopyAllocationReceipt) error {
	if !m.matchesCopyAllocationInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := s.roots.cleanup(m); e != nil {
		return e
	}
	*s = integerCopyAllocationScope{invocation: s.invocation}
	return nil
}
