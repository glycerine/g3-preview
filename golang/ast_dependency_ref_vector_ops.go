// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

var ErrASTDependencyCleanupPending = errors.New("g3: interrupted dependency cleanup requires completion")

var ErrASTDependencySelfAppend = errors.New("g3: nonterminating source dependency vector self append")

func (m *ASTManager) dependencyAppendVector(dst, src ASTDependencyRefVector) error {
	if dst == src {
		state, e := m.dependencyVectorState(src)
		if e != nil {
			return e
		}
		if state.size != 0 {
			return ErrASTDependencySelfAppend
		}
	}
	for i := uint32(0); ; i++ {
		s, e := m.dependencyVectorState(src)
		if e != nil {
			return e
		}
		if i >= s.size {
			return nil
		}
		r, e := m.dependencyVectorValue(s, i)
		if e != nil {
			return e
		}
		if e = m.dependencyPushVector(dst, r); e != nil {
			return e
		}
	}
}
func (m *ASTManager) DependencyRefVectorAppend(dst, src ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyVectorState(dst); e != nil {
		return e
	}
	return m.dependencyAppendVector(dst, src)
}
func (m *ASTManager) CopyDependencyRefVector(src ASTDependencyRefVector) (ASTDependencyRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefVector{}, e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyVectorState(src); e != nil {
		return ASTDependencyRefVector{}, e
	}
	v, e := m.newDependencyRefVector()
	if e != nil {
		return ASTDependencyRefVector{}, e
	}
	if e = m.dependencyAppendVector(v, src); e != nil {
		return m.dependencyFailedVectorConstructor(v, e)
	}
	return v, nil
}
func (m *ASTManager) DependencyRefVectorAssign(dst, src ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyVectorState(dst); e != nil {
		return e
	}
	if _, e := m.dependencyVectorState(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.dependencyResizeVector(dst, 0); e != nil {
		return e
	}
	return m.dependencyAppendVector(dst, src)
}
func (m *ASTManager) DependencyRefVectorReset(v ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyResizeVector(v, 0)
}
func (m *ASTManager) dependencyFinalizeVector(v ASTDependencyRefVector) error {
	failure := m.dependencyResizeVector(v, 0)
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if failure != nil && s.size != 0 {
		return failure
	}
	if s.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+astDependencyRefVectorHeaderBytes, s.payload); e != nil {
			return e
		}
	}
	if e = m.dependencyStoreVector(v, astDependencyRefVectorState{}); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) DependencyRefVectorFinalize(v ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyFinalizeVector(v)
}
func (m *ASTManager) DependencyRefVectorShrink(v ASTDependencyRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if s.decPending {
		return m.dependencyResizeVector(v, n)
	}
	if astSourceDebug && n > s.size {
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
		if e = m.dependencyVectorPutWord(s.payload, 4, n); e != nil {
			return e
		}
		return m.dependencyStoreVector(v, s)
	}
	return m.dependencyResizeVector(v, n)
}
func (m *ASTManager) DependencyRefVectorResizeFill(v ASTDependencyRefVector, n uint32, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if n <= s.size {
		if n == s.size {
			return nil
		}
		return m.dependencyResizeVector(v, n)
	}
	r, e := m.dependencyVectorInput(h)
	if e != nil {
		return e
	}
	for i := s.size; i < n; i++ {
		if e = m.dependencyPushVector(v, r); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) DependencyRefVectorPop(v ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if s.size == 0 {
		if astSourceDebug {
			return ErrASTDomain
		}
		return arena.ErrBounds
	}
	r, e := m.dependencyVectorValue(s, s.size-1)
	if e != nil {
		return e
	}
	s.size--
	m.dependencyVectorPutWord(s.payload, 4, s.size)
	if e = m.dependencyStoreVector(v, s); e != nil {
		return e
	}
	_, e = m.dependencyArrays.valueDec(dependencyRefPointer(r))
	return e
}
func (m *ASTManager) dependencySwapVectorElements(s astDependencyRefVectorState, a, b uint32) error {
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	x, e := m.dependencyVectorValue(s, a)
	if e != nil {
		return e
	}
	y, e := m.dependencyVectorValue(s, b)
	if e != nil {
		return e
	}
	if e = m.dependencyVectorPut(s, a, y); e != nil {
		return e
	}
	return m.dependencyVectorPut(s, b, x)
}
func (m *ASTManager) DependencyRefVectorSwapElements(v ASTDependencyRefVector, a, b uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	return m.dependencySwapVectorElements(s, a, b)
}
func (m *ASTManager) DependencyRefVectorReverse(v ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	for i := uint32(0); i < s.size/2; i++ {
		if e = m.dependencySwapVectorElements(s, i, s.size-i-1); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) SwapDependencyRefVectors(a, b ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	x, e := m.dependencyVectorState(a)
	if e != nil {
		return e
	}
	y, e := m.dependencyVectorState(b)
	if e != nil {
		return e
	}
	if e = m.dependencyStoreVector(a, y); e != nil {
		return e
	}
	return m.dependencyStoreVector(b, x)
}
func (m *ASTManager) DependencyRefVectorContains(v ASTDependencyRefVector, h ASTDependency) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	if !h.IsNull() && h.origin != m.origin {
		return false, ErrASTHandle
	}
	r := astNodeRef{uint32(h.record.Slot), uint32(h.record.Generation)}
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < s.size; i++ {
		value, e := m.dependencyVectorValue(s, i)
		if e != nil {
			return false, e
		}
		if value == r {
			return true, nil
		}
	}
	return false, nil
}
func (m *ASTManager) DependencyRefVectorBack(v ASTDependencyRefVector) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return ASTDependency{}, e
	}
	if s.size == 0 {
		if astSourceDebug {
			return ASTDependency{}, ErrASTDomain
		}
		return ASTDependency{}, arena.ErrBounds
	}
	r, e := m.dependencyVectorValue(s, s.size-1)
	return m.dependencies.external(dependencyRefPointer(r)), e
}
func (m *ASTManager) DependencyRefVectorManager(v ASTDependencyRefVector) (*ASTManager, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyVectorState(v); e != nil {
		return nil, e
	}
	return m, nil
}

func (m *ASTManager) DependencyRefVectorCapacity(v ASTDependencyRefVector) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	return s.capacity, e
}

// A refused DEC leaves a charged, resumable owner visible to the caller.
// This exception-publication adapter does not undo source completed callbacks.
func (m *ASTManager) dependencyFailedVectorConstructor(v ASTDependencyRefVector, failure error) (ASTDependencyRefVector, error) {
	m.dependencyFinalizeVector(v)
	s, e := m.dependencyVectorState(v)
	if e != nil || s.size != 0 || s.decPending {
		return v, failure
	}
	if e = m.allocator.Deallocate(astDependencyRefVectorRecordBytes, v.record); e != nil {
		return v, failure
	}
	return ASTDependencyRefVector{}, failure
}
