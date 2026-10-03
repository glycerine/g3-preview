// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// mpq.cpp28-34 destructor body followed by mpz_manager base destruction.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// sourceNumeralDel checks the actual capacity access. The permanently dead
// failure token is accepted by owning cleanup, not by literal source destruction.
func sourceNumeralDel(z *IntegerManager, a *Int) (bool, error) {
	if a.cell != 0 {
		if !validNumericRef(z.cells, a.cell) {
			return true, arena.ErrInvalidSlot
		}
		c := z.cell(a.cell)
		if c.cleanupPending {
			// Persisted descriptor/release authority acknowledges an interrupted
			// owning cleanup step; a completed free must not reread capacity.
			// This is Go continuation, not another C++ destructor invocation.
			return false, z.Del(a)
		}
		if c.released {
			return true, arena.ErrInvalidSlot
		}
		if _, e := c.backing.readWord(z.allocator, 4); e != nil {
			return true, e
		}
	}
	return false, z.Del(a)
}
func (m *RationalManager) cleanupOuter(source bool) error {
	roots := [9]*Int{&m.nTmp, &m.addTmp1, &m.addTmp2, &m.ltTmp1.num, &m.ltTmp1.den, &m.ltTmp2.num, &m.ltTmp2.den, &m.addMulTmp.num, &m.addMulTmp.den}
	for m.cleanupStage < 9 {
		var e error
		if source {
			var guard bool
			guard, e = sourceNumeralDel(m.z, roots[m.cleanupStage])
			if guard {
				m.sourceGuard = true
				m.sourceCause = e
			}
		} else {
			e = m.z.Del(roots[m.cleanupStage])
		}
		if e != nil {
			return e
		}
		m.cleanupStage++
	}
	return nil
}

// SourceFinalize preserves surviving caller direct blocks and all charged cold
// ownership metadata. CloseChecked is the separate owning cleanup operation.
func (m *RationalManager) SourceFinalize() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceFinalized {
		return nil
	}
	if m.originalCause != nil || (m.closing && !m.sourceFinalizing) {
		return arena.ErrClosed
	}
	m.sourceFinalizing = true
	if m.sourceGuard {
		return m.sourceCause
	}
	fail := func(e error) error { m.sourceCause = e; m.cleanupCause = e; return e }
	if e := m.cleanupOuter(true); e != nil {
		return fail(e)
	}
	if e := m.z.SourceFinalize(); e != nil {
		if m.z.sourceGuard {
			m.sourceGuard = true
		}
		return fail(e)
	}
	m.sourceFinalized = true
	m.sourceCause = nil
	m.cleanupCause = nil
	return nil
}
