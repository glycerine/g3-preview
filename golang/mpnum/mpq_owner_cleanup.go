// Licensed under the MIT license.
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
	if m.z != nil && (m.z.copyAllocation.executing || m.z.normalizationExecuting || m.z.factorNumeralExecuting || m.z.factorNumeralVectorCallerExecuting() || m.z.factorUnivariateCallerExecuting() || m.z.factorCoreCallerExecuting()) {
		return ErrPolynomialPending
	}
	if m.closed {
		m.cleanupCause = nil
		return nil
	}
	if m.sourceFinalizing && !m.sourceFinalized && !m.sourceGuard {
		if e := m.SourceFinalize(); e != nil {
			m.cleanupCause = e
			return m
		}
	}
	m.closing = true
	if e := m.cleanupRawArithmetic(); e != nil {
		m.cleanupCause = e
		return m
	}
	if e := m.lexical.cleanup(m.z); e != nil {
		m.cleanupCause = e
		return m
	}
	if !m.sourceGuard {
		if e := m.cleanupOuter(false); e != nil {
			m.cleanupCause = e
			return m
		}
	}
	if e := m.z.CloseChecked(); e != nil {
		m.cleanupCause = e
		return m
	}
	// Owning abort releases only child-known roots; stopped source history stays.
	if m.sourceGuard {
		m.nTmp = Int{}
		m.addTmp1 = Int{}
		m.addTmp2 = Int{}
		m.ltTmp1 = Rational{}
		m.ltTmp2 = Rational{}
		m.addMulTmp = Rational{}
	}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
	m.cleanupCause = nil
	return nil
}
