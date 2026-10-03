// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Actual non-owning ptr_vector<expr> output; codecs are shared with the concrete
// ref vector, but these methods never invoke its reference-count callbacks.
package g3

import "github.com/glycerine/g3/golang/arena"

type ASTExprVector struct {
	origin uint64
	record arena.SmallBytesRef
}

func (v ASTExprVector) codec() ASTRefVector { return ASTRefVector{v.origin, v.record} }
func (m *ASTManager) newExprVector() (ASTExprVector, error) {
	v, e := m.newRefVector()
	return ASTExprVector{v.origin, v.record}, e
}
func (m *ASTManager) exprVectorPush(v ASTExprVector, r astNodeRef) error {
	s, e := m.vectorState(v.codec())
	if e != nil {
		return e
	}
	if s.size == s.capacity {
		if e = m.growVector(v.codec(), &s); e != nil {
			return e
		}
	}
	if e = m.vectorPut(s, s.size, r); e != nil {
		return e
	}
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
		if e = m.vectorPutWord(s.payload, 8, s.initialized); e != nil {
			return e
		}
	}
	if e = m.vectorPutWord(s.payload, 4, s.size); e != nil {
		return e
	}
	return m.storeVector(v.codec(), s)
}
func (m *ASTManager) delExprVector(v *ASTExprVector) error {
	if v == nil {
		return ErrASTHandle
	}
	s, e := m.vectorState(v.codec())
	if e != nil {
		return e
	}
	if s.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+astRefVectorHeaderBytes, s.payload); e != nil {
			return e
		}
	}
	if e = m.allocator.Deallocate(astRefVectorRecordBytes, v.record); e != nil {
		return e
	}
	*v = ASTExprVector{}
	return nil
}
func (m *ASTManager) NewExprVector() (ASTExprVector, error) {
	if e := m.lock(); e != nil {
		return ASTExprVector{}, e
	}
	defer m.mu.Unlock()
	return m.newExprVector()
}
func (m *ASTManager) ExprVectorPush(v ASTExprVector, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	var r astNodeRef
	var e error
	if !h.IsNull() {
		r, e = m.requireExpr(h)
		if e != nil {
			return e
		}
	}
	return m.exprVectorPush(v, r)
}
func (m *ASTManager) ExprVectorGet(v ASTExprVector, i uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v.codec())
	if e != nil {
		return ASTHandle{}, e
	}
	r, e := m.vectorValue(s, i)
	if e != nil {
		return ASTHandle{}, e
	}
	if r.slot != 0 {
		if e = m.validNode(r); e != nil {
			return ASTHandle{}, e
		}
	}
	return m.external(r), nil
}
func (m *ASTManager) ExprVectorSize(v ASTExprVector) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v.codec())
	return s.size, e
}
func (m *ASTManager) ExprVectorReset(v ASTExprVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v.codec())
	if e != nil {
		return e
	}
	s.size = 0
	if s.capacity != 0 {
		if e = m.vectorPutWord(s.payload, 4, 0); e != nil {
			return e
		}
	}
	return m.storeVector(v.codec(), s)
}
func (m *ASTManager) DelExprVector(v *ASTExprVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.delExprVector(v)
}
