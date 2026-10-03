// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete ast_ref_vector specialization; encoded cold owners, scalar entries.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

type ASTRefVector struct {
	origin uint64
	record arena.SmallBytesRef
}
type astRefVectorState struct {
	size, capacity uint32
	payload        arena.SmallBytesRef
	initialized    uint32
}

const astRefVectorRecordBytes uint32 = 16
const astRefVectorHeaderBytes uint32 = 12

func (m *ASTManager) vectorState(v ASTRefVector) (astRefVectorState, error) {
	if v.origin != m.origin || !m.allocator.IsLive(v.record) {
		return astRefVectorState{}, ErrASTHandle
	}
	var words [4]uint32
	for i := range words {
		x, e := m.readWord(v.record, uint32(i)*4)
		if e != nil {
			return astRefVectorState{}, e
		}
		words[i] = x
	}
	s := astRefVectorState{size: words[0], capacity: words[1], payload: arena.SmallBytesRef{Slot: arena.Slot(words[2]), Generation: arena.Generation(words[3])}}
	if s.capacity != 0 {
		var e error
		s.initialized, e = m.vectorWord(s.payload, 8)
		if e != nil {
			return astRefVectorState{}, e
		}
	}
	return s, nil
}
func (m *ASTManager) storeVector(v ASTRefVector, s astRefVectorState) error {
	for i, x := range [4]uint32{s.size, s.capacity, uint32(s.payload.Slot), uint32(s.payload.Generation)} {
		if e := m.writeWord(v.record, uint32(i)*4, x); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) vectorWord(p arena.SmallBytesRef, offset uint32) (uint32, error) {
	var x uint32
	for i := uint32(0); i < 4; i++ {
		b, e := m.vectorAllocator.Read(p, offset+i)
		if e != nil {
			return 0, e
		}
		x |= uint32(b) << (i * 8)
	}
	return x, nil
}
func (m *ASTManager) vectorPutWord(p arena.SmallBytesRef, offset, x uint32) error {
	for i := uint32(0); i < 4; i++ {
		if e := m.vectorAllocator.Write(p, offset+i, byte(x>>(i*8))); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) vectorValue(s astRefVectorState, i uint32) (astNodeRef, error) {
	if astSourceDebug && i >= s.size {
		return astNodeRef{}, ErrASTDomain
	}
	return m.vectorPhysicalValue(s, i)
}
func (m *ASTManager) vectorPhysicalValue(s astRefVectorState, i uint32) (astNodeRef, error) {
	if i >= s.capacity {
		return astNodeRef{}, arena.ErrBounds
	}
	if i >= s.initialized {
		return astNodeRef{}, ErrASTDomain
	}
	offset, e := astPayloadOffset(i, 8)
	if e != nil {
		return astNodeRef{}, e
	}
	if offset > math.MaxUint32-astRefVectorHeaderBytes-8 {
		return astNodeRef{}, arena.ErrBounds
	}
	slot, e := m.vectorWord(s.payload, offset+astRefVectorHeaderBytes)
	if e != nil {
		return astNodeRef{}, e
	}
	g, e := m.vectorWord(s.payload, offset+astRefVectorHeaderBytes+4)
	return astNodeRef{slot, g}, e
}
func (m *ASTManager) vectorPut(s astRefVectorState, i uint32, r astNodeRef) error {
	if i >= s.capacity {
		return arena.ErrBounds
	}
	offset, e := astPayloadOffset(i, 8)
	if e != nil || offset > math.MaxUint32-astRefVectorHeaderBytes-8 {
		return arena.ErrBounds
	}
	if e = m.vectorPutWord(s.payload, offset+astRefVectorHeaderBytes, r.slot); e != nil {
		return e
	}
	return m.vectorPutWord(s.payload, offset+astRefVectorHeaderBytes+4, r.generation)
}
func (m *ASTManager) newRefVector() (ASTRefVector, error) {
	p, e := m.allocator.Allocate(astRefVectorRecordBytes)
	if e != nil {
		return ASTRefVector{}, e
	}
	v := ASTRefVector{m.origin, p}
	if e = m.storeVector(v, astRefVectorState{}); e != nil {
		m.allocator.Deallocate(astRefVectorRecordBytes, p)
		return ASTRefVector{}, e
	}
	return v, nil
}
func (m *ASTManager) NewRefVector() (ASTRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTRefVector{}, e
	}
	defer m.mu.Unlock()
	return m.newRefVector()
}
func (m *ASTManager) growVector(v ASTRefVector, s *astRefVectorState) error {
	n := uint64(2)
	if s.capacity != 0 {
		n = (3*uint64(s.capacity) + 1) >> 1
	}
	if n > uint64(math.MaxUint32-8)/8 || n <= uint64(s.capacity) {
		return arena.ErrBounds
	}
	if n > uint64(math.MaxUint32-astRefVectorHeaderBytes)/8 {
		return arena.ErrBounds
	}
	p, e := m.vectorAllocator.Allocate(uint32(n)*8 + astRefVectorHeaderBytes)
	if e != nil {
		return e
	}
	next := astRefVectorState{size: s.size, capacity: uint32(n), payload: p, initialized: s.initialized}
	m.vectorPutWord(p, 0, uint32(n))
	m.vectorPutWord(p, 4, s.size)
	m.vectorPutWord(p, 8, s.initialized)
	// Source trivial pointer reallocation preserves all physical capacity words,
	// including inactive payloads, without retaining/freeing any AST references.
	for i := uint32(0); i < s.capacity; i++ {
		for off := uint32(0); off < 8; off++ {
			b, e := m.vectorAllocator.Read(s.payload, astRefVectorHeaderBytes+i*8+off)
			if e != nil {
				m.vectorAllocator.Deallocate(uint32(n)*8+astRefVectorHeaderBytes, p)
				return e
			}
			if e = m.vectorAllocator.Write(p, astRefVectorHeaderBytes+i*8+off, b); e != nil {
				m.vectorAllocator.Deallocate(uint32(n)*8+astRefVectorHeaderBytes, p)
				return e
			}
		}
	}
	if s.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+astRefVectorHeaderBytes, s.payload); e != nil {
			return e
		}
	}
	*s = next
	return m.storeVector(v, *s)
}
func (m *ASTManager) pushVector(v ASTRefVector, r astNodeRef) error {
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	if e = m.incNode(r); e != nil {
		return e
	}
	// Preserve source's retained INC prefix on growth failure. There is no
	// source-missing cleanup on vector retry/Close; manager teardown owns it.
	if s.size == s.capacity {
		if e = m.growVector(v, &s); e != nil {
			return e
		}
	}
	if e = m.vectorPut(s, s.size, r); e != nil {
		return e
	}
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
		m.vectorPutWord(s.payload, 8, s.initialized)
	}
	m.vectorPutWord(s.payload, 4, s.size)
	return m.storeVector(v, s)
}
func (m *ASTManager) vectorInput(h ASTHandle) (astNodeRef, error) {
	if h.IsNull() {
		return astNodeRef{}, nil
	}
	return m.internal(h)
}
func (m *ASTManager) RefVectorPush(v ASTRefVector, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.vectorInput(h)
	if e != nil {
		return e
	}
	return m.pushVector(v, r)
}
func (m *ASTManager) RefVectorSize(v ASTRefVector) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	return s.size, e
}
func (m *ASTManager) RefVectorGet(v ASTRefVector, i uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTHandle{}, e
	}
	r, e := m.vectorValue(s, i)
	return m.external(r), e
}
func (m *ASTManager) RefVectorSet(v ASTRefVector, i uint32, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	r, e := m.vectorInput(h)
	if e != nil {
		return e
	}
	if e = m.incNode(r); e != nil {
		return e
	}
	old, e := m.vectorValue(s, i)
	if e != nil {
		return e
	}
	consumed, err := m.decNode(old)
	if !consumed {
		return err
	}
	if e = m.vectorPut(s, i, r); e != nil {
		return e
	}
	return err
}
func (m *ASTManager) resizeVector(v ASTRefVector, n uint32) error {
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	var failure error
	if n < s.size {
		for i := n; i < s.size; i++ {
			r, e := m.vectorValue(s, i)
			if e == nil {
				_, e = m.decNode(r)
			}
			if failure == nil {
				failure = e
			}
		}
	} else {
		for n > s.capacity {
			if e = m.growVector(v, &s); e != nil {
				return e
			}
		}
		for i := s.size; i < n; i++ {
			if e = m.vectorPut(s, i, astNodeRef{}); e != nil {
				return e
			}
		}
	}
	s.size = n
	if n > s.initialized {
		s.initialized = n
		m.vectorPutWord(s.payload, 8, n)
	}
	if s.capacity != 0 {
		m.vectorPutWord(s.payload, 4, n)
	}
	if e = m.storeVector(v, s); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) RefVectorReserve(v ASTRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil || n <= s.size {
		return e
	}
	return m.resizeVector(v, n)
}
func (m *ASTManager) RefVectorResize(v ASTRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.resizeVector(v, n)
}
func (m *ASTManager) RefVectorErase(v ASTRefVector, i uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return e
	}
	old, e := m.vectorValue(s, i)
	if e != nil {
		return e
	}
	if i >= s.size {
		return ErrASTDomain
	} // source erase's forward !=end loop has no valid continuation.
	for j := i + 1; j < s.size; j++ {
		r, e := m.vectorValue(s, j)
		if e != nil {
			return e
		}
		if e = m.vectorPut(s, j-1, r); e != nil {
			return e
		}
	}
	s.size--
	m.vectorPutWord(s.payload, 4, s.size)
	if e = m.storeVector(v, s); e != nil {
		return e
	}
	_, e = m.decNode(old)
	return e
}
func (m *ASTManager) DelRefVector(v *ASTRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if v == nil {
		return ErrASTHandle
	}
	if e := m.resizeVector(*v, 0); e != nil {
		return e
	}
	s, e := m.vectorState(*v)
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
	*v = ASTRefVector{}
	return nil
}
