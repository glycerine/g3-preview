// Licensed under the MIT license.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

func (m *ASTManager) appendVector(dst, src ASTRefVector) error {
	for i := uint32(0); ; i++ {
		s, e := m.vectorState(src)
		if e != nil {
			return e
		}
		if i >= s.size {
			return nil
		}
		r, e := m.vectorValue(s, i)
		if e != nil {
			return e
		}
		if e = m.pushVector(dst, r); e != nil {
			return e
		}
	}
}
func (m *ASTManager) RefVectorAppend(dst, src ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.vectorState(dst); e != nil {
		return e
	}
	return m.appendVector(dst, src)
}
func (m *ASTManager) CopyRefVector(src ASTRefVector) (ASTRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTRefVector{}, e
	}
	defer m.mu.Unlock()
	if _, e := m.vectorState(src); e != nil {
		return ASTRefVector{}, e
	}
	v, e := m.newRefVector()
	if e != nil {
		return ASTRefVector{}, e
	}
	if e = m.appendVector(v, src); e != nil {
		m.finalizeVector(v)
		m.allocator.Deallocate(astRefVectorRecordBytes, v.record)
		return ASTRefVector{}, e
	}
	return v, nil
}
func (m *ASTManager) RefVectorAssign(dst, src ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.vectorState(dst); e != nil {
		return e
	}
	if _, e := m.vectorState(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.resizeVector(dst, 0); e != nil {
		return e
	}
	return m.appendVector(dst, src)
}
func (m *ASTManager) RefVectorReset(v ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.resizeVector(v, 0)
}
func (m *ASTManager) finalizeVector(v ASTRefVector) error {
	failure := m.resizeVector(v, 0)
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	if s.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+astRefVectorHeaderBytes, s.payload); e != nil {
			return e
		}
	}
	if e = m.storeVector(v, astRefVectorState{}); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) RefVectorFinalize(v ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.finalizeVector(v)
}
func (m *ASTManager) RefVectorShrink(v ASTRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	if ASTSourceDebug && n > s.size {
		return ErrASTDomain
	}
	if n > s.size {
		if s.capacity == 0 {
			return nil
		} // source null backing publishes nothing.
		if n > s.capacity {
			return arena.ErrBounds
		}
		if n > s.initialized {
			return ErrASTDomain
		}
		// Source dec_range(begin+n,end) is empty when n exceeds old size.
		// Underlying scalar shrink publishes size without INC or construction.
		s.size = n
		if e = m.vectorPutWord(s.payload, 4, n); e != nil {
			return e
		}
		return m.storeVector(v, s)
	}
	return m.resizeVector(v, n)
}
func (m *ASTManager) RefVectorResizeFill(v ASTRefVector, n uint32, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	if n < s.size {
		return m.resizeVector(v, n)
	}
	r, e := m.vectorInput(h)
	if e != nil {
		return e
	}
	for i := s.size; i < n; i++ {
		if e = m.pushVector(v, r); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) RefVectorPop(v ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	if s.size == 0 {
		return arena.ErrBounds
	}
	r, e := m.vectorValue(s, s.size-1)
	if e != nil {
		return e
	}
	s.size--
	m.vectorPutWord(s.payload, 4, s.size)
	if e = m.storeVector(v, s); e != nil {
		return e
	}
	_, e = m.decNode(r)
	return e
}
func (m *ASTManager) swapVectorElements(s astRefVectorState, a, b uint32) error {
	x, e := m.vectorValue(s, a)
	if e != nil {
		return e
	}
	y, e := m.vectorValue(s, b)
	if e != nil {
		return e
	}
	if e = m.vectorPut(s, a, y); e != nil {
		return e
	}
	return m.vectorPut(s, b, x)
}
func (m *ASTManager) RefVectorSwapElements(v ASTRefVector, a, b uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	return m.swapVectorElements(s, a, b)
}
func (m *ASTManager) RefVectorReverse(v ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	for i := uint32(0); i < s.size/2; i++ {
		if e = m.swapVectorElements(s, i, s.size-i-1); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) SwapRefVectors(a, b ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	x, e := m.vectorState(a)
	if e != nil {
		return e
	}
	y, e := m.vectorState(b)
	if e != nil {
		return e
	}
	if e = m.storeVector(a, y); e != nil {
		return e
	}
	return m.storeVector(b, x)
}
func (m *ASTManager) RefVectorContains(v ASTRefVector, h ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.vectorInput(h)
	if e != nil {
		return false, e
	}
	s, e := m.vectorState(v)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < s.size; i++ {
		value, e := m.vectorValue(s, i)
		if e != nil {
			return false, e
		}
		if value == r {
			return true, nil
		}
	}
	return false, nil
}
func (m *ASTManager) RefVectorBack(v ASTRefVector) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTHandle{}, e
	}
	if s.size == 0 {
		return ASTHandle{}, arena.ErrBounds
	}
	r, e := m.vectorValue(s, s.size-1)
	return m.external(r), e
}
func (m *ASTManager) RefVectorManager(v ASTRefVector) (*ASTManager, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	if _, e := m.vectorState(v); e != nil {
		return nil, e
	}
	return m, nil
}
