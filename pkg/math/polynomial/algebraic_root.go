// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577,2933-2935.
// The standalone numerical core borrows services and materializes workspaces
// on demand. AlgebraicManager supplies eager source-ordered ownership around
// these same numerical operations.
package polynomial

import (
	"errors"
	"math"
	"reflect"
	"sync/atomic"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

var ErrAlgebraicOwner = errors.New("mpnum: invalid algebraic owner or value")
var ErrAlgebraicAssertion = errors.New("mpnum: algebraic source assertion")
var ErrAlgebraicLive = errors.New("mpnum: live caller algebraic owners")
var ErrAlgebraicPending = errors.New("mpnum: algebraic cleanup pending")
var algebraicOrigins atomic.Uint64

// Null is source rational zero. Owning nonzero values must not be shallow-copied.
type Algebraic struct {
	owner            uint64
	slot, generation uint32
}
type AlgebraicRootConfig struct {
	Rationals   *unumeral.RationalManager
	Polynomials *UPolynomialManager
	Factors     *UPolynomialFactorizationService
	Binary      *unumeral.BinaryRationalManager
	Intervals   *unumeral.BinaryIntervalManager
	Allocator   *umemory.SmallAllocator
	Checkpoint  func() bool
	Cooperation func() error
}

// Only checked locations live here. Numeral fields are encoded in real object
// bytes; coefficient arrays are actual owned UPolynomial typed backings.
type algebraicLocation struct {
	raw                                       umemory.ObjectRef
	coefficientInitialized, coefficientCursor uint32
	sourceGuard                               bool
	generation                                uint32
	kind, stage                               uint8
	live, initialized, published              bool
}
type AlgebraicRootCore struct {
	representationNeg                                          algebraicRepresentationNegNumeric
	representationNormalize                                    unumeral.IntegerNormalizeInvocationReceipt
	representationCopy                                         unumeral.IntegerCopyAllocationReceipt
	comparisonTraceRunning                                     bool
	rationalCopy                                               unumeral.IntegerCopyAllocationReceipt
	manager                                                    *AlgebraicManager
	coefficientCopy                                            unumeral.IntegerCopyAllocationReceipt
	cellAllocation                                             umemory.SmallAllocatorAllocationOutcome
	cellPending                                                umemory.SmallAllocatorPendingAllocation
	cellAllocationRunning                                      bool
	zero                                                       unumeral.Rational
	rootConstruction                                           algebraicRootConstructionScope
	isolationSigns                                             algebraicIsolationSignsScope
	signEvaluationCaller                                       *algebraicIsolationSignsScope
	localSelection                                             *algebraicSelectionScope
	multivariateIsolationFrames                                []*algebraicMultivariateIsolationFrame
	multivariateIsolationDepth                                 uint32
	multivariateIsolationRunning                               bool
	isolateRootsVariables                                      PolynomialVariableDegrees
	polynomialCaller, localPolynomialCaller                    *polynomialAlgebraicEvaluationScope
	polynomialEvaluationBorrowers, polynomialEvaluationRunning uint32
	integerSet                                                 algebraicIntegerSetScope
	integerAdd                                                 algebraicIntegerAddScope
	localIntegerSet                                            *algebraicIntegerSetScope
	localIntegerAdd                                            *algebraicIntegerAddScope
	isolationConversion                                        algebraicIsolationConversionScope
	isolateTmp1                                                UPolynomial
	filterRootsRunning                                         bool
	signEvaluationRunning                                      bool
	signResultant                                              algebraicSignResultantScope
	evalSignTmp                                                UPolynomial
	signInterval                                               algebraicSignIntervalScope
	evalSignVariables                                          PolynomialVariableDegrees
	zeroAccuracy                                               int32
	signSubstitution                                           algebraicSignSubstitutionScope
	rationalSign                                               algebraicRationalSignScope
	rootDisplay                                                algebraicRootDisplayScope
	selection                                                  algebraicSelectionScope
	basicRootScope                                             algebraicBasicRootScope
	division                                                   algebraicDivisionScope
	binary                                                     algebraicBinaryScope
	multivariate                                               *MultivariatePolynomialManager
	resultant                                                  algebraicResultantPolynomialScope
	resultantVariables                                         [2]uint32
	resultantVariableCount                                     uint8
	arithmetic                                                 algebraicRationalArithmeticScope
	addTmp                                                     UPolynomial
	bounds                                                     algebraicBoundScope
	inversion                                                  algebraicInverseScope
	negation                                                   algebraicNegScope
	negationHistory                                            []algebraicNegHistory
	predicate                                                  algebraicPredicateScope
	isRationalTmp                                              unumeral.Int
	rootSelection                                              *AlgebraicVector
	self                                                       *AlgebraicRootCore
	config                                                     AlgebraicRootConfig
	origin, header                                             uint64
	locations                                                  []algebraicLocation
	closed, closing                                            bool
	factors                                                    *UPolynomialFactors
	roots, lowers, uppers                                      *BinaryRationalVector
	tmp2, tmp3                                                 UPolynomial
	pending                                                    Algebraic
	minMagnitude                                               int32
	factor                                                     bool
	factorParams                                               UPolynomialFactorParams
	compareCheap, comparePoly, compareRefine, compareSturm     uint32
	compareSequence                                            *UPolynomialSequence
	callers, vectors                                           uint32
	localQ                                                     unumeral.Rational
	localActive                                                bool
	localOriginal, localCleanup                                error
	constructionStage                                          uint8
}

// Scalar codec is actual Go-adapter extent: basic 24, root 56 bytes. It is not
// C++ sizeof. No duplicate owning numerals exist in the locator directory.
type algebraicRecord struct {
	value       unumeral.Rational
	poly        UPolynomial
	interval    unumeral.BinaryInterval
	flags, size uint32
}

func NewAlgebraicRootCore(c AlgebraicRootConfig) (*AlgebraicRootCore, error) {
	if c.Rationals == nil || c.Polynomials == nil || c.Factors == nil || c.Binary == nil || c.Intervals == nil || c.Allocator == nil || c.Checkpoint == nil || c.Cooperation == nil {
		return nil, arena.ErrConfig
	}
	if e := c.Rationals.CheckOwnerReady(); e != nil {
		return nil, e
	}
	z := c.Rationals.Integers()
	if c.Polynomials.config.Integer != z || c.Factors.manager != c.Polynomials || c.Binary.Integers() != z || c.Intervals.BorrowBounds() != c.Binary {
		return nil, arena.ErrConfig
	}
	if e := c.Polynomials.check(); e != nil {
		return nil, e
	}
	if e := c.Factors.check(); e != nil {
		return nil, e
	}
	if e := c.Binary.Open(); e != nil {
		return nil, e
	}
	if c.Allocator.RuntimeOwner() != z.RuntimeOwner() {
		return nil, arena.ErrConfig
	}
	if _, e := c.Allocator.NumFreeObjects(); e != nil {
		return nil, e
	}
	return allocateAlgebraicRootCore(c)
}
func allocateAlgebraicRootCore(c AlgebraicRootConfig) (*AlgebraicRootCore, error) {
	n := uint64(reflect.TypeOf(AlgebraicRootCore{}).Size())
	if e := c.Rationals.Integers().Budget().Reserve(n); e != nil {
		return nil, e
	}
	var id uint64
	for {
		old := algebraicOrigins.Load()
		if old == math.MaxUint64 {
			c.Rationals.Integers().Budget().Release(n)
			return nil, arena.ErrSlotLimit
		}
		if algebraicOrigins.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	m := &AlgebraicRootCore{config: c, origin: id, header: n, zero: unumeral.NewRational(), minMagnitude: -16, factor: true, factorParams: UPolynomialFactorParams{MaxPrime: 31, PrimeTrials: 1, MaxSearchSize: 5000}}
	m.self = m
	return m, nil
}
func (m *AlgebraicRootCore) check() error {
	if m != nil && m.self == m && m.comparisonTraceRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.cellAllocationRunning || m.cellPending.Block() != (umemory.Block{})) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.binary.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.resultant.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.arithmetic.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.bounds.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.inversion.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.negationExecuting() {
		return ErrAlgebraicPending
	}
	if m == nil || m.self != m {
		return ErrAlgebraicOwner
	}
	if m.closed || m.closing || m.manager != nil && !m.manager.completed {
		return arena.ErrClosed
	}
	return m.config.Rationals.CheckOwnerReady()
}
func (m *AlgebraicRootCore) location(a *Algebraic) (*algebraicLocation, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.locationCleanup(a)
}
func (m *AlgebraicRootCore) locationCleanup(a *Algebraic) (*algebraicLocation, error) {
	if m == nil || m.self != m {
		return nil, ErrAlgebraicOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if a == nil {
		return nil, ErrAlgebraicOwner
	}
	if a.slot == 0 {
		if a.owner != 0 || a.generation != 0 {
			return nil, ErrAlgebraicOwner
		}
		return nil, nil
	}
	if a.owner != m.origin || uint64(a.slot) > uint64(len(m.locations)) {
		return nil, ErrAlgebraicOwner
	}
	l := &m.locations[a.slot-1]
	if !l.live || l.generation != a.generation {
		return nil, ErrAlgebraicOwner
	}
	return l, nil
}
func (m *AlgebraicRootCore) allocate(kind uint8) (Algebraic, error) {
	if e := m.check(); e != nil {
		return Algebraic{}, e
	}
	i := len(m.locations)
	for j := range m.locations {
		if !m.locations[j].live && m.locations[j].generation != math.MaxUint32 {
			i = j
			break
		}
	}
	if i == len(m.locations) {
		if uint64(len(m.locations)) >= math.MaxUint32 || len(m.locations) == int(^uint(0)>>1) {
			return Algebraic{}, arena.ErrSlotLimit
		}
		m.locations = append(m.locations, algebraicLocation{})
	}
	l := &m.locations[i]
	l.generation++
	l.kind = kind
	l.stage = 0
	l.live = true
	l.initialized = false
	l.coefficientInitialized = 0
	l.coefficientCursor = 0
	l.sourceGuard = false
	l.published = false
	a := Algebraic{m.origin, uint32(i + 1), l.generation}
	n := uint64(24)
	if kind == 1 {
		n = 56
	}
	m.cellAllocationRunning = true
	defer func() { m.cellAllocationRunning = false }()
	raw, e := m.config.Allocator.AllocateWithOutcome(n, &m.cellAllocation)
	l.raw = raw
	if e != nil {
		if raw.LogicalAddress() == 0 {
			l.live = false
		}
		return a, e
	}
	r := algebraicRecord{value: unumeral.NewRational()}
	if e = m.store(l, &r); e != nil {
		return a, e
	}
	l.initialized = true
	return a, nil
}
func algebraicIntWords(a unumeral.Int) [3]uint32 {
	return a.NativeWords()
}
func algebraicWordsInt(w []uint32) unumeral.Int {
	return unumeral.IntFromNativeWords([3]uint32{uint32(int32(w[0])), uint32(uint64(w[1]) | uint64(w[2])<<32), uint32(uint64(uint64(w[1])|uint64(w[2])<<32) >> 32)})
}
func (m *AlgebraicRootCore) store(l *algebraicLocation, r *algebraicRecord) error {
	var w [14]uint32
	n := 6
	if l.kind == 0 {
		a, b := algebraicIntWords((*r.value.BorrowNumerator())), algebraicIntWords((*r.value.BorrowDenominator()))
		copy(w[:3], a[:])
		copy(w[3:6], b[:])
	} else {
		n = 14
		w[0] = uint32(r.poly.origin)
		w[1] = uint32(r.poly.origin >> 32)
		w[2] = r.poly.slot
		w[3] = r.poly.generation
		w[4] = r.flags
		w[5] = r.size
		a, b := algebraicIntWords((*(*r.interval.BorrowLower()).BorrowNumerator())), algebraicIntWords((*(*r.interval.BorrowUpper()).BorrowNumerator()))
		copy(w[6:9], a[:])
		w[9] = (*r.interval.BorrowLower()).K()
		copy(w[10:13], b[:])
		w[13] = (*r.interval.BorrowUpper()).K()
	}
	for i := 0; i < n; i++ {
		if e := m.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w[i]); e != nil {
			return e
		}
	}
	return nil
}
func (m *AlgebraicRootCore) load(l *algebraicLocation) (algebraicRecord, error) {
	r := algebraicRecord{value: unumeral.NewRational()}
	if !l.initialized {
		return r, umemory.ErrRuntimeUninitialized
	}
	var w [14]uint32
	n := 6
	if l.kind == 1 {
		n = 14
	}
	for i := 0; i < n; i++ {
		v, e := m.config.Allocator.ReadUint32(l.raw, uint64(i)*4)
		if e != nil {
			return r, e
		}
		w[i] = v
	}
	if l.kind == 0 {
		r.value = unumeral.RationalFromBorrowedNumerals(algebraicWordsInt(w[:3]), algebraicWordsInt(w[3:6]))
	} else {
		r.poly = UPolynomial{uint64(w[0]) | uint64(w[1])<<32, w[2], w[3]}
		r.flags = w[4]
		r.size = w[5]
		r.interval = unumeral.BinaryIntervalFromBorrowedBounds(unumeral.BinaryRationalFromBorrowedNumeral(algebraicWordsInt(w[6:9]), w[9]), unumeral.BinaryRationalFromBorrowedNumeral(algebraicWordsInt(w[10:13]), w[13]))
	}
	return r, nil
}
func (m *AlgebraicRootCore) IsZero(a *Algebraic) (bool, error) {
	_, e := m.location(a)
	return a != nil && a.slot == 0, e
}
func (m *AlgebraicRootCore) Del(a *Algebraic) error {
	if m.negationProtectsCaller(a) {
		return ErrAlgebraicPending
	}
	l, e := m.location(a)
	if e != nil || l == nil {
		return e
	}
	return m.destroyCell(a, l, false)
}

// Identity-only private destruction remains callable after owning Close stops
// ordinary operations; it retains source Del guards and their completed cursor.
func (m *AlgebraicRootCore) delClosing(a *Algebraic) error {
	if m.negationProtectsCaller(a) {
		return ErrAlgebraicPending
	}
	l, e := m.locationCleanup(a)
	if e != nil || l == nil {
		return e
	}
	return m.destroyCell(a, l, false)
}

// The cursor is cold lifetime metadata for actual completed source actions.
// Owning abort skips only source-unconstructed slots; source Del preserves the
// endangered read and its guard history, without rebuilding an array.
func (m *AlgebraicRootCore) destroyCell(a *Algebraic, l *algebraicLocation, owning bool) error {
	if l.initialized {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		z := m.config.Rationals.Integers()
		if l.kind == 0 {
			for l.stage < 2 {
				n := &(*r.value.BorrowNumerator())
				if l.stage == 1 {
					n = &(*r.value.BorrowDenominator())
				}
				e = z.Del(n)
				if e = m.save(l, &r, e); e != nil {
					return e
				}
				l.stage++
			}
		} else {
			if l.stage == 0 {
				if e = m.destroyPolynomial(l, &r, owning); e != nil {
					return e
				}
				l.stage = 1
			}
			for l.stage < 3 {
				n := &(*(*r.interval.BorrowLower()).BorrowNumerator())
				if l.stage == 2 {
					n = &(*(*r.interval.BorrowUpper()).BorrowNumerator())
				}
				e = z.Del(n)
				if e = m.save(l, &r, e); e != nil {
					return e
				}
				l.stage++
			}
		}
	}
	n := uint64(24)
	if l.kind == 1 {
		n = 56
	}
	if l.raw.LogicalAddress() != 0 {
		progress, e := m.config.Allocator.DeallocateTracked(n, l.raw)
		if progress.PhysicalReleased || progress.PooledLinked {
			l.raw = umemory.ObjectRef{}
			l.initialized = false
		}
		if e != nil {
			return e
		}
	}
	l.live = false
	if l.published {
		m.callers--
		l.published = false
	}
	*a = Algebraic{}
	return nil
}
func (m *AlgebraicRootCore) deleteLocation(a *Algebraic, l *algebraicLocation) error {
	return m.destroyCell(a, l, true)
}
func (m *AlgebraicRootCore) destroyPolynomial(l *algebraicLocation, r *algebraicRecord, owning bool) error {
	p := m.config.Polynomials
	n := r.size
	if owning {
		n = l.coefficientInitialized
	}
	for l.coefficientCursor < n {
		if r.poly.slot == 0 || l.coefficientCursor >= l.coefficientInitialized {
			l.sourceGuard = true
			return arena.ErrBounds
		}
		v, e := m.sourcePolynomial(r)
		if e != nil {
			return e
		}
		a, e := p.read(v, l.coefficientCursor)
		if e != nil {
			l.sourceGuard = true
			return e
		}
		if e = p.config.Integer.Del(a); e != nil {
			return e
		}
		l.coefficientCursor++
	}
	if r.poly.slot != 0 {
		d, e := p.descriptor(r.poly)
		if e != nil {
			return e
		}
		if owning {
			values := p.values(&d.state)
			for i := l.coefficientCursor; uint64(i) < uint64(len(values)); i++ {
				if e = p.config.Integer.Del(&values[i]); e != nil {
					return e
				}
			}
		}
		p.freeBacking(&d.state)
		d.live = false
		if d.generation != math.MaxUint32 {
			d.generation++
		}
		p.live--
		r.poly = UPolynomial{}
	}
	r.size = 0
	l.coefficientCursor = 0
	l.coefficientInitialized = 0
	return m.store(l, r)
}
func (m *AlgebraicRootCore) beginQ() error {
	if m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.rationalCopy) {
		return ErrAlgebraicPending
	}
	if m.negation.active {
		return ErrAlgebraicPending
	}
	if m.localActive {
		return ErrAlgebraicPending
	}
	m.localQ = unumeral.NewRational()
	m.localActive = true
	m.localPolynomialCaller = m.polynomialCaller
	m.localSelection = nil
	if m.selection.running {
		m.localSelection = &m.selection
	}
	m.localIntegerSet = nil
	m.localIntegerAdd = nil
	if m.integerSet.running {
		m.localIntegerSet = &m.integerSet
	}
	if m.integerAdd.running {
		m.localIntegerAdd = &m.integerAdd
	}
	m.localOriginal = nil
	m.localCleanup = nil
	return nil
}
func (m *AlgebraicRootCore) finishQ(cause error) error {
	if !m.localActive {
		return cause
	}
	m.localOriginal = cause
	e := m.config.Rationals.Del(&m.localQ)
	if e != nil {
		m.localCleanup = e
		if cause != nil {
			return cause
		}
		return e
	}
	m.localActive = false
	m.localOriginal = nil
	m.localCleanup = nil
	return cause
}
func (m *AlgebraicRootCore) publish(a *Algebraic, next Algebraic) {
	*a = next
	l := &m.locations[next.slot-1]
	if !l.published {
		l.published = true
		m.callers++
	}
	if m.pending == next {
		m.pending = Algebraic{}
		m.constructionStage = 0
	}
}
func (m *AlgebraicRootCore) save(l *algebraicLocation, r *algebraicRecord, cause error) error {
	e := m.store(l, r)
	if e != nil {
		return e
	}
	return cause
}

// Const source overload makes the scoped copy BEFORE mutable setter actions.
func (m *AlgebraicRootCore) SetRational(a *Algebraic, q *unumeral.Rational) (err error) {
	if _, err = m.location(a); err != nil {
		return err
	}
	if err = m.beginQ(); err != nil {
		return err
	}
	defer func() { err = m.finishQ(err) }()
	if err = m.config.Rationals.CopyWithAllocationReceipt(&m.localQ, q, &m.rationalCopy); err != nil {
		return err
	}
	return m.SetTake(a, &m.localQ)
}
func (m *AlgebraicRootCore) SetTake(a *Algebraic, q *unumeral.Rational) error {
	if m != nil && m.self == m && m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.rationalCopy) {
		return ErrAlgebraicPending
	}
	l, e := m.location(a)
	if e != nil {
		return e
	}
	zero, e := m.config.Rationals.IsZero(q)
	if e != nil {
		return e
	}
	if zero {
		return m.Del(a)
	}
	if l != nil && l.kind == 1 {
		if e = m.Del(a); e != nil {
			return e
		}
		l = nil
	}
	if l != nil {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		e = m.config.Rationals.CopyWithAllocationReceipt(&r.value, q, &m.rationalCopy)
		return m.save(l, &r, e)
	}
	if m.pending.slot != 0 {
		return ErrAlgebraicPending
	}
	next, e := m.allocate(0)
	if next.slot != 0 && m.locations[next.slot-1].live {
		m.pending = next
	}
	if e != nil {
		return e
	}
	l = &m.locations[next.slot-1]
	r, e := m.load(l)
	if e != nil {
		return e
	}
	e = m.config.Rationals.Swap(&r.value, q)
	if e = m.save(l, &r, e); e != nil {
		return e
	}
	m.publish(a, next)
	return nil
}
func (m *AlgebraicRootCore) copyPolynomial(l *algebraicLocation, r *algebraicRecord, v UPolynomialView) error {
	if m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.coefficientCopy) {
		return ErrAlgebraicPending
	}
	r.size = v.count
	l.coefficientInitialized = 0
	l.coefficientCursor = 0
	if e := m.store(l, r); e != nil {
		return e
	}
	p := m.config.Polynomials
	q, e := p.NewPolynomial()
	if e != nil {
		return e
	}
	r.poly = q
	if e = m.store(l, r); e != nil {
		return e
	}
	d, e := p.descriptor(q)
	if e != nil {
		return e
	}
	if e = p.reserve(&d.state, v.count); e != nil {
		return e
	}
	// Typed backing has zeroed physical spare capacity; only this explicit
	// prefix denotes source placement construction, then ascending numeral copy.
	for i := uint32(0); i < v.count; i++ {
		l.coefficientInitialized = i + 1
		a, e := p.read(v, i)
		if e != nil {
			return e
		}
		if e = m.config.Rationals.Integers().CopyFromOwnerWithReceipt(&p.values(&d.state)[i], m.config.Rationals.Integers(), a, &m.coefficientCopy); e != nil {
			return e
		}
	}
	return nil
}
func (m *AlgebraicRootCore) deletePolynomial(l *algebraicLocation, r *algebraicRecord) error {
	return m.destroyPolynomial(l, r, false)
}
func (m *AlgebraicRootCore) updateSign(l *algebraicLocation, r *algebraicRecord) error {
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return e
	}
	s, e := m.config.Polynomials.EvalSignBinary(v, &(*r.interval.BorrowLower()))
	if e != nil {
		return e
	}
	if polynomialSourceDebug {
		if s == 0 {
			return ErrUPolynomialAssertion
		}
		u, e := m.config.Polynomials.EvalSignBinary(v, &(*r.interval.BorrowUpper()))
		if e != nil {
			return e
		}
		if u != -s {
			return ErrUPolynomialAssertion
		}
	}
	r.flags &^= 2
	if s < 0 {
		r.flags |= 2
	}
	return m.store(l, r)
}
func (m *AlgebraicRootCore) normalizeCoefficients(l *algebraicLocation, r *algebraicRecord) error {
	v, e := m.sourcePolynomial(r)
	if e != nil {
		return e
	}
	if e = m.config.Polynomials.NormalizeView(v); e != nil {
		return e
	}
	if r.size == 0 {
		return arena.ErrBounds
	}
	lead, e := m.config.Polynomials.read(v, r.size-1)
	if e != nil {
		return e
	}
	negative, e := m.config.Rationals.Integers().IsNeg(lead)
	if e != nil {
		return e
	}
	if negative {
		if e = m.config.Polynomials.NegViewInPlace(v); e != nil {
			return e
		}
		r.flags ^= 2
	}
	return m.store(l, r)
}
func (m *AlgebraicRootCore) SetRoot(a *Algebraic, p UPolynomialView, lower, upper *unumeral.BinaryRational, minimal bool) (err error) {
	l, err := m.location(a)
	if err != nil {
		return err
	}
	if polynomialSourceDebug && p.count <= 1 {
		return ErrUPolynomialAssertion
	}
	if p.count < 2 {
		return arena.ErrBounds
	}
	if p.count == 2 {
		if err = m.beginQ(); err != nil {
			return err
		}
		defer func() { err = m.finishQ(err) }()
		x, e := m.config.Polynomials.read(p, 0)
		if e != nil {
			return e
		}
		y, e := m.config.Polynomials.read(p, 1)
		if e != nil {
			return e
		}
		if err = m.config.Rationals.SetParts(&m.localQ, x, y); err != nil {
			return err
		}
		if err = m.config.Rationals.Neg(&m.localQ); err != nil {
			return err
		}
		return m.SetTake(a, &m.localQ)
	}
	existing := l != nil && l.kind == 1
	if !existing {
		if err = m.Del(a); err != nil {
			return err
		}
		if m.pending.slot != 0 {
			return ErrAlgebraicPending
		}
		next, e := m.allocate(1)
		if next.slot != 0 && m.locations[next.slot-1].live {
			m.pending = next
		}
		if e != nil {
			return e
		}
		l = &m.locations[next.slot-1]
	}
	r, err := m.load(l)
	if err != nil {
		return err
	}
	if existing {
		if err = m.deletePolynomial(l, &r); err != nil {
			return err
		}
	}
	if err = m.copyPolynomial(l, &r, p); err != nil {
		return err
	}
	if err = m.config.Intervals.SetBounds(&r.interval, lower, upper); err != nil {
		return m.save(l, &r, err)
	}
	if err = m.store(l, &r); err != nil {
		return err
	}
	if existing {
		r.flags &= 2
		if minimal {
			r.flags |= 5
		}
		if err = m.store(l, &r); err != nil {
			return err
		}
	}
	if err = m.updateSign(l, &r); err != nil {
		return err
	}
	if !existing {
		r.flags &= 2
		if minimal {
			r.flags |= 5
		}
		if err = m.store(l, &r); err != nil {
			return err
		}
	}
	if err = m.normalizeCoefficients(l, &r); err != nil {
		return err
	}
	if !existing {
		m.publish(a, m.pending)
	}
	if polynomialSourceDebug {
		v, e := m.config.Polynomials.wholeView(r.poly)
		if e != nil {
			return e
		}
		s, e := m.config.Polynomials.EvalSignBinary(v, &(*r.interval.BorrowLower()))
		if e != nil {
			return e
		}
		want := 1
		if r.flags&2 != 0 {
			want = -1
		}
		if s != want {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (m *AlgebraicRootCore) Copy(dst, src *Algebraic) error {
	if m != nil && m.self == m && m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.coefficientCopy) {
		return ErrAlgebraicPending
	}
	sl, e := m.location(src)
	if e != nil {
		return e
	}
	dl, e := m.location(dst)
	if e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if sl == nil {
		return m.Del(dst)
	}
	r, e := m.load(sl)
	if e != nil {
		return e
	}
	if sl.kind == 0 {
		return m.SetRational(dst, &r.value)
	}
	if dl == nil || dl.kind == 0 {
		if e = m.Del(dst); e != nil {
			return e
		}
		next, e := m.allocate(1)
		if e != nil {
			if next.slot != 0 && m.locations[next.slot-1].live {
				m.pending = next
			}
			return e
		}
		m.publish(dst, next)
		dl = &m.locations[dst.slot-1]
	} else {
		old, e := m.load(dl)
		if e != nil {
			return e
		}
		if e = m.deletePolynomial(dl, &old); e != nil {
			return e
		}
		e = m.config.Intervals.Del(&old.interval)
		if e = m.save(dl, &old, e); e != nil {
			return e
		}
	}
	out, e := m.load(dl)
	if e != nil {
		return e
	}
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return e
	}
	if e = m.copyPolynomial(dl, &out, v); e != nil {
		return e
	}
	e = m.config.Intervals.Copy(&out.interval, &r.interval)
	if e = m.save(dl, &out, e); e != nil {
		return e
	}
	out.flags = r.flags
	return m.store(dl, &out)
}
func (m *AlgebraicRootCore) CleanupRetained() error {
	if m != nil && m.self == m && m.manager != nil && (m.manager.constructing || m.manager.finishing) {
		return ErrAlgebraicPending
	}
	return m.cleanupRetainedCore()
}
func (m *AlgebraicRootCore) cleanupRetainedCore() error {
	if m != nil && m.self == m && m.comparisonTraceRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.cellAllocationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.rootConstruction.running || m.rootConstruction.finishing) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.isolationSigns.running || m.isolationSigns.finishing) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.multivariateIsolationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.polynomialEvaluationRunning != 0 {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.integerSet.running || m.integerAdd.running) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.isolationConversion.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.filterRootsRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signEvaluationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signResultant.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signInterval.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signSubstitution.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.rationalSign.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.rootDisplay.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.selection.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.basicRootScope.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.division.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.binary.running || m.binary.cleaning) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.resultant.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.arithmetic.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.bounds.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.inversion.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.negationExecuting() {
		return ErrAlgebraicPending
	}
	if m == nil || m.self != m || m.closed {
		return ErrAlgebraicOwner
	}
	if e := m.cleanupCellAllocation(); e != nil {
		return e
	}
	if e := m.representationNeg.roots.Cleanup(m.config.Rationals.Integers()); e != nil {
		return e
	}
	if p := m.config.Polynomials; p != nil {
		if e := p.modular.CleanupPNormalizeInvocation(&m.representationNormalize, true); e != nil {
			return e
		}
	}
	if e := m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&m.representationCopy); e != nil {
		return e
	}
	if e := m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&m.rationalCopy); e != nil {
		return e
	}
	if e := m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&m.coefficientCopy); e != nil {
		return e
	}
	if e := m.cleanupResultantPolynomial(); e != nil {
		return e
	}
	if e := m.cleanupBinary(); e != nil {
		return e
	}
	if e := m.cleanupRationalArithmetic(); e != nil {
		return e
	}
	if e := m.cleanupInverse(); e != nil {
		return e
	}
	if e := m.cleanupBasicRoot(); e != nil {
		return e
	}
	if e := m.cleanupRootDisplay(); e != nil {
		return e
	}
	if e := m.cleanupRationalSign(); e != nil {
		return e
	}
	if e := m.cleanupSignResultant(); e != nil {
		return e
	}
	if e := m.cleanupSignInterval(); e != nil {
		return e
	}
	if e := m.cleanupSignSubstitution(); e != nil {
		return e
	}
	if e := m.cleanupIsolationConversion(); e != nil {
		return e
	}
	if e := m.cleanupDivision(); e != nil {
		return e
	}
	if e := m.cleanupBounds(); e != nil {
		return e
	}
	if e := m.CleanupNegation(); e != nil {
		return e
	}
	if m.localActive {
		if e := m.finishQ(nil); e != nil {
			return e
		}
	}
	if e := m.cleanupIntegerSet(); e != nil {
		return e
	}
	if e := m.cleanupIntegerAdd(true); e != nil {
		return e
	}
	if e := m.cleanupSelection(); e != nil {
		return e
	}
	if e := m.finishIsolationSigns(true); e != nil {
		return e
	}
	if m.pending.slot != 0 {
		l := &m.locations[m.pending.slot-1]
		if e := m.deleteLocation(&m.pending, l); e != nil {
			return e
		}
		m.constructionStage = 0
	}
	if e := m.finishPredicate(nil); e != nil {
		return e
	}
	if e := m.comparisonCleanup(); e != nil {
		return e
	}
	if e := m.cleanupRootConstruction(); e != nil {
		return e
	}
	return m.cleanupMultivariateIsolation()
}
func (m *AlgebraicRootCore) Close() error {
	if m != nil && m.self == m && m.manager != nil {
		return m.manager.Close()
	}
	return m.closeCoreMembers()
}
func (m *AlgebraicRootCore) closeCoreMembers() error {
	if m != nil && m.self == m && m.comparisonTraceRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.cellAllocationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.rootConstruction.running || m.rootConstruction.finishing) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.isolationSigns.running || m.isolationSigns.finishing) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.multivariateIsolationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.polynomialEvaluationBorrowers != 0 {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.integerSet.running || m.integerAdd.running) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.isolationConversion.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.filterRootsRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signEvaluationRunning {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signResultant.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signInterval.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.signSubstitution.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.rationalSign.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.rootDisplay.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.selection.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.basicRootScope.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.division.running {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && (m.binary.running || m.binary.cleaning) {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.resultant.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.arithmetic.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.bounds.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.inversion.executing {
		return ErrAlgebraicPending
	}
	if m != nil && m.self == m && m.negationExecuting() {
		return ErrAlgebraicPending
	}
	if m == nil || m.self != m {
		return ErrAlgebraicOwner
	}
	if m.closed {
		return nil
	}
	// Only named private selection elements are excluded from caller ownership.
	var privateCallers, privateVectors uint32
	if m.rootSelection != nil {
		privateVectors = 1
		for i := uint32(0); i < uint32(len(m.rootSelection.values)); i++ {
			if m.rootSelection.values[i].slot != 0 {
				privateCallers++
			}
		}
	}
	if m.callers != privateCallers || m.vectors != privateVectors {
		return ErrAlgebraicLive
	}
	if m.rootConstruction.active {
		if e := m.cleanupRetainedCore(); e != nil {
			return e
		}
	}
	if e := m.CleanupNegation(); e != nil {
		return e
	}
	m.closing = true
	// The later lexical comparison sequence is destroyed before older owners.
	if e := m.comparisonCleanup(); e != nil {
		return e
	}
	if m.rootSelection != nil {
		if e := m.rootSelection.Close(); e != nil {
			return e
		}
		m.rootSelection = nil
	}
	if e := m.cleanupRetainedCore(); e != nil {
		return e
	}
	// m_add_tmp is younger than the isolation vector members.
	if m.addTmp.slot != 0 {
		if e := m.config.Polynomials.DelPolynomial(&m.addTmp); e != nil {
			return e
		}
	}
	// Reverse declaration order of the selected actual source members.
	for _, v := range []**BinaryRationalVector{&m.uppers, &m.lowers, &m.roots} {
		if *v != nil {
			if e := (*v).Close(); e != nil {
				return e
			}
			*v = nil
		}
	}
	if m.factors != nil {
		if e := m.factors.Close(); e != nil {
			return e
		}
		m.factors = nil
	}
	if m.evalSignTmp.slot != 0 {
		if e := m.config.Polynomials.DelPolynomial(&m.evalSignTmp); e != nil {
			return e
		}
	}
	for _, p := range []*UPolynomial{&m.tmp3, &m.tmp2, &m.isolateTmp1} {
		if p.slot != 0 {
			if e := m.config.Polynomials.DelPolynomial(p); e != nil {
				return e
			}
		}
	}
	if m.manager == nil || m.manager.rationalTemporaryConstructed {
		if e := m.config.Rationals.Integers().Del(&m.isRationalTmp); e != nil {
			return e
		}
	}
	if m.manager != nil {
		return nil
	}
	if m.multivariate != nil {
		if e := m.multivariate.CloseChecked(); e != nil {
			return e
		}
		m.multivariate = nil
	}
	return m.releaseCoreStorage()
}
func (m *AlgebraicRootCore) releaseCoreStorage() error {
	b := m.config.Rationals.Budget()
	m.negationHistory = nil
	m.evalSignVariables.values = nil
	m.isolateRootsVariables.values = nil
	b.Release(uint64(len(m.multivariateIsolationFrames)) * uint64(reflect.TypeOf(algebraicMultivariateIsolationFrame{}).Size()))
	m.multivariateIsolationFrames = nil
	b.Release(m.header)
	m.locations = nil
	m.header = 0
	m.closed = true
	return nil
}

func numericRationalService_NewAlgebraicRootCore(s *unumeral.RationalService, c AlgebraicRootConfig) (out *AlgebraicRootCore, cause error) {
	cause = s.WithNumericBorrow(func(b *unumeral.RationalNumericBorrow) error {
		if e := b.Ready(); e != nil {
			return e
		}
		if c.Rationals != nil && c.Rationals != b.Rationals() {
			return ErrAlgebraicOwner
		}
		c.Rationals = b.Rationals()
		var e error
		out, e = NewAlgebraicRootCore(c)
		return e
	})
	return out, cause
}
func (m *AlgebraicRootCore) sourcePolynomial(r *algebraicRecord) (UPolynomialView, error) {
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return v, e
	}
	v.count = r.size
	return v, nil
}

// CleanupValue is the explicit owning abort of one caller-provided reachable
// value. It does not repair source metadata or discover roots from locations.
func (m *AlgebraicRootCore) CleanupValue(a *Algebraic) error {
	if m.negationProtectsCaller(a) {
		return ErrAlgebraicPending
	}
	l, e := m.location(a)
	if e != nil || l == nil {
		return e
	}
	return m.deleteLocation(a, l)
}
