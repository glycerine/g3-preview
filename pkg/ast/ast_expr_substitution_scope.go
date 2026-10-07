package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

const astScopedExprSubstBytes uint32 = 48
const astScopedExprSubstMagic uint32 = 0x53455355

type astScopedExprSubstState struct {
	subst                                         ASTExprSubstitution
	trail                                         ASTRefVector
	limits                                        arena.SmallBytesRef
	size, capacity, initialized, operation, index uint32
}

func (m *ASTManager) scopedSubstState(s ASTScopedExprSubstitution) (astScopedExprSubstState, error) {
	if s.origin != m.origin {
		return astScopedExprSubstState{}, ErrASTHandle
	}
	if size, e := m.allocator.Size(s.record); e != nil || size != 48 {
		return astScopedExprSubstState{}, ErrASTHandle
	}
	var a [12]uint32
	for i := range a {
		x, e := m.readWord(s.record, uint32(i)*4)
		if e != nil {
			return astScopedExprSubstState{}, e
		}
		a[i] = x
	}
	if a[11] != astScopedExprSubstMagic {
		return astScopedExprSubstState{}, ErrASTHandle
	}
	return astScopedExprSubstState{ASTExprSubstitution{m.origin, arena.SmallBytesRef{Slot: arena.Slot(a[0]), Generation: arena.Generation(a[1])}}, ASTRefVector{m.origin, arena.SmallBytesRef{Slot: arena.Slot(a[2]), Generation: arena.Generation(a[3])}}, arena.SmallBytesRef{Slot: arena.Slot(a[4]), Generation: arena.Generation(a[5])}, a[6], a[7], a[8], a[9], a[10]}, nil
}
func (m *ASTManager) scopedSubstStore(s ASTScopedExprSubstitution, a astScopedExprSubstState) error {
	for i, x := range [12]uint32{uint32(a.subst.record.Slot), uint32(a.subst.record.Generation), uint32(a.trail.record.Slot), uint32(a.trail.record.Generation), uint32(a.limits.Slot), uint32(a.limits.Generation), a.size, a.capacity, a.initialized, a.operation, a.index, astScopedExprSubstMagic} {
		if e := m.writeWord(s.record, uint32(i)*4, x); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) NewScopedExprSubstitution(subst ASTExprSubstitution) (ASTScopedExprSubstitution, error) {
	if e := m.lock(); e != nil {
		return ASTScopedExprSubstitution{}, e
	}
	defer m.mu.Unlock()
	if _, e := m.substState(subst); e != nil {
		return ASTScopedExprSubstitution{}, e
	}
	r, e := m.allocator.Allocate(48)
	if e != nil {
		return ASTScopedExprSubstitution{}, e
	}
	trail, e := m.newRefVector()
	if e != nil {
		m.allocator.Deallocate(48, r)
		return ASTScopedExprSubstitution{}, e
	}
	s := ASTScopedExprSubstitution{m.origin, r}
	if e = m.scopedSubstStore(s, astScopedExprSubstState{subst: subst, trail: trail}); e != nil {
		m.allocator.Deallocate(astRefVectorRecordBytes, trail.record)
		m.allocator.Deallocate(48, r)
		return ASTScopedExprSubstitution{}, e
	}
	return s, nil
}
func (m *ASTManager) scopedSubstLimit(a astScopedExprSubstState, i uint32) (uint32, error) {
	if ASTSourceDebug && i >= a.size {
		return 0, ErrASTDomain
	}
	if i >= a.capacity {
		return 0, arena.ErrBounds
	}
	if i >= a.initialized {
		return 0, ErrASTDomain
	}
	return m.vectorWord(a.limits, 12+i*4)
}
func astScopedExprSubstCapacity(old uint32, hostMax uint64) (uint32, error) {
	n := uint32(2)
	if old != 0 {
		n = (3*old + 1) >> 1
		if n <= old || n*4+8 <= old*4+8 {
			return 0, arena.ErrBounds
		}
	}
	if uint64(n)*4+12 > uint64(^uint32(0)) || uint64(n)*4+12 > hostMax {
		return 0, arena.ErrBounds
	}
	return n, nil
}
func (m *ASTManager) scopedSubstGrow(s ASTScopedExprSubstitution, a *astScopedExprSubstState) error {
	n, e := astScopedExprSubstCapacity(a.capacity, uint64(^uint(0)>>1))
	if e != nil {
		return e
	}
	p, e := m.vectorAllocator.Allocate(n*4 + 12)
	if e != nil {
		return e
	}
	for i := uint32(0); i < a.capacity*4; i++ {
		b, e := m.vectorAllocator.Read(a.limits, 12+i)
		if e != nil {
			m.vectorAllocator.Deallocate(n*4+12, p)
			return e
		}
		m.vectorAllocator.Write(p, 12+i, b)
	}
	m.vectorPutWord(p, 0, n)
	m.vectorPutWord(p, 4, a.size)
	m.vectorPutWord(p, 8, a.initialized)
	if a.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(a.capacity*4+12, a.limits); e != nil {
			m.vectorAllocator.Deallocate(n*4+12, p)
			return e
		}
	}
	a.limits = p
	a.capacity = n
	return m.scopedSubstStore(s, *a)
}
func (m *ASTManager) ScopedExprSubstitutionInsert(s ASTScopedExprSubstitution, key, def, proof ASTHandle, dep ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return e
	}
	if a.operation != 0 {
		return ErrASTPending
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return e
	}
	present, e := m.substContains(a.subst, k)
	if e != nil || present {
		return e
	}
	v, e := m.substRawExpr(def)
	if e != nil {
		return e
	}
	flags, e := m.substState(a.subst)
	if e != nil {
		return e
	}
	var p, d astNodeRef
	if flags&1 != 0 {
		p, e = m.substRawExpr(proof)
		if e != nil {
			return e
		}
	}
	if flags&2 != 0 {
		d, e = m.substRawDependency(dep)
		if e != nil {
			return e
		}
	}
	if e = m.substInsert(a.subst, k, v, p, d); e != nil {
		return e
	}
	return m.pushVector(a.trail, k)
}
func (m *ASTManager) ScopedExprSubstitutionPush(s ASTScopedExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return e
	}
	if a.operation != 0 {
		return ErrASTPending
	}
	trail, e := m.vectorState(a.trail)
	if e != nil {
		return e
	}
	if a.size == a.capacity {
		if e = m.scopedSubstGrow(s, &a); e != nil {
			return e
		}
	}
	if e = m.vectorPutWord(a.limits, 12+a.size*4, trail.size); e != nil {
		return e
	}
	a.size++
	if a.size > a.initialized {
		a.initialized = a.size
	}
	m.vectorPutWord(a.limits, 4, a.size)
	m.vectorPutWord(a.limits, 8, a.initialized)
	return m.scopedSubstStore(s, a)
}
func (m *ASTManager) scopedSubstPop(s ASTScopedExprSubstitution, a *astScopedExprSubstState, n uint32) error {
	if n == 0 && a.operation == 0 {
		return nil
	}
	newSize := a.size - n
	if a.operation != 0 {
		newSize = a.operation & ((1 << 30) - 1)
	}
	old, e := m.scopedSubstLimit(*a, newSize)
	if e != nil {
		return e
	}
	trail, e := m.vectorState(a.trail)
	if e != nil {
		return e
	}
	if a.operation == 0 {
		a.operation = (1 << 30) | newSize
		a.index = old
		if e = m.scopedSubstStore(s, *a); e != nil {
			return e
		}
	}
	var failure error
	if a.operation>>30 == 1 {
		for a.index < trail.size {
			r, e := m.vectorValue(trail, a.index)
			if e != nil {
				return e
			}
			if e = m.substErase(a.subst, r); e != nil {
				table, x := m.substTable(a.subst, 0)
				if x == nil {
					_, present, x := m.substFind(table, r)
					if x == nil && !present {
						a.index++
						m.scopedSubstStore(s, *a)
					}
				}
				return e
			}
			a.index++
			if e = m.scopedSubstStore(s, *a); e != nil {
				return e
			}
		}
		a.operation = (2 << 30) | newSize
		a.index = old
		m.scopedSubstStore(s, *a)
	}
	for a.index < trail.size {
		r, e := m.vectorPhysicalValue(trail, a.index)
		if e != nil {
			return e
		}
		consumed, err := m.decNode(r)
		if !consumed {
			if failure != nil {
				return failure
			}
			return err
		}
		a.index++
		m.scopedSubstStore(s, *a)
		if failure == nil {
			failure = err
		}
	}
	if old >= trail.size {
		if e = m.resizeVector(a.trail, old); e != nil {
			return e
		}
	} else {
		trail.size = old
		if trail.capacity != 0 {
			m.vectorPutWord(trail.payload, 4, old)
		}
		if e = m.storeVector(a.trail, trail); e != nil {
			return e
		}
	}
	for i := a.size; i < newSize; i++ {
		if e = m.vectorPutWord(a.limits, 12+i*4, 0); e != nil {
			return e
		}
	}
	a.size = newSize
	if a.capacity != 0 {
		m.vectorPutWord(a.limits, 4, newSize)
	}
	a.operation = 0
	a.index = 0
	if e = m.scopedSubstStore(s, *a); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) ScopedExprSubstitutionPop(s ASTScopedExprSubstitution, n uint32) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return e
	}
	return m.scopedSubstPop(s, &a, n)
}
func (m *ASTManager) ScopedExprSubstitutionScopeLevel(s ASTScopedExprSubstitution) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	return a.size, e
}
func (m *ASTManager) ScopedExprSubstitutionEmpty(s ASTScopedExprSubstitution) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return false, e
	}
	t, e := m.substTable(a.subst, 0)
	return t.size == 0, e
}
func (m *ASTManager) ScopedExprSubstitutionContains(s ASTScopedExprSubstitution, key ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return false, e
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substContains(a.subst, k)
}
func (m *ASTManager) ScopedExprSubstitutionFindProof(s ASTScopedExprSubstitution, key ASTHandle, def, proof *ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return false, e
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substFindOutputs(a.subst, k, def, proof, nil)
}
func (m *ASTManager) ScopedExprSubstitutionFindDependency(s ASTScopedExprSubstitution, key ASTHandle, def, proof *ASTHandle, dep *ASTDependency) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	if dep == nil {
		return false, ErrASTDomain
	}
	a, e := m.scopedSubstState(s)
	if e != nil {
		return false, e
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substFindOutputs(a.subst, k, def, proof, dep)
}
func (m *ASTManager) ScopedExprSubstitutionFindValue(s ASTScopedExprSubstitution, key ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return ASTHandle{}, e
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return ASTHandle{}, e
	}
	var def, pr ASTHandle
	found, e := m.substFindOutputs(a.subst, k, &def, &pr, nil)
	if e != nil {
		return ASTHandle{}, e
	}
	if found {
		return def, nil
	}
	return key, nil
}
func (m *ASTManager) ScopedExprSubstitutionCleanup(s ASTScopedExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return e
	}
	return m.substCleanup(a.subst)
}
func astScopedExprSubstDestructionWord(prefix uint32) (uint32, error) {
	// Existing actual ptr-vector payload limits prove prefix < 2^29.
	if prefix >= 1<<29 {
		return 0, arena.ErrBounds
	}
	return 3<<30 | prefix, nil
}
func (m *ASTManager) scopedSubstReleaseTrail(s ASTScopedExprSubstitution, a *astScopedExprSubstState) error {
	trail, e := m.vectorState(a.trail)
	if e != nil {
		return e
	}
	var failure error
	if a.operation>>30 == 3 && a.operation != ^uint32(0) {
		target := a.operation & ((1 << 30) - 1)
		if target > trail.size || a.index < target || a.index > trail.size {
			return ErrASTDomain
		}
		for a.index < trail.size {
			r, e := m.vectorPhysicalValue(trail, a.index)
			if e != nil {
				return e
			}
			consumed, err := m.decNode(r)
			if !consumed {
				if failure != nil {
					return failure
				}
				return err
			}
			a.index++
			m.scopedSubstStore(s, *a)
			if failure == nil {
				failure = err
			}
		}
		trail.size = target
		if trail.capacity != 0 {
			m.vectorPutWord(trail.payload, 4, target)
		}
		m.storeVector(a.trail, trail)
		a.operation = ^uint32(0)
		a.index = 0
		m.scopedSubstStore(s, *a)
	}
	if a.operation != ^uint32(0) {
		a.operation = ^uint32(0)
		a.index = 0
		m.scopedSubstStore(s, *a)
	}
	for a.index < trail.size {
		r, e := m.vectorPhysicalValue(trail, a.index)
		if e != nil {
			return e
		}
		consumed, err := m.decNode(r)
		if !consumed {
			if failure != nil {
				return failure
			}
			return err
		}
		a.index++
		m.scopedSubstStore(s, *a)
		if failure == nil {
			failure = err
		}
	}
	trail.size = 0
	if trail.capacity != 0 {
		m.vectorPutWord(trail.payload, 4, 0)
	}
	m.storeVector(a.trail, trail)
	a.operation = 0
	a.index = 0
	m.scopedSubstStore(s, *a)
	return failure
}
func (m *ASTManager) ScopedExprSubstitutionReset(s ASTScopedExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	a, e := m.scopedSubstState(s)
	if e != nil {
		return e
	}
	if a.operation != 0 && a.operation != ^uint32(0) {
		return ErrASTPending
	}
	if a.operation == 0 {
		if e = m.substReset(a.subst); e != nil {
			return e
		}
	}
	failure := m.scopedSubstReleaseTrail(s, &a)
	if a.operation != 0 {
		return failure
	}
	a.size = 0
	if a.capacity != 0 {
		m.vectorPutWord(a.limits, 4, 0)
	}
	m.scopedSubstStore(s, a)
	return failure
}
func (m *ASTManager) DelScopedExprSubstitution(s *ASTScopedExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if s == nil {
		return ErrASTHandle
	}
	a, e := m.scopedSubstState(*s)
	if e != nil {
		return e
	}
	if a.operation>>30 == 2 {
		newSize := a.operation & ((1 << 30) - 1)
		target, e := m.scopedSubstLimit(a, newSize)
		if e != nil {
			return e
		}
		trail, e := m.vectorState(a.trail)
		if e != nil {
			return e
		}
		// A refused growth has consumed no tail references. The still-owned
		// prefix remains the actual old logical size, never the requested size.
		if target > trail.size {
			target = trail.size
			a.index = trail.size
		}
		word, e := astScopedExprSubstDestructionWord(target)
		if e != nil {
			return e
		}
		a.operation = word
		if e = m.scopedSubstStore(*s, a); e != nil {
			return e
		}
	}
	// m_trail_lim is destroyed before m_trail; neither resets m_subst.
	if a.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(a.capacity*4+12, a.limits); e != nil {
			return e
		}
		a.capacity = 0
		a.size = 0
		a.initialized = 0
		a.limits = arena.SmallBytesRef{}
		m.scopedSubstStore(*s, a)
	}
	failure := m.scopedSubstReleaseTrail(*s, &a)
	if a.operation != 0 {
		return failure
	}
	trail, e := m.vectorState(a.trail)
	if e != nil {
		return e
	}
	if trail.capacity != 0 {
		if e = m.vectorAllocator.Deallocate(trail.capacity*8+astRefVectorHeaderBytes, trail.payload); e != nil {
			return e
		}
	}
	if e = m.allocator.Deallocate(astRefVectorRecordBytes, a.trail.record); e != nil {
		return e
	}
	if e = m.allocator.Deallocate(48, s.record); e != nil {
		return e
	}
	*s = ASTScopedExprSubstitution{}
	return failure
}
