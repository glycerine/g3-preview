// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete expr_dependency_ref_vector specialization; encoded cold owners, scalar entries.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

type ASTDependencyRefVector struct {
	origin uint64
	record arena.SmallBytesRef
}
type astDependencyRefVectorState struct {
	size, capacity     uint32
	decTarget, decNext uint32
	decPending         bool
	payload            arena.SmallBytesRef
	initialized        uint32
}

const astDependencyRefVectorRecordBytes uint32 = 24
const astDependencyRefVectorHeaderBytes uint32 = 12

func (m *ASTManager) dependencyVectorState(v ASTDependencyRefVector) (astDependencyRefVectorState, error) {
	if v.origin != m.origin || !m.allocator.IsLive(v.record) {
		return astDependencyRefVectorState{}, ErrASTHandle
	}
	var words [6]uint32
	for i := range words {
		x, e := m.readWord(v.record, uint32(i)*4)
		if e != nil {
			return astDependencyRefVectorState{}, e
		}
		words[i] = x
	}
	s := astDependencyRefVectorState{size: words[0], capacity: words[1], payload: arena.SmallBytesRef{Slot: arena.Slot(words[2]), Generation: arena.Generation(words[3])}, decPending: words[4]&(1<<31) != 0, decTarget: words[4] &^ (uint32(1) << 31), decNext: words[5]}
	if s.capacity != 0 {
		var e error
		s.initialized, e = m.dependencyVectorWord(s.payload, 8)
		if e != nil {
			return astDependencyRefVectorState{}, e
		}
	}
	return s, nil
}
func (m *ASTManager) dependencyStoreVector(v ASTDependencyRefVector, s astDependencyRefVectorState) error {
	word, e := astDependencyRefCleanupWord(s.decTarget, s.decPending)
	if e != nil {
		return e
	}
	for i, x := range [6]uint32{s.size, s.capacity, uint32(s.payload.Slot), uint32(s.payload.Generation), word, s.decNext} {
		if e := m.writeWord(v.record, uint32(i)*4, x); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) dependencyVectorWord(p arena.SmallBytesRef, offset uint32) (uint32, error) {
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
func (m *ASTManager) dependencyVectorPutWord(p arena.SmallBytesRef, offset, x uint32) error {
	for i := uint32(0); i < 4; i++ {
		if e := m.vectorAllocator.Write(p, offset+i, byte(x>>(i*8))); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) dependencyVectorValue(s astDependencyRefVectorState, i uint32) (astNodeRef, error) {
	if astSourceDebug && i >= s.size {
		return astNodeRef{}, ErrASTDomain
	}
	return m.dependencyVectorPhysicalValue(s, i)
}
func (m *ASTManager) dependencyVectorPhysicalValue(s astDependencyRefVectorState, i uint32) (astNodeRef, error) {
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
	if offset > math.MaxUint32-astDependencyRefVectorHeaderBytes-8 {
		return astNodeRef{}, arena.ErrBounds
	}
	slot, e := m.dependencyVectorWord(s.payload, offset+astDependencyRefVectorHeaderBytes)
	if e != nil {
		return astNodeRef{}, e
	}
	g, e := m.dependencyVectorWord(s.payload, offset+astDependencyRefVectorHeaderBytes+4)
	return astNodeRef{slot, g}, e
}
func (m *ASTManager) dependencyVectorPut(s astDependencyRefVectorState, i uint32, r astNodeRef) error {
	if i >= s.capacity {
		return arena.ErrBounds
	}
	offset, e := astPayloadOffset(i, 8)
	if e != nil || offset > math.MaxUint32-astDependencyRefVectorHeaderBytes-8 {
		return arena.ErrBounds
	}
	if e = m.dependencyVectorPutWord(s.payload, offset+astDependencyRefVectorHeaderBytes, r.slot); e != nil {
		return e
	}
	return m.dependencyVectorPutWord(s.payload, offset+astDependencyRefVectorHeaderBytes+4, r.generation)
}
func (m *ASTManager) newDependencyRefVector() (ASTDependencyRefVector, error) {
	p, e := m.allocator.Allocate(astDependencyRefVectorRecordBytes)
	if e != nil {
		return ASTDependencyRefVector{}, e
	}
	v := ASTDependencyRefVector{m.origin, p}
	if e = m.dependencyStoreVector(v, astDependencyRefVectorState{}); e != nil {
		m.allocator.Deallocate(astDependencyRefVectorRecordBytes, p)
		return ASTDependencyRefVector{}, e
	}
	return v, nil
}
func (m *ASTManager) NewDependencyRefVector() (ASTDependencyRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefVector{}, e
	}
	defer m.mu.Unlock()
	return m.newDependencyRefVector()
}
func (m *ASTManager) dependencyGrowVector(v ASTDependencyRefVector, s *astDependencyRefVectorState) error {
	n := uint64(2)
	if s.capacity != 0 {
		n = (3*uint64(s.capacity) + 1) >> 1
	}
	if n > uint64(math.MaxUint32-8)/8 || n <= uint64(s.capacity) {
		return arena.ErrBounds
	}
	if n > uint64(math.MaxUint32-astDependencyRefVectorHeaderBytes)/8 {
		return arena.ErrBounds
	}
	p, e := m.vectorAllocator.Allocate(uint32(n)*8 + astDependencyRefVectorHeaderBytes)
	if e != nil {
		return e
	}
	next := astDependencyRefVectorState{size: s.size, capacity: uint32(n), payload: p, initialized: s.initialized}
	m.dependencyVectorPutWord(p, 0, uint32(n))
	m.dependencyVectorPutWord(p, 4, s.size)
	m.dependencyVectorPutWord(p, 8, s.initialized)
	// Source trivial pointer reallocation preserves all physical capacity words,
	// including inactive payloads, without retaining/freeing any dependency references.
	for i := uint32(0); i < s.capacity; i++ {
		for off := uint32(0); off < 8; off++ {
			b, e := m.vectorAllocator.Read(s.payload, astDependencyRefVectorHeaderBytes+i*8+off)
			if e != nil {
				m.vectorAllocator.Deallocate(uint32(n)*8+astDependencyRefVectorHeaderBytes, p)
				return e
			}
			if e = m.vectorAllocator.Write(p, astDependencyRefVectorHeaderBytes+i*8+off, b); e != nil {
				m.vectorAllocator.Deallocate(uint32(n)*8+astDependencyRefVectorHeaderBytes, p)
				return e
			}
		}
	}
	if s.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+astDependencyRefVectorHeaderBytes, s.payload); e != nil {
			return e
		}
	}
	*s = next
	return m.dependencyStoreVector(v, *s)
}
func (m *ASTManager) dependencyPushVector(v ASTDependencyRefVector, r astNodeRef) error {
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if e = m.dependencies.inc(dependencyRefPointer(r)); e != nil {
		return e
	}
	// Preserve source's retained INC prefix on growth failure. There is no
	// source-missing cleanup on vector retry/Close; manager teardown owns it.
	if s.size == s.capacity {
		if e = m.dependencyGrowVector(v, &s); e != nil {
			return e
		}
	}
	if e = m.dependencyVectorPut(s, s.size, r); e != nil {
		return e
	}
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
		m.dependencyVectorPutWord(s.payload, 8, s.initialized)
	}
	m.dependencyVectorPutWord(s.payload, 4, s.size)
	return m.dependencyStoreVector(v, s)
}
func (m *ASTManager) dependencyVectorInput(h ASTDependency) (astNodeRef, error) {
	if h.IsNull() {
		return astNodeRef{}, nil
	}
	r, e := m.dependencies.valid(h)
	return astNodeRef{uint32(r.Slot), uint32(r.Generation)}, e
}
func (m *ASTManager) DependencyRefVectorPush(v ASTDependencyRefVector, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencyVectorInput(h)
	if e != nil {
		return e
	}
	return m.dependencyPushVector(v, r)
}
func (m *ASTManager) DependencyRefVectorSize(v ASTDependencyRefVector) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	return s.size, e
}
func (m *ASTManager) DependencyRefVectorGet(v ASTDependencyRefVector, i uint32) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return ASTDependency{}, e
	}
	r, e := m.dependencyVectorValue(s, i)
	return m.dependencies.external(dependencyRefPointer(r)), e
}
func (m *ASTManager) DependencyRefVectorSet(v ASTDependencyRefVector, i uint32, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	r, e := m.dependencyVectorInput(h)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if e = m.dependencies.inc(dependencyRefPointer(r)); e != nil {
		return e
	}
	old, e := m.dependencyVectorValue(s, i)
	if e != nil {
		return e
	}
	consumed, err := m.dependencyArrays.valueDec(dependencyRefPointer(old))
	if !consumed {
		return err
	}
	if e = m.dependencyVectorPut(s, i, r); e != nil {
		return e
	}
	return err
}
func (m *ASTManager) dependencyResizeVector(v ASTDependencyRefVector, n uint32) error {
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	var prior error
	if s.decPending {
		e = m.dependencyDrainVector(v, &s)
		if s.decPending {
			return e
		}
		prior = e
	}
	if n < s.size {
		s.decPending = true
		s.decTarget = n
		s.decNext = n
		if e = m.dependencyStoreVector(v, s); e != nil {
			return e
		}
		e = m.dependencyDrainVector(v, &s)
		if prior != nil {
			return prior
		}
		return e
	}
	if prior != nil {
		return prior
	}
	for n > s.capacity {
		if e = m.dependencyGrowVector(v, &s); e != nil {
			return e
		}
	}
	for i := s.size; i < n; i++ {
		if e = m.dependencyVectorPut(s, i, astNodeRef{}); e != nil {
			return e
		}
	}
	s.size = n
	if n > s.initialized {
		s.initialized = n
		m.dependencyVectorPutWord(s.payload, 8, n)
	}
	if s.capacity != 0 {
		m.dependencyVectorPutWord(s.payload, 4, n)
	}
	return m.dependencyStoreVector(v, s)
}
func (m *ASTManager) dependencyDrainVector(v ASTDependencyRefVector, s *astDependencyRefVectorState) error {
	var failure error
	for s.decNext < s.size {
		r, e := m.dependencyVectorPhysicalValue(*s, s.decNext)
		if e != nil {
			if failure != nil {
				return failure
			}
			return e
		}
		consumed, e := m.dependencyArrays.valueDec(dependencyRefPointer(r))
		if !consumed {
			if failure != nil {
				return failure
			}
			return e
		}
		s.decNext++
		if x := m.dependencyStoreVector(v, *s); x != nil {
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
	if s.capacity != 0 {
		if e := m.dependencyVectorPutWord(s.payload, 4, s.size); e != nil {
			return e
		}
	}
	if e := m.dependencyStoreVector(v, *s); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) DependencyRefVectorReserve(v ASTDependencyRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil || n <= s.size {
		return e
	}
	return m.dependencyResizeVector(v, n)
}
func (m *ASTManager) DependencyRefVectorResize(v ASTDependencyRefVector, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyResizeVector(v, n)
}
func (m *ASTManager) DependencyRefVectorErase(v ASTDependencyRefVector, i uint32) error {
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
	old, e := m.dependencyVectorValue(s, i)
	if e != nil {
		return e
	}
	if i >= s.size {
		return ErrASTDomain
	} // source erase's forward !=end loop has no valid continuation.
	for j := i + 1; j < s.size; j++ {
		r, e := m.dependencyVectorValue(s, j)
		if e != nil {
			return e
		}
		if e = m.dependencyVectorPut(s, j-1, r); e != nil {
			return e
		}
	}
	s.size--
	m.dependencyVectorPutWord(s.payload, 4, s.size)
	if e = m.dependencyStoreVector(v, s); e != nil {
		return e
	}
	_, e = m.dependencyArrays.valueDec(dependencyRefPointer(old))
	return e
}
func (m *ASTManager) DelDependencyRefVector(v *ASTDependencyRefVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if v == nil {
		return ErrASTHandle
	}
	failure := m.dependencyResizeVector(*v, 0)
	s, e := m.dependencyVectorState(*v)
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
	if e = m.allocator.Deallocate(astDependencyRefVectorRecordBytes, v.record); e != nil {
		return e
	}
	*v = ASTDependencyRefVector{}
	return failure
}

func dependencyRefPointer(r astNodeRef) arena.SmallBytesRef {
	return arena.SmallBytesRef{Slot: arena.Slot(r.slot), Generation: arena.Generation(r.generation)}
}
