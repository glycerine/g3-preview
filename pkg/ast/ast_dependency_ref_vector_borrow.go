package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

// These caller-owned scalar borrows model element_ref and c_ptr iterators.
// Growth/finalization invalidates their physical payload binding. Checked
// errors replace dereferencing dangling C++ pointers; no references are owned.
type ASTDependencyRefElement struct {
	origin  uint64
	payload arena.SmallBytesRef
	index   uint32
}
type ASTDependencyRefVectorIterator struct {
	origin     uint64
	payload    arena.SmallBytesRef
	index, end uint32
}

func (m *ASTManager) dependencyBorrowedVectorState(origin uint64, payload arena.SmallBytesRef) (astDependencyRefVectorState, error) {
	if origin != m.origin || !m.vectorAllocator.IsLive(payload) {
		return astDependencyRefVectorState{}, ErrASTHandle
	}
	capacity, e := m.dependencyVectorWord(payload, 0)
	if e != nil {
		return astDependencyRefVectorState{}, e
	}
	size, e := m.dependencyVectorWord(payload, 4)
	if e != nil {
		return astDependencyRefVectorState{}, e
	}
	initialized, e := m.dependencyVectorWord(payload, 8)
	if e != nil {
		return astDependencyRefVectorState{}, e
	}
	return astDependencyRefVectorState{size: size, capacity: capacity, payload: payload, initialized: initialized}, nil
}
func (m *ASTManager) dependencyElementState(view ASTDependencyRefElement) (astDependencyRefVectorState, error) {
	s, e := m.dependencyBorrowedVectorState(view.origin, view.payload)
	if e != nil {
		return s, e
	}
	if view.index >= s.capacity || view.index >= s.initialized {
		return astDependencyRefVectorState{}, ErrASTHandle
	}
	return s, nil
}
func (m *ASTManager) DependencyRefVectorElement(v ASTDependencyRefVector, index uint32) (ASTDependencyRefElement, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefElement{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return ASTDependencyRefElement{}, e
	}
	if _, e = m.dependencyVectorValue(s, index); e != nil {
		return ASTDependencyRefElement{}, e
	}
	return ASTDependencyRefElement{origin: m.origin, payload: s.payload, index: index}, nil
}
func (m *ASTManager) DependencyRefElementGet(view ASTDependencyRefElement) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyElementState(view)
	if e != nil {
		return ASTDependency{}, e
	}
	r, e := m.dependencyVectorPhysicalValue(s, view.index)
	return m.dependencies.external(dependencyRefPointer(r)), e
}
func (m *ASTManager) DependencyRefElementSet(view ASTDependencyRefElement, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyElementState(view)
	if e != nil {
		return e
	}
	if ASTSourceDebug && h.IsNull() {
		return ErrASTDomain
	} // element_ref::operator= SASSERT(n)
	r, e := m.dependencyVectorInput(h)
	if e != nil {
		return e
	}
	if e = m.dependencies.inc(dependencyRefPointer(r)); e != nil {
		return e
	}
	old, e := m.dependencyVectorPhysicalValue(s, view.index)
	if e != nil {
		return e
	}
	consumed, failure := m.dependencyArrays.valueDec(dependencyRefPointer(old))
	if failure != nil && !consumed {
		return failure
	}
	if e = m.dependencyVectorPut(s, view.index, r); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) DependencyRefElementAssign(dst, src ASTDependencyRefElement) error {
	h, e := m.DependencyRefElementGet(src)
	if e != nil {
		return e
	}
	return m.DependencyRefElementSet(dst, h)
}
func (m *ASTManager) DependencyRefElementEqual(view ASTDependencyRefElement, h ASTDependency) (bool, error) {
	got, e := m.DependencyRefElementGet(view)
	return got == h, e
}
func (m *ASTManager) DependencyRefVectorBegin(v ASTDependencyRefVector) (ASTDependencyRefVectorIterator, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefVectorIterator{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return ASTDependencyRefVectorIterator{}, e
	}
	return ASTDependencyRefVectorIterator{origin: m.origin, payload: s.payload, end: s.size}, nil
}
func (m *ASTManager) DependencyRefVectorNext(it *ASTDependencyRefVectorIterator) (ASTDependency, bool, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, false, e
	}
	defer m.mu.Unlock()
	if it == nil {
		return ASTDependency{}, false, ErrASTHandle
	}
	if it.origin != m.origin {
		return ASTDependency{}, false, ErrASTHandle
	}
	// A null begin/end of an empty source vector can be compared without
	// dereferencing storage, including after moving its vector object.
	if it.payload == (arena.SmallBytesRef{}) && it.index >= it.end {
		return ASTDependency{}, false, nil
	}
	s, e := m.dependencyBorrowedVectorState(it.origin, it.payload)
	if e != nil {
		return ASTDependency{}, false, e
	}
	if it.index >= it.end {
		return ASTDependency{}, false, nil
	}
	r, e := m.dependencyVectorPhysicalValue(s, it.index)
	if e != nil {
		return ASTDependency{}, false, e
	}
	it.index++
	return m.dependencies.external(dependencyRefPointer(r)), true, nil
}
