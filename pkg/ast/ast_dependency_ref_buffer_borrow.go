package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Concrete physical binding: payload null means the owner's actual inline16.
// No reference counters are owned by these caller-owned scalar borrows.
type ASTDependencyRefBufferBorrow struct {
	origin         uint64
	owner, payload arena.SmallBytesRef
	index          uint32
}
type ASTDependencyRefBufferIterator struct {
	view ASTDependencyRefBufferBorrow
	end  uint32
}

func (m *ASTManager) dependencyBufferBorrowState(v ASTDependencyRefBufferBorrow) (ASTDependencyRefBuffer, astDependencyRefBufferState, error) {
	b := ASTDependencyRefBuffer{v.origin, v.owner}
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return b, s, e
	}
	s.payload = v.payload
	if dependencyNull(v.payload) {
		s.capacity = 16
		s.initialized = s.inlineInitialized
	} else {
		if !m.vectorAllocator.IsLive(v.payload) {
			return b, s, ErrASTHandle
		}
		s.capacity, e = m.vectorWord(v.payload, 0)
		if e != nil {
			return b, s, e
		}
		s.initialized, e = m.vectorWord(v.payload, 8)
	}
	return b, s, e
}
func (m *ASTManager) dependencyBufferBorrowValue(v ASTDependencyRefBufferBorrow) (arena.SmallBytesRef, error) {
	b, s, e := m.dependencyBufferBorrowState(v)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	return m.dependencyBufferPhysical(b, s, v.index)
}
func (m *ASTManager) DependencyRefBufferElement(b ASTDependencyRefBuffer, i uint32) (ASTDependencyRefBufferBorrow, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	if _, e = m.dependencyBufferPhysical(b, s, i); e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	return ASTDependencyRefBufferBorrow{m.origin, b.buffer, s.payload, i}, nil
}
func (m *ASTManager) DependencyRefBufferBorrowGet(v ASTDependencyRefBufferBorrow) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	r, e := m.dependencyBufferBorrowValue(v)
	return m.dependencies.external(r), e
}
func (m *ASTManager) DependencyRefBufferBorrowWriteRaw(v ASTDependencyRefBufferBorrow, value ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if !value.IsNull() && value.origin != m.origin {
		return ErrASTHandle
	}
	b, s, e := m.dependencyBufferBorrowState(v)
	if e != nil {
		return e
	}
	if _, e = m.dependencyBufferPhysical(b, s, v.index); e != nil {
		return e
	}
	return m.dependencyBufferPut(b, s, v.index, value.record)
}
func (m *ASTManager) DependencyRefBufferBegin(b ASTDependencyRefBuffer) (ASTDependencyRefBufferIterator, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefBufferIterator{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return ASTDependencyRefBufferIterator{}, e
	}
	return ASTDependencyRefBufferIterator{view: ASTDependencyRefBufferBorrow{origin: m.origin, owner: b.buffer, payload: s.payload}, end: s.size}, nil
}
func (m *ASTManager) DependencyRefBufferNext(it *ASTDependencyRefBufferIterator) (ASTDependency, bool, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, false, e
	}
	defer m.mu.Unlock()
	if it == nil || it.view.origin != m.origin {
		return ASTDependency{}, false, ErrASTHandle
	}
	if it.view.index >= it.end {
		return ASTDependency{}, false, nil
	}
	r, e := m.dependencyBufferBorrowValue(it.view)
	if e != nil {
		return ASTDependency{}, false, e
	}
	it.view.index++
	return m.dependencies.external(r), true, nil
}

func (m *ASTManager) DependencyRefBufferBackBorrow(b ASTDependencyRefBuffer) (ASTDependencyRefBufferBorrow, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	if ASTSourceDebug && s.size == 0 {
		return ASTDependencyRefBufferBorrow{}, ErrASTDomain
	}
	if _, e = m.dependencyBufferPhysical(b, s, s.size-1); e != nil {
		return ASTDependencyRefBufferBorrow{}, e
	}
	return ASTDependencyRefBufferBorrow{origin: m.origin, owner: b.buffer, payload: s.payload, index: s.size - 1}, nil
}
