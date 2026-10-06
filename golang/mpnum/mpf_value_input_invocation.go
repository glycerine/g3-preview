// Source: src/util/mpf.cpp:69-106,114-181; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// The borrowed receipt names an actual allocation-call record. It contains no
// copied owning numeral. Inline input statements never materialize a record.
type ArbitraryFloatValueInputReceipt struct {
	manager    *ArbitraryFloatManager
	frame      *arbitraryFloatValueInputCall
	invocation uint64
}
type arbitraryFloatValueInputCall struct {
	self                *arbitraryFloatValueInputCall
	manager             *ArbitraryFloatManager
	next                *arbitraryFloatValueInputCall
	issued              ArbitraryFloatValueInputReceipt
	header              uint64
	numeric             polynomialNumericRoots
	transferred         memory.SmallAllocatorPendingAllocation
	originalCell        sourceCellBacking
	originalPending     memory.Block
	cellRetired         bool
	executing, retired  bool
	bodyCause, ownCause error
}
type arbitraryFloatInputBinding struct {
	manager    *ArbitraryFloatManager
	invocation uint64
	active     bool
	record     *arbitraryFloatValueInputCall
	receipt    *ArbitraryFloatValueInputReceipt
}

func (m *ArbitraryFloatManager) beginValueInput(receipt *ArbitraryFloatValueInputReceipt) (integerAllocationInvocationContext, error) {
	if m.inputBinding.active || m.inputInvocation == ^uint64(0) {
		return integerAllocationInvocationContext{}, ErrPolynomialPending
	}
	m.inputInvocation++
	m.inputBinding = arbitraryFloatInputBinding{manager: m, invocation: m.inputInvocation, active: true, receipt: receipt}
	return integerAllocationInvocationContext{mpfInput: &m.inputBinding, mpfInputInvocation: m.inputInvocation}, nil
}
func (c integerAllocationInvocationContext) inputBinding(z *IntegerManager) *arbitraryFloatInputBinding {
	a := c.mpfInput
	if z == nil || z.self != z || a == nil || a.manager == nil || a.manager.self != a.manager {
		return nil
	}
	m := a.manager
	if m.integer != z || a != &m.inputBinding || c.mpfInputInvocation == 0 || c.mpfInputInvocation != a.invocation || a.invocation != m.inputInvocation || !a.active || !m.valueExecuting || m.closed || m.sourceEnded || z.memory.FailureTransportActive() {
		return nil
	}
	return a
}
func (c integerAllocationInvocationContext) prepareInputAllocation(z *IntegerManager) error {
	if c.mpfInput == nil {
		return nil
	}
	a := c.inputBinding(z)
	if a == nil {
		return memory.ErrRuntimeOwner
	}
	if a.record != nil {
		return nil
	}
	m := a.manager
	bytes := uint64(reflect.TypeOf(arbitraryFloatValueInputCall{}).Size())
	if e := m.budget.Reserve(bytes); e != nil {
		return e
	}
	q := &arbitraryFloatValueInputCall{manager: m, next: m.inputCalls, header: bytes, executing: true}
	q.self = q
	q.issued = ArbitraryFloatValueInputReceipt{m, q, a.invocation}
	m.inputCalls = q
	m.inputHistoryBytes += bytes
	a.record = q
	if a.receipt != nil {
		*a.receipt = q.issued
	}
	return nil
}
func (c integerAllocationInvocationContext) inputRoots(z *IntegerManager) *polynomialNumericRoots {
	a := c.inputBinding(z)
	if a == nil || a.record == nil {
		return nil
	}
	q := a.record
	if q.self != q || q.manager != a.manager || q.issued.frame != q || q.issued.invocation != a.invocation || !q.executing {
		return nil
	}
	return &q.numeric
}
func (z *IntegerManager) captureInputAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, out *memory.SmallAllocatorAllocationOutcome) error {
	a := c.inputBinding(z)
	if a == nil || a.record == nil {
		return nil
	}
	q := a.record
	if backing.object != (memory.ObjectRef{}) {
		q.originalCell = backing
		if z.pendingCell == backing {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if out.Pending != (memory.Block{}) {
		q.originalPending = out.Pending
		if e := z.allocator.TransferAllocationPending(out, &q.transferred); e != nil {
			return e
		}
		if q.numeric.allocator[0] == out.Pending {
			q.numeric.allocator[0] = memory.Block{}
		}
	}
	return nil
}
func (m *ArbitraryFloatManager) finishValueInput(cause error) {
	a := &m.inputBinding
	if q := a.record; q != nil {
		q.executing = false
		q.bodyCause = cause
		q.retired = !polynomialNegNumericPending(&q.numeric) && q.transferred.Block() == (memory.Block{})
	}
	a.active = false
}
func (m *ArbitraryFloatManager) valueInputCall(r *ArbitraryFloatValueInputReceipt) (*arbitraryFloatValueInputCall, error) {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame == nil || r.invocation == 0 {
		return nil, memory.ErrRuntimeOwner
	}
	for q := m.inputCalls; q != nil; q = q.next {
		if q == r.frame {
			if q.self != q || q.manager != m || q.issued != *r {
				return nil, memory.ErrRuntimeOwner
			}
			return q, nil
		}
	}
	return nil, memory.ErrRuntimeOwner
}

// false reads only completion metadata; true releases the exact unjoined leaf
// after the caller owns its older passed MPF NUM. No output numeral is owned here.
func (m *ArbitraryFloatManager) CleanupValueInput(r *ArbitraryFloatValueInputReceipt, abort bool) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if m.valueExecuting || m.inputBinding.active || m.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	if r == nil || *r == (ArbitraryFloatValueInputReceipt{}) {
		return nil
	}
	q, e := m.valueInputCall(r)
	if e != nil {
		return e
	}
	if q.executing {
		return ErrPolynomialPending
	}
	if q.retired || !abort {
		return nil
	}
	q.executing = true
	defer func() { q.executing = false }()
	e = m.integer.cleanupFormatAllocation(&q.numeric, &q.transferred, &q.cellRetired)
	if e != nil {
		if q.ownCause == nil {
			q.ownCause = e
		}
		return e
	}
	q.retired = true
	return nil
}
func (m *ArbitraryFloatManager) cleanupValueInputs() error {
	for q := m.inputCalls; q != nil; q = q.next {
		if e := m.CleanupValueInput(&q.issued, true); e != nil {
			return e
		}
	}
	return nil
}
func (m *ArbitraryFloatManager) releaseValueInputHistory() {
	for q := m.inputCalls; q != nil; q = q.next {
		m.budget.Release(q.header)
		q.header = 0
	}
	m.inputHistoryBytes = 0
	m.inputCalls = nil
}
