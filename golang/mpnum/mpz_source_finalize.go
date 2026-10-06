// Licensed under the MIT license.
// Source: src/util/mpz.cpp:142–161 and reverse member destruction in src/util/mpz.h and src/util/mpn.h.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// SourceFinalize runs only the source destructor, preserving caller-owned
// direct allocations. CloseChecked is the separate owning Go cleanup adapter.
// The charged scalar cursor records completed actions across cleanup refusal;
// it does not claim C++ recovery from an interrupted destructor.
func (m *IntegerManager) SourceFinalize() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.rawInvocationExecuting() || m.copyAllocation.executing || m.normalizationExecuting || m.factorNumeralExecuting || m.factorNumeralVectorCallerExecuting() || m.factorUnivariateCallerExecuting() || m.factorCoreCallerExecuting() || m.factorFrontendCallerExecuting() || m.upolyDisplayCallerExecuting() {
		return ErrPolynomialPending
	}
	if m.closed || m.sourceFinalized {
		return nil
	}
	if m.smt2Display.active {
		return ErrSMT2DisplayPending
	}
	// Constructor failure is not a completed C++ object destructor invocation.
	if m.originalCause != nil {
		return arena.ErrClosed
	}
	m.sourceFinalizing = true
	fail := func(e error) error { m.cleanupCause = e; return e }
	for m.sourceFinalizeStage < 8 {
		switch m.sourceFinalizeStage {
		case 0, 1:
			value := &m.two64
			if m.sourceFinalizeStage == 1 {
				value = &m.intMin
			}
			if value.cell == m.failedNullRef {
				value.cell = 0
			}
			if value.cell != 0 {
				if !validNumericRef(m.cells, value.cell) {
					m.sourceGuard = true
					return fail(arena.ErrInvalidSlot)
				}
				if e := m.free(value.cell); e != nil {
					return fail(e)
				}
				value.cell = 0 // del preserves m_val.
			}
		case 2, 3, 4, 5:
			index := int(m.sourceFinalizeStage-2) / 2
			ref := &m.tmp[index]
			if m.sourceFinalizeStage%2 == 1 {
				ref = &m.arg[index]
			}
			// Unlike del(numeral), source deallocate(ptr) reads the capacity header.
			// A missing scratch pointer is guarded at this actual destructor access.
			if *ref == 0 || !validNumericRef(m.cells, *ref) {
				m.sourceGuard = true
				return fail(arena.ErrInvalidSlot)
			}
			if e := m.free(*ref); e != nil {
				return fail(e)
			}
			*ref = 0
		case 6:
			if m.natural != nil {
				if e := m.natural.SourceFinalize(); e != nil {
					return fail(e)
				}
			}
		case 7:
			e := m.allocator.Close()
			// Chunk frees may complete before a later TLS sync/diagnostic refusal.
			// Mark only physically consumed pooled roots, preserving surviving chunks.
			if mark := m.markSourceFreedPools(); mark != nil {
				return fail(mark)
			}
			if e != nil {
				return fail(e)
			}
		}
		m.sourceFinalizeStage++
	}
	m.sourceFinalized = true
	m.cleanupCause = nil
	return nil
}

func (m *IntegerManager) markSourceFreedPools() error {
	if parent := m.rationalServiceParent; parent != nil {
		if parent.self != parent {
			return ErrPolynomialOwner
		}
		for c := parent.lexicalInvocations; c != nil; c = c.next {
			if !c.valid() {
				return ErrPolynomialOwner
			}
			if c.integer != m {
				continue
			}
			if e := m.markSourceConsumedDetachedCell(&c.numeric.cell, &c.cellRetired); e != nil {
				return e
			}
		}
	}

	if parent := m.mpfParent; parent != nil && parent.self == parent && parent.integer == m {
		for _, s := range []*arbitraryFloatRoundScope{&parent.roundScope, &parent.roundSqrtScope} {
			if s.manager == parent && s.invocation != 0 {
				if e := m.markSourceConsumedDetachedCell(&s.numeric.cell, &s.cellRetired); e != nil {
					return e
				}
			}
		}
		for h := parent.rationalSetHistories; h != nil; h = h.next {
			if h.self != h || h.manager != parent || h.invocation == 0 {
				return memory.ErrRuntimeOwner
			}
			if e := m.markSourceConsumedDetachedCell(&h.numeric.cell, &h.cellRetired); e != nil {
				return e
			}
		}
		for h := parent.formattedSetHistories; h != nil; h = h.next {
			if h.self != h || h.manager != parent || h.issued.manager != parent || h.issued.frame != h || h.issued.invocation != h.invocation || h.invocation == 0 {
				return memory.ErrRuntimeOwner
			}
			if e := m.markSourceConsumedDetachedCell(&h.numeric.cell, &h.cellRetired); e != nil {
				return e
			}
		}
		for q := parent.inputCalls; q != nil; q = q.next {
			if q.self != q || q.manager != parent || q.issued.manager != parent || q.issued.frame != q || q.issued.invocation == 0 {
				return memory.ErrRuntimeOwner
			}
			if e := m.markSourceConsumedDetachedCell(&q.numeric.cell, &q.cellRetired); e != nil {
				return e
			}
		}
	}

	for s := m.conversionDetachedOutputs; s != nil; s = s.outputDetachedNext {
		if s.manager == nil || !s.manager.validConversionScope(s) || s.outputManager.z != m || s.outputDetachedInvocation == 0 || s.outputDetachedInvocation != s.invocation {
			return memory.ErrRuntimeOwner
		}
		if e := m.markSourceConsumedDetachedCell(&s.outputAllocation.numeric.cell, &s.outputAllocation.cellRetired); e != nil {
			return e
		}
	}
	if parent := m.mpfParent; parent != nil {
		s := &parent.rationalConversion
		if parent.validConversionScope(s) {
			if e := m.markSourceConsumedDetachedCell(&s.numeric.cell, &s.cellRetired); e != nil {
				return e
			}
		}
	}

	if parent := m.mpfParent; parent != nil {
		for s := parent.formatScopes; s != nil; s = s.next {
			if !parent.matchesFormatFrame(s) {
				return ErrPolynomialOwner
			}
			if e := m.markSourceConsumedDetachedCell(&s.numeric.cell, &s.cellRetired); e != nil {
				return e
			}
			if e := m.markSourceConsumedDetachedCell(&s.decimal.numeric.cell, &s.decimal.cellRetired); e != nil {
				return e
			}
		}
	}

	if parent := m.mpfParent; parent != nil {
		for q := parent.valueCalls; q != nil; q = q.next {
			if q.retired || !m.matchesCopyAllocationInvocation(&q.issued.child) {
				continue
			}
			child := &q.childFrame
			if e := m.markSourceConsumedDetachedCell(&child.roots.cell, &child.cellRetired); e != nil {
				return e
			}
		}
	}
	if parent := m.mpfParent; parent != nil {
		for w := parent.powerWrappers; w != nil; w = w.next {
			if !w.valid() || w.manager.integer != m {
				return ErrPolynomialOwner
			}
			if e := m.markSourceConsumedDetachedCell(&w.roots.cell, &w.cellRetired); e != nil {
				return e
			}
		}
	}
	for n := uint64(1); n <= uint64(m.cells.Stats().Slots); n++ {
		slot := arena.Slot(n)
		c, e := m.cells.Read(slot)
		if e != nil || c.released || c.backing.offset == 0 {
			continue
		}
		_, e = m.memory.ReadUint32(c.backing.raw, c.backing.offset+4)
		if errors.Is(e, memory.ErrRuntimeOwner) {
			c.released = true
			if e := m.cells.Write(slot, c); e != nil {
				return e
			}
		} else if e != nil {
			return e
		}
	}
	return nil
}

// After source allocator destruction its locator directory no longer exists.
// Each charged descriptor retains the real raw allocation token; owning cleanup
// frees direct blocks through Runtime, without an invented allocator event.
func (m *IntegerManager) freeAfterSourceFinalize(ref numericCellRef) error {
	if m.isFailureRef(ref) {
		return nil
	} // actual owning metadata, never a raw source free

	c := m.cell(ref)
	if !c.released {
		if c.backing.offset != 0 {
			return arena.ErrInvalidSlot
		}
		raw := c.backing.raw
		e := m.memory.Deallocate(&raw)
		if raw == (memory.Block{}) {
			c.released = true
			m.store(ref, c)
		}
		if e != nil {
			c.cleanupPending = true
			m.store(ref, c)
			return e
		}
	}
	return m.cells.Recycle(numericSlot(ref))
}

// markSourceConsumedDetachedCell checks only a previously acquired exact cell
// token at the reached allocator destructor frontier. Direct blocks survive.
func (m *IntegerManager) markSourceConsumedDetachedCell(c *sourceCellBacking, retired *bool) error {
	if c.object == (memory.ObjectRef{}) || c.offset == 0 {
		return nil
	}
	_, e := m.memory.ReadUint32(c.raw, c.offset+4)
	if errors.Is(e, memory.ErrRuntimeOwner) {
		*c = sourceCellBacking{}
		*retired = true
		return nil
	}
	return e
}
