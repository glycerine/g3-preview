// Source: src/util/mpzzp.h:58-68,111-114; src/util/mpz.cpp:quot_rem_core.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// Direct transport of actual integer-cell and source-buffer acquisition leaves.
// Whole-parent and exhaustive prefix validation remain open.
type integerPNormalizeScope struct {
	owner             *IntegerModularManager
	invocation        uint64
	stage             uint8
	active, executing bool
	roots             polynomialNumericRoots
	buffers           [4]naturalPNBufferReceipt
	verification      naturalPNVerificationReceipt
	sourceCause       error
}
type integerPNormalizeInvocationReceipt struct {
	manager    *IntegerModularManager
	frame      *integerPNormalizeScope
	invocation uint64
}

func (m *IntegerModularManager) matchesPNormalizeInvocation(r *integerPNormalizeInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.normalization && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *IntegerModularManager) cleanupPNormalizeInvocation(r *integerPNormalizeInvocationReceipt, abort bool) error {
	if !m.matchesPNormalizeInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.manager.natural.cleanupPNVerification(&s.verification, abort); e != nil {
		return e
	}
	if !abort {
		return nil
	}
	for i := len(s.buffers) - 1; i >= 0; i-- {
		if e := s.buffers[i].cleanup(); e != nil {
			return e
		}
	}
	if e := s.roots.cleanup(m.manager); e != nil {
		return e
	}
	*s = integerPNormalizeScope{owner: m, invocation: s.invocation, verification: s.verification, sourceCause: s.sourceCause}
	return nil
}

// A receipt exists only for a real pending growth of the named source buffer.
// Successful publication returns authority to the persistent Natural member.
type naturalPNBufferReceipt struct {
	owner      *NaturalManager
	context    *integerPNormalizeScope
	invocation uint64
	buffer     *sourceLimbBuffer
	pending    memory.Block
	stage      uint8
	private    bool
	published  memory.Block
}

func (m *NaturalManager) pnBufferReceipt(pn *integerPNormalizeScope, i int) *naturalPNBufferReceipt {
	if pn == nil || pn.owner == nil || pn.owner.manager.natural != m || pn != &pn.owner.normalization || !pn.active || !pn.executing {
		return nil
	}
	r := &pn.buffers[i]
	if r.context == nil {
		*r = naturalPNBufferReceipt{owner: m, context: pn, invocation: pn.invocation, buffer: &m.division[i], stage: pn.stage}
	}
	return r
}
func (r *naturalPNBufferReceipt) acquire(s *sourceLimbBuffer, block memory.Block) {
	if r.buffer == s && r.context.active && r.context.executing && r.invocation == r.context.invocation {
		r.pending = block
	}
}
func (r *naturalPNBufferReceipt) publish(s *sourceLimbBuffer, block memory.Block) {
	if r.buffer == s && r.pending == block {
		if r.private {
			r.published = block
		}
		r.pending = memory.Block{}
	}
}
func (r *naturalPNBufferReceipt) cleanup() error {
	if r.pending == (memory.Block{}) {
		return nil
	}
	if r.context == nil || !r.context.active || r.invocation != r.context.invocation || r.buffer.pending != r.pending {
		return nil
	}
	e := r.owner.runtime.Deallocate(&r.buffer.pending)
	if r.buffer.pending == (memory.Block{}) {
		r.pending = memory.Block{}
	}
	return e
}
func (s *integerPNormalizeScope) pending() bool {
	if s.owner.manager.natural.matchesPNVerification(&s.verification) {
		return true
	}
	if polynomialNegNumericPending(&s.roots) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (memory.Block{}) {
			return true
		}
	}
	return false
}
