// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

// These caller-owned scalar borrows model element_ref and c_ptr iterators.
// Growth/finalization invalidates their physical payload binding. Checked
// errors replace dereferencing dangling C++ pointers; no references are owned.
type ASTRefElement struct {
	origin  uint64
	payload arena.SmallBytesRef
	index   uint32
}
type ASTRefVectorIterator struct {
	origin     uint64
	payload    arena.SmallBytesRef
	index, end uint32
}

func (m *ASTManager) borrowedVectorState(origin uint64, payload arena.SmallBytesRef) (astRefVectorState, error) {
	if origin != m.origin || !m.vectorAllocator.IsLive(payload) {
		return astRefVectorState{}, ErrASTHandle
	}
	capacity, e := m.vectorWord(payload, 0)
	if e != nil {
		return astRefVectorState{}, e
	}
	size, e := m.vectorWord(payload, 4)
	if e != nil {
		return astRefVectorState{}, e
	}
	initialized, e := m.vectorWord(payload, 8)
	if e != nil {
		return astRefVectorState{}, e
	}
	return astRefVectorState{size: size, capacity: capacity, payload: payload, initialized: initialized}, nil
}
func (m *ASTManager) elementState(view ASTRefElement) (astRefVectorState, error) {
	s, e := m.borrowedVectorState(view.origin, view.payload)
	if e != nil {
		return s, e
	}
	if view.index >= s.capacity || view.index >= s.initialized {
		return astRefVectorState{}, ErrASTHandle
	}
	return s, nil
}
func (m *ASTManager) RefVectorElement(v ASTRefVector, index uint32) (ASTRefElement, error) {
	if e := m.lock(); e != nil {
		return ASTRefElement{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTRefElement{}, e
	}
	if _, e = m.vectorValue(s, index); e != nil {
		return ASTRefElement{}, e
	}
	return ASTRefElement{origin: m.origin, payload: s.payload, index: index}, nil
}
func (m *ASTManager) RefElementGet(view ASTRefElement) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.elementState(view)
	if e != nil {
		return ASTHandle{}, e
	}
	r, e := m.vectorPhysicalValue(s, view.index)
	return m.external(r), e
}
func (m *ASTManager) RefElementSet(view ASTRefElement, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.elementState(view)
	if e != nil {
		return e
	}
	if astSourceDebug && h.IsNull() {
		return ErrASTDomain
	} // element_ref::operator= SASSERT(n)
	r, e := m.vectorInput(h)
	if e != nil {
		return e
	}
	if e = m.incNode(r); e != nil {
		return e
	}
	old, e := m.vectorPhysicalValue(s, view.index)
	if e != nil {
		return e
	}
	consumed, failure := m.decNode(old)
	if failure != nil && !consumed {
		return failure
	}
	if e = m.vectorPut(s, view.index, r); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) RefElementAssign(dst, src ASTRefElement) error {
	h, e := m.RefElementGet(src)
	if e != nil {
		return e
	}
	return m.RefElementSet(dst, h)
}
func (m *ASTManager) RefElementEqual(view ASTRefElement, h ASTHandle) (bool, error) {
	got, e := m.RefElementGet(view)
	return got == h, e
}
func (m *ASTManager) RefVectorBegin(v ASTRefVector) (ASTRefVectorIterator, error) {
	if e := m.lock(); e != nil {
		return ASTRefVectorIterator{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTRefVectorIterator{}, e
	}
	return ASTRefVectorIterator{origin: m.origin, payload: s.payload, end: s.size}, nil
}
func (m *ASTManager) RefVectorNext(it *ASTRefVectorIterator) (ASTHandle, bool, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, false, e
	}
	defer m.mu.Unlock()
	if it == nil {
		return ASTHandle{}, false, ErrASTHandle
	}
	if it.origin != m.origin {
		return ASTHandle{}, false, ErrASTHandle
	}
	// A null begin/end of an empty source vector can be compared without
	// dereferencing storage, including after moving its vector object.
	if it.payload == (arena.SmallBytesRef{}) && it.index >= it.end {
		return ASTHandle{}, false, nil
	}
	s, e := m.borrowedVectorState(it.origin, it.payload)
	if e != nil {
		return ASTHandle{}, false, e
	}
	if it.index >= it.end {
		return ASTHandle{}, false, nil
	}
	r, e := m.vectorPhysicalValue(s, it.index)
	if e != nil {
		return ASTHandle{}, false, e
	}
	it.index++
	return m.external(r), true, nil
}
