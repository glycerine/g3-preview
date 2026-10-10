package ast

func (m *ASTManager) substMutation(s ASTExprSubstitution) error {
	if _, e := m.substState(s); e != nil {
		return e
	}
	v, e := m.readWord(s.record, 8)
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
		m.writeWord(s.record, 8, 1)
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
func (m *ASTManager) substStoreValue(t *astExprSubstTable, which uint32, token uint64, new astNodeRef, replacing bool) error {
	a, _ := t.entries.Get2(token)
	if e := m.substInc(which, new); e != nil {
		return e
	}
	var failure error
	if replacing {
		consumed, e := m.substDec(which, a.value)
		if !consumed {
			return e
		}
		failure = e
	}
	a.value = new
	a.owned |= 2
	t.entries.Set(token, a)
	return failure
}
func (m *ASTManager) substInsert(s ASTExprSubstitution, key, def, proof, dep astNodeRef) error {
	if e := m.substMutation(s); e != nil {
		return e
	}
	token, e := m.substKey(key)
	if e != nil {
		return e
	}
	flags, _ := m.substState(s)
	t, _ := m.substTable(s, 0)
	entry, found := t.entries.Get2(token)
	if !found {
		entry = astExprSubstEntry{key: key}
		t.entries.Put(token, entry)
	}
	// A null definition retains the source's repeated-new-branch behavior.
	fresh := entry.value.slot == 0
	if fresh {
		if e = m.substInc(0, key); e != nil {
			return m.substNoteFailure(s, e)
		}
		entry.owned |= 1
		t.entries.Set(token, entry)
	}
	if e = m.substStoreValue(t, 0, token, def, !fresh); e != nil {
		return m.substNoteFailure(s, e)
	}
	for which := uint32(1); which < 3; which++ {
		if flags&(1<<(which-1)) == 0 {
			continue
		}
		t, _ = m.substTable(s, which)
		_, found = t.entries.Get2(token)
		value := proof
		if which == 2 {
			value = dep
		}
		if fresh {
			if ASTSourceDebug && found {
				return m.substNoteFailure(s, ErrASTDomain)
			}
			// Optional pointers are published before their actual INC, as in source.
			entry = astExprSubstEntry{key: key, value: value}
			t.entries.Put(token, entry)
			if e = m.substInc(which, value); e != nil {
				return m.substNoteFailure(s, e)
			}
			entry.owned = 2
			t.entries.Set(token, entry)
		} else {
			if !found {
				return m.substNoteFailure(s, ErrASTDomain)
			}
			if e = m.substStoreValue(t, which, token, value, true); e != nil {
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
	t, _ := m.substTable(s, 0)
	token, found, e := m.substFind(t, key)
	if e != nil || !found {
		return found, e
	}
	a, _ := t.entries.Get2(token)
	*def = m.external(a.value)
	if flags&1 != 0 {
		t, _ = m.substTable(s, 1)
		if a, found := t.entries.Get2(token); found {
			*proof = m.external(a.value)
		}
	}
	if dep != nil && flags&2 != 0 {
		t, _ = m.substTable(s, 2)
		if a, found := t.entries.Get2(token); found {
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
func (m *ASTManager) substReleaseEntry(t *astExprSubstTable, which uint32, token uint64) error {
	a, _ := t.entries.Get2(token)
	var failure error
	for bit := uint32(1); bit <= 2; bit <<= 1 {
		if a.owned&bit == 0 {
			continue
		}
		r, target := a.key, uint32(0)
		if bit == 2 {
			r, target = a.value, which
		}
		consumed, e := m.substDec(target, r)
		if !consumed {
			if failure != nil {
				return failure
			}
			return e
		}
		a.owned &^= bit
		t.entries.Set(token, a)
		if failure == nil {
			failure = e
		}
	}
	return failure
}

func (m *ASTManager) substErase(s ASTExprSubstitution, key astNodeRef) error {
	if e := m.substMutation(s); e != nil {
		return e
	}
	token, e := m.substKey(key)
	if e != nil {
		return e
	}
	flags, _ := m.substState(s)
	var failure error
	for _, which := range [3]uint32{1, 2, 0} {
		if which != 0 && flags&(1<<(which-1)) == 0 {
			continue
		}
		t, _ := m.substTable(s, which)
		if _, found := t.entries.Get2(token); !found {
			continue
		}
		e = m.substReleaseEntry(t, which, token)
		a, _ := t.entries.Get2(token)
		if a.owned != 0 {
			return m.substNoteFailure(s, e)
		}
		// Native erase uses captured identity, including after the last key DEC.
		t.entries.Del(token)
		if failure == nil {
			failure = e
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
	for token := range t.entries.All() {
		e = m.substReleaseEntry(t, which, token)
		a, _ := t.entries.Get2(token)
		if a.owned == 0 {
			t.entries.Del(token)
		}
		if e != nil {
			return e
		}
	}
	t.entries.Clear()
	return nil
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
	return m.writeWord(s.record, 8, 0)
}
func (m *ASTManager) ExprSubstitutionReset(s ASTExprSubstitution) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.substReset(s)
}
func (m *ASTManager) substCleanup(s ASTExprSubstitution) error { return m.substReset(s) }

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
	// Owning destruction drains fully consumed errors but preserves the token
	// whenever a callback still owns a key or value reference.
	for which := uint32(0); which < 3; which++ {
		if which != 0 && flags&(1<<(which-1)) == 0 {
			continue
		}
		t, _ := m.substTable(*s, which)
		for token := range t.entries.All() {
			e = m.substReleaseEntry(t, which, token)
			a, _ := t.entries.Get2(token)
			if a.owned != 0 {
				return m.substNoteFailure(*s, e)
			}
			t.entries.Del(token)
			if failure == nil {
				failure = e
			}
		}
		t.entries.Clear()
	}
	state := m.exprSubstitutions[s.record]
	if e = m.allocator.Deallocate(astExprSubstBytes, s.record); e != nil {
		return e
	}
	m.BudgetField.Release(state.header)
	delete(m.exprSubstitutions, s.record)
	*s = ASTExprSubstitution{}
	return failure
}
