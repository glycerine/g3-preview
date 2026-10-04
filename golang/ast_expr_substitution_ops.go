// Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

func (m *ASTManager) substMutation(s ASTExprSubstitution) error {
	if _, e := m.substState(s); e != nil {
		return e
	}
	v, e := m.readWord(s.record, 104)
	if e != nil {
		return e
	}
	if v != 0 {
		return ErrASTPending
	}
	return nil
}
func (m *ASTManager) substNoteFailure(s ASTExprSubstitution, e error) error {
	if e != nil {
		m.writeWord(s.record, 104, 1)
	}
	return e
}
func (m *ASTManager) substInc(which uint32, r astNodeRef) error {
	if r.slot == 0 {
		return nil
	}
	if which == 2 {
		return m.dependencies.inc(dependencyRefPointer(r))
	}
	if e := m.validNode(r); e != nil {
		return e
	}
	kind := m.nodeAt(r.slot).kind
	if which == 1 {
		if kind != ASTApp {
			return ErrASTDomain
		}
	} else if kind != ASTApp && kind != ASTVar && kind != ASTQuantifier {
		return ErrASTDomain
	}
	return m.incNode(r)
}
func (m *ASTManager) substDec(which uint32, r astNodeRef) (bool, error) {
	if which == 2 {
		return m.dependencyArrays.valueDec(dependencyRefPointer(r))
	}
	return m.decNode(r)
}
func (m *ASTManager) substStoreValue(s ASTExprSubstitution, which, index uint32, new astNodeRef, replacing bool) error {
	t, e := m.substTable(s, which)
	if e != nil {
		return e
	}
	a, e := m.substEntry(t.payload, index)
	if e != nil {
		return e
	}
	if e = m.substInc(which, new); e != nil {
		return e
	}
	var failure error
	if replacing {
		consumed, err := m.substDec(which, a.value)
		if !consumed {
			return err
		}
		failure = err
	}
	a.value = new
	a.owned |= 2
	if e = m.substPutEntry(t.payload, index, a); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) substInsert(s ASTExprSubstitution, key, def, proof astNodeRef, dep astNodeRef) error {
	if e := m.substMutation(s); e != nil {
		return e
	}
	flags, _ := m.substState(s)
	index, e := m.substInsertSlot(s, 0, key, astNodeRef{}, false)
	if e != nil {
		return e
	}
	t, e := m.substTable(s, 0)
	if e != nil {
		return e
	}
	entry, e := m.substEntry(t.payload, index)
	if e != nil {
		return e
	}
	fresh := entry.value.slot == 0
	if fresh {
		if e = m.substInc(0, key); e != nil {
			return m.substNoteFailure(s, e)
		}
		entry.owned |= 1
		if e = m.substPutEntry(t.payload, index, entry); e != nil {
			return e
		}
	}
	if e = m.substStoreValue(s, 0, index, def, !fresh); e != nil {
		return m.substNoteFailure(s, e)
	}
	for which := uint32(1); which < 3; which++ {
		if flags&(1<<(which-1)) == 0 {
			continue
		}
		t, e = m.substTable(s, which)
		if e != nil {
			return e
		}
		j, found, e := m.substFind(t, key)
		if e != nil {
			return m.substNoteFailure(s, e)
		}
		value := proof
		if which == 2 {
			value = dep
		}
		if fresh {
			if astSourceDebug && found {
				return m.substNoteFailure(s, ErrASTDomain)
			}
			j, e = m.substInsertSlot(s, which, key, value, true)
			if e != nil {
				return m.substNoteFailure(s, e)
			}
			// Source optional-map insert publishes the pointer BEFORE its INC.
			if e = m.substInc(which, value); e != nil {
				return m.substNoteFailure(s, e)
			}
			t, _ = m.substTable(s, which)
			a, err := m.substEntry(t.payload, j)
			if err != nil {
				return err
			}
			a.owned |= 2
			if e = m.substPutEntry(t.payload, j, a); e != nil {
				return e
			}
		} else {
			if !found {
				return m.substNoteFailure(s, ErrASTDomain)
			}
			if e = m.substStoreValue(s, which, j, value, true); e != nil {
				return m.substNoteFailure(s, e)
			}
		}
	}
	return nil
}
func (m *ASTManager) ExprSubstitutionInsert(s ASTExprSubstitution, key, def, proof ASTHandle, dep ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	k, e := m.substRawExpr(key)
	if e != nil {
		return e
	}
	v, e := m.substRawExpr(def)
	if e != nil {
		return e
	}
	flags, e := m.substState(s)
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
	return m.substInsert(s, k, v, p, d)
}
func (m *ASTManager) substContains(s ASTExprSubstitution, key astNodeRef) (bool, error) {
	t, e := m.substTable(s, 0)
	if e != nil {
		return false, e
	}
	_, found, e := m.substFind(t, key)
	return found, e
}
func (m *ASTManager) ExprSubstitutionContains(s ASTExprSubstitution, key ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substContains(s, k)
}
func (m *ASTManager) substFindOutputs(s ASTExprSubstitution, key astNodeRef, def, proof *ASTHandle, dep *ASTDependency) (bool, error) {
	if def == nil || proof == nil {
		return false, ErrASTDomain
	}
	flags, e := m.substState(s)
	if e != nil {
		return false, e
	}
	t, e := m.substTable(s, 0)
	if e != nil {
		return false, e
	}
	i, found, e := m.substFind(t, key)
	if e != nil || !found {
		return found, e
	}
	a, e := m.substEntry(t.payload, i)
	if e != nil {
		return false, e
	}
	*def = m.external(a.value)
	if flags&1 != 0 {
		t, e = m.substTable(s, 1)
		if e != nil {
			return false, e
		}
		i, found, e = m.substFind(t, key)
		if e != nil {
			return false, e
		}
		if found {
			a, e = m.substEntry(t.payload, i)
			if e != nil {
				return false, e
			}
			*proof = m.external(a.value)
		}
	}
	if dep != nil && flags&2 != 0 {
		t, e = m.substTable(s, 2)
		if e != nil {
			return false, e
		}
		i, found, e = m.substFind(t, key)
		if e != nil {
			return false, e
		}
		if found {
			a, e = m.substEntry(t.payload, i)
			if e != nil {
				return false, e
			}
			*dep = m.substDependency(a.value)
		}
	}
	return true, nil
}
func (m *ASTManager) ExprSubstitutionFindProof(s ASTExprSubstitution, key ASTHandle, def, proof *ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substFindOutputs(s, k, def, proof, nil)
}
func (m *ASTManager) ExprSubstitutionFindDependency(s ASTExprSubstitution, key ASTHandle, def, proof *ASTHandle, dep *ASTDependency) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	if dep == nil {
		return false, ErrASTDomain
	}
	k, e := m.substRawExpr(key)
	if e != nil {
		return false, e
	}
	return m.substFindOutputs(s, k, def, proof, dep)
}
func (m *ASTManager) substReleaseEntry(s ASTExprSubstitution, which, index uint32) error {
	t, e := m.substTable(s, which)
	if e != nil {
		return e
	}
	a, e := m.substEntry(t.payload, index)
	if e != nil {
		return e
	}
	var failure error
	for bit := uint32(1); bit <= 2; bit <<= 1 {
		if a.owned&bit == 0 {
			continue
		}
		r := a.key
		target := uint32(0)
		if bit == 2 {
			r = a.value
			target = which
		}
		consumed, err := m.substDec(target, r)
		if !consumed {
			if failure != nil {
				return failure
			}
			return err
		}
		a.owned &^= bit
		if e = m.substPutEntry(t.payload, index, a); e != nil {
			return e
		}
		if failure == nil {
			failure = err
		}
	}
	return failure
}
func (m *ASTManager) substErase(s ASTExprSubstitution, key astNodeRef) error {
	if e := m.substMutation(s); e != nil {
		return e
	}
	flags, _ := m.substState(s)
	var failure error
	for _, which := range [3]uint32{1, 2, 0} {
		if which != 0 && flags&(1<<(which-1)) == 0 {
			continue
		}
		t, e := m.substTable(s, which)
		if e != nil {
			return e
		}
		i, found, e := m.substFind(t, key)
		if e != nil {
			return m.substNoteFailure(s, e)
		}
		if !found {
			continue
		}
		e = m.substReleaseEntry(s, which, i)
		if e != nil {
			a, _ := m.substEntry(t.payload, i)
			if a.owned != 0 {
				return m.substNoteFailure(s, e)
			}
			if failure == nil {
				failure = e
			}
		}
		// Source erase hashes the key AFTER DEC; actual lifetime guards retain
		// the consumed callback prefix, instead of dereferencing freed storage.
		if e = m.substEraseSlot(s, which, key); e != nil {
			return m.substNoteFailure(s, e)
		}
	}
	return failure
}
func (m *ASTManager) ExprSubstitutionErase(s ASTExprSubstitution, key ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	k, e := m.substRawExpr(key)
	if e != nil {
		return e
	}
	return m.substErase(s, k)
}
func (m *ASTManager) substResetTable(s ASTExprSubstitution, which uint32) error {
	t, e := m.substTable(s, which)
	if e != nil {
		return e
	}
	if t.capacity == 0 {
		return nil
	}
	var failure error
	if t.payload != (arena.SmallBytesRef{}) {
		for t.index < t.capacity {
			a, e := m.substEntry(t.payload, t.index)
			if e != nil {
				return e
			}
			if a.state == 1 {
				e = m.substReleaseEntry(s, which, t.index)
				if e != nil {
					a, _ = m.substEntry(t.payload, t.index)
					if a.owned != 0 {
						m.substStoreTable(s, which, t)
						return e
					}
					if failure == nil {
						failure = e
					}
				}
			}
			t.index++
			if e = m.substStoreTable(s, which, t); e != nil {
				return e
			}
			if failure != nil {
				return failure
			}
		}
		if t.size != 0 || t.deleted != 0 {
			overhead := uint32(0)
			for i := uint32(0); i < t.capacity; i++ {
				a, e := m.substEntry(t.payload, i)
				if e != nil {
					return e
				}
				if a.state == 0 {
					overhead++
				} else {
					a.state = 0
					a.owned = 0
					if e = m.substPutEntry(t.payload, i, a); e != nil {
						return e
					}
				}
			}
			if t.capacity > 16 && overhead<<2 > t.capacity*3 {
				if e = m.vectorAllocator.Deallocate(t.capacity*24, t.payload); e != nil {
					return e
				}
				t.payload = arena.SmallBytesRef{}
				t.capacity >>= 1
				t.flags = 1
				if e = m.substStoreTable(s, which, t); e != nil {
					return e
				}
			}
		}
	}
	if t.payload == (arena.SmallBytesRef{}) {
		requested := t.capacity
		if t.flags == 2 {
			requested = 64
		}
		p, e := m.substAllocTable(requested)
		if e != nil {
			return e
		}
		t.payload = p
		t.capacity = requested
	}
	t.size = 0
	t.deleted = 0
	t.index = 0
	t.phase = 0
	t.flags = 0
	if e = m.substStoreTable(s, which, t); e != nil {
		return e
	}
	return failure
}
func (m *ASTManager) substReset(s ASTExprSubstitution) error {
	flags, e := m.substState(s)
	if e != nil {
		return e
	}
	for i := uint32(0); i < 3; i++ {
		if i != 0 && flags&(1<<(i-1)) == 0 {
			continue
		}
		if e = m.substResetTable(s, i); e != nil {
			return m.substNoteFailure(s, e)
		}
	}
	return m.writeWord(s.record, 104, 0)
}
func (m *ASTManager) ExprSubstitutionReset(s ASTExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.substReset(s)
}
func (m *ASTManager) substCleanup(s ASTExprSubstitution) error {
	if e := m.substReset(s); e != nil {
		return e
	}
	flags, _ := m.substState(s)
	for i := uint32(0); i < 3; i++ {
		if i != 0 && flags&(1<<(i-1)) == 0 {
			continue
		}
		t, e := m.substTable(s, i)
		if e != nil {
			return e
		}
		if t.capacity > 64 {
			if e = m.vectorAllocator.Deallocate(t.capacity*24, t.payload); e != nil {
				return e
			}
			t.payload = arena.SmallBytesRef{}
			// Source publishes capacity64 only AFTER alloc_table64 succeeds.
			t.flags = 2
			m.substStoreTable(s, i, t)
			p, e := m.substAllocTable(64)
			if e != nil {
				return m.substNoteFailure(s, e)
			}
			t.payload = p
			t.capacity = 64
			t.flags = 0
			m.substStoreTable(s, i, t)
		} else {
			if e = m.substResetTable(s, i); e != nil {
				return m.substNoteFailure(s, e)
			}
		}
	}
	return nil
}
func (m *ASTManager) ExprSubstitutionCleanup(s ASTExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.substCleanup(s)
}
func (m *ASTManager) DelExprSubstitution(s *ASTExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if s == nil {
		return ErrASTHandle
	}
	flags, e := m.substState(*s)
	if e != nil {
		return e
	}
	failure := m.substReset(*s)
	// Destructor reset followed by table destructors. Consume finite cleanup
	// errors before refund; preserve the token on an unconsumed callback.
	for i := uint32(0); i < 3; i++ {
		if i != 0 && flags&(1<<(i-1)) == 0 {
			continue
		}
		t, _ := m.substTable(*s, i)
		if t.payload != (arena.SmallBytesRef{}) {
			for j := t.index; j < t.capacity; j++ {
				a, e := m.substEntry(t.payload, j)
				if e != nil {
					return e
				}
				if a.state == 1 {
					e = m.substReleaseEntry(*s, i, j)
					a, _ = m.substEntry(t.payload, j)
					if a.owned != 0 {
						return m.substNoteFailure(*s, e)
					}
					if failure == nil {
						failure = e
					}
				}
				t.index = j + 1
				m.substStoreTable(*s, i, t)
			}
		}
	}
	// Member destructors run dependency, proof, primary in reverse order.
	for i := int(2); i >= 0; i-- {
		t, _ := m.substTable(*s, uint32(i))
		if t.payload != (arena.SmallBytesRef{}) {
			if e = m.vectorAllocator.Deallocate(t.capacity*24, t.payload); e != nil {
				return e
			}
		}
	}
	if e = m.allocator.Deallocate(astExprSubstBytes, s.record); e != nil {
		return e
	}
	*s = ASTExprSubstitution{}
	return failure
}
