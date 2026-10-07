// Source: src/util/mpz.cpp, mpn.cpp, mpzzp.h: primitive invocation boundaries.
// Licensed under the MIT license.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// NumericInvocationHooks transports caller-owned lifetime policy. The hooks
// borrow the caller only during the operation; utility managers never store
// concrete polynomial or algebraic owners. Roots must validate caller identity
// and generation before returning ownership of allocation prefixes.

type NumericInvocationHooks struct {
	Policy       NumericCallerPolicy
	Roots        func(*IntegerManager, *NumericAllocationRoots) *NumericAllocationRoots
	Prepare      func(*IntegerManager, string) error
	Record       func(*IntegerManager, string, NumericInvocationReceipt)
	Buffer       func(*NaturalManager, int) *NaturalBufferReceipt
	Verification func(*NaturalManager) (*NaturalVerificationScope, error)
	ScalarRaw    func(*IntegerManager, string) (*IntegerScalarRawScope, error)
	Normalize    func(*IntegerModularManager, Int) (Int, error)
	Power        func(*IntegerModularManager) (*Int, func() error, func(error), error)
	SourceCause  func(*IntegerManager) error
	Bound        func(bool) bool
}

type NumericInvocationReceipt struct {
	Raw           *IntegerRawInvocationReceipt
	Divides       *IntegerDividesNumeralInvocationReceipt
	PerfectSquare *IntegerPerfectSquareInvocationReceipt
}
type NumericAllocationContext = integerAllocationInvocationContext
type IntegerRawInvocationReceipt = integerRawInvocationReceipt
type IntegerDividesNumeralInvocationReceipt = integerDividesNumeralInvocationReceipt
type IntegerPerfectSquareInvocationReceipt = integerPerfectSquareInvocationReceipt
type NaturalBufferReceipt = naturalPNBufferReceipt
type NaturalVerificationScope = naturalPNVerificationScope
type IntegerScalarRawScope = integerScalarRawScope

func (c integerAllocationInvocationContext) WithCaller(h *NumericInvocationHooks) integerAllocationInvocationContext {
	c.Caller = h
	return c
}
func (c integerAllocationInvocationContext) roots(m *IntegerManager) *NumericAllocationRoots {
	r := c.localRoots(m)
	if c.hasLocalInvocation() && r == nil {
		return nil
	}
	if c.Caller != nil {
		callerRoots := c.Caller.callRoots(m, r)
		if callerRoots == nil {
			return nil
		}
		if r != nil {
			return r
		}
		return callerRoots
	}
	return r
}
func (c integerAllocationInvocationContext) hasLocalInvocation() bool {
	return c.rationalService != nil || c.mpfRationalSet != nil || c.mpfFormattedSet != nil || c.mpfInput != nil || c.mpfConversion != nil || c.mpfFormat != nil || c.mpfPower != nil || c.mpfRound != nil || c.mpfRoundSqrt != nil || c.BoolDivides != nil || c.raw != nil || c.pn != nil || c.addMul != nil || c.perfectSquare != nil
}
func (c integerAllocationInvocationContext) prepareCaller(m *IntegerManager, phase string) error {
	if c.Caller != nil {
		return c.Caller.callPrepare(m, phase)
	}
	return nil
}
func (c integerAllocationInvocationContext) recordCaller(m *IntegerManager, event string, r NumericInvocationReceipt) {
	if c.Caller != nil && r.Raw != nil && r.Raw.Frame != nil {
		r.Raw.Frame.caller = c.Caller
	}
	if c.Caller != nil {
		c.Caller.callRecord(m, event, r)
	}
}
func (m *IntegerManager) numericCallerExecuting() bool {
	return m != nil && m.self == m && (m.callerExecuting != nil && m.callerExecuting() || callerBindingsExecuting(m.callerBindings))
}
func (m *NaturalManager) numericCallerExecuting() bool {
	return m != nil && m.self == m && (m.callerExecuting != nil && m.callerExecuting() || callerBindingsExecuting(m.callerBindings))
}

// SetCallerExecutionGuard binds the higher owner's synchronous reentry guard.
// The owner must clear its guard before destroying the borrowed manager.
func (m *IntegerManager) SetCallerExecutionGuard(guard func() bool) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if m.numericCallerExecuting() {
		return ErrNumericPending
	}
	m.callerExecuting = guard
	return nil
}
func (m *NaturalManager) SetCallerExecutionGuard(guard func() bool) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.ClosedField {
		return arena.ErrClosed
	}
	if m.numericCallerExecuting() {
		return ErrNumericPending
	}
	m.callerExecuting = guard
	return nil
}

// NumericCharacterCaller borrows primitive formatting storage from a caller.
type NumericCharacterPolicy interface {
	NumericCharacterValid(*NaturalManager, bool) bool
	NumericCharacterPrepare() error
}

type NumericCharacterCaller struct {
	Policy         NumericCharacterPolicy
	Characters     *IntegerCharacterScope
	Formatting     *NaturalFormattingScope
	Generation     uint64
	Valid          func(*NaturalManager, bool) bool
	PrepareStorage func() error
}
type IntegerCharacterScope = integerCharacterScope
type NaturalFormattingScope = naturalFormattingScope

// BindCallerBuffer attaches a higher caller's receipt to the actual persistent
// Natural division buffer. Bound validates the caller's receipt address and
// invocation, so copies and expired generations cannot acquire pending storage.
func (m *NaturalManager) BindCallerBuffer(r *NaturalBufferReceipt, index int, invocation uint64, h *NumericInvocationHooks) error {
	if m == nil || m.self != m || r == nil || h == nil || !h.callBound(true) || invocation == 0 {
		return ErrNumericOwner
	}
	if m.ClosedField {
		return arena.ErrClosed
	}
	if index < 0 || index >= len(m.Division) {
		return arena.ErrBounds
	}
	if r.Owner != nil {
		if r.Owner != m || r.Buffer != &m.Division[index] || r.Invocation != invocation || r.Caller != h {
			return ErrNumericOwner
		}
		return nil
	}
	*r = naturalPNBufferReceipt{CallerSelf: r, Owner: m, Buffer: &m.Division[index], Invocation: invocation, Caller: h}
	return nil
}
func (r *naturalPNBufferReceipt) Cleanup() error {
	if r == nil {
		return nil
	}
	return r.cleanup()
}

// BeginCallerVerification constructs the real source lexical verification
// holder; successful source destruction remains in the utility algorithm.
func (m *NaturalManager) BeginCallerVerification(h *NumericInvocationHooks) (*NaturalVerificationScope, error) {
	if m == nil || m.self != m || h == nil || !h.callBound(true) {
		return nil, ErrNumericOwner
	}
	if m.ClosedField {
		return nil, arena.ErrClosed
	}
	s := &m.VerificationInvocation
	if s.active || s.invocation == ^uint64(0) {
		return nil, ErrNumericPending
	}
	*s = naturalPNVerificationScope{caller: h, active: true, executing: true, invocation: s.invocation + 1}
	s.Buffer = naturalPNBufferReceipt{CallerSelf: &s.Buffer, Owner: m, Buffer: &m.Verification, private: true, Invocation: s.invocation, Caller: h}
	return s, nil
}
func (s *naturalPNVerificationScope) Cleanup(m *NaturalManager, abort bool) error {
	if m == nil || m.self != m || s != &m.VerificationInvocation {
		return ErrNumericOwner
	}
	r := naturalPNVerificationReceipt{m, s, s.invocation}
	return m.CleanupPNVerification(&r, abort)
}

func (m *IntegerModularManager) SetCallerExecutionGuard(guard func() bool) error {
	if m == nil || m.self != m {
		return ErrNumericOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if m.numericCallerExecuting() {
		return ErrNumericPending
	}
	m.CallerExecuting = guard
	return nil
}

func (c *NumericCharacterCaller) valid(n *NaturalManager, executing bool) bool {
	if c == nil {
		return false
	}
	if c.Policy != nil {
		return c.Policy.NumericCharacterValid(n, executing)
	}
	return c.Valid != nil && c.Valid(n, executing)
}
func (c *NumericCharacterCaller) prepare() error {
	if c.Policy != nil {
		return c.Policy.NumericCharacterPrepare()
	}
	if c.PrepareStorage != nil {
		return c.PrepareStorage()
	}
	return nil
}
