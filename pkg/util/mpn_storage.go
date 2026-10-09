// Source: src/util/mpn.h manager scratch and util/buffer.h growth policy.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// NaturalManager owns source division scratch, an embedded lexical formatting
// scope, and the explicitly separate retained output assembly adapter.
// Operands/output slices are borrowed only during a call. A manager may not be
// copied or used concurrently. Close invalidates the service. Requested source
// capacities are kept separate from arena geometric allocation classes.
type NaturalManager struct {
	callerExecuting func() bool
	callerBindings  *NumericCallerBinding

	self                              *NaturalManager
	runtime                           *Runtime
	ownsRuntime                       bool
	Division                          [4]SourceLimbBuffer
	Verification                      SourceLimbBuffer
	VerificationInvocation            naturalPNVerificationScope
	Formatting                        naturalFormattingScope
	CharFormatting                    integerCharacterScope
	characterCalls                    *integerCharacterScope
	SourceFinalizing, sourceFinalized bool
	SourceStage                       uint8
	sourceCause                       error
	VerificationActive                bool
	originalCause, cleanupCause       error
	Text                              *arena.SmallBytes
	textRef                           arena.SmallBytesRef
	textCapacity                      uint32
	config                            arena.VectorConfig
	budget                            *arena.Budget
	HeaderBytes                       uint64
	ClosedField, Closing              bool
}

// NewNaturalManager owns a real memory runtime and the source four inline16
// buffers. Formatting locals own separate Runtime buffers; retained text
// is an output assembly adapter and character scratch uses native Go bytes.
func NewNaturalManager(c arena.VectorConfig) (*NaturalManager, error) {
	return newNaturalManagerWithMemory(c, nil)
}
func newNaturalManagerWithMemory(c arena.VectorConfig, runtime *Runtime) (*NaturalManager, error) {
	header := uint64(reflect.TypeOf(NaturalManager{}).Size())
	if e := c.Descriptors.Budget.Reserve(header); e != nil {
		return nil, e
	}
	m := &NaturalManager{config: c, budget: c.Descriptors.Budget, HeaderBytes: header, runtime: runtime}
	m.self = m
	if runtime == nil {
		m.ownsRuntime = true
		var e error
		m.runtime, e = NewRuntime(RuntimeConfig{Budget: m.budget, SizeTBits: uint8(32 << (^uint(0) >> 63))})
		if e != nil {
			m.originalCause = e
			if cleanup := m.CloseChecked(); cleanup != nil {
				return nil, m
			}
			return nil, e
		}
	}
	return m, nil
}
func (m *NaturalManager) RuntimeOwner() *Runtime {
	if m == nil || m.self != m {
		return nil
	}
	return m.runtime
}
func (m *NaturalManager) Error() string { return "mpnum: retained Natural cleanup owner" }
func (m *NaturalManager) Unwrap() error {
	if m == nil {
		return nil
	}
	if m.originalCause != nil {
		return m.originalCause
	}
	return m.cleanupCause
}
func (m *NaturalManager) OriginalCause() error {
	if m == nil {
		return nil
	}
	return m.originalCause
}
func (m *NaturalManager) CleanupCause() error {
	if m == nil {
		return nil
	}
	return m.cleanupCause
}
func (m *NaturalManager) check() error {
	if m == nil || m.self == nil {
		return arena.ErrClosed
	}
	if m.self != m {
		return ErrRuntimeOwner
	}
	if m.ClosedField || m.Closing || m.runtime == nil || m.SourceFinalizing || m.originalCause != nil {
		return arena.ErrClosed
	}
	return nil
}
func (m *NaturalManager) Close() {
	if m == nil {
		return
	}
	if e := m.CloseChecked(); e != nil {
		panic(m)
	}
}
func (m *NaturalManager) CloseChecked() error {
	if m.numericCallerExecuting() {
		return arena.ErrInvalidSlot
	}
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.ClosedField {
		return nil
	}
	if m.characterCalls != nil {
		return arena.ErrInvalidSlot
	}
	m.Closing = true
	fail := func(e error) error { m.cleanupCause = e; return e }
	if m.SourceFinalizing && !m.sourceFinalized {
		if e := m.sourceFinalizeMembers(); e != nil {
			return fail(e)
		}
	}
	if m.runtime != nil {
		if e := m.cleanupOutputFormatting(); e != nil {
			return fail(e)
		}
		if m.VerificationActive {
			if e := m.Verification.Finalize(m.runtime); e != nil {
				return fail(e)
			}
			m.VerificationActive = false
		}
		for i := 3; i >= 0; i-- {
			if e := m.Division[i].Finalize(m.runtime); e != nil {
				return fail(e)
			}
		}
	}
	if m.Text != nil {
		m.Text.Close()
		m.Text = nil
	}
	if m.ownsRuntime && m.runtime != nil {
		if e := m.runtime.Close(); e != nil {
			return fail(e)
		}
	}
	m.budget.Release(m.HeaderBytes)
	m.HeaderBytes = 0
	m.ClosedField = true
	m.cleanupCause = nil
	return nil
}
func (m *NaturalManager) ResizeDiv(i int, n uint32) error { return m.resizeDivWithPNReceipt(i, n, nil) }
func (m *NaturalManager) resizeDivWithPNReceipt(i int, n uint32, pn *integerPNormalizeScope) error {
	return m.resizeDivWithAllocationContext(i, n, pnAllocationContext(pn))
}
func (m *NaturalManager) resizeDivWithAllocationContext(i int, n uint32, context integerAllocationInvocationContext) error {
	if i < 0 || i >= 4 {
		return arena.ErrBounds
	}
	e := m.Division[i].resizeWithPNReceipt(m.runtime, n, m.DivisionBufferReceipt(context, i))
	return e
}
func (m *NaturalManager) sourceDivRead(r int, i uint32) uint32 {
	v, e := m.Division[r].read(m.runtime, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *NaturalManager) sourceDivWrite(r int, i, v uint32) {
	if e := m.Division[r].write(m.runtime, i, v); e != nil {
		panic(e)
	}
}

// overflowPayloads counts actual live/pending source allocations. Inline16
// buffers have no physical payload, and released Runtime blocks clear tokens.
func (m *NaturalManager) overflowPayloads() uint32 {
	if m == nil || m.self != m || m.ClosedField {
		return 0
	}
	var count uint32
	for i := range m.Division {
		if m.Division[i].Raw != (Block{}) {
			count++
		}
		if m.Division[i].Pending != (Block{}) {
			count++
		}
	}
	for i := range m.Formatting.Locals {
		if m.Formatting.Locals[i].Raw != (Block{}) {
			count++
		}
		if m.Formatting.Locals[i].Pending != (Block{}) {
			count++
		}
	}
	if m.VerificationActive {
		if m.Verification.Raw != (Block{}) {
			count++
		}
		if m.Verification.Pending != (Block{}) {
			count++
		}
	}
	return count
}
