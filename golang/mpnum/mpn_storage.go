// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Licensed under the MIT license.
// Source: src/util/mpn.h manager scratch and util/buffer.h growth policy.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// NaturalManager owns source division scratch, an embedded lexical formatting
// scope, and the explicitly separate retained output assembly adapter.
// Operands/output slices are borrowed only during a call. A manager may not be
// copied or used concurrently. Close invalidates the service. Requested source
// capacities are kept separate from arena geometric allocation classes.
type NaturalManager struct {
	knuthCaller                       *UPolynomialManager
	oneHalfRootCaller                 *UPolynomialManager
	rootUpperBoundCaller              *UPolynomialManager
	self                              *NaturalManager
	runtime                           *memory.Runtime
	ownsRuntime                       bool
	division                          [4]sourceLimbBuffer
	verification                      sourceLimbBuffer
	verificationInvocation            naturalPNVerificationScope
	formatting                        naturalFormattingScope
	charFormatting                    integerCharacterScope
	characterCalls                    *integerCharacterScope
	sourceFinalizing, sourceFinalized bool
	sourceStage                       uint8
	sourceCause                       error
	verificationActive                bool
	originalCause, cleanupCause       error
	words                             *arena.Words
	divScratch                        [4]arena.VectorRef
	divCapacity                       [4]uint32
	// Zero compatibility fields: no numeral/formatting payload is allocated here.
	formatScratch              [3]arena.VectorRef
	formatCapacity             [3]uint32
	text                       *arena.SmallBytes
	textRef                    arena.SmallBytesRef
	textCapacity               uint32
	config                     arena.VectorConfig
	budget                     *arena.Budget
	headerBytes                uint64
	ownsWords, closed, closing bool
}

// NewNaturalManager owns a real memory runtime and the source four inline16
// buffers. Formatting locals use real lexical source buffers; retained text is
// a separate output assembly adapter until char/ostream integration lands.
func NewNaturalManager(c arena.VectorConfig) (*NaturalManager, error) {
	words, e := arena.NewWords(c)
	if e != nil {
		return nil, e
	}
	m, e := newNaturalManager(c, words, true)
	if e != nil && m == nil {
		words.Close()
	}
	return m, e
}
func newNaturalManager(c arena.VectorConfig, words *arena.Words, owns bool) (*NaturalManager, error) {
	return newNaturalManagerWithMemory(c, words, owns, nil)
}
func newNaturalManagerWithMemory(c arena.VectorConfig, words *arena.Words, owns bool, runtime *memory.Runtime) (*NaturalManager, error) {
	header := uint64(reflect.TypeOf(NaturalManager{}).Size())
	if e := c.Descriptors.Budget.Reserve(header); e != nil {
		return nil, e
	}
	m := &NaturalManager{words: words, config: c, budget: c.Descriptors.Budget, headerBytes: header, ownsWords: owns, runtime: runtime}
	m.self = m
	for i := range m.divCapacity {
		m.divCapacity[i] = 16
	}
	if runtime == nil {
		m.ownsRuntime = true
		var e error
		m.runtime, e = memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget, SizeTBits: uint8(32 << (^uint(0) >> 63))})
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
func (m *NaturalManager) RuntimeOwner() *memory.Runtime {
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
	if m == nil || m.words == nil {
		return arena.ErrClosed
	}
	if m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.closing || m.runtime == nil || m.sourceFinalizing || m.originalCause != nil {
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
	if m.oneHalfRootCallerExecuting() || m.knuthCallerExecuting() || m.rootUpperBoundCallerExecuting() {
		return arena.ErrInvalidSlot
	}
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed {
		return nil
	}
	if m.characterCalls != nil {
		return arena.ErrInvalidSlot
	}
	m.closing = true
	fail := func(e error) error { m.cleanupCause = e; return e }
	if m.sourceFinalizing && !m.sourceFinalized {
		if e := m.sourceFinalizeMembers(); e != nil {
			return fail(e)
		}
	}
	if m.runtime != nil {
		if e := m.cleanupOutputFormatting(); e != nil {
			return fail(e)
		}
		if m.verificationActive {
			if e := m.verification.finalize(m.runtime); e != nil {
				return fail(e)
			}
			m.verificationActive = false
		}
		for i := 3; i >= 0; i-- {
			if e := m.division[i].finalize(m.runtime); e != nil {
				return fail(e)
			}
		}
	}
	if m.words != nil {
		for i, r := range m.formatScratch {
			if r != (arena.VectorRef{}) {
				if e := m.words.Free(r); e != nil {
					return fail(e)
				}
				m.formatScratch[i] = arena.VectorRef{}
			}
		}
	}
	if m.text != nil {
		m.text.Close()
		m.text = nil
	}
	if m.ownsWords && m.words != nil {
		m.words.Close()
	}
	if m.ownsRuntime && m.runtime != nil {
		if e := m.runtime.Close(); e != nil {
			return fail(e)
		}
	}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
	m.cleanupCause = nil
	return nil
}
func sourceCapacity(current, needed uint32) (uint32, error) {
	if current == 0 {
		current = 16
	}
	for current < needed {
		if current > ^uint32(0)/2 {
			return 0, arena.ErrBounds
		}
		current *= 2
	}
	return current, nil
}
func (m *NaturalManager) resizeScratch(ref *arena.VectorRef, capacity *uint32, size uint32) error {
	c, e := sourceCapacity(*capacity, size)
	if e != nil {
		return e
	}
	var r arena.VectorRef
	if *ref == (arena.VectorRef{}) {
		r, e = m.words.Allocate(size, c)
	} else {
		r, e = m.words.Reserve(*ref, c)
		if e == nil {
			*ref = r
			r, e = m.words.Resize(r, size)
		}
	}
	if e != nil {
		return e
	}
	*ref = r
	*capacity = c
	return nil
}
func (m *NaturalManager) resizeDiv(i int, n uint32) error { return m.resizeDivWithPNReceipt(i, n, nil) }
func (m *NaturalManager) resizeDivWithPNReceipt(i int, n uint32, pn *integerPNormalizeScope) error {
	return m.resizeDivWithAllocationContext(i, n, pnAllocationContext(pn))
}
func (m *NaturalManager) resizeDivWithAllocationContext(i int, n uint32, context integerAllocationInvocationContext) error {
	if i < 0 || i >= 4 {
		return arena.ErrBounds
	}
	e := m.division[i].resizeWithPNReceipt(m.runtime, n, m.divisionBufferReceipt(context, i))
	m.divCapacity[i] = m.division[i].capacity()
	return e
}
func (m *NaturalManager) sourceDivRead(r int, i uint32) uint32 {
	v, e := m.division[r].read(m.runtime, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *NaturalManager) sourceDivWrite(r int, i, v uint32) {
	if e := m.division[r].write(m.runtime, i, v); e != nil {
		panic(e)
	}
}

// Legacy test diagnostic only. Division never stores Words references.
func (m *NaturalManager) divRead(r arena.VectorRef, i uint32) uint32 {
	v, e := m.words.Read(r, i)
	if e != nil {
		panic(e)
	}
	return v
}

// overflowPayloads counts actual live/pending source allocations. Inline16
// buffers have no physical payload, and released Runtime blocks clear tokens.
func (m *NaturalManager) overflowPayloads() uint32 {
	if m == nil || m.self != m || m.closed {
		return 0
	}
	var count uint32
	for i := range m.division {
		if m.division[i].raw != (memory.Block{}) {
			count++
		}
		if m.division[i].pending != (memory.Block{}) {
			count++
		}
	}
	for i := range m.formatting.locals {
		if m.formatting.locals[i].raw != (memory.Block{}) {
			count++
		}
		if m.formatting.locals[i].pending != (memory.Block{}) {
			count++
		}
	}
	if m.verificationActive {
		if m.verification.raw != (memory.Block{}) {
			count++
		}
		if m.verification.pending != (memory.Block{}) {
			count++
		}
	}
	if m.charFormatting.buffer.raw != (memory.Block{}) {
		count++
	}
	if m.charFormatting.buffer.pending != (memory.Block{}) {
		count++
	}
	return count
}
