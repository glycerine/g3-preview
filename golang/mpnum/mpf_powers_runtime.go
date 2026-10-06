// Source: src/util/mpf.h:241-290; src/util/mpf.cpp:1821-1847.
// Source author: Christoph Wintersteiger.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Cold acquisition metadata. The sole numeral payload is the 16-byte Runtime
// wrapper, with twelve meaningful codec bytes and four unread padding bytes.
// Preparing this actual Go receipt header is an explicit adapter reservation.
type arbitraryFloatPowerWrapper struct {
	self                                                             *arbitraryFloatPowerWrapper
	manager                                                          *ArbitraryFloatManager
	next                                                             *arbitraryFloatPowerWrapper
	block, original                                                  memory.Block
	roots                                                            polynomialNumericRoots
	transferred                                                      memory.SmallAllocatorPendingAllocation
	originalCell                                                     sourceCellBacking
	cellRetired                                                      bool
	id                                                               uint32
	invocation, header                                               uint64
	executing, published, initialized, sourceDelDone, sourceFreeDone bool
	bodyCause, sourceCause, ownCause                                 error
}

func (w *arbitraryFloatPowerWrapper) valid() bool {
	return w != nil && w.self == w && w.manager != nil && w.manager.self == w.manager && !w.manager.closed && w.id != 0
}
func (w *arbitraryFloatPowerWrapper) capture() (Int, error) {
	if !w.valid() || w.block == (memory.Block{}) {
		return Int{}, memory.ErrRuntimeOwner
	}
	r := w.manager.memory
	val, e := r.ReadUint32(w.block, 0)
	if e != nil {
		return Int{}, e
	}
	lo, e := r.ReadUint32(w.block, 4)
	if e != nil {
		return Int{}, e
	}
	hi, e := r.ReadUint32(w.block, 8)
	if e != nil {
		return Int{}, e
	}
	return Int{val: int32(val), cell: numericCellRef(uint64(lo) | uint64(hi)<<32)}, nil
}
func (w *arbitraryFloatPowerWrapper) receiver() *integerRuntimeValueReceiver {
	return &integerRuntimeValueReceiver{runtime: w.manager.memory, block: w.block}
}
func (m *ArbitraryFloatManager) powerWrapperByID(id uint32) *arbitraryFloatPowerWrapper {
	for w := m.powerWrappers; w != nil; w = w.next {
		if w.id == id && w.valid() {
			return w
		}
	}
	return nil
}
func (m *ArbitraryFloatManager) powerTwoWrapper(n uint32, negated bool) (*arbitraryFloatPowerWrapper, error) {
	return m.powerWrapper(n, false, negated)
}
func (m *ArbitraryFloatManager) powerTwoMinusOneWrapper(n uint32, negated bool) (*arbitraryFloatPowerWrapper, error) {
	return m.powerWrapper(n, true, negated)
}
func (m *ArbitraryFloatManager) powerWrapper(n uint32, minusOne, negated bool) (*arbitraryFloatPowerWrapper, error) {
	if m == nil || m.self != m || m.integer == nil || m.integer.self != m.integer {
		return nil, memory.ErrRuntimeOwner
	}
	if !m.constructorComplete || m.closed || m.sourceEnded {
		return nil, arena.ErrClosed
	}
	if m.valueExecuting || m.integer.mpfBorrowedSourceCallbackActive() {
		return nil, ErrPolynomialPending
	}
	index := 0
	if minusOne {
		index = 2
	}
	if negated {
		index++
	}
	if id, found := m.powers[index].Find(n); found {
		w := m.powerWrapperByID(id)
		if w == nil {
			return nil, memory.ErrRuntimeOwner
		}
		return w, nil
	}
	if m.powerNextID == ^uint32(0) {
		return nil, arena.ErrBounds
	}
	header := uint64(reflect.TypeOf(arbitraryFloatPowerWrapper{}).Size())
	if e := m.budget.Reserve(header); e != nil {
		return nil, e
	}
	m.powerNextID++
	w := &arbitraryFloatPowerWrapper{manager: m, id: m.powerNextID, header: header, next: m.powerWrappers, invocation: 1}
	w.self = w
	m.powerWrappers = w
	m.powerWrapperBytes += header
	w.executing = true
	m.valueExecuting = true
	defer func() { w.executing = false; m.valueExecuting = false }()
	fail := func(e error) (*arbitraryFloatPowerWrapper, error) {
		if w.bodyCause == nil {
			w.bodyCause = e
		}
		return nil, e
	}
	block, e := m.memory.Allocate(16)
	w.block = block
	w.original = block
	if e != nil {
		return fail(e)
	}
	for _, offset := range []uint64{0, 4, 8} {
		if e = m.memory.WriteUint32(block, offset, 0); e != nil {
			return fail(e)
		}
	}
	w.initialized = true
	if e = m.powers[index].Insert(n, w.id); e != nil {
		return fail(e)
	}
	w.published = true
	out, e := w.capture()
	if e != nil {
		return fail(e)
	}
	two := NewInt(2)
	context := integerAllocationInvocationContext{mpfPower: w, mpfPowerInvocation: w.invocation}
	if e = m.integer.powerWithPhysicalAllocationContext(&two, n, &out, nil, context, w.receiver()); e != nil {
		return fail(e)
	}
	if minusOne {
		minus := NewInt(-1)
		e = m.integer.addSubWithAllocationContext(&out, &minus, &out, false, context)
		// This actual positive-power decrement has no callback after output
		// publication: BIG output swaps a nonnull old cell into scratch; small
		// output performs no allocation. Scratch growth occurs before the swap.
		if out.IsSmall() {
			if write := w.receiver().publishCell(&out); write != nil {
				return fail(write)
			}
			if write := w.receiver().publishValue(&out); write != nil {
				return fail(write)
			}
		} else {
			if write := w.receiver().publishValue(&out); write != nil {
				return fail(write)
			}
			if write := w.receiver().publishCell(&out); write != nil {
				return fail(write)
			}
		}
		if e != nil {
			return fail(e)
		}
	}
	if negated {
		if e = m.integer.Neg(&out); e != nil {
			return fail(e)
		}
		if e = w.receiver().publishValue(&out); e != nil {
			return fail(e)
		}
	}
	return w, nil
}
func (c integerAllocationInvocationContext) mpfPowerRoots(m *IntegerManager) *polynomialNumericRoots {
	w := c.mpfPower
	if !w.valid() || w.manager.powerWrapperByID(w.id) != w || w.manager.integer != m || !w.executing || !w.manager.valueExecuting || c.mpfPowerInvocation == 0 || c.mpfPowerInvocation != w.invocation || w.sourceDelDone || w.sourceFreeDone {
		return nil
	}
	return &w.roots
}
func (w *arbitraryFloatPowerWrapper) destroy(own bool) error {
	if !w.valid() {
		return memory.ErrRuntimeOwner
	}
	if !own && w.sourceCause != nil {
		return w.sourceCause
	}
	stop := func(e error) error {
		if own {
			if w.ownCause == nil {
				w.ownCause = e
			}
		} else if w.sourceCause == nil {
			w.sourceCause = e
		}
		return e
	}
	if !w.sourceDelDone && w.initialized && w.block != (memory.Block{}) {
		a, e := w.capture()
		if e != nil {
			return stop(e)
		}
		if own {
			e = w.manager.integer.cleanupRawNumeral(&a)
		} else {
			e = w.manager.integer.Del(&a)
		}
		if write := w.receiver().publishCell(&a); write != nil {
			return stop(write)
		}
		if e != nil {
			return stop(e)
		}
		w.sourceDelDone = true
	}
	if own {
		if w.roots.cell.object != (memory.ObjectRef{}) {
			e := w.manager.integer.cleanupDetachedCopyCell(&w.roots.cell)
			if w.roots.cell.object == (memory.ObjectRef{}) {
				w.cellRetired = true
			}
			if e != nil {
				return stop(e)
			}
		}
		if w.transferred.Block() != (memory.Block{}) {
			if e := w.manager.integer.allocator.CleanupTransferredAllocation(&w.transferred); e != nil {
				return stop(e)
			}
		}
	}
	// Source memory_manager.h:112 skips nullptr. A failed wrapper new has
	// no Runtime payload to release; its cold metadata remains until owner end.
	if !w.sourceFreeDone && w.block == (memory.Block{}) {
		w.sourceFreeDone = true
	}
	if !w.sourceFreeDone {
		e := w.manager.memory.Deallocate(&w.block)
		if w.block == (memory.Block{}) {
			w.sourceFreeDone = true
		}
		if e != nil {
			return stop(e)
		}
	}
	return nil
}
func (m *ArbitraryFloatManager) destroyPowerWrappers(own bool) error {
	if own {
		for w := m.powerWrappers; w != nil; w = w.next {
			if e := w.destroy(true); e != nil {
				return e
			}
		}
		return nil
	}
	for i := range m.powers {
		for it, end := m.powers[i].Begin(), m.powers[i].End(); !it.Equal(end); it.Next() {
			pair, e := it.Entry()
			if e != nil {
				return e
			}
			w := m.powerWrapperByID(pair.Value)
			if w == nil {
				return memory.ErrRuntimeOwner
			}
			if e = w.destroy(false); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *ArbitraryFloatManager) releasePowerWrapperHistory() {
	for w := m.powerWrappers; w != nil; w = w.next {
		m.budget.Release(w.header)
		w.header = 0
	}
	m.powerWrapperBytes = 0
}

// Called only at the actual allocating leaf return. It transfers the exact
// returned outcome; no before/after allocator census participates in authority.
func (m *IntegerManager) capturePowerAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, outcome *memory.SmallAllocatorAllocationOutcome) error {
	w := c.mpfPower
	if c.mpfPowerRoots(m) == nil {
		return ErrUPolynomialOwner
	}
	if outcome != nil && outcome.Pending != (memory.Block{}) {
		if e := m.allocator.TransferAllocationPending(outcome, &w.transferred); e != nil {
			return e
		}
	}
	if backing.object != (memory.ObjectRef{}) {
		w.originalCell = backing
		m.pendingCell = sourceCellBacking{}
	}
	return nil
}
