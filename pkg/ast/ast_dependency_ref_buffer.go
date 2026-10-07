// Concrete AST-valued util/ref_buffer.h specialization with actual inline16.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

type ASTDependencyRefBuffer struct {
	origin uint64
	buffer arena.SmallBytesRef
}
type astDependencyRefBufferState struct {
	size, capacity, initialized, inlineInitialized uint32
	decTarget, decNext                             uint32
	decPending                                     bool
	payload                                        arena.SmallBytesRef
}

const astDependencyRefBufferBytes uint32 = 160
const astDependencyRefBufferInlineOffset uint32 = 32
const astDependencyRefBufferTag uint32 = 0x44425546

func (m *ASTManager) dependencyBufferState(b ASTDependencyRefBuffer) (astDependencyRefBufferState, error) {
	if b.origin != m.origin || !m.allocator.IsLive(b.buffer) {
		return astDependencyRefBufferState{}, ErrASTHandle
	}
	bytes, e := m.allocator.Size(b.buffer)
	if e != nil || bytes != astDependencyRefBufferBytes {
		return astDependencyRefBufferState{}, ErrASTHandle
	}
	tag, e := m.readWord(b.buffer, 28)
	if e != nil || tag != astDependencyRefBufferTag {
		return astDependencyRefBufferState{}, ErrASTHandle
	}
	var a [7]uint32
	for i := range a {
		v, e := m.readWord(b.buffer, uint32(i)*4)
		if e != nil {
			return astDependencyRefBufferState{}, e
		}
		a[i] = v
	}
	s := astDependencyRefBufferState{size: a[0], capacity: a[1], payload: arena.SmallBytesRef{Slot: arena.Slot(a[2]), Generation: arena.Generation(a[3])}, inlineInitialized: a[4], initialized: a[4], decPending: a[5]&(1<<31) != 0, decTarget: a[5] &^ (uint32(1) << 31), decNext: a[6]}
	if !dependencyNull(s.payload) {
		var e error
		s.initialized, e = m.vectorWord(s.payload, 8)
		if e != nil {
			return s, e
		}
	}
	return s, nil
}
func (m *ASTManager) dependencyStoreBuffer(b ASTDependencyRefBuffer, s astDependencyRefBufferState) error {
	word, e := astDependencyRefCleanupWord(s.decTarget, s.decPending)
	if e != nil {
		return e
	}
	for i, v := range [7]uint32{s.size, s.capacity, uint32(s.payload.Slot), uint32(s.payload.Generation), s.inlineInitialized, word, s.decNext} {
		if e := m.writeWord(b.buffer, uint32(i)*4, v); e != nil {
			return e
		}
	}
	if !dependencyNull(s.payload) {
		if e := m.vectorPutWord(s.payload, 4, s.size); e != nil {
			return e
		}
		return m.vectorPutWord(s.payload, 8, s.initialized)
	}
	return nil
}
func (m *ASTManager) newDependencyRefBuffer() (ASTDependencyRefBuffer, error) {
	p, e := m.allocator.Allocate(astDependencyRefBufferBytes)
	if e != nil {
		return ASTDependencyRefBuffer{}, e
	}
	b := ASTDependencyRefBuffer{m.origin, p}
	m.writeWord(p, 28, astDependencyRefBufferTag)
	if e = m.dependencyStoreBuffer(b, astDependencyRefBufferState{capacity: 16}); e != nil {
		panic(e)
	}
	return b, nil
}
func (m *ASTManager) NewDependencyRefBuffer() (ASTDependencyRefBuffer, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefBuffer{}, e
	}
	defer m.mu.Unlock()
	return m.newDependencyRefBuffer()
}
func (m *ASTManager) dependencyBufferPhysical(b ASTDependencyRefBuffer, s astDependencyRefBufferState, i uint32) (arena.SmallBytesRef, error) {
	if i >= s.capacity {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	if i >= s.initialized {
		return arena.SmallBytesRef{}, ErrASTDomain
	}
	var lo, hi uint32
	var e error
	if dependencyNull(s.payload) {
		lo, e = m.readWord(b.buffer, astDependencyRefBufferInlineOffset+i*8)
		if e == nil {
			hi, e = m.readWord(b.buffer, astDependencyRefBufferInlineOffset+i*8+4)
		}
	} else {
		off := uint64(12) + uint64(i)*8
		if off+8 > math.MaxUint32 {
			return arena.SmallBytesRef{}, arena.ErrBounds
		}
		lo, e = m.vectorWord(s.payload, uint32(off))
		if e == nil {
			hi, e = m.vectorWord(s.payload, uint32(off)+4)
		}
	}
	return arena.SmallBytesRef{Slot: arena.Slot(lo), Generation: arena.Generation(hi)}, e
}
func (m *ASTManager) dependencyBufferValue(b ASTDependencyRefBuffer, s astDependencyRefBufferState, i uint32) (arena.SmallBytesRef, error) {
	if ASTSourceDebug && i >= s.size {
		return arena.SmallBytesRef{}, ErrASTDomain
	}
	return m.dependencyBufferPhysical(b, s, i)
}
func (m *ASTManager) dependencyBufferPut(b ASTDependencyRefBuffer, s astDependencyRefBufferState, i uint32, v arena.SmallBytesRef) error {
	if i >= s.capacity {
		return arena.ErrBounds
	}
	if dependencyNull(s.payload) {
		if e := m.writeWord(b.buffer, astDependencyRefBufferInlineOffset+i*8, uint32(v.Slot)); e != nil {
			return e
		}
		return m.writeWord(b.buffer, astDependencyRefBufferInlineOffset+i*8+4, uint32(v.Generation))
	}
	off := uint64(12) + uint64(i)*8
	if off+8 > math.MaxUint32 {
		return arena.ErrBounds
	}
	if e := m.vectorPutWord(s.payload, uint32(off), uint32(v.Slot)); e != nil {
		return e
	}
	return m.vectorPutWord(s.payload, uint32(off)+4, uint32(v.Generation))
}
func astDependencyBufferCapacity(old, hostMax uint64) (uint32, uint32, error) {
	if old > math.MaxUint32 {
		return 0, 0, arena.ErrBounds
	}
	n := uint32(old) << 1
	if uint64(n) <= old || uint64(n) > (math.MaxUint32-12)/8 || uint64(n)*8+12 > hostMax {
		return 0, 0, arena.ErrBounds
	}
	return n, n*8 + 12, nil
}
func (m *ASTManager) dependencyGrowBuffer(b ASTDependencyRefBuffer, s *astDependencyRefBufferState) error {
	n, bytes, e := astDependencyBufferCapacity(uint64(s.capacity), uint64(^uint(0)>>1))
	if e != nil {
		return e
	}
	p, e := m.vectorAllocator.Allocate(bytes)
	if e != nil {
		return e
	}
	next := *s
	next.payload = p
	next.capacity = n
	next.initialized = s.size
	m.vectorPutWord(p, 0, n)
	m.vectorPutWord(p, 4, s.size)
	m.vectorPutWord(p, 8, s.size)
	// Source buffer memcpy copies only logical m_pos, not the inactive tail.
	for i := uint32(0); i < s.size; i++ {
		v, e := m.dependencyBufferPhysical(b, *s, i)
		if e == nil {
			e = m.dependencyBufferPut(b, next, i, v)
		}
		if e != nil {
			m.vectorAllocator.Deallocate(bytes, p)
			return e
		}
	}
	if !dependencyNull(s.payload) {
		if e = m.vectorAllocator.Deallocate(s.capacity*8+12, s.payload); e != nil {
			return e
		}
	}
	*s = next
	return m.dependencyStoreBuffer(b, *s)
}
func (m *ASTManager) dependencyPushBuffer(b ASTDependencyRefBuffer, v arena.SmallBytesRef) error {
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if e = m.dependencies.inc(v); e != nil {
		return e
	}
	if s.size >= s.capacity {
		if e = m.dependencyGrowBuffer(b, &s); e != nil {
			return e
		}
	}
	if e = m.dependencyBufferPut(b, s, s.size, v); e != nil {
		return e
	}
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
	}
	if dependencyNull(s.payload) && s.initialized > s.inlineInitialized {
		s.inlineInitialized = s.initialized
	}
	return m.dependencyStoreBuffer(b, s)
}
func (m *ASTManager) DependencyRefBufferPush(b ASTDependencyRefBuffer, v ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(v)
	if e != nil {
		return e
	}
	return m.dependencyPushBuffer(b, r)
}
func (m *ASTManager) DependencyRefBufferSize(b ASTDependencyRefBuffer) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	return s.size, e
}
func (m *ASTManager) DependencyRefBufferCapacity(b ASTDependencyRefBuffer) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	return s.capacity, e
}
func (m *ASTManager) DependencyRefBufferEmpty(b ASTDependencyRefBuffer) (bool, error) {
	n, e := m.DependencyRefBufferSize(b)
	return n == 0, e
}
func (m *ASTManager) DependencyRefBufferManager(b ASTDependencyRefBuffer) (*ASTManager, error) {
	if _, e := m.DependencyRefBufferSize(b); e != nil {
		return nil, e
	}
	return m, nil
}
func (m *ASTManager) DependencyRefBufferGet(b ASTDependencyRefBuffer, i uint32) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return ASTDependency{}, e
	}
	v, e := m.dependencyBufferValue(b, s, i)
	return m.dependencies.external(v), e
}
func (m *ASTManager) DependencyRefBufferBack(b ASTDependencyRefBuffer) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return ASTDependency{}, e
	}
	if ASTSourceDebug && s.size == 0 {
		return ASTDependency{}, ErrASTDomain
	}
	v, e := m.dependencyBufferValue(b, s, s.size-1)
	return m.dependencies.external(v), e
}
func (m *ASTManager) dependencyBufferSet(b ASTDependencyRefBuffer, i uint32, v arena.SmallBytesRef) error {
	s, e := m.dependencyBufferState(b)
	if e != nil {
		return e
	}
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	if e = m.dependencies.inc(v); e != nil {
		return e
	}
	old, e := m.dependencyBufferValue(b, s, i)
	if e != nil {
		return e
	}
	consumed, e := m.dependencyArrays.valueDec(old)
	if !consumed {
		return e
	}
	if x := m.dependencyBufferPut(b, s, i, v); x != nil {
		return x
	}
	return e
}
func (m *ASTManager) DependencyRefBufferSet(b ASTDependencyRefBuffer, i uint32, v ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(v)
	if e != nil {
		return e
	}
	return m.dependencyBufferSet(b, i, r)
}
