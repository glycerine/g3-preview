// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/ast.cpp:1397-1479; src/ast/ast.h:1454-1490.
// Source author: Leonardo de Moura.
package g3

func (m *ASTManager) SourceFinalize() error {
	if m == nil || m.self != m {
		return ErrASTHandle
	}
	if m.traceExecuting {
		return ErrASTPending
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.constructorComplete {
		return ErrASTDomain
	}
	if m.sourceDTCause != nil {
		return m.sourceDTCause
	}
	if m.sourceDTCompleted {
		return nil
	}
	if m.closed {
		return ErrASTHandle
	}
	m.sourceDTStarted = true
	e := m.closeCore(true)
	if e != nil && m.sourceDTCause == nil {
		m.sourceDTCause = e
	}
	return e
}

// Actual source DT consumed member storage. This explicit owning continuation
// drains exact remaining global parameter/info allocations and the Go header;
// it never replays the source destructor or its EOF action.
func (m *ASTManager) closeSourceOwnedStorageCore() error {
	if e := m.llFrameDestroy(true); e != nil {
		return e
	}
	if e := m.cleanupTraceAcquisitions(); e != nil {
		return e
	}
	stop := func(e error) error {
		if e != nil && m.errorOwnClose == nil {
			m.errorOwnClose = e
		}
		return e
	}
	for slot := uint32(1); slot < m.nextInfo; slot++ {
		r := m.infoAt(slot)
		if r.live {
			if e := m.dropInfoStorage(slot, r.generation); e != nil {
				return stop(e)
			}
		}
	}
	for slot := uint32(1); slot < m.nextParameter; slot++ {
		r := m.parameterAt(slot)
		if r.live {
			if e := m.dropParameter(ASTParameter{m.origin, slot, r.generation}); e != nil {
				return stop(e)
			}
		}
	}
	if m.vectorAllocator != nil {
		m.vectorAllocator.Close()
		m.vectorAllocator = nil
	}
	m.releaseTraceAcquisitionHistory()
	m.releasePages()
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	m.closed = true
	return nil
}
