// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

func (m *ASTManager) appendVectorValues(v ASTRefVector, values []ASTHandle) error {
	for _, h := range values {
		r, e := m.vectorInput(h)
		if e != nil {
			return e
		}
		if e = m.pushVector(v, r); e != nil {
			return e
		}
	}
	return nil
}

// RefVectorAppendValues borrows the caller's scalar handles for this call only.
func (m *ASTManager) RefVectorAppendValues(v ASTRefVector, values []ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.vectorState(v); e != nil {
		return e
	}
	return m.appendVectorValues(v, values)
}

func (m *ASTManager) NewRefVectorFrom(values []ASTHandle) (ASTRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTRefVector{}, e
	}
	defer m.mu.Unlock()
	v, e := m.newRefVector()
	if e != nil {
		return ASTRefVector{}, e
	}
	if e = m.appendVectorValues(v, values); e != nil {
		// Constructor unwinding releases initialized entries, but cannot
		// release push_back's retained increment before failed growth.
		m.finalizeVector(v)
		m.allocator.Deallocate(astRefVectorRecordBytes, v.record)
		return ASTRefVector{}, e
	}
	return v, nil
}

func (m *ASTManager) RefVectorGetDefault(v ASTRefVector, index uint32, fallback ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTHandle{}, e
	}
	if index >= s.size {
		return fallback, nil
	}
	r, e := m.vectorValue(s, index)
	return m.external(r), e
}

func (m *ASTManager) RefVectorEmpty(v ASTRefVector) (bool, error) {
	n, e := m.RefVectorSize(v)
	return n == 0, e
}

func (m *ASTManager) eraseVector(v ASTRefVector, s astRefVectorState, index uint32) error {
	old, e := m.vectorValue(s, index)
	if e != nil {
		return e
	}
	if index >= s.size {
		return ErrASTDomain
	}
	for i := index + 1; i < s.size; i++ {
		r, e := m.vectorValue(s, i)
		if e != nil {
			return e
		}
		if e = m.vectorPut(s, i-1, r); e != nil {
			return e
		}
	}
	s.size--
	if e = m.vectorPutWord(s.payload, 4, s.size); e != nil {
		return e
	}
	if e = m.storeVector(v, s); e != nil {
		return e
	}
	_, e = m.decNode(old)
	return e
}

func (m *ASTManager) RefVectorEraseValue(v ASTRefVector, h ASTHandle) error {
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
	for i := uint32(0); i < s.size; i++ {
		value, e := m.vectorValue(s, i)
		if e != nil {
			return e
		}
		if value == r {
			return m.eraseVector(v, s, i)
		}
	}
	return nil
}

// MoveRefVector constructs a new scalar owner record and transfers backing
// without reference callbacks. The source vector remains valid and empty.
func (m *ASTManager) MoveRefVector(src ASTRefVector) (ASTRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTRefVector{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(src)
	if e != nil {
		return ASTRefVector{}, e
	}
	v, e := m.newRefVector()
	if e != nil {
		return ASTRefVector{}, e
	}
	if e = m.storeVector(v, s); e != nil {
		m.allocator.Deallocate(astRefVectorRecordBytes, v.record)
		return ASTRefVector{}, e
	}
	if e = m.storeVector(src, astRefVectorState{}); e != nil {
		m.storeVector(v, astRefVectorState{})
		m.allocator.Deallocate(astRefVectorRecordBytes, v.record)
		return ASTRefVector{}, e
	}
	return v, nil
}
