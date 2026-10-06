// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577; src/util/mpz.h:124-149.
// Source: src/util/mpz.h:698-707.
// Licensed under the MIT license.
// Translated from src/util/mpz.h and src/util/mpz.cpp (_MP_INTERNAL, 32-bit digits).
// Original author: Leonardo de Moura, 2010-06-17.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"runtime"
)

var ErrConversion = errors.New("mpnum: integer conversion out of range")
var ErrIntegerAssertion = errors.New("mpnum: source integer assertion failed")

// Int is a pointer-free, owner-local mutable integer. With a zero cell ref,
// val is an inline int32. Otherwise val is the sign (-1/+1). A big Int must
// NEVER be shallow-copied: use Copy, Move, or Swap through its owning manager.
// The implicit manager owner is a caller contract; refs do not identify owners.
type Int struct {
	val  int32
	cell numericCellRef
}

func NewInt(v int32) Int     { return Int{val: v} }
func (a *Int) IsSmall() bool { return a.cell == 0 }

type numericCell struct {
	payload                               arena.VectorRef // zero compatibility field; numerical bytes live only in backing
	backing                               sourceCellBacking
	size, capacity, span                  uint32
	released, unpublished, cleanupPending bool
}
type IntegerConfig struct {
	Cells   arena.Config
	Words   arena.VectorConfig
	Memory  *memory.Runtime
	Profile memory.SmallAllocatorProfile
}

// IntegerManager owns all persistent source cells, two reusable result
// cells, two one-limb argument cells, and the source intMin/two64 constants.
// Single-owner confinement replaces the source synchronized OpenMP variant.
// Cells, transitional Natural Words and the actual memory backend share one Budget.
// Close invalidates every live numeral; a supplied Runtime is borrowed.
type IntegerManager struct {
	algebraicNegCaller                     *AlgebraicRootCore
	dividesNumeral                         integerDividesNumeralScope
	divisionRaw, extendedRaw, gcdRaw       integerRawScope
	addMulInvocation                       integerAddMulScope
	copyAllocation                         integerCopyAllocationScope
	normalizationExecuting                 bool
	factorNumeralExecuting                 bool
	factorCoreCaller                       *polynomialFactorCoreFrame
	factorUnivariateCaller                 *MultivariatePolynomialManager
	factorNumeralVectorCaller              *MultivariatePolynomialManager
	factorFrontendCaller                   *upolyFactorFrontendScope
	upolyDisplayCaller                     *upolyDisplayScope
	upolyDisplaySequenceCaller             *UPolynomialSequence
	factorFrontendConstantCaller           *UPolynomialFactorizationService
	self                                   *IntegerManager
	memory                                 *memory.Runtime
	allocator                              *memory.SmallAllocator
	ownsMemory                             bool
	pendingCell                            sourceCellBacking
	originalCause, cleanupCause            error
	cells                                  *numericCellSlab
	words                                  *arena.Words
	tmp, arg                               [2]numericCellRef
	failedRef, failedNullRef               numericCellRef  // permanently dead, charged auxiliary descriptor
	natural                                *NaturalManager // child borrows this manager's Words and Budget
	intMin, two64                          Int
	powerRaw                               integerPowerScope
	powerGeneration                        uint64
	rootRaw                                integerRootScope
	rootGeneration                         uint64
	perfectSquare                          integerPerfectSquareScope
	perfectSquareGeneration                uint64
	rawDisplay                             rawDisplayScope
	smt2Display                            smt2DisplayScope
	closed                                 bool
	sourceFinalizing, sourceFinalized      bool
	sourceGuard                            bool
	sourceFinalizeStage                    uint8
	constructionFailed, constructorUnwound bool
	constructorUnwindStage                 uint8
	budget                                 *arena.Budget
	headerBytes                            uint64
}

func NewIntegerManager(config IntegerConfig) (*IntegerManager, error) {
	if config.Cells.Budget != config.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	if config.Memory != nil {
		if e := config.Memory.CheckOpen(); e != nil {
			return nil, e
		}
		if config.Memory.Budget() != config.Cells.Budget {
			return nil, arena.ErrConfig
		}
	}
	header := uint64(reflect.TypeOf(IntegerManager{}).Size())
	if e := config.Cells.Budget.Reserve(header); e != nil {
		return nil, e
	}
	m := &IntegerManager{budget: config.Cells.Budget, headerBytes: header, memory: config.Memory}
	m.self = m
	fail := func(err error) (*IntegerManager, error) {
		m.originalCause = err
		m.constructionFailed = true
		if cleanup := m.unwindConstructorMembers(); cleanup != nil {
			m.cleanupCause = cleanup
			return nil, m
		}
		// Source member unwind never executes the completed manager's body.
		// Direct physical cells remain an explicit owning cleanup obligation.
		if m.hasConstructorPhysicalRoots() {
			return nil, m
		}
		if cleanup := m.CloseChecked(); cleanup != nil {
			return nil, m
		}
		return nil, err
	}
	var e error
	if m.memory == nil {
		m.memory, e = memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget, SizeTBits: uint8(32 << (^uint(0) >> 63))})
		if e != nil {
			return fail(e)
		}
		m.ownsMemory = true
	}
	profile := config.Profile
	if profile.PointerBits == 0 {
		profile = memory.SmallAllocatorProfile{PointerBits: m.memory.SizeTBits(), Debug: modularSourceDebug, Windows: runtime.GOOS == "windows"}
	}
	m.allocator, e = memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: m.memory, Profile: profile})
	if e != nil {
		return fail(e)
	}
	m.cells, e = newNumericCellSlab(config.Cells)
	if e != nil {
		return fail(e)
	}
	slot, gen, e := m.cells.reserveDead()
	if e != nil {
		return fail(e)
	}
	m.failedRef = makeNumericRef(slot, gen)
	// Both source pointer failure states use real permanently dead Go metadata
	// descriptors. They never contribute source cells or allocator counters.
	slot, gen, e = m.cells.reserveDead()
	if e != nil {
		m.originalCause = e
		m.constructionFailed = true
		m.cleanupCause = m.unwindConstructorMembers()
		return nil, m // the already charged owner retains the actual cold roots
	}
	m.failedNullRef = makeNumericRef(slot, gen)
	// Actual Natural Words scratch is retained only during the staged backend
	// migration. It does not hold numeric cells or duplicate physical digits.
	m.words, e = arena.NewWords(config.Words)
	if e != nil {
		return fail(e)
	}
	m.natural, e = newNaturalManagerWithMemory(config.Words, m.words, false, m.memory)
	if e != nil {
		return fail(e)
	}
	for i := 0; i < 2; i++ {
		m.tmp[i], e = m.allocate(6, 0)
		if e != nil {
			return fail(e)
		}
		m.publishCell(m.tmp[i])
		m.arg[i], e = m.allocate(6, 0)
		if e != nil {
			return fail(e)
		}
		m.publishCell(m.arg[i])
		m.resizeCell(m.arg[i], 1)
	}
	if e = m.SetInt64(&m.intMin, -int64(math.MinInt32)); e != nil {
		return fail(e)
	}
	if e = m.SetUint64(&m.two64, math.MaxUint64); e != nil {
		return fail(e)
	}
	one := NewInt(1)
	if e = m.Add(&m.two64, &one, &m.two64); e != nil {
		return fail(e)
	}
	return m, nil
}

func (m *IntegerManager) RuntimeOwner() *memory.Runtime {
	if m == nil || m.self != m {
		return nil
	}
	return m.memory
}
func (m *IntegerManager) cellObject(ref numericCellRef) (memory.ObjectRef, error) {
	if m == nil || m.self != m {
		return memory.ObjectRef{}, memory.ErrRuntimeOwner
	}
	if !validNumericRef(m.cells, ref) {
		return memory.ObjectRef{}, arena.ErrInvalidSlot
	}
	c := m.cell(ref)
	if c.released {
		return memory.ObjectRef{}, arena.ErrInvalidSlot
	}
	return c.backing.object, nil
}
func (m *IntegerManager) Error() string {
	if m.originalCause != nil {
		return m.originalCause.Error()
	}
	if m.cleanupCause != nil {
		return m.cleanupCause.Error()
	}
	return "mpnum: retained integer owner"
}
func (m *IntegerManager) Unwrap() error {
	if m.originalCause != nil {
		return m.originalCause
	}
	return m.cleanupCause
}
func (m *IntegerManager) OriginalCause() error { return m.originalCause }
func (m *IntegerManager) CleanupCause() error  { return m.cleanupCause }
func (m *IntegerManager) Cleanup() error       { return m.CloseChecked() }

// checkCleanup accepts the owner's permanently dead allocation token only for
// release/reset after a source allocation prefix. Ordinary reads use check.
func (m *IntegerManager) checkCleanup(a *Int) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceFinalizing || m.constructionFailed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if a.cell == 0 || (m.failedRef != 0 && (a.cell == m.failedRef || a.cell == m.failedNullRef)) {
		return nil
	}
	// A retained descriptor whose physical release completed remains cleanup
	// authority only; ordinary reads reject it until the descriptor is consumed.
	if !validNumericRef(m.cells, a.cell) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *IntegerManager) check(a *Int) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceFinalizing || m.constructionFailed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if a.cell != 0 && (!validNumericRef(m.cells, a.cell) || m.cell(a.cell).released) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *IntegerManager) cell(ref numericCellRef) numericCell {
	c, e := m.cells.Read(numericSlot(ref))
	if e != nil {
		panic(e)
	}
	return c
}
func (m *IntegerManager) store(ref numericCellRef, c numericCell) {
	if e := m.cells.Write(numericSlot(ref), c); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) digit(ref numericCellRef, i uint32) uint32 {
	c := m.cell(ref)
	if i >= max(c.size, c.span) {
		return 0
	}
	v, e := c.backing.readWord(m.allocator, 8+4*uint64(i))
	if e != nil {
		panic(e)
	}
	return v
}
func (m *IntegerManager) writeDigit(ref numericCellRef, i, v uint32) {
	c := m.cell(ref)
	if e := c.backing.writeWord(m.allocator, 8+4*uint64(i), v); e != nil {
		panic(e)
	}
}

// A scratch access span is bookkeeping, never a source m_size publication.
func (m *IntegerManager) prepareTmpSpan(idx int, size uint32) error {
	if idx < 0 || idx >= len(m.tmp) || m.tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	c := m.cell(m.tmp[idx])
	if size > c.capacity || uint64(size)*4+8 > c.backing.extent {
		return arena.ErrBounds
	}
	c.span = size
	m.store(m.tmp[idx], c)
	return nil
}
func (m *IntegerManager) resizeCell(ref numericCellRef, size uint32) {
	c := m.cell(ref)
	if size > c.capacity {
		panic(arena.ErrBounds)
	}
	if e := c.backing.writeWord(m.allocator, 0, size); e != nil {
		panic(e)
	}
	c.size = size
	c.span = 0
	m.store(ref, c)
}
func (m *IntegerManager) allocate(capacity, size uint32) (numericCellRef, error) {
	return m.allocateWithCopyReceipt(capacity, size, nil)
}
func (m *IntegerManager) allocateWithCopyReceipt(capacity, size uint32, s *integerCopyAllocationScope) (numericCellRef, error) {
	return m.allocateWithInvocationReceipts(capacity, size, s, nil)
}
func (m *IntegerManager) allocateWithInvocationReceipts(capacity, size uint32, s *integerCopyAllocationScope, pn *integerPNormalizeScope) (numericCellRef, error) {
	return m.allocateWithSourceContext(capacity, size, s, pnAllocationContext(pn))
}
func (m *IntegerManager) allocateWithSourceContext(capacity, size uint32, s *integerCopyAllocationScope, context integerAllocationInvocationContext) (numericCellRef, error) {
	if modularSourceDebug && capacity < 6 {
		return 0, ErrIntegerAssertion
	}
	if m.pendingCell.object != (memory.ObjectRef{}) {
		return 0, memory.ErrAllocatorPending
	}
	if e := context.prepareFactorFrontendConstantHistory(m); e != nil {
		return 0, e
	}
	var outcome memory.SmallAllocatorAllocationOutcome
	var out *memory.SmallAllocatorAllocationOutcome
	var roots *polynomialNumericRoots
	if s != nil && s == &m.copyAllocation && s.active && s.executing {
		roots = &s.roots
	}
	if r := context.roots(m); r != nil {
		roots = r
	}
	if roots != nil {
		out = &outcome
	}
	if q := context.upolyDisplayFrame(m); q != nil {
		if e := q.prepareStorage(); e != nil {
			return 0, e
		}
	}
	if e := context.prepareFactorHistory(m); e != nil {
		return 0, e
	}
	backing, e := allocateSourceCellWithOutcome(m.allocator, capacity, out)
	// Capture the physical prefix before charged descriptor preparation.
	m.pendingCell = backing
	if out != nil {
		if backing.object != (memory.ObjectRef{}) {
			roots.cell = backing
		}
		if outcome.Pending != (memory.Block{}) {
			roots.allocator[0] = outcome.Pending
		}
	}
	if e != nil {
		return 0, e
	}
	slot, gen, e := m.cells.Allocate()
	if e != nil {
		return 0, e
	}
	ref := makeNumericRef(slot, gen)
	m.store(ref, numericCell{backing: backing, capacity: capacity, unpublished: true})
	m.pendingCell = sourceCellBacking{}
	if out != nil && roots.cell == backing {
		roots.cell = sourceCellBacking{}
	}
	return ref, nil
}

// Publication is scalar ownership metadata, not a source allocation or write.
func (m *IntegerManager) publishCell(ref numericCellRef) {
	c := m.cell(ref)
	c.unpublished = false
	m.store(ref, c)
}
func (m *IntegerManager) free(ref numericCellRef) error {
	c := m.cell(ref)
	if !c.released {
		progress, e := c.backing.deallocate(m.allocator)
		if progress.PhysicalReleased || progress.PooledLinked {
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
func (m *IntegerManager) Close() {
	if e := m.CloseChecked(); e != nil {
		panic(m)
	}
}
func (m *IntegerManager) CloseChecked() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.rawInvocationExecuting() || m.copyAllocation.executing || m.normalizationExecuting || m.factorNumeralExecuting || m.factorNumeralVectorCallerExecuting() || m.factorUnivariateCallerExecuting() || m.factorCoreCallerExecuting() || m.factorFrontendCallerExecuting() || m.upolyDisplayCallerExecuting() {
		return ErrPolynomialPending
	}
	if m.closed {
		m.cleanupCause = nil
		return nil
	}
	fail := func(e error) error { m.cleanupCause = e; return m }
	if m.constructionFailed && !m.constructorUnwound {
		if m.allocator != nil {
			if e := m.allocator.CleanupPending(); e != nil {
				return fail(e)
			}
		}
		if e := m.unwindConstructorMembers(); e != nil {
			return fail(e)
		}
	}
	if e := m.CleanupSMT2Display(); e != nil {
		return fail(e)
	}
	// Resume every explicitly begun source cursor before owning cleanup. A
	// missing/stale private pointer stops the source body at an undefined access;
	// owning cleanup then aborts that body without repairing it or advancing its
	// history, and releases only known actual roots.
	if m.sourceFinalizing && !m.sourceGuard && !m.sourceFinalized {
		if e := m.SourceFinalize(); e != nil {
			return fail(e)
		}
	}

	if e := m.CleanupRoot(); e != nil {
		return fail(e)
	}
	if e := m.CleanupPower(); e != nil {
		return fail(e)
	}
	if e := m.CleanupPerfectSquare(); e != nil {
		return fail(e)
	}
	if e := m.CleanupRawDisplay(); e != nil {
		return fail(e)
	}
	if e := m.CleanupDivision(); e != nil {
		return fail(e)
	}
	if e := m.CleanupExtendedGCD(); e != nil {
		return fail(e)
	}
	if e := m.CleanupGCDRaw(); e != nil {
		return fail(e)
	}
	if e := m.CleanupAddMulRaw(); e != nil {
		return fail(e)
	}
	if e := m.CleanupDividesNumeral(); e != nil {
		return fail(e)
	}
	if m.cells != nil {
		for n := uint64(1); n <= uint64(m.cells.Stats().Slots); n++ {
			slot := arena.Slot(n)
			if _, e := m.cells.Read(slot); e == nil {
				// Slab slots retain their actual generation. Only live records are read.
				gen := m.cells.state(slot).generation
				ref := makeNumericRef(slot, gen)
				if m.sourceFinalized || m.constructorUnwound {
					e = m.freeAfterSourceFinalize(ref)
				} else {
					e = m.free(ref)
				}
				if e != nil {
					return fail(e)
				}
			}
		}
	}
	if e := m.cleanupPendingAllocation(); e != nil {
		return fail(e)
	}
	if m.natural != nil {
		if e := m.natural.CloseChecked(); e != nil {
			return fail(e)
		}
		m.natural = nil
	}
	if m.words != nil {
		m.words.Close()
	}
	if m.allocator != nil {
		if e := m.allocator.CleanupPending(); e != nil {
			return fail(e)
		}
		if e := m.allocator.Close(); e != nil {
			return fail(e)
		}
	}
	if m.ownsMemory && m.memory != nil {
		if e := m.memory.Close(); e != nil {
			return fail(e)
		}
	}
	if m.cells != nil {
		m.cells.Close()
	}
	m.tmp = [2]numericCellRef{}
	m.arg = [2]numericCellRef{}
	m.intMin = Int{}
	m.two64 = Int{}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
	m.cleanupCause = nil
	return nil
}

// Del preserves the source's m_val after releasing a big cell. Thus deleting a
// big positive/negative value leaves inline +1/-1. Reset additionally writes 0.
func (m *IntegerManager) Del(a *Int) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if m.failedRef != 0 && (a.cell == m.failedRef || a.cell == m.failedNullRef) {
		a.cell = 0
		return nil
	}
	if a.cell != 0 {
		if e := m.free(a.cell); e != nil {
			return e
		}
		a.cell = 0
	}
	return nil
}
func (m *IntegerManager) Reset(a *Int) error {
	if e := m.Del(a); e != nil {
		return e
	}
	a.val = 0
	return nil
}

// allocateIfNeeded is the source destructive capacity helper. It never copies
// old limbs. A failed replacement leaves the already-published positive sign
// and the owner's dead allocation token; only cleanup may consume that token.
func (m *IntegerManager) allocateIfNeeded(a *Int, capacity uint32) error {
	return m.allocateIfNeededWithAllocationContext(a, capacity, integerAllocationInvocationContext{})
}
func (m *IntegerManager) allocateIfNeededWithAllocationContext(a *Int, capacity uint32, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if capacity < 6 {
		capacity = 6
	}
	aWasSmall := a.IsSmall()
	if aWasSmall || m.cell(a.cell).capacity < capacity {
		if a.cell != 0 {
			if e := m.free(a.cell); e != nil {
				return e
			}
		}
		a.val = 1
		a.cell = m.failedRef
		if aWasSmall {
			a.cell = m.failedNullRef
		}
		ref, e := m.allocateWithSourceContext(capacity, 0, nil, context)
		if e != nil {
			return e
		}
		a.cell = ref
		m.publishCell(ref)
	}
	return nil
}

func (m *IntegerManager) setBig64(a *Int, v uint64, sign int32) error {
	return m.setBig64WithAllocationContext(a, v, sign, integerAllocationInvocationContext{})
}
func (m *IntegerManager) setBig64WithAllocationContext(a *Int, v uint64, sign int32, context integerAllocationInvocationContext) error {
	ref := a.cell
	if ref == 0 {
		var e error
		ref, e = m.allocateWithSourceContext(6, 0, nil, context)
		if e != nil {
			return e
		}
	}
	// mpz.cpp set_big_i64/set_big_ui64 assert capacity after optional
	// allocation and before any sign or limb publication.
	if modularSourceDebug && m.cell(ref).capacity < 6 {
		return ErrIntegerAssertion
	}
	size := uint32(1)
	if v>>32 != 0 {
		size = 2
	}
	a.cell = ref
	m.publishCell(ref)
	a.val = sign
	m.writeDigit(ref, 0, uint32(v))
	m.writeDigit(ref, 1, uint32(v>>32))
	m.resizeCell(ref, size)
	return nil
}
func (m *IntegerManager) SetInt64(a *Int, v int64) error {
	return m.setInt64WithAllocationContext(a, v, integerAllocationInvocationContext{})
}
func (m *IntegerManager) setInt64WithAllocationContext(a *Int, v int64, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if v >= math.MinInt32 && v <= math.MaxInt32 {
		if e := m.Del(a); e != nil {
			return e
		}
		a.val = int32(v)
		return nil
	}
	sign := int32(1)
	magnitude := uint64(v)
	if v < 0 {
		sign = -1
		magnitude = 0 - uint64(v)
	}
	return m.setBig64WithAllocationContext(a, magnitude, sign, context)
}

// SetUint64 preserves the source's strict < INT_MAX inline threshold.
func (m *IntegerManager) SetUint64(a *Int, v uint64) error {
	return m.setUint64WithAllocationContext(a, v, integerAllocationInvocationContext{})
}
func (m *IntegerManager) setUint64WithAllocationContext(a *Int, v uint64, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if v < math.MaxInt32 {
		return m.setInt64WithAllocationContext(a, int64(v), context)
	}
	return m.setBig64WithAllocationContext(a, v, 1, context)
}

// SetLimbs is the source unsigned digit-vector setter, with trailing zeros
// removed. The caller's slice is borrowed and never retained.
func (m *IntegerManager) SetLimbs(a *Int, digits []uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	for len(digits) > 0 && digits[len(digits)-1] == 0 {
		digits = digits[:len(digits)-1]
	}
	if len(digits) == 0 {
		return m.Reset(a)
	}
	if len(digits) == 1 {
		return m.SetInt64(a, int64(digits[0]))
	}
	if uint64(len(digits)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	size := uint32(len(digits))
	a.val = 1 // source publication precedes the allocation branches
	ref := a.cell
	if ref == 0 || m.cell(ref).capacity < size {
		if ref != 0 {
			if e := m.free(ref); e != nil {
				return e
			}
		}
		a.cell = m.failedRef
		if ref == 0 {
			a.cell = m.failedNullRef
		}
		var e error
		ref, e = m.allocate(max(size, 6), size)
		if e != nil {
			return e
		}
	}
	a.cell = ref
	m.publishCell(ref)
	m.resizeCell(ref, size)
	for i, v := range digits {
		m.writeDigit(ref, uint32(i), v)
	}
	a.cell = ref
	a.val = 1
	return nil
}
func (m *IntegerManager) Copy(dst, src *Int) error { return m.CopyFromOwner(dst, m, src) }

// CopyFromOwner is the same literal SET engine with the actual borrowed source
// cell owner. It differs from the staged ergonomic CopyFrom adapter: growth
// publishes sign and destroys the old destination BEFORE allocation.
func (m *IntegerManager) CopyFromOwner(dst *Int, sourceManager *IntegerManager, src *Int) error {
	return m.copyFromOwnerWithReceipt(dst, sourceManager, src, nil)
}
func (m *IntegerManager) copyFromOwnerWithReceipt(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt) error {
	return m.copyFromOwnerWithReceipts(dst, sourceManager, src, receipt, nil)
}
func (m *IntegerManager) copyFromOwnerWithReceipts(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, pn *integerPNormalizeScope) error {
	return m.copyFromOwnerWithAllocationContext(dst, sourceManager, src, receipt, pnAllocationContext(pn))
}
func (m *IntegerManager) copyFromOwnerWithAllocationContext(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, context integerAllocationInvocationContext) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if sourceManager == nil {
		return arena.ErrConfig
	}
	if e := sourceManager.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	var owned *integerCopyAllocationScope
	if src.IsSmall() {
		v := src.val
		if e := m.Del(dst); e != nil {
			return e
		}
		dst.val = v
		return nil
	}
	source := sourceManager.cell(src.cell)
	dst.val = src.val // big_set publishes sign before allocation/free
	ref := dst.cell
	if ref == 0 || m.cell(ref).capacity < source.size {
		if ref != 0 {
			if e := m.free(ref); e != nil {
				return e
			}
		}
		dst.cell = m.failedRef
		if ref == 0 {
			dst.cell = m.failedNullRef
		}
		if receipt != nil {
			owned = &m.copyAllocation
			if owned.active || owned.invocation == ^uint64(0) {
				return ErrPolynomialPending
			}
			*owned = integerCopyAllocationScope{active: true, executing: true, invocation: owned.invocation + 1}
			*receipt = integerCopyAllocationReceipt{m, owned, owned.invocation}
			defer func() {
				owned.executing = false
				if !polynomialNegNumericPending(&owned.roots) {
					owned.active = false
				}
			}()
		}
		var e error
		ref, e = m.allocateWithSourceContext(source.capacity, source.size, owned, context)
		if e != nil {
			return e
		}
	}
	dst.cell = ref
	m.publishCell(ref)
	m.resizeCell(ref, source.size)
	for i := uint32(0); i < source.size; i++ {
		m.writeDigit(ref, i, sourceManager.digit(src.cell, i))
	}
	dst.cell = ref
	dst.val = src.val
	return nil
}

// Move transfers the cell and preserves the source moved-from inline m_val.
// Self move is explicitly a no-op, an added Go contract.
func (m *IntegerManager) Move(dst, src *Int) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if e := m.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.Del(dst); e != nil {
		return e
	}
	dst.val = src.val
	dst.cell = src.cell
	src.cell = 0
	return nil
}
func (m *IntegerManager) Swap(a, b *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	*a, *b = *b, *a
	return nil
}
func (m *IntegerManager) Neg(a *Int) error {
	return m.negWithAllocationContext(a, integerAllocationInvocationContext{})
}
func (m *IntegerManager) negWithAllocationContext(a *Int, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.IsSmall() && a.val == math.MinInt32 {
		return m.setInt64WithAllocationContext(a, -int64(math.MinInt32), context)
	}
	a.val = -a.val
	return nil
}
func (m *IntegerManager) Abs(a *Int) error {
	return m.absWithAllocationContext(a, integerAllocationInvocationContext{})
}
func (m *IntegerManager) absWithAllocationContext(a *Int, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.IsSmall() {
		if a.val < 0 {
			return m.negWithAllocationContext(a, context)
		}
	} else {
		a.val = 1
	}
	return nil
}

func (m *IntegerManager) signCell(a *Int, idx int) (int32, numericCellRef) {
	if !a.IsSmall() {
		return a.val, a.cell
	}
	if a.val == math.MinInt32 {
		return -1, m.intMin.cell
	}
	sign := int32(1)
	v := a.val
	if v < 0 {
		sign = -1
		v = -v
	}
	m.writeDigit(m.arg[idx], 0, uint32(v))
	return sign, m.arg[idx]
}
func (m *IntegerManager) compareCells(a, b numericCellRef) int {
	for i := max(m.cell(a).size, m.cell(b).size); i > 0; i-- {
		u, v := m.digit(a, i-1), m.digit(b, i-1)
		if u > v {
			return 1
		}
		if u < v {
			return -1
		}
	}
	return 0
}
func (m *IntegerManager) Compare(a, b *Int) (int, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if e := m.check(b); e != nil {
		return 0, e
	}
	if a.IsSmall() && b.IsSmall() {
		if a.val < b.val {
			return -1, nil
		}
		if a.val > b.val {
			return 1, nil
		}
		return 0, nil
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	if sa > 0 {
		if sb > 0 {
			return m.compareCells(ca, cb), nil
		}
		return 1, nil
	}
	if sb > 0 {
		return -1, nil
	}
	return m.compareCells(cb, ca), nil
}

func (m *IntegerManager) ensureTmp(idx int, size uint32) error {
	return m.ensureTmpWithPNReceipt(idx, size, nil)
}
func (m *IntegerManager) ensureTmpWithPNReceipt(idx int, size uint32, pn *integerPNormalizeScope) error {
	return m.ensureTmpWithAllocationContext(idx, size, pnAllocationContext(pn))
}
func (m *IntegerManager) ensureTmpWithAllocationContext(idx int, size uint32, context integerAllocationInvocationContext) error {
	if idx < 0 || idx >= len(m.tmp) || m.tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	// Source: src/util/mpz.h:156-162. A returning release guard can follow
	// physical consumption before replacement publication. The retained token
	// still owns its descriptor for disposal, but cannot name readable scratch.
	// Refuse the failed lifetime before the capacity shortcut, without repair.
	if !validNumericRef(m.cells, m.tmp[idx]) || m.cell(m.tmp[idx]).released {
		return arena.ErrInvalidSlot
	}
	if m.cell(m.tmp[idx]).capacity >= size {
		return nil
	}
	if e := m.free(m.tmp[idx]); e != nil {
		return e
	}
	m.tmp[idx] = 0                // source leaves a dangling pointer; Go keeps a guarded hole
	capacity := (3*size + 1) >> 1 // actual unsigned32 source arithmetic
	ref, e := m.allocateWithSourceContext(capacity, 0, nil, context)
	if e != nil {
		return e
	}
	m.tmp[idx] = ref
	m.publishCell(ref)
	if modularSourceDebug && m.cell(ref).capacity < size {
		return ErrIntegerAssertion
	}
	return nil
}
func (m *IntegerManager) publishTmp(dst *Int, sign int32, size uint32) error {
	return m.publishTmpAt(0, dst, sign, size)
}

// publishTmpAt publishes the source sign/pointer/size prefix before requesting
// missing scratch. Paired division calls the setters sequentially.
func (m *IntegerManager) publishTmpAt(idx int, dst *Int, sign int32, size uint32) error {
	return m.publishTmpAtWithPNReceipt(idx, dst, sign, size, nil)
}
func (m *IntegerManager) publishTmpAtWithPNReceipt(idx int, dst *Int, sign int32, size uint32, pn *integerPNormalizeScope) error {
	return m.publishTmpAtWithAllocationContext(idx, dst, sign, size, pnAllocationContext(pn))
}
func (m *IntegerManager) publishTmpAtWithAllocationContext(idx int, dst *Int, sign int32, size uint32, context integerAllocationInvocationContext) error {
	if idx < 0 || idx >= len(m.tmp) || m.tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	ref := m.tmp[idx]
	for size > 0 && m.digit(ref, size-1) == 0 {
		size--
	}
	if size == 0 {
		return m.Reset(dst)
	}
	if size == 1 && m.digit(ref, 0) <= math.MaxInt32 {
		v := int32(m.digit(ref, 0))
		if sign < 0 {
			v = -v
		}
		return m.setInt64WithAllocationContext(dst, int64(v), context)
	}
	dst.val = sign
	dst.cell, m.tmp[idx] = m.tmp[idx], dst.cell
	m.resizeCell(ref, size)
	if m.tmp[idx] == 0 {
		var e error
		m.tmp[idx], e = m.allocateWithSourceContext(6, 0, nil, context)
		if e == nil {
			m.publishCell(m.tmp[idx])
		}
		return e
	}
	return nil
}
func (m *IntegerManager) Add(a, b, c *Int) error { return m.addSub(a, b, c, false) }
func (m *IntegerManager) Sub(a, b, c *Int) error { return m.addSub(a, b, c, true) }
func (m *IntegerManager) addSub(a, b, c *Int, subtract bool) error {
	return m.addSubWithPNReceipt(a, b, c, subtract, nil)
}
func (m *IntegerManager) addSubWithPNReceipt(a, b, c *Int, subtract bool, pn *integerPNormalizeScope) error {
	return m.addSubWithAllocationContext(a, b, c, subtract, pnAllocationContext(pn))
}
func (m *IntegerManager) addSubWithAllocationContext(a, b, c *Int, subtract bool, context integerAllocationInvocationContext) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.IsSmall() && b.IsSmall() {
		v := int64(b.val)
		if subtract {
			v = -v
		}
		return m.setInt64WithAllocationContext(c, int64(a.val)+v, context)
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	if subtract {
		sb = -sb
	}
	na, nb := m.cell(ca).size, m.cell(cb).size
	n := max(na, nb)
	sign := sa
	if sa == sb {
		if n == math.MaxUint32 {
			return arena.ErrBounds
		}
		if e := m.ensureTmpWithAllocationContext(0, n+1, context); e != nil {
			return e
		}
		if e := m.prepareTmpSpan(0, n+1); e != nil {
			return e
		}
		var k uint32
		for j := uint32(0); j < n; j++ {
			u, v := m.digit(ca, j), m.digit(cb, j)
			r := u + v
			c1 := r < u
			out := r + k
			c2 := out < r
			m.writeDigit(m.tmp[0], j, out)
			k = 0
			if c1 || c2 {
				k = 1
			}
		}
		m.writeDigit(m.tmp[0], n, k)
		return m.publishTmpAtWithAllocationContext(0, c, sign, n+1, context)
	}
	cmp := m.compareCells(ca, cb)
	if cmp == 0 {
		return m.Reset(c)
	}
	if cmp < 0 {
		ca, cb = cb, ca
		sign = sb
		n = nb
	} else {
		n = na
	}
	if e := m.ensureTmpWithAllocationContext(0, n, context); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(0, n); e != nil {
		return e
	}
	var k uint32
	for j := uint32(0); j < n; j++ {
		u, v := m.digit(ca, j), m.digit(cb, j)
		r := u - v
		c1 := r > u
		out := r - k
		c2 := out > r
		m.writeDigit(m.tmp[0], j, out)
		k = 0
		if c1 || c2 {
			k = 1
		}
	}
	if k != 0 {
		panic("mpnum: unexpected subtraction borrow")
	}
	return m.publishTmpAtWithAllocationContext(0, c, sign, n, context)
}

func (m *IntegerManager) absUint64(a *Int) (uint64, error) {
	c := m.cell(a.cell)
	if modularSourceDebug && c.size > 2 {
		return 0, ErrIntegerAssertion
	}
	// The source size==1 branch reads only D0; every other size reads D1/D0.
	// These are physical reads, including retained words under a malformed
	// size0/>2 header, rather than the arithmetic accessor's size padding.
	if c.size == 1 {
		v, e := c.backing.readWord(m.allocator, 8)
		return uint64(v), e
	}
	hi, e := c.backing.readWord(m.allocator, 12)
	if e != nil {
		return 0, e
	}
	lo, e := c.backing.readWord(m.allocator, 8)
	return uint64(hi)<<32 | uint64(lo), e
}
func (m *IntegerManager) IsUint64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.val < 0 {
		return false, nil
	}
	return a.IsSmall() || m.cell(a.cell).size <= 2, nil
}
func (m *IntegerManager) IsInt64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.IsSmall() {
		return true, nil
	}
	if m.cell(a.cell).size > 2 {
		return false, nil
	}
	v, e := m.absUint64(a)
	if e != nil {
		return false, e
	}
	msb := v & (uint64(1) << 63)
	if a.val >= 0 {
		return msb == 0, nil
	}
	return msb == 0 || msb == v, nil
}

// Uint64 preserves the source accessor: inline negatives cast to uint64,
// big negatives return their absolute magnitude. Use IsUint64 for fit/sign.
func (m *IntegerManager) Uint64(a *Int) (uint64, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		return uint64(int64(a.val)), nil
	}
	if modularSourceDebug && m.cell(a.cell).size == 0 {
		return 0, ErrIntegerAssertion
	}
	return m.absUint64(a)
}
func (m *IntegerManager) Int64(a *Int) (int64, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		return int64(a.val), nil
	}
	if modularSourceDebug {
		ok, e := m.IsInt64(a)
		if e != nil {
			return 0, e
		}
		if !ok {
			return 0, ErrIntegerAssertion
		}
	}
	v, e := m.absUint64(a)
	if e != nil {
		return 0, e
	}
	if a.val < 0 {
		if v != 0 && v<<1 == 0 {
			return math.MinInt64, nil
		}
		return -int64(v), nil
	}
	return int64(v), nil
}

// isFailureRef recognizes only this owner's two permanently dead, charged
// descriptors. Ordinary reads still reject both through the normal live guard.
func (m *IntegerManager) isFailureRef(ref numericCellRef) bool {
	return ref != 0 && m.failedRef != 0 && (ref == m.failedRef || ref == m.failedNullRef)
}
