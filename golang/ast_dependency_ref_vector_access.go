// Licensed under the MIT license.
package g3

func (m *ASTManager) dependencyAppendVectorValues(v ASTDependencyRefVector, values []ASTDependency) error {
	for _, h := range values {
		r, e := m.dependencyVectorInput(h)
		if e != nil {
			return e
		}
		if e = m.dependencyPushVector(v, r); e != nil {
			return e
		}
	}
	return nil
}

// RefVectorAppendValues borrows the caller's scalar handles for this call only.
func (m *ASTManager) DependencyRefVectorAppendValues(v ASTDependencyRefVector, values []ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if _, e := m.dependencyVectorState(v); e != nil {
		return e
	}
	return m.dependencyAppendVectorValues(v, values)
}

func (m *ASTManager) NewDependencyRefVectorFrom(values []ASTDependency) (ASTDependencyRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefVector{}, e
	}
	defer m.mu.Unlock()
	v, e := m.newDependencyRefVector()
	if e != nil {
		return ASTDependencyRefVector{}, e
	}
	if e = m.dependencyAppendVectorValues(v, values); e != nil {
		// Constructor unwinding releases initialized entries, but cannot
		// release push_back's retained increment before failed growth.
		return m.dependencyFailedVectorConstructor(v, e)
	}
	return v, nil
}

func (m *ASTManager) DependencyRefVectorGetDefault(v ASTDependencyRefVector, index uint32, fallback ASTDependency) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return ASTDependency{}, e
	}
	if index >= s.size {
		return fallback, nil
	}
	r, e := m.dependencyVectorValue(s, index)
	return m.dependencies.external(dependencyRefPointer(r)), e
}

func (m *ASTManager) DependencyRefVectorEmpty(v ASTDependencyRefVector) (bool, error) {
	n, e := m.DependencyRefVectorSize(v)
	return n == 0, e
}

func (m *ASTManager) dependencyEraseVector(v ASTDependencyRefVector, s astDependencyRefVectorState, index uint32) error {
	if s.decPending {
		return ErrASTDependencyCleanupPending
	}
	old, e := m.dependencyVectorValue(s, index)
	if e != nil {
		return e
	}
	if index >= s.size {
		return ErrASTDomain
	}
	for i := index + 1; i < s.size; i++ {
		r, e := m.dependencyVectorValue(s, i)
		if e != nil {
			return e
		}
		if e = m.dependencyVectorPut(s, i-1, r); e != nil {
			return e
		}
	}
	s.size--
	if e = m.dependencyVectorPutWord(s.payload, 4, s.size); e != nil {
		return e
	}
	if e = m.dependencyStoreVector(v, s); e != nil {
		return e
	}
	_, e = m.dependencyArrays.valueDec(dependencyRefPointer(old))
	return e
}

func (m *ASTManager) DependencyRefVectorEraseValue(v ASTDependencyRefVector, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(v)
	if e != nil {
		return e
	}
	if !h.IsNull() && h.origin != m.origin {
		return ErrASTHandle
	}
	r := astNodeRef{uint32(h.record.Slot), uint32(h.record.Generation)}
	for i := uint32(0); i < s.size; i++ {
		value, e := m.dependencyVectorValue(s, i)
		if e != nil {
			return e
		}
		if value == r {
			return m.dependencyEraseVector(v, s, i)
		}
	}
	return nil
}

// MoveRefVector constructs a new scalar owner record and transfers backing
// without reference callbacks. The source vector remains valid and empty.
func (m *ASTManager) MoveDependencyRefVector(src ASTDependencyRefVector) (ASTDependencyRefVector, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyRefVector{}, e
	}
	defer m.mu.Unlock()
	s, e := m.dependencyVectorState(src)
	if e != nil {
		return ASTDependencyRefVector{}, e
	}
	v, e := m.newDependencyRefVector()
	if e != nil {
		return ASTDependencyRefVector{}, e
	}
	if e = m.dependencyStoreVector(v, s); e != nil {
		m.allocator.Deallocate(astDependencyRefVectorRecordBytes, v.record)
		return ASTDependencyRefVector{}, e
	}
	if e = m.dependencyStoreVector(src, astDependencyRefVectorState{}); e != nil {
		m.dependencyStoreVector(v, astDependencyRefVectorState{})
		m.allocator.Deallocate(astDependencyRefVectorRecordBytes, v.record)
		return ASTDependencyRefVector{}, e
	}
	return v, nil
}
