// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/memory"

func (m *RationalManager) Error() string { return "mpnum: retained rational manager" }
func (m *RationalManager) Unwrap() error {
	if m.originalCause != nil {
		return m.originalCause
	}
	return m.cleanupCause
}
func (m *RationalManager) OriginalCause() error { return m.originalCause }
func (m *RationalManager) CleanupCause() error  { return m.cleanupCause }
func (m *RationalManager) RuntimeOwner() *memory.Runtime {
	if m == nil || m.self != m || m.z == nil {
		return nil
	}
	return m.z.RuntimeOwner()
}
func (m *RationalManager) Cleanup() error { return m.CloseChecked() }
func (m *RationalManager) CloseChecked() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed {
		m.cleanupCause = nil
		return nil
	}
	m.closing = true
	// mpq.cpp destructor del order; each mpq releases NUM then DEN.
	roots := [9]*Int{&m.nTmp, &m.addTmp1, &m.addTmp2, &m.ltTmp1.num, &m.ltTmp1.den, &m.ltTmp2.num, &m.ltTmp2.den, &m.addMulTmp.num, &m.addMulTmp.den}
	for m.cleanupStage < uint8(len(roots)) {
		if e := m.z.Del(roots[m.cleanupStage]); e != nil {
			m.cleanupCause = e
			return m
		}
		m.cleanupStage++
	}
	if e := m.z.CloseChecked(); e != nil {
		m.cleanupCause = e
		return m
	}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
	m.cleanupCause = nil
	return nil
}
