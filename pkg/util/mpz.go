// Source: src/util/mpz.h:698-707.
// Translated from src/util/mpz.h and src/util/mpz.cpp (_MP_INTERNAL, 32-bit digits).
package util

import (
	"errors"
	"math"
	"reflect"
	"runtime"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrConversion = errors.New("mpnum: integer conversion out of range")
var ErrIntegerAssertion = errors.New("mpnum: source integer assertion failed")

// Int is a pointer-free, owner-local mutable integer. With a zero cell ref,
// val is an inline int32. Otherwise val is the sign (-1/+1). A big Int must
// NEVER be shallow-copied: use Copy, Move, or Swap through its owning manager.
// The implicit manager owner is a caller contract; refs do not identify owners.
type Int struct {
	Val  int32
	Cell NumericCellRef
}

func NewInt(v int32) Int     { return Int{Val: v} }
func (a *Int) IsSmall() bool { return a.Cell == 0 }

type NumericCell struct {
	Payload                               arena.VectorRef // zero compatibility field; numerical bytes live only in backing
	Backing                               SourceCellBacking
	Size, Capacity, span                  uint32
	Released, Unpublished, CleanupPending bool
}
type IntegerConfig struct {
	Cells   arena.Config
	Words   arena.VectorConfig
	Memory  *Runtime
	Profile SmallAllocatorProfile
}

// IntegerManager owns all persistent source cells, two reusable result
// cells, two one-limb argument cells, and the source intMin/two64 constants.
// Single-owner confinement replaces the source synchronized OpenMP variant.
// Cells, transitional Natural Words and the actual memory backend share one Budget.
// Close invalidates every live numeral; a supplied Runtime is borrowed.
type IntegerManager struct {
	stringSet       integerStringSetScope
	smt2Stream      integerSMT2StreamScope
	mySqrt          integerMPFSqrtHelperScope
	callerExecuting func() bool
	callerBindings  *NumericCallerBinding

	rationalServiceParent *RationalService
	rationalServiceRead   *RationalServiceInvocation

	conversionDetachedOutputs                       *arbitraryFloatRationalConversionScope
	mpfConversionCaller                             *arbitraryFloatRationalConversionScope
	DividesNumeral                                  integerDividesNumeralScope
	DivisionRaw, ExtendedRaw, GcdRaw, BitwiseNotRaw IntegerRawScope
	BitwiseOrRaw, BitwiseAndRaw, BitwiseXorRaw      IntegerRawScope
	AddMulInvocation                                integerAddMulScope
	CopyAllocation                                  integerCopyAllocationScope
	NormalizationExecuting                          bool

	self                                   *IntegerManager
	mpfValueRead                           *arbitraryFloatCopyBinding
	MpfParent                              *ArbitraryFloatManager
	Memory                                 *Runtime
	Allocator                              *SmallAllocator
	ownsMemory                             bool
	PendingCell                            SourceCellBacking
	originalCause, CleanupCauseField       error
	Cells                                  *numericCellSlab
	Words                                  *arena.Words
	Tmp, Arg                               [2]NumericCellRef
	FailedRef, FailedNullRef               NumericCellRef  // permanently dead, charged auxiliary descriptor
	Natural                                *NaturalManager // child borrows this manager's Words and Budget
	IntMin, Two64                          Int
	PowerRaw                               integerPowerScope
	powerGeneration                        uint64
	RootRaw                                integerRootScope
	rootGeneration                         uint64
	PerfectSquare                          integerPerfectSquareScope
	PerfectSquareGeneration                uint64
	RawDisplay                             rawDisplayScope
	Smt2Display                            smt2DisplayScope
	closed                                 bool
	SourceFinalizing, SourceFinalized      bool
	sourceGuard                            bool
	SourceFinalizeStage                    uint8
	constructionFailed, ConstructorUnwound bool
	constructorUnwindStage                 uint8
	BudgetField                            *arena.Budget
	HeaderBytes                            uint64
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
	m := &IntegerManager{BudgetField: config.Cells.Budget, HeaderBytes: header, Memory: config.Memory}
	m.self = m
	fail := func(err error) (*IntegerManager, error) {
		m.originalCause = err
		m.constructionFailed = true
		if cleanup := m.unwindConstructorMembers(); cleanup != nil {
			m.CleanupCauseField = cleanup
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
	if m.Memory == nil {
		m.Memory, e = NewRuntime(RuntimeConfig{Budget: m.BudgetField, SizeTBits: uint8(32 << (^uint(0) >> 63))})
		if e != nil {
			return fail(e)
		}
		m.ownsMemory = true
	}
	profile := config.Profile
	if profile.PointerBits == 0 {
		profile = SmallAllocatorProfile{PointerBits: m.Memory.SizeTBits(), Debug: modularSourceDebug, Windows: runtime.GOOS == "windows"}
	}
	m.Allocator, e = NewSmallAllocator(SmallAllocatorConfig{Runtime: m.Memory, Profile: profile})
	if e != nil {
		return fail(e)
	}
	m.Cells, e = newNumericCellSlab(config.Cells)
	if e != nil {
		return fail(e)
	}
	slot, gen, e := m.Cells.reserveDead()
	if e != nil {
		return fail(e)
	}
	m.FailedRef = MakeNumericRef(slot, gen)
	// Both source pointer failure states use real permanently dead Go metadata
	// descriptors. They never contribute source cells or allocator counters.
	slot, gen, e = m.Cells.reserveDead()
	if e != nil {
		m.originalCause = e
		m.constructionFailed = true
		m.CleanupCauseField = m.unwindConstructorMembers()
		return nil, m // the already charged owner retains the actual cold roots
	}
	m.FailedNullRef = MakeNumericRef(slot, gen)
	// Actual Natural Words scratch is retained only during the staged backend
	// migration. It does not hold numeric cells or duplicate physical digits.
	m.Words, e = arena.NewWords(config.Words)
	if e != nil {
		return fail(e)
	}
	m.Natural, e = newNaturalManagerWithMemory(config.Words, m.Words, false, m.Memory)
	if e != nil {
		return fail(e)
	}
	for i := 0; i < 2; i++ {
		m.Tmp[i], e = m.Allocate(6, 0)
		if e != nil {
			return fail(e)
		}
		m.PublishCell(m.Tmp[i])
		m.Arg[i], e = m.Allocate(6, 0)
		if e != nil {
			return fail(e)
		}
		m.PublishCell(m.Arg[i])
		m.ResizeCell(m.Arg[i], 1)
	}
	if e = m.SetInt64(&m.IntMin, -int64(math.MinInt32)); e != nil {
		return fail(e)
	}
	if e = m.SetUint64(&m.Two64, math.MaxUint64); e != nil {
		return fail(e)
	}
	one := NewInt(1)
	if e = m.Add(&m.Two64, &one, &m.Two64); e != nil {
		return fail(e)
	}
	return m, nil
}

func (m *IntegerManager) RuntimeOwner() *Runtime {
	if m == nil || m.self != m {
		return nil
	}
	return m.Memory
}
func (m *IntegerManager) CellObject(ref NumericCellRef) (ObjectRef, error) {
	if m == nil || m.self != m {
		return ObjectRef{}, ErrRuntimeOwner
	}
	if !ValidNumericRef(m.Cells, ref) {
		return ObjectRef{}, arena.ErrInvalidSlot
	}
	c := m.Cell(ref)
	if c.Released {
		return ObjectRef{}, arena.ErrInvalidSlot
	}
	return c.Backing.Object, nil
}
func (m *IntegerManager) Error() string {
	if m.originalCause != nil {
		return m.originalCause.Error()
	}
	if m.CleanupCauseField != nil {
		return m.CleanupCauseField.Error()
	}
	return "mpnum: retained integer owner"
}
func (m *IntegerManager) Unwrap() error {
	if m.originalCause != nil {
		return m.originalCause
	}
	return m.CleanupCauseField
}
func (m *IntegerManager) OriginalCause() error { return m.originalCause }
func (m *IntegerManager) CleanupCause() error  { return m.CleanupCauseField }
func (m *IntegerManager) Cleanup() error       { return m.CloseChecked() }

// checkCleanup accepts the owner's permanently dead allocation token only for
// release/reset after a source allocation prefix. Ordinary reads use check.
func (m *IntegerManager) checkCleanup(a *Int) error {
	if m.stringSetTransportActive() {
		return ErrNumericPending
	}
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.rationalServiceReadCallbackActive() || (m.mpfValueRead != nil && m.mpfBorrowedSourceCallbackActive()) {
		return ErrNumericPending
	}
	if m.closed || m.SourceFinalizing || m.constructionFailed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if a.Cell == 0 || (m.FailedRef != 0 && (a.Cell == m.FailedRef || a.Cell == m.FailedNullRef)) {
		return nil
	}
	// A retained descriptor whose physical release completed remains cleanup
	// authority only; ordinary reads reject it until the descriptor is consumed.
	if !ValidNumericRef(m.Cells, a.Cell) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *IntegerManager) check(a *Int) error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.stringSetTransportActive() || m.decimalStreamExecuting(true) || m.smt2StreamExecuting(true) || m.rationalStringExecuting(true) || m.mpfToMPZExecuting(true) || m.mpfArithmeticTransportActive() || m.myMPZSqrtTransportActive() {
		return ErrNumericPending
	}
	if m.rationalServiceReadCallbackActive() || (m.mpfValueRead != nil && m.mpfBorrowedSourceCallbackActive()) {
		return ErrNumericPending
	}
	if m.closed || m.SourceFinalizing || m.constructionFailed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if a.Cell != 0 && (!ValidNumericRef(m.Cells, a.Cell) || m.Cell(a.Cell).Released) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *IntegerManager) Cell(ref NumericCellRef) NumericCell {
	c, e := m.Cells.Read(NumericSlot(ref))
	if e != nil {
		panic(e)
	}
	return c
}
func (m *IntegerManager) Store(ref NumericCellRef, c NumericCell) {
	if e := m.Cells.Write(NumericSlot(ref), c); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) Digit(ref NumericCellRef, i uint32) uint32 {
	c := m.Cell(ref)
	if i >= max(c.Size, c.span) {
		return 0
	}
	v, e := c.Backing.ReadWord(m.Allocator, 8+4*uint64(i))
	if e != nil {
		panic(e)
	}
	return v
}
func (m *IntegerManager) writeDigit(ref NumericCellRef, i, v uint32) {
	c := m.Cell(ref)
	if e := c.Backing.writeWord(m.Allocator, 8+4*uint64(i), v); e != nil {
		panic(e)
	}
}

// A scratch access span is bookkeeping, never a source m_size publication.
func (m *IntegerManager) prepareTmpSpan(idx int, size uint32) error {
	if idx < 0 || idx >= len(m.Tmp) || m.Tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	c := m.Cell(m.Tmp[idx])
	if size > c.Capacity || uint64(size)*4+8 > c.Backing.extent {
		return arena.ErrBounds
	}
	c.span = size
	m.Store(m.Tmp[idx], c)
	return nil
}
func (m *IntegerManager) ResizeCell(ref NumericCellRef, size uint32) {
	c := m.Cell(ref)
	if size > c.Capacity {
		panic(arena.ErrBounds)
	}
	if e := c.Backing.writeWord(m.Allocator, 0, size); e != nil {
		panic(e)
	}
	c.Size = size
	c.span = 0
	m.Store(ref, c)
}
func (m *IntegerManager) Allocate(capacity, size uint32) (NumericCellRef, error) {
	return m.allocateWithCopyReceipt(capacity, size, nil)
}
func (m *IntegerManager) allocateWithCopyReceipt(capacity, size uint32, s *integerCopyAllocationScope) (NumericCellRef, error) {
	return m.allocateWithInvocationReceipts(capacity, size, s, nil)
}
func (m *IntegerManager) allocateWithInvocationReceipts(capacity, size uint32, s *integerCopyAllocationScope, pn *integerPNormalizeScope) (NumericCellRef, error) {
	return m.allocateWithSourceContext(capacity, size, s, pnAllocationContext(pn))
}
func (m *IntegerManager) allocateWithSourceContext(capacity, size uint32, s *integerCopyAllocationScope, context integerAllocationInvocationContext) (NumericCellRef, error) {
	if context.rationalService != nil && context.rationalServiceFrame(m) == nil {
		return 0, arena.ErrInvalidSlot
	}
	if modularSourceDebug && capacity < 6 {
		return 0, ErrIntegerAssertion
	}
	if m.PendingCell.Object != (ObjectRef{}) {
		return 0, ErrAllocatorPending
	}
	if e := context.prepareCaller(m, "constant-history"); e != nil {
		return 0, e
	}
	if e := context.prepareRationalSetHistory(m); e != nil {
		return 0, e
	}
	if e := context.prepareFormattedSetHistory(m); e != nil {
		return 0, e
	}
	if e := context.prepareRoundHistory(m); e != nil {
		return 0, e
	}
	if e := context.prepareInputAllocation(m); e != nil {
		return 0, e
	}
	var outcome SmallAllocatorAllocationOutcome
	var out *SmallAllocatorAllocationOutcome
	var roots *NumericAllocationRoots
	if s != nil && m.executingCopyAllocationScope(s) {
		roots = &s.Roots
	}
	if r := context.roots(m); r != nil {
		roots = r
	}
	if roots != nil {
		out = &outcome
	}
	if e := context.prepareCaller(m, "display-storage"); e != nil {
		return 0, e
	}
	if e := context.prepareConversionHistory(m); e != nil {
		return 0, e
	}
	if e := context.prepareCaller(m, "factor-history"); e != nil {
		return 0, e
	}
	backing, e := allocateSourceCellWithOutcome(m.Allocator, capacity, out)
	// Capture the physical prefix before charged descriptor preparation.
	m.PendingCell = backing
	if out != nil {
		if backing.Object != (ObjectRef{}) {
			roots.Cell = backing
			if s != nil && roots == &s.Roots {
				s.originalCell = backing
			}
		}
		if outcome.Pending != (Block{}) {
			roots.Allocator[0] = outcome.Pending
		}
	}
	if e != nil {
		if transfer := m.captureRationalServiceAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureRationalSetAllocation(context, backing, &outcome); transfer != nil {
			return 0, transfer
		}
		if transfer := m.captureFormattedSetAllocation(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureRoundAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureInputAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureConversionAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureFormatAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if context.mpfPower != nil && roots == context.mpfPowerRoots(m) {
			if transfer := m.capturePowerAllocationOutcome(context, backing, &outcome); transfer != nil {
				return 0, JoinNumericErrors(e, transfer)
			}
		}
		if s != nil && s.mpfCaller != nil && roots == &s.Roots {
			if outcome.Pending != (Block{}) {
				if transfer := m.Allocator.TransferAllocationPending(&outcome, &s.Transferred); transfer != nil {
					return 0, JoinNumericErrors(e, transfer)
				}
			}
			if backing.Object != (ObjectRef{}) {
				m.PendingCell = SourceCellBacking{}
			}
		}
		return 0, e
	}
	slot, gen, e := m.Cells.Allocate()
	if e != nil {
		if transfer := m.captureRationalServiceAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureRationalSetAllocation(context, backing, &outcome); transfer != nil {
			return 0, transfer
		}
		if transfer := m.captureFormattedSetAllocation(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureRoundAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureInputAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureConversionAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if transfer := m.captureFormatAllocationOutcome(context, backing, &outcome); transfer != nil {
			return 0, JoinNumericErrors(e, transfer)
		}
		if context.mpfPower != nil && roots == context.mpfPowerRoots(m) {
			if transfer := m.capturePowerAllocationOutcome(context, backing, &outcome); transfer != nil {
				return 0, JoinNumericErrors(e, transfer)
			}
		}
		if s != nil && s.mpfCaller != nil && roots == &s.Roots {
			m.PendingCell = SourceCellBacking{}
		}
		return 0, e
	}
	ref := MakeNumericRef(slot, gen)
	m.Store(ref, NumericCell{Backing: backing, Capacity: capacity, Unpublished: true})
	m.PendingCell = SourceCellBacking{}
	if out != nil && roots.Cell == backing {
		roots.Cell = SourceCellBacking{}
	}
	return ref, nil
}

// Publication is scalar ownership metadata, not a source allocation or write.
func (m *IntegerManager) PublishCell(ref NumericCellRef) {
	c := m.Cell(ref)
	c.Unpublished = false
	m.Store(ref, c)
}
func (m *IntegerManager) free(ref NumericCellRef) error {
	c := m.Cell(ref)
	if !c.Released {
		progress, e := c.Backing.deallocate(m.Allocator)
		if progress.PhysicalReleased || progress.PooledLinked {
			c.Released = true
			m.Store(ref, c)
		}
		if e != nil {
			c.CleanupPending = true
			m.Store(ref, c)
			return e
		}
	}
	return m.Cells.Recycle(NumericSlot(ref))
}
func (m *IntegerManager) Close() {
	if e := m.CloseChecked(); e != nil {
		panic(m)
	}
}
func (m *IntegerManager) CloseChecked() error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.rawInvocationExecuting() || m.CopyAllocation.Executing || m.NormalizationExecuting || m.numericCallerExecuting() {
		return ErrNumericPending
	}
	if m.closed {
		m.CleanupCauseField = nil
		return nil
	}
	fail := func(e error) error { m.CleanupCauseField = e; return m }
	if m.constructionFailed && !m.ConstructorUnwound {
		if m.Allocator != nil {
			if e := m.Allocator.CleanupPending(); e != nil {
				return fail(e)
			}
		}
		if e := m.unwindConstructorMembers(); e != nil {
			return fail(e)
		}
	}
	if e := m.CleanupStringSet(); e != nil {
		return fail(e)
	}
	if e := m.CleanupSMT2Stream(); e != nil {
		return fail(e)
	}
	if e := m.CleanupSMT2Display(); e != nil {
		return fail(e)
	}
	// Resume every explicitly begun source cursor before owning cleanup. A
	// missing/stale private pointer stops the source body at an undefined access;
	// owning cleanup then aborts that body without repairing it or advancing its
	// history, and releases only known actual roots.
	if m.SourceFinalizing && !m.sourceGuard && !m.SourceFinalized {
		if e := m.SourceFinalize(); e != nil {
			return fail(e)
		}
	}

	if e := m.CleanupMyMPZSqrt(); e != nil {
		return fail(e)
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
	if e := m.CleanupBitwiseNot(); e != nil {
		return fail(e)
	}
	for _, raw := range []*IntegerRawScope{&m.BitwiseOrRaw, &m.BitwiseAndRaw, &m.BitwiseXorRaw} {
		r := integerRawInvocationReceipt{m, raw, raw.Invocation}
		if e := m.CleanupRawInvocation(&r, true); e != nil {
			return fail(e)
		}
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
	if m.Cells != nil {
		for n := uint64(1); n <= uint64(m.Cells.Stats().Slots); n++ {
			slot := arena.Slot(n)
			if _, e := m.Cells.Read(slot); e == nil {
				// Slab slots retain their actual generation. Only live records are read.
				gen := m.Cells.State(slot).Generation
				ref := MakeNumericRef(slot, gen)
				if m.SourceFinalized || m.ConstructorUnwound {
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
	if m.Natural != nil {
		if e := m.Natural.CloseChecked(); e != nil {
			return fail(e)
		}
		m.Natural = nil
	}
	if m.Words != nil {
		m.Words.Close()
	}
	if m.Allocator != nil {
		if e := m.Allocator.CleanupPending(); e != nil {
			return fail(e)
		}
		if e := m.Allocator.Close(); e != nil {
			return fail(e)
		}
	}
	if m.ownsMemory && m.Memory != nil {
		if e := m.Memory.Close(); e != nil {
			return fail(e)
		}
	}
	if m.Cells != nil {
		m.Cells.Close()
	}
	m.Tmp = [2]NumericCellRef{}
	m.Arg = [2]NumericCellRef{}
	m.IntMin = Int{}
	m.Two64 = Int{}
	m.BudgetField.Release(m.HeaderBytes)
	m.HeaderBytes = 0
	m.closed = true
	m.CleanupCauseField = nil
	return nil
}

// Del preserves the source's m_val after releasing a big cell. Thus deleting a
// big positive/negative value leaves inline +1/-1. Reset additionally writes 0.
func (m *IntegerManager) Del(a *Int) error {
	if m.stringSetTransportActive() || m.decimalStreamExecuting(true) || m.smt2StreamExecuting(true) || m.rationalStringExecuting(true) || m.mpfToMPZExecuting(true) || m.mpfArithmeticTransportActive() || m.myMPZSqrtTransportActive() {
		return ErrNumericPending
	}
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if m.FailedRef != 0 && (a.Cell == m.FailedRef || a.Cell == m.FailedNullRef) {
		a.Cell = 0
		return nil
	}
	if a.Cell != 0 {
		if e := m.free(a.Cell); e != nil {
			return e
		}
		a.Cell = 0
	}
	return nil
}
func (m *IntegerManager) Reset(a *Int) error {
	if e := m.Del(a); e != nil {
		return e
	}
	a.Val = 0
	return nil
}

// allocateIfNeeded is the source destructive capacity helper. It never copies
// old limbs. A failed replacement leaves the already-published positive sign
// and the owner's dead allocation token; only cleanup may consume that token.
func (m *IntegerManager) allocateIfNeeded(a *Int, capacity uint32) error {
	return m.allocateIfNeededWithAllocationContext(a, capacity, integerAllocationInvocationContext{})
}
func (m *IntegerManager) allocateIfNeededWithAllocationContext(a *Int, capacity uint32, context integerAllocationInvocationContext) error {
	return m.allocateIfNeededWithPhysicalAllocationContext(a, capacity, context, nil)
}
func (m *IntegerManager) allocateIfNeededWithPhysicalAllocationContext(a *Int, capacity uint32, context integerAllocationInvocationContext, receiver *integerRuntimeValueReceiver) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := receiver.validate(m, a); e != nil {
		return e
	}
	if capacity < 6 {
		capacity = 6
	}
	aWasSmall := a.IsSmall()
	if aWasSmall || m.Cell(a.Cell).Capacity < capacity {
		if a.Cell != 0 {
			if e := m.free(a.Cell); e != nil {
				return e
			}
		}
		a.Val = 1
		a.Cell = m.FailedRef
		if aWasSmall {
			a.Cell = m.FailedNullRef
		}
		if e := receiver.publishValue(a); e != nil {
			return e
		}
		if e := receiver.publishCell(a); e != nil {
			return e
		}
		ref, e := m.allocateWithSourceContext(capacity, 0, nil, context)
		if e != nil {
			return e
		}
		a.Cell = ref
		if e := receiver.publishCell(a); e != nil {
			return e
		}
		m.PublishCell(ref)
	}
	return nil
}

func (m *IntegerManager) setBig64(a *Int, v uint64, sign int32) error {
	return m.setBig64WithAllocationContext(a, v, sign, integerAllocationInvocationContext{})
}
func (m *IntegerManager) setBig64WithAllocationContext(a *Int, v uint64, sign int32, context integerAllocationInvocationContext) error {
	ref := a.Cell
	if ref == 0 {
		var e error
		ref, e = m.allocateWithSourceContext(6, 0, nil, context)
		if e != nil {
			return e
		}
	}
	// mpz.cpp set_big_i64/set_big_ui64 assert capacity after optional
	// allocation and before any sign or limb publication.
	if modularSourceDebug && m.Cell(ref).Capacity < 6 {
		return ErrIntegerAssertion
	}
	size := uint32(1)
	if v>>32 != 0 {
		size = 2
	}
	a.Cell = ref
	m.PublishCell(ref)
	a.Val = sign
	m.writeDigit(ref, 0, uint32(v))
	m.writeDigit(ref, 1, uint32(v>>32))
	m.ResizeCell(ref, size)
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
		a.Val = int32(v)
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
	a.Val = 1 // source publication precedes the allocation branches
	ref := a.Cell
	if ref == 0 || m.Cell(ref).Capacity < size {
		if ref != 0 {
			if e := m.free(ref); e != nil {
				return e
			}
		}
		a.Cell = m.FailedRef
		if ref == 0 {
			a.Cell = m.FailedNullRef
		}
		var e error
		ref, e = m.Allocate(max(size, 6), size)
		if e != nil {
			return e
		}
	}
	a.Cell = ref
	m.PublishCell(ref)
	m.ResizeCell(ref, size)
	for i, v := range digits {
		m.writeDigit(ref, uint32(i), v)
	}
	a.Cell = ref
	a.Val = 1
	return nil
}
func (m *IntegerManager) Copy(dst, src *Int) error { return m.CopyFromOwner(dst, m, src) }

// CopyFromOwner is the same literal SET engine with the actual borrowed source
// cell owner. It differs from the staged ergonomic CopyFrom adapter: growth
// publishes sign and destroys the old destination BEFORE allocation.
func (m *IntegerManager) CopyFromOwner(dst *Int, sourceManager *IntegerManager, src *Int) error {
	return m.CopyFromOwnerWithReceipt(dst, sourceManager, src, nil)
}
func (m *IntegerManager) CopyFromOwnerWithReceipt(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt) error {
	return m.copyFromOwnerWithReceipts(dst, sourceManager, src, receipt, nil)
}
func (m *IntegerManager) copyFromOwnerWithReceipts(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, pn *integerPNormalizeScope) error {
	return m.copyFromOwnerWithAllocationContext(dst, sourceManager, src, receipt, pnAllocationContext(pn))
}
func (m *IntegerManager) copyFromOwnerWithAllocationContext(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, context integerAllocationInvocationContext) error {
	return m.copyFromOwnerWithPhysicalAllocationContext(dst, sourceManager, src, receipt, context, nil)
}
func (m *IntegerManager) copyFromOwnerWithPhysicalAllocationContext(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, context integerAllocationInvocationContext, receiver *integerRuntimeValueReceiver) error {
	return m.copyFromOwnerWithPhysicalFloatCaller(dst, sourceManager, src, receipt, context, receiver, nil)
}
func (m *IntegerManager) copyFromOwnerWithPhysicalFloatCaller(dst *Int, sourceManager *IntegerManager, src *Int, receipt *integerCopyAllocationReceipt, context integerAllocationInvocationContext, receiver *integerRuntimeValueReceiver, caller *arbitraryFloatCopyBinding) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if sourceManager == nil {
		return arena.ErrConfig
	}
	if e := sourceManager.check(src); e != nil {
		return e
	}
	if e := receiver.validate(m, dst); e != nil {
		return e
	}
	if e := m.validateFloatCopyBinding(caller); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	var owned *integerCopyAllocationScope
	if src.IsSmall() {
		v := src.Val
		delError := m.Del(dst)
		if e := receiver.publishCell(dst); e != nil {
			return e
		}
		if e := receiver.publishValue(dst); e != nil {
			return e
		}
		if delError != nil {
			return delError
		}
		dst.Val = v
		if e := receiver.publishValue(dst); e != nil {
			return e
		}
		return nil
	}
	source := sourceManager.Cell(src.Cell)
	dst.Val = src.Val // big_set publishes sign before allocation/free
	if e := receiver.publishValue(dst); e != nil {
		return e
	}
	ref := dst.Cell
	if ref == 0 || m.Cell(ref).Capacity < source.Size {
		if ref != 0 {
			if e := m.free(ref); e != nil {
				return e
			}
		}
		dst.Cell = m.FailedRef
		if ref == 0 {
			dst.Cell = m.FailedNullRef
		}
		if e := receiver.publishCell(dst); e != nil {
			return e
		}
		if receipt != nil {
			owned = &m.CopyAllocation
			if caller == nil && (owned.active || owned.invocation == ^uint64(0)) {
				return ErrNumericPending
			}
			if e := m.prepareFloatCopyRecord(caller); e != nil {
				return e
			}
			var mpfCaller *arbitraryFloatValueCopyCall
			invocation := owned.invocation + 1
			if caller != nil {
				mpfCaller = caller.record
				owned = &mpfCaller.ChildFrame
				invocation = caller.invocation
			}
			*owned = integerCopyAllocationScope{active: true, Executing: true, invocation: invocation, mpfCaller: mpfCaller}
			*receipt = integerCopyAllocationReceipt{m, owned, owned.invocation}
			m.bindFloatCopyChild(caller, *receipt)
			defer func() {
				owned.Executing = false
				if !NumericAllocationPending(&owned.Roots) {
					owned.active = false
				}
			}()
		}
		var e error
		ref, e = m.allocateWithSourceContext(source.Capacity, source.Size, owned, context)
		if e != nil {
			return e
		}
	}
	dst.Cell = ref
	if e := receiver.publishCell(dst); e != nil {
		return e
	}
	m.PublishCell(ref)
	m.ResizeCell(ref, source.Size)
	for i := uint32(0); i < source.Size; i++ {
		m.writeDigit(ref, i, sourceManager.Digit(src.Cell, i))
	}
	dst.Cell = ref
	if e := receiver.publishCell(dst); e != nil {
		return e
	}
	dst.Val = src.Val
	return receiver.publishValue(dst)
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
	dst.Val = src.Val
	dst.Cell = src.Cell
	src.Cell = 0
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
	if a.IsSmall() && a.Val == math.MinInt32 {
		return m.setInt64WithAllocationContext(a, -int64(math.MinInt32), context)
	}
	a.Val = -a.Val
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
		if a.Val < 0 {
			return m.negWithAllocationContext(a, context)
		}
	} else {
		a.Val = 1
	}
	return nil
}

func (m *IntegerManager) signCell(a *Int, idx int) (int32, NumericCellRef) {
	if !a.IsSmall() {
		return a.Val, a.Cell
	}
	if a.Val == math.MinInt32 {
		return -1, m.IntMin.Cell
	}
	sign := int32(1)
	v := a.Val
	if v < 0 {
		sign = -1
		v = -v
	}
	m.writeDigit(m.Arg[idx], 0, uint32(v))
	return sign, m.Arg[idx]
}
func (m *IntegerManager) compareCells(a, b NumericCellRef) int {
	for i := max(m.Cell(a).Size, m.Cell(b).Size); i > 0; i-- {
		u, v := m.Digit(a, i-1), m.Digit(b, i-1)
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
		if a.Val < b.Val {
			return -1, nil
		}
		if a.Val > b.Val {
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

func (m *IntegerManager) EnsureTmp(idx int, size uint32) error {
	return m.ensureTmpWithPNReceipt(idx, size, nil)
}
func (m *IntegerManager) ensureTmpWithPNReceipt(idx int, size uint32, pn *integerPNormalizeScope) error {
	return m.ensureTmpWithAllocationContext(idx, size, pnAllocationContext(pn))
}
func (m *IntegerManager) ensureTmpWithAllocationContext(idx int, size uint32, context integerAllocationInvocationContext) error {
	if idx < 0 || idx >= len(m.Tmp) || m.Tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	// Source: src/util/mpz.h:156-162. A returning release guard can follow
	// physical consumption before replacement publication. The retained token
	// still owns its descriptor for disposal, but cannot name readable scratch.
	// Refuse the failed lifetime before the capacity shortcut, without repair.
	if !ValidNumericRef(m.Cells, m.Tmp[idx]) || m.Cell(m.Tmp[idx]).Released {
		return arena.ErrInvalidSlot
	}
	if m.Cell(m.Tmp[idx]).Capacity >= size {
		return nil
	}
	if e := m.free(m.Tmp[idx]); e != nil {
		return e
	}
	m.Tmp[idx] = 0                // source leaves a dangling pointer; Go keeps a guarded hole
	capacity := (3*size + 1) >> 1 // actual unsigned32 source arithmetic
	ref, e := m.allocateWithSourceContext(capacity, 0, nil, context)
	if e != nil {
		return e
	}
	m.Tmp[idx] = ref
	m.PublishCell(ref)
	if modularSourceDebug && m.Cell(ref).Capacity < size {
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
	if idx < 0 || idx >= len(m.Tmp) || m.Tmp[idx] == 0 {
		return arena.ErrInvalidSlot
	}
	ref := m.Tmp[idx]
	for size > 0 && m.Digit(ref, size-1) == 0 {
		size--
	}
	if size == 0 {
		return m.Reset(dst)
	}
	if size == 1 && m.Digit(ref, 0) <= math.MaxInt32 {
		v := int32(m.Digit(ref, 0))
		if sign < 0 {
			v = -v
		}
		return m.setInt64WithAllocationContext(dst, int64(v), context)
	}
	dst.Val = sign
	dst.Cell, m.Tmp[idx] = m.Tmp[idx], dst.Cell
	m.ResizeCell(ref, size)
	if m.Tmp[idx] == 0 {
		var e error
		m.Tmp[idx], e = m.allocateWithSourceContext(6, 0, nil, context)
		if e == nil {
			m.PublishCell(m.Tmp[idx])
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
		v := int64(b.Val)
		if subtract {
			v = -v
		}
		return m.setInt64WithAllocationContext(c, int64(a.Val)+v, context)
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	if subtract {
		sb = -sb
	}
	na, nb := m.Cell(ca).Size, m.Cell(cb).Size
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
			u, v := m.Digit(ca, j), m.Digit(cb, j)
			r := u + v
			c1 := r < u
			out := r + k
			c2 := out < r
			m.writeDigit(m.Tmp[0], j, out)
			k = 0
			if c1 || c2 {
				k = 1
			}
		}
		m.writeDigit(m.Tmp[0], n, k)
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
		u, v := m.Digit(ca, j), m.Digit(cb, j)
		r := u - v
		c1 := r > u
		out := r - k
		c2 := out > r
		m.writeDigit(m.Tmp[0], j, out)
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
	c := m.Cell(a.Cell)
	if modularSourceDebug && c.Size > 2 {
		return 0, ErrIntegerAssertion
	}
	// The source size==1 branch reads only D0; every other size reads D1/D0.
	// These are physical reads, including retained words under a malformed
	// size0/>2 header, rather than the arithmetic accessor's size padding.
	if c.Size == 1 {
		v, e := c.Backing.ReadWord(m.Allocator, 8)
		return uint64(v), e
	}
	hi, e := c.Backing.ReadWord(m.Allocator, 12)
	if e != nil {
		return 0, e
	}
	lo, e := c.Backing.ReadWord(m.Allocator, 8)
	return uint64(hi)<<32 | uint64(lo), e
}
func (m *IntegerManager) IsUint64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.Val < 0 {
		return false, nil
	}
	return a.IsSmall() || m.Cell(a.Cell).Size <= 2, nil
}
func (m *IntegerManager) IsInt64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.IsSmall() {
		return true, nil
	}
	if m.Cell(a.Cell).Size > 2 {
		return false, nil
	}
	v, e := m.absUint64(a)
	if e != nil {
		return false, e
	}
	msb := v & (uint64(1) << 63)
	if a.Val >= 0 {
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
		return uint64(int64(a.Val)), nil
	}
	if modularSourceDebug && m.Cell(a.Cell).Size == 0 {
		return 0, ErrIntegerAssertion
	}
	return m.absUint64(a)
}
func (m *IntegerManager) Int64(a *Int) (int64, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		return int64(a.Val), nil
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
	if a.Val < 0 {
		if v != 0 && v<<1 == 0 {
			return math.MinInt64, nil
		}
		return -int64(v), nil
	}
	return int64(v), nil
}

// isFailureRef recognizes only this owner's two permanently dead, charged
// descriptors. Ordinary reads still reject both through the normal live guard.
func (m *IntegerManager) IsFailureRef(ref NumericCellRef) bool {
	return ref != 0 && m.FailedRef != 0 && (ref == m.FailedRef || ref == m.FailedNullRef)
}
