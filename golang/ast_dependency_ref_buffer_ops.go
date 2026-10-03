// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

func (m *ASTManager) dependencyResizeBuffer(b ASTDependencyRefBuffer, n uint32) error {
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	var prior error
	if s.decPending {
		e = m.dependencyDrainBuffer(b, &s)
		if s.decPending {
			return e
		}
		prior = e
	}
	if prior != nil && n >= s.size {
		return prior
	}
	if n > s.size {
		for i := s.size; i < n; i++ {
			if e = m.dependencyPushBuffer(b, arena.SmallBytesRef{}); e != nil {
				return e
			}
		}
		return nil
	}
	if n == s.size {
		return nil
	}
	s.decPending = true
	s.decTarget = n
	s.decNext = n
	if e = m.dependencyStoreBuffer(b, s); e != nil {
		return e
	}
	e = m.dependencyDrainBuffer(b, &s)
	if prior != nil {
		return prior
	}
	return e
}
func (m *ASTManager) dependencyDrainBuffer(b ASTDependencyRefBuffer, s *astDependencyRefBufferState) error {
	var failure error
	for s.decNext < s.size {
		v, e := m.dependencyBufferPhysical(b, *s, s.decNext)
		if e != nil {
			if failure != nil {
				return failure
			}
			return e
		}
		consumed, e := m.dependencyArrays.valueDec(v)
		if !consumed {
			if failure != nil {
				return failure
			}
			return e
		}
		s.decNext++
		if x := m.dependencyStoreBuffer(b, *s); x != nil {
			return x
		}
		if failure == nil {
			failure = e
		}
	}
	s.size = s.decTarget
	s.decPending = false
	s.decTarget = 0
	s.decNext = 0
	if e := m.dependencyStoreBuffer(b, *s); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) DependencyRefBufferResize(b ASTDependencyRefBuffer, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyResizeBuffer(b, n)
}
func (m *ASTManager) DependencyRefBufferShrink(b ASTDependencyRefBuffer, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if astSourceDebug && n > s.size {
		return ErrASTDomain
	}
	return m.dependencyResizeBuffer(b, n)
}
func (m *ASTManager) DependencyRefBufferReset(b ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyResizeBuffer(b, 0)
}
func (m *ASTManager) dependencyFinalizeBuffer(b ASTDependencyRefBuffer) error {
	failure := m.dependencyResizeBuffer(b, 0)
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if failure != nil && s.size != 0 {
		return failure
	}
	if !dependencyNull(s.payload) {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+12, s.payload); e != nil {
			return e
		}
	}
	s.payload = arena.SmallBytesRef{}
	s.capacity = 16
	s.size = 0
	s.initialized = s.inlineInitialized
	if e = m.dependencyStoreBuffer(b, s); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) DependencyRefBufferFinalize(b ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyFinalizeBuffer(b)
}
func (m *ASTManager) DelDependencyRefBuffer(b *ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if b == nil {
		return ErrASTHandle
	}
	failure := m.dependencyFinalizeBuffer(*b)
	s, e := m.dependencyBufferState(*b)
	if e != nil {
		return e
	}
	if failure != nil && s.size != 0 {
		return failure
	}
	if e = m.allocator.Deallocate(astDependencyRefBufferBytes, b.buffer); e != nil {
		return e
	}
	*b = ASTDependencyRefBuffer{}
	return failure
}
func (m *ASTManager) DependencyRefBufferPop(b ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if astSourceDebug && s.size == 0 {
		return ErrASTDomain
	}
	v, e := m.dependencyBufferValue(b, s, s.size-1)
	if e != nil {
		return e
	}
	s.size--
	if e = m.dependencyStoreBuffer(b, s); e != nil {
		return e
	}
	_, e = m.dependencyArrays.valueDec(v)
	return e
}
func (m *ASTManager) DependencyRefBufferSetX(b ASTDependencyRefBuffer, i uint32, v ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if !v.IsNull() && v.origin != m.origin {
		return ErrASTHandle
	}
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if i >= s.size {
		if e = m.dependencyResizeBuffer(b, i+1); e != nil {
			return e
		}
	}
	return m.dependencyBufferSet(b, i, v.record)
}
func (m *ASTManager) dependencyAppendBuffer(dst, src ASTDependencyRefBuffer) error {
	s, e := m.dependencyBufferState(src)
	if e != nil {
		return e
	}
	// Source captures count and c_ptr once. Inline memory remains alive after
	// expansion; dynamic memory is checked only when the next captured read occurs.
	view := ASTDependencyRefBufferBorrow{origin: m.origin, owner: src.buffer, payload: s.payload}
	for i := uint32(0); i < s.size; i++ {
		view.index = i
		v, e := m.dependencyBufferBorrowValue(view)
		if e != nil {
			return e
		}
		if e = m.dependencyPushBuffer(dst, v); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) DependencyRefBufferAppend(dst, src ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyBufferState(dst); e != nil {
		return e
	}
	return m.dependencyAppendBuffer(dst, src)
}
func (m *ASTManager) DependencyRefBufferAppendValues(b ASTDependencyRefBuffer, values []ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyBufferState(b); e != nil {
		return e
	}
	for _, v := range values {
		r, e := m.dependencies.valid(v)
		if e != nil {
			return e
		}
		if e = m.dependencyPushBuffer(b, r); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) DependencyRefBufferAssign(dst, src ASTDependencyRefBuffer) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyBufferState(dst); e != nil {
		return e
	}
	if _, e := m.dependencyBufferState(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.dependencyResizeBuffer(dst, 0); e != nil {
		return e
	}
	return m.dependencyAppendBuffer(dst, src)
}
