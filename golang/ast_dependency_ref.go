// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete expr_dependency_ref specialization of util/obj_ref.h.
package g3

import "github.com/glycerine/g3/golang/arena"

type ASTDependencyRef struct {
	origin uint64
	object arena.SmallBytesRef
}

const astDependencyRefBytes uint32 = 16
const astDependencyRefTag uint32 = 0x44524546

func (m *ASTManager) dependencyRefGet(r ASTDependencyRef) (arena.SmallBytesRef, error) {
	if r.origin != m.origin || !m.allocator.IsLive(r.object) {
		return arena.SmallBytesRef{}, ErrASTHandle
	}
	n, e := m.allocator.Size(r.object)
	if e != nil || n != astDependencyRefBytes {
		return arena.SmallBytesRef{}, ErrASTHandle
	}
	tag, e := m.readWord(r.object, 8)
	if e != nil || tag != astDependencyRefTag {
		return arena.SmallBytesRef{}, ErrASTHandle
	}
	s, e := m.readWord(r.object, 0)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	g, e := m.readWord(r.object, 4)
	return arena.SmallBytesRef{Slot: arena.Slot(s), Generation: arena.Generation(g)}, e
}
func (m *ASTManager) dependencyRefPut(r ASTDependencyRef, v arena.SmallBytesRef) error {
	if e := m.writeWord(r.object, 0, uint32(v.Slot)); e != nil {
		return e
	}
	return m.writeWord(r.object, 4, uint32(v.Generation))
}
func (m *ASTManager) newDependencyRef(v arena.SmallBytesRef, retain bool) (ASTDependencyRef, error) {
	p, e := m.allocator.Allocate(astDependencyRefBytes)
	if e != nil {
		return ASTDependencyRef{}, e
	}
	r := ASTDependencyRef{m.origin, p}
	m.writeWord(p, 8, astDependencyRefTag)
	if retain {
		if e = m.dependencies.inc(v); e != nil {
			m.allocator.Deallocate(astDependencyRefBytes, p)
			return ASTDependencyRef{}, e
		}
	}
	if e = m.dependencyRefPut(r, v); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *ASTManager) NewDependencyRef(v ASTDependency) (ASTDependencyRef, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRef{}, e
	}
	defer m.mu.Unlock()
	p, e := m.dependencies.valid(v)
	if e != nil {
		return ASTDependencyRef{}, e
	}
	return m.newDependencyRef(p, true)
}
func (m *ASTManager) CopyDependencyRef(src ASTDependencyRef) (ASTDependencyRef, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRef{}, e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(src)
	if e != nil {
		return ASTDependencyRef{}, e
	}
	return m.newDependencyRef(p, true)
}
func (m *ASTManager) MoveDependencyRef(src ASTDependencyRef) (ASTDependencyRef, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRef{}, e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(src)
	if e != nil {
		return ASTDependencyRef{}, e
	}
	r, e := m.newDependencyRef(p, false)
	if e == nil {
		e = m.dependencyRefPut(src, arena.SmallBytesRef{})
	}
	return r, e
}
func (m *ASTManager) DependencyRefGet(r ASTDependencyRef) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(r)
	return m.dependencies.external(p), e
}
func (m *ASTManager) DependencyRefPresent(r ASTDependencyRef) (bool, error) {
	v, e := m.DependencyRefGet(r)
	return !v.IsNull(), e
}
func (m *ASTManager) DependencyRefEqual(a, b ASTDependencyRef) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	x, e := m.dependencyRefGet(a)
	if e != nil {
		return false, e
	}
	y, e := m.dependencyRefGet(b)
	return x == y, e
}
func (m *ASTManager) DependencyRefManager(r ASTDependencyRef) (*ASTManager, error) {
	if _, e := m.DependencyRefGet(r); e != nil {
		return nil, e
	}
	return m, nil
}
func (m *ASTManager) dependencyRefSet(r ASTDependencyRef, v arena.SmallBytesRef) error {
	old, e := m.dependencyRefGet(r)
	if e != nil {
		return e
	}
	if e = m.dependencies.inc(v); e != nil {
		return e
	}
	consumed, e := m.dependencyArrays.valueDec(old)
	if !consumed {
		return e
	}
	if x := m.dependencyRefPut(r, v); x != nil {
		return x
	}
	return e
}
func (m *ASTManager) DependencyRefSet(r ASTDependencyRef, v ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	p, e := m.dependencies.valid(v)
	if e != nil {
		return e
	}
	return m.dependencyRefSet(r, p)
}
func (m *ASTManager) DependencyRefAssign(dst, src ASTDependencyRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(src)
	if e != nil {
		return e
	}
	return m.dependencyRefSet(dst, p)
}
func (m *ASTManager) DependencyRefReset(r ASTDependencyRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(r)
	if e != nil {
		return e
	}
	consumed, e := m.dependencyArrays.valueDec(p)
	if consumed {
		if x := m.dependencyRefPut(r, arena.SmallBytesRef{}); x != nil {
			return x
		}
	}
	return e
}
func (m *ASTManager) SwapDependencyRefs(a, b ASTDependencyRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	x, e := m.dependencyRefGet(a)
	if e != nil {
		return e
	}
	y, e := m.dependencyRefGet(b)
	if e != nil {
		return e
	}
	if e = m.dependencyRefPut(a, y); e != nil {
		return e
	}
	return m.dependencyRefPut(b, x)
}
func (m *ASTManager) DependencyRefSteal(r ASTDependencyRef) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	p, e := m.dependencyRefGet(r)
	if e != nil {
		return ASTDependency{}, e
	}
	if e = m.dependencyRefPut(r, arena.SmallBytesRef{}); e != nil {
		return ASTDependency{}, e
	}
	return m.dependencies.external(p), nil
}
func (m *ASTManager) DelDependencyRef(r *ASTDependencyRef) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if r == nil {
		return ErrASTHandle
	}
	p, e := m.dependencyRefGet(*r)
	if e != nil {
		return e
	}
	consumed, e := m.dependencyArrays.valueDec(p)
	if !consumed {
		return e
	}
	if x := m.allocator.Deallocate(astDependencyRefBytes, r.object); x != nil {
		return x
	}
	*r = ASTDependencyRef{}
	return e
}

// Actual selected capacities are below2^29; active metadata never overlaps a
// representable range target. The source pointer/value bits are not modified.
func astDependencyRefCleanupWord(target uint32, active bool) (uint32, error) {
	if target >= 1<<29 {
		return 0, arena.ErrBounds
	}
	if active {
		return target | 1<<31, nil
	}
	return target, nil
}
