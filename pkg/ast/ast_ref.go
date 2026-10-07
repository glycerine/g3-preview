// Licensed under the MIT license.
// Concrete ast_ref specialization of src/util/obj_ref.h.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

func (m *ASTManager) reference(r ASTRef) (*astRefRecord, error) {
	if r.origin != m.origin || r.slot == 0 || r.slot >= m.nextReference {
		return nil, ErrASTHandle
	}
	v := m.referenceAt(r.slot)
	if !v.live || v.generation != r.generation {
		return nil, ErrASTHandle
	}
	return v, nil
}
func (m *ASTManager) allocateReference() (ASTRef, error) {
	for m.freeReference != 0 {
		slot := m.freeReference
		r := m.referenceAt(slot)
		m.freeReference = r.nextFree
		if r.generation == math.MaxUint32 {
			continue
		}
		gen := r.generation + 1
		*r = astRefRecord{generation: gen, live: true}
		m.liveReferences++
		return ASTRef{m.origin, slot, gen}, nil
	}
	if m.nextReference == math.MaxUint32 {
		return ASTRef{}, arena.ErrBounds
	}
	slot := m.nextReference
	if e := m.ensureRefPage(slot); e != nil {
		return ASTRef{}, e
	}
	m.nextReference++
	*m.referenceAt(slot) = astRefRecord{generation: 1, live: true}
	m.liveReferences++
	return ASTRef{m.origin, slot, 1}, nil
}
func (m *ASTManager) releaseReferenceRecord(ref ASTRef) {
	r := m.referenceAt(ref.slot)
	r.live = false
	r.node = astNodeRef{}
	r.nextFree = m.freeReference
	m.freeReference = ref.slot
	m.liveReferences--
}
func (m *ASTManager) NewRef(h ASTHandle) (ASTRef, error) {
	if e := m.lock(); e != nil {
		return ASTRef{}, e
	}
	defer m.mu.Unlock()
	var node astNodeRef
	var e error
	if !h.IsNull() {
		node, e = m.internal(h)
		if e != nil {
			return ASTRef{}, e
		}
	}
	ref, e := m.allocateReference()
	if e != nil {
		return ref, e
	}
	if e = m.incNode(node); e != nil {
		m.releaseReferenceRecord(ref)
		return ASTRef{}, e
	}
	m.referenceAt(ref.slot).node = node
	return ref, nil
}
func (m *ASTManager) RefGet(ref ASTRef) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.reference(ref)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.external(r.node), nil
}
func (m *ASTManager) CopyRef(source ASTRef) (ASTRef, error) {
	if e := m.lock(); e != nil {
		return ASTRef{}, e
	}
	defer m.mu.Unlock()
	src, e := m.reference(source)
	if e != nil {
		return ASTRef{}, e
	}
	node := src.node
	ref, e := m.allocateReference()
	if e != nil {
		return ref, e
	}
	if e = m.incNode(node); e != nil {
		m.releaseReferenceRecord(ref)
		return ASTRef{}, e
	}
	m.referenceAt(ref.slot).node = node
	return ref, nil
}
func (m *ASTManager) assignRef(dst ASTRef, node astNodeRef) error {
	r, e := m.reference(dst)
	if e != nil {
		return e
	}
	if e = m.incNode(node); e != nil {
		return e
	}
	consumed, e := m.decNode(r.node)
	if consumed {
		r.node = node
		return e
	}
	m.decNode(node)
	return e
}
func (m *ASTManager) AssignRef(dst, src ASTRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.reference(src)
	if e != nil {
		return e
	}
	return m.assignRef(dst, r.node)
}
func (m *ASTManager) SetRef(dst ASTRef, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	var r astNodeRef
	var e error
	if !h.IsNull() {
		r, e = m.internal(h)
		if e != nil {
			return e
		}
	}
	return m.assignRef(dst, r)
}
func (m *ASTManager) ResetRef(ref ASTRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.reference(ref)
	if e != nil {
		return e
	}
	consumed, e := m.decNode(r.node)
	if consumed {
		r.node = astNodeRef{}
	}
	return e
}
func (m *ASTManager) DelRef(ref *ASTRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if ref == nil {
		return ErrASTHandle
	}
	r, e := m.reference(*ref)
	if e != nil {
		return e
	}
	consumed, e := m.decNode(r.node)
	if consumed {
		m.releaseReferenceRecord(*ref)
		*ref = ASTRef{}
	}
	return e
}
func (m *ASTManager) MoveRef(source ASTRef) (ASTRef, error) {
	if e := m.lock(); e != nil {
		return ASTRef{}, e
	}
	defer m.mu.Unlock()
	src, e := m.reference(source)
	if e != nil {
		return ASTRef{}, e
	}
	node := src.node
	ref, e := m.allocateReference()
	if e != nil {
		return ref, e
	}
	m.referenceAt(ref.slot).node = node
	m.referenceAt(source.slot).node = astNodeRef{}
	return ref, nil
}
func (m *ASTManager) SwapRefs(a, b ASTRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	x, e := m.reference(a)
	if e != nil {
		return e
	}
	y, e := m.reference(b)
	if e != nil {
		return e
	}
	x.node, y.node = y.node, x.node
	return nil
}
func (m *ASTManager) StealRef(source ASTRef) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	src, e := m.reference(source)
	if e != nil {
		return ASTHandle{}, e
	}
	node := src.node
	src.node = astNodeRef{}
	return m.external(node), nil
}
