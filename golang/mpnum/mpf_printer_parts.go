// Source: src/util/mpf.h:152-153,192-193;
// src/ast/ast_smt2_pp.cpp:277-288.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// ExponentBias borrows the actual cached m_powers2.m1 numeral and reaches the
// same signed get_int64 leaf used by the printer. The cache remains its owner.
func (m *ArbitraryFloatManager) ExponentBias(ebits uint32) (int64, error) {
	if e := m.valueGuard(&ArbitraryFloat{}); e != nil {
		return 0, e
	}
	w, e := m.powerTwoMinusOneWrapper(ebits-1, false)
	if e != nil {
		return 0, e
	}
	a, e := w.capture()
	if e != nil {
		return 0, e
	}
	return m.integer.Int64(&a)
}

// CopySignificand is scoped_mpz assignment from fm.sig(v), not an MPF copy.
// Its opaque child is issued solely by the existing actual Integer SET leaf.
func (m *ArbitraryFloatManager) CopySignificand(out *Int, a *ArbitraryFloat, receipt *ArbitraryFloatValueCopyReceipt) (cause error) {
	if e := m.valueGuard(a); e != nil {
		return e
	}
	if out == nil {
		return memory.ErrRuntimeOwner
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	if e := m.beginValueCopyBinding(receipt, m.integer); e != nil {
		return e
	}
	defer func() { m.finishValueCopyBinding(cause) }()
	return m.copyValueSignificand(out, &a.significand)
}

// BorrowScopedInteger exposes the explicit source manager for rational(mpz).
// This grants no allocation or cleanup authority to the borrowing caller.
func (m *ArbitraryFloatManager) BorrowScopedInteger(a *Int) (*IntegerManager, error) {
	if e := m.valueGuard(&ArbitraryFloat{}); e != nil {
		return nil, e
	}
	if e := m.integer.check(a); e != nil {
		return nil, e
	}
	return m.integer, nil
}

func (m *ArbitraryFloatManager) DeleteScopedInteger(a *Int) error {
	if e := m.valueGuard(&ArbitraryFloat{}); e != nil {
		return e
	}
	if a == nil {
		return memory.ErrRuntimeOwner
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	return m.integer.Del(a)
}

// CleanupScopedInteger is the separate owning continuation for a failed SET
// or source-ended numeral. It does not acknowledge an unjoined copy child.
func (m *ArbitraryFloatManager) CleanupScopedInteger(a *Int) error {
	if m == nil || m.self != m || m.integer == nil || m.integer.self != m.integer || a == nil {
		return memory.ErrRuntimeOwner
	}
	if !m.constructorComplete || m.closed {
		return arena.ErrClosed
	}
	if m.valueExecuting || m.integer.mpfBorrowedSourceCallbackActive() {
		return ErrPolynomialPending
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	return m.integer.cleanupRawNumeral(a)
}
