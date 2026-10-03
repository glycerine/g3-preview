// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete decl_info, sort_info and func_decl_info storage/action translation.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"math"
)

func (m *ASTManager) info(slot, generation uint32) (*astInfoRecord, error) {
	if slot == 0 || slot >= m.nextInfo {
		return nil, ErrASTHandle
	}
	r := m.infoAt(slot)
	if !r.live || r.generation != generation {
		return nil, ErrASTHandle
	}
	return r, nil
}
func (m *ASTManager) sortInfo(p ASTSortInfo) (*astInfoRecord, error) {
	if p.origin != m.origin {
		return nil, ErrASTHandle
	}
	r, e := m.info(p.slot, p.generation)
	if e == nil && !r.isSort {
		return nil, ErrASTHandle
	}
	return r, e
}
func (m *ASTManager) funcInfo(p ASTFuncDeclInfo) (*astInfoRecord, error) {
	if p.origin != m.origin {
		return nil, ErrASTHandle
	}
	r, e := m.info(p.slot, p.generation)
	if e == nil && r.isSort {
		return nil, ErrASTHandle
	}
	return r, e
}
func (m *ASTManager) allocInfo() (uint32, uint32, error) {
	for m.freeInfo != 0 {
		s := m.freeInfo
		r := m.infoAt(s)
		m.freeInfo = r.nextFree
		if r.generation == math.MaxUint32 {
			continue
		}
		g := r.generation + 1
		*r = astInfoRecord{generation: g, live: true}
		m.liveInfos++
		return s, g, nil
	}
	if m.nextInfo == math.MaxUint32 {
		return 0, 0, arena.ErrBounds
	}
	s := m.nextInfo
	if e := m.ensureInfoPage(s); e != nil {
		return 0, 0, e
	}
	m.nextInfo++
	*m.infoAt(s) = astInfoRecord{generation: 1, live: true}
	m.liveInfos++
	return s, 1, nil
}
func (m *ASTManager) infoParameter(r *astInfoRecord, i uint32) (ASTParameter, error) {
	if i >= r.count {
		return ASTParameter{}, ErrASTDomain
	}
	s, e := m.readWord(r.parameters, 8+i*8)
	if e != nil {
		return ASTParameter{}, e
	}
	g, e := m.readWord(r.parameters, 12+i*8)
	return ASTParameter{m.origin, s, g}, e
}
func (m *ASTManager) growInfoParameters(slot uint32) error {
	r := m.infoAt(slot)
	n, e := idFreeGrowth(r.capacity)
	if e != nil {
		return e
	}
	bytes := uint64(n)*8 + 8
	if bytes > math.MaxUint32 {
		return arena.ErrBounds
	}
	p, e := m.allocator.Allocate(uint32(bytes))
	if e != nil {
		return e
	}
	if e = m.writeWord(p, 0, n); e == nil {
		e = m.writeWord(p, 4, r.count)
	}
	for i := uint32(0); e == nil && i < r.count; i++ {
		v, x := m.infoParameter(r, i)
		e = x
		if e == nil {
			e = m.writeWord(p, 8+i*8, v.slot)
		}
		if e == nil {
			e = m.writeWord(p, 12+i*8, v.generation)
		}
	}
	if e != nil {
		m.allocator.Deallocate(uint32(bytes), p)
		return e
	}
	if r.capacity != 0 {
		if e = m.allocator.Deallocate(r.capacity*8+8, r.parameters); e != nil {
			return e
		}
	}
	r.parameters = p
	r.capacity = n
	return nil
}
func (m *ASTManager) pushInfoParameter(slot uint32, p ASTParameter) error {
	r := m.infoAt(slot)
	if r.count == r.capacity {
		if e := m.growInfoParameters(slot); e != nil {
			return e
		}
	}
	copy, e := m.copyParameter(p)
	if e != nil {
		return e
	}
	r = m.infoAt(slot)
	if e = m.writeWord(r.parameters, 8+r.count*8, copy.slot); e == nil {
		e = m.writeWord(r.parameters, 12+r.count*8, copy.generation)
	}
	if e != nil {
		m.dropParameter(copy)
		return e
	}
	r.count++
	return m.writeWord(r.parameters, 4, r.count)
}

// dropInfoStorage does not run attachment callbacks. The caller explicitly
// performs del_eh first; caller-owned unattached information never acquired RC.
func (m *ASTManager) dropInfoStorage(slot, generation uint32) error {
	r, e := m.info(slot, generation)
	if e != nil {
		return e
	}
	for i := uint32(0); i < r.count; i++ {
		p, e := m.infoParameter(r, i)
		if e != nil {
			return e
		}
		if e = m.dropParameter(p); e != nil {
			return e
		}
	}
	if r.capacity != 0 {
		if e = m.allocator.Deallocate(r.capacity*8+8, r.parameters); e != nil {
			return e
		}
	}
	r.live = false
	r.nextFree = m.freeInfo
	m.freeInfo = slot
	m.liveInfos--
	return nil
}
func (m *ASTManager) newInfo(isSort bool, fid FamilyID, kind int32, ps []ASTParameter, size SortSize, flags ASTFuncFlags, private bool) (uint32, uint32, error) {
	if uint64(len(ps)) > math.MaxUint32 {
		return 0, 0, arena.ErrBounds
	}
	s, g, e := m.allocInfo()
	if e != nil {
		return 0, 0, e
	}
	r := m.infoAt(s)
	r.isSort = isSort
	r.family = fid
	r.kind = kind
	r.size = size
	r.flags = flags
	r.private = private
	for _, p := range ps {
		if e = m.pushInfoParameter(s, p); e != nil {
			m.dropInfoStorage(s, g)
			return 0, 0, e
		}
	}
	return s, g, nil
}
func (m *ASTManager) NewSortInfo(fid FamilyID, kind int32, ps []ASTParameter, size SortSize, private bool) (ASTSortInfo, error) {
	if e := m.lock(); e != nil {
		return ASTSortInfo{}, e
	}
	defer m.mu.Unlock()
	s, g, e := m.newInfo(true, fid, kind, ps, size, ASTFuncFlags{}, private)
	if e != nil {
		return ASTSortInfo{}, e
	}
	return ASTSortInfo{m.origin, s, g}, nil
}
func (m *ASTManager) NewFuncDeclInfo(fid FamilyID, kind int32, ps []ASTParameter, flags ASTFuncFlags, private bool) (ASTFuncDeclInfo, error) {
	if e := m.lock(); e != nil {
		return ASTFuncDeclInfo{}, e
	}
	defer m.mu.Unlock()
	s, g, e := m.newInfo(false, fid, kind, ps, SortSize{}, flags, private)
	if e != nil {
		return ASTFuncDeclInfo{}, e
	}
	return ASTFuncDeclInfo{m.origin, s, g}, nil
}
func (m *ASTManager) copyInfo(slot, generation uint32) (uint32, uint32, error) {
	src, e := m.info(slot, generation)
	if e != nil {
		return 0, 0, e
	}
	saved := *src
	s, g, e := m.allocInfo()
	if e != nil {
		return 0, 0, e
	}
	r := m.infoAt(s)
	r.isSort = saved.isSort
	r.family = saved.family
	r.kind = saved.kind
	r.size = saved.size
	r.flags = saved.flags
	r.private = saved.private
	for i := uint32(0); i < saved.count; i++ {
		p, x := m.infoParameter(&saved, i)
		e = x
		if e == nil {
			e = m.pushInfoParameter(s, p)
		}
		if e != nil {
			m.dropInfoStorage(s, g)
			return 0, 0, e
		}
	}
	return s, g, nil
}
func (m *ASTManager) DelSortInfo(p *ASTSortInfo) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if p == nil {
		return ErrASTHandle
	}
	r, e := m.sortInfo(*p)
	if e != nil {
		return e
	}
	if r.attached {
		return ErrASTDomain
	}
	if e = m.dropInfoStorage(p.slot, p.generation); e == nil {
		*p = ASTSortInfo{}
	}
	return e
}
func (m *ASTManager) DelFuncDeclInfo(p *ASTFuncDeclInfo) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if p == nil {
		return ErrASTHandle
	}
	r, e := m.funcInfo(*p)
	if e != nil {
		return e
	}
	if r.attached {
		return ErrASTDomain
	}
	if e = m.dropInfoStorage(p.slot, p.generation); e == nil {
		*p = ASTFuncDeclInfo{}
	}
	return e
}
func astFuncInfoNull(r *astInfoRecord) bool {
	f := r.flags
	return r.family == NullFamilyID && !f.LeftAssociative && !f.RightAssociative && !f.Commutative && !f.Chainable && !f.Pairwise && !f.Injective && !f.Idempotent && !f.Skolem
}
func (m *ASTManager) infoHash(slot, generation uint32) (uint32, error) {
	r, e := m.info(slot, generation)
	if e != nil {
		return 0, e
	}
	var childErr error
	c := uint32(0)
	if r.count != 0 {
		c = utilhash.Composite(17, r.count, func(i uint32) uint32 {
			p, x := m.infoParameter(r, i)
			if x != nil {
				childErr = x
				return 0
			}
			h, x := m.parameterHash(p)
			if x != nil {
				childErr = x
			}
			return h
		})
	}
	if childErr != nil {
		return 0, childErr
	}
	return utilhash.Mix(uint32(r.family), uint32(r.kind), c), nil
}
func astFuncFlagsEqual(a, b ASTFuncFlags) bool {
	return a.LeftAssociative == b.LeftAssociative && a.RightAssociative == b.RightAssociative && a.FlatAssociative == b.FlatAssociative && a.Commutative == b.Commutative && a.Chainable == b.Chainable && a.Pairwise == b.Pairwise && a.Injective == b.Injective && a.Skolem == b.Skolem
}
func (m *ASTManager) infoEqual(leftSlot, leftGen, rightSlot, rightGen uint32) (bool, error) {
	a, e := m.info(leftSlot, leftGen)
	if e != nil {
		return false, e
	}
	b, e := m.info(rightSlot, rightGen)
	if e != nil {
		return false, e
	}
	if a.family != b.family || a.kind != b.kind {
		return false, nil
	}
	// Preserve source left-length traversal. A shorter RHS would read outside
	// its initialized vector: typed invalid-continuation adapter, not equality.
	for i := uint32(0); i < a.count; i++ {
		if i >= b.count {
			return false, ErrASTDomain
		}
		p, e := m.infoParameter(a, i)
		if e != nil {
			return false, e
		}
		q, e := m.infoParameter(b, i)
		if e != nil {
			return false, e
		}
		same, e := m.parameterEqual(p, q)
		if e != nil || !same {
			return same, e
		}
	}
	if !a.isSort {
		return astFuncFlagsEqual(a.flags, b.flags), nil
	}
	return true, nil
}
