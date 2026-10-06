// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Licensed under the MIT license.
// Source mpn.h manager member destruction: t_ab,t_ms,v,u.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// SourceFinalize performs only the actual completed manager's source member
// destructors. Leaked failed-constructor locals and unpublished member expansion
// blocks remain in the charged owner for separate explicit owning CloseChecked.
func (m *NaturalManager) SourceFinalize() error {
	if m.oneHalfRootCallerExecuting() || m.knuthCallerExecuting() || m.removeOneHalfRootCallerExecuting() || m.rootUpperBoundCallerExecuting() {
		return arena.ErrInvalidSlot
	}
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.closing || m.runtime == nil || m.originalCause != nil {
		return arena.ErrClosed
	}
	return m.sourceFinalizeMembers()
}

// Owning Close resumes an explicitly begun source cursor without reopening
// the public half-destroyed owner.
func (m *NaturalManager) sourceFinalizeMembers() error {
	if m.sourceFinalized {
		return nil
	}
	if m.formatting.active || m.charFormatting.active || m.characterCalls != nil {
		return arena.ErrInvalidSlot
	}
	m.sourceFinalizing = true
	for m.sourceStage < 4 {
		b := &m.division[3-int(m.sourceStage)]
		if b.raw != (memory.Block{}) {
			// Runtime updates the real token after physical free, even if its later
			// TLS synchronization fails. Retry acknowledges that completed prefix.
			if e := m.runtime.Deallocate(&b.raw); e != nil {
				m.sourceCause = e
				return e
			}
		}
		m.sourceStage++
	}
	m.sourceFinalized = true
	m.sourceCause = nil
	return nil
}
