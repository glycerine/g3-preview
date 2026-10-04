// Licensed under the MIT license.
// Full default scoped_mark action order and actual AST owning callbacks.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type ASTScopedMark struct {
	state      astMarkState
	stack      ASTRefVector
	limits     []uint32
	limitBytes uint64
}

func NewASTScopedMark(m *ASTManager) (*ASTScopedMark, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	bytes := uint64(reflect.TypeOf(ASTScopedMark{}).Size())
	id, e := prepareASTMarkOwner(m, bytes)
	if e != nil {
		return nil, e
	}
	v := &ASTScopedMark{}
	v.state.owner.bind(m, bytes, id)
	stack, e := m.newRefVector()
	if e != nil {
		v.state.owner.finish()
		return nil, e
	}
	v.stack = stack
	return v, nil
}
func (v *ASTScopedMark) IsMarked(h ASTHandle) (bool, error) {
	if v == nil {
		return false, ErrASTHandle
	}
	m, e := v.state.owner.lock()
	if e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	return v.state.marked(m, h)
}
func (v *ASTScopedMark) Mark(h ASTHandle, flag bool) error {
	if v == nil {
		return ErrASTHandle
	}
	m, e := v.state.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	if astSourceDebug && !flag {
		return ErrASTDomain
	}
	yes, e := v.state.marked(m, h)
	if e != nil || yes {
		return e
	}
	r, e := m.internal(h)
	if e != nil {
		return e
	}
	if e = m.pushVector(v.stack, r); e != nil {
		return e
	}
	return v.state.mark(m, h, true)
}
func (v *ASTScopedMark) Reset() error {
	if v == nil {
		return ErrASTHandle
	}
	m, e := v.state.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	v.state.reset()
	e = m.resizeVector(v.stack, 0)
	// Valid-source deletion reserve failures use the manager's consumed
	// continuation; source reset's limits action still follows that release.
	if e != nil {
		s, err := m.vectorState(v.stack)
		if err != nil || s.size != 0 {
			return e
		}
	}
	v.limits = v.limits[:0]
	return e
}
func (v *ASTScopedMark) PushScope() error {
	if v == nil {
		return ErrASTHandle
	}
	m, e := v.state.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v.stack)
	if e != nil {
		return e
	}
	if len(v.limits) == cap(v.limits) {
		n := uint64(2)
		if cap(v.limits) != 0 {
			n = (3*uint64(cap(v.limits)) + 1) / 2
		}
		bytes := n * 4
		if n > uint64(^uint32(0))/4 || bytes > uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e = m.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint32, len(v.limits), int(n))
		copy(next[:cap(next)], v.limits[:cap(v.limits)])
		m.budget.Release(v.limitBytes)
		v.limits = next
		v.limitBytes = bytes
	}
	v.limits = append(v.limits, s.size)
	return nil
}
func (v *ASTScopedMark) pop(m *ASTManager) error {
	if len(v.limits) == 0 {
		if astSourceDebug {
			return ErrASTDomain
		}
		return arena.ErrBounds
	}
	s, e := m.vectorState(v.stack)
	if e != nil {
		return e
	}
	old := v.limits[len(v.limits)-1]
	for i := old; i < s.size; i++ {
		r, e := m.vectorValue(s, i)
		if e != nil {
			return e
		}
		if e = v.state.mark(m, m.external(r), false); e != nil {
			return e
		}
	}
	v.limits = v.limits[:len(v.limits)-1]
	return m.resizeVector(v.stack, old)
}
func (v *ASTScopedMark) PopScope() error { return v.PopScopes(1) }
func (v *ASTScopedMark) PopScopes(n uint32) error {
	if v == nil {
		return ErrASTHandle
	}
	m, e := v.state.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	for i := uint32(0); i < n; i++ {
		if e = v.pop(m); e != nil {
			return e
		}
	}
	return nil
}
func (v *ASTScopedMark) Close() error {
	if v == nil || v.state.owner.self != &v.state.owner {
		return ErrASTHandle
	}
	if v.state.owner.closed {
		return nil
	}
	m := v.state.owner.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	// Empty derived destructor: reverse member destruction is limits, then
	// stack (ascending DEC); only then base ast_mark's decl/expr bitsets.
	m.budget.Release(v.limitBytes)
	v.limits = nil
	v.limitBytes = 0
	var e error
	if !m.closed {
		e = m.finalizeVector(v.stack)
		if e != nil {
			s, err := m.vectorState(v.stack)
			if err != nil || s.size != 0 {
				return e
			}
		}
		if err := m.allocator.Deallocate(astRefVectorRecordBytes, v.stack.record); e == nil {
			e = err
		}
	}
	v.stack = ASTRefVector{}
	v.state.close(m)
	return e
}
