// Source mpn.h manager member destruction: t_ab,t_ms,v,u.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// SourceFinalize performs only the actual completed manager's source member
// destructors. Leaked failed-constructor locals and unpublished member expansion
// blocks remain in the charged owner for separate explicit owning CloseChecked.

func (m *NaturalManager) SourceFinalize() error {
	if m.numericCallerExecuting() {
		return arena.ErrInvalidSlot
	}
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.ClosedField || m.Closing || m.runtime == nil || m.originalCause != nil {
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
	if m.Formatting.Active || m.CharFormatting.active || m.characterCalls != nil {
		return arena.ErrInvalidSlot
	}
	m.SourceFinalizing = true
	for m.SourceStage < 4 {
		b := &m.Division[3-int(m.SourceStage)]
		if b.Raw != (Block{}) {
			// Runtime updates the real token after physical free, even if its later
			// TLS synchronization fails. Retry acknowledges that completed prefix.
			if e := m.runtime.Deallocate(&b.Raw); e != nil {
				m.sourceCause = e
				return e
			}
		}
		m.SourceStage++
	}
	m.sourceFinalized = true
	m.sourceCause = nil
	return nil
}
