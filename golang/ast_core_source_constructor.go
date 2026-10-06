// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/ast.cpp:1297-1334; src/ast/ast.h:1454-1487.
// Source author: Leonardo de Moura.
package g3

import "errors"

const (
	astCtorAllocator uint32 = 1 << iota
	astCtorFamilies
	astCtorVectorAllocator
	astCtorArrays
	astCtorTable
	astCtorIDGenerators
	astCtorDebugFree
)

type ASTConstructorState struct {
	Complete                         bool
	CompletedMembers                 uint32
	UnwindStage                      uint8
	BodyCause, UnwindCause, OwnCause error
}

// ConstructorState is explicit diagnostic storage inspection of a returned
// unfinished Go context, not permission to call the incomplete C++ object.
func (m *ASTManager) ConstructorState() ASTConstructorState {
	if m == nil || m.self != m {
		return ASTConstructorState{BodyCause: ErrASTHandle}
	}
	if m.traceExecuting {
		panic(ErrASTPending)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return ASTConstructorState{m.constructorComplete, m.constructorMembers, m.constructorMemberStage, m.constructorBodyCause, m.constructorUnwindCause, m.constructorOwnCause}
}

func (m *ASTManager) unwindConstructorMembers() error {
	if m.constructorUnwindCause != nil {
		return m.constructorUnwindCause
	}
	stop := func(e error) error {
		if e != nil && m.constructorUnwindCause == nil {
			m.constructorUnwindCause = e
		}
		return e
	}
	// Body-owned trace and format pointers are deliberately not source members.
	// Plugins' pointer records also have POD destruction, not plugin finalize.
	if m.constructorMemberStage == 0 {
		if m.debugFreeComplete {
			m.debugFreeIndices.Close()
			m.debugFreeComplete = false
		}
		m.constructorMemberStage = 1
	}
	if m.constructorMemberStage == 1 {
		if m.constructorMembers&astCtorIDGenerators != 0 {
			m.declIDs.cleanup(1 << 31)
			m.exprIDs.cleanup(0)
		}
		m.constructorMemberStage = 2
	}
	if m.constructorMemberStage == 2 {
		if m.constructorMembers&astCtorTable != 0 {
			m.budget.Release(astChainBytes(uint32(len(m.table.cells))))
			m.table.cells = nil
			m.table.size = 0
			m.traceConstructorTableEnded()
		}
		m.constructorMemberStage = 3
	}
	if m.constructorMemberStage == 3 {
		if m.constructorMembers&astCtorArrays != 0 {
			m.dependencyArrays.closeAfterNodeTeardown()
			m.dependencies.closeAfterNodeTeardown()
			m.exprArrays.closeAfterNodeTeardown()
		}
		m.constructorMemberStage = 4
	}
	if m.constructorMemberStage == 4 {
		if m.families != nil {
			if e := m.families.Close(); e != nil {
				return stop(e)
			}
			m.families = nil
		}
		m.constructorMemberStage = 5
	}
	if m.constructorMemberStage == 5 {
		// Concrete Go vector payload provider is a separate owned adapter child.
		// This Go provider represents separately allocated global vector/parameter
		// backing; it is not a source member of ast_manager and remains owned until
		// explicit abort, including body-acquired plugin info arrays.

		if m.allocator != nil {
			m.allocator.Close()
			m.traceConstructorNodesEnded()
			m.allocator = nil
		}
		m.constructorMemberStage = 6
	}
	return nil
}

func (m *ASTManager) AbortConstruction() error {
	if m == nil || m.self != m {
		return ErrASTHandle
	}
	if m.traceExecuting {
		return ErrASTPending
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	return m.abortConstructionCore()
}

func (m *ASTManager) abortConstructionCore() error {
	if m.constructorComplete || m.constructorBodyCause == nil {
		return ErrASTDomain
	}
	stop := func(e error) error {
		if e != nil && m.constructorOwnCause == nil {
			m.constructorOwnCause = e
		}
		return e
	}
	if e := m.llFrameDestroy(true); e != nil {
		return stop(e)
	}
	if e := m.retireConstructorTraceAcquisitions(); e != nil {
		return stop(e)
	}
	// No successful constructor body is replayed. The exact returned child owner
	// distinguishes incomplete placement work from a completed format pointer.
	if m.constructorPendingFormat != nil {
		if e := m.constructorPendingFormat.AbortConstruction(); e != nil {
			return stop(e)
		}
		m.constructorPendingFormat = nil
	}
	if m.format != nil {
		if e := m.format.Close(); e != nil {
			return stop(e)
		}
		m.format = nil
	}
	if m.traceFile != nil {
		if e := m.traceFile.Close(); e != nil {
			return stop(e)
		}
		m.traceFile = nil
		m.traceStream = nil
		m.traceOwner = false
	}
	if m.constructorPendingFile != nil {
		if e := m.constructorPendingFile.Close(); e != nil {
			return stop(e)
		}
		m.constructorPendingFile = nil
	}
	if m.constructorUnwindCause != nil {
		return stop(errors.Join(m.constructorBodyCause, m.constructorUnwindCause))
	}
	// Automatic members have already physically consumed their payloads. Body
	// plugin cache storage is still an exact acquired Go owner; no node DEC or
	// ast_manager destructor is authorized for this incomplete outer object.
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
	m.destroyPluginBacking()
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
