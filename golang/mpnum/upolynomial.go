// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Source: src/math/polynomial/upolynomial.h; src/math/polynomial/upolynomial.cpp.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
	"sync/atomic"
)

var ErrUPolynomialOwner = errors.New("mpnum: invalid polynomial owner")
var ErrUPolynomialLive = errors.New("mpnum: live polynomial owners")
var ErrUPolynomialCanceled = errors.New("mpnum: polynomial resource canceled")
var ErrUPolynomialAssertion = errors.New("mpnum: polynomial source assertion")
var upolyOrigins atomic.Uint64

type UPolynomialConfig struct {
	Integer     *IntegerManager
	Checkpoint  func() bool
	Cooperation func() error
}

// The two callbacks borrow actual source resource/cooperation owners. They
// are required, even though operations without a checkpoint do not call them.
// Cold ownership and every actual typed backing are charged to Integer's budget.
type UPolynomialManager struct {
	oneHalfRoot                    upolyOneHalfRootScope
	oneHalfRootHistory             *upolyOneHalfRootHistory
	oneHalfRootHistoryBytes        uint64
	knuth                          upolyKnuthScope
	knuthHistory                   *upolyKnuthHistory
	knuthHistoryBytes              uint64
	rootUpperBound                 upolyRootUpperBoundScope
	rootUpperBoundHistory          *upolyRootUpperBoundHistory
	rootUpperBoundHistoryBytes     uint64
	integerEvaluation              upolyIntegerEvaluationScope
	integerEvaluationHistory       *upolyIntegerEvaluationHistory
	integerEvaluationHistoryBytes  uint64
	rationalEvaluation             upolyRationalEvaluationScope
	rationalEvaluationHistory      *upolyRationalEvaluationHistory
	rationalEvaluationHistoryBytes uint64
	binaryEvaluation               upolyBinaryEvaluationScope
	binaryEvaluationHistory        *upolyBinaryEvaluationHistory
	binaryEvaluationHistoryBytes   uint64
	algebraicNegCaller             *AlgebraicRootCore
	display                        upolyDisplayScope
	displaySequence                *UPolynomialSequence
	currentDisplay, displayFrames  *upolyDisplayScope
	displayNonce                   uint64

	factorFrontendCaller         *upolyFactorFrontendScope
	factorFrontendConstantCaller *UPolynomialFactorizationService
	primePreparationCaller       *upolyFactorPrimePreparationScope
	preparationCaller            *upolyFactorPreparationScope
	henselPairCaller             *upolyHenselPairScope
	henselCallerGuard            *upolyHenselCallerScope
	henselCallerInvocation       uint64
	callbackDepth                uint32
	mignotteCaller               *upolyMignotteScope
	trialCaller                  *upolyRecombinationTrialScope
	recombinationCaller          *upolyRecombinationStepScope
	primeTrialCaller             *upolyPrimeTrialScope
	squareFreeCaller             *upolyFactorSquareFreeScope
	self                         *UPolynomialManager
	origin                       uint64
	config                       UPolynomialConfig
	modular                      *IntegerModularManager
	// Actual source members: 16 core and five derived numeral vectors.
	basic, div1, div2, exactDiv, gcd1, gcd2, cra   upolyVectorState
	mgcd                                           [6]upolyVectorState
	sqf1, sqf2, pw                                 upolyVectorState
	db, dbab1, dbab2, tr, push                     upolyVectorState
	pages                                          []*upolyDescriptorPage
	store                                          *UPolynomialCoefficientStore
	bindings                                       []upolyStoreBinding
	live                                           uint32
	liveSequences                                  uint32
	liveFactors                                    uint32
	header                                         uint64
	closed, closing                                bool
	sourceFinalizing, sourceFinalized, sourceGuard bool
	sourceStage                                    uint8
	sourceIndex                                    uint32
	sourceCause                                    error
}

func NewUPolynomialManager(c UPolynomialConfig) (*UPolynomialManager, error) {
	return newUPolynomialManager(c, nil)
}
func newUPolynomialManager(c UPolynomialConfig, store *UPolynomialCoefficientStore) (*UPolynomialManager, error) {
	if c.Integer == nil || c.Checkpoint == nil || c.Cooperation == nil {
		return nil, arena.ErrConfig
	}
	if c.Integer.closed {
		return nil, arena.ErrClosed
	}
	n := uint64(reflect.TypeOf(UPolynomialManager{}).Size())
	if e := c.Integer.budget.Reserve(n); e != nil {
		return nil, e
	}
	z, e := NewIntegerModularManager(c.Integer)
	if e != nil {
		c.Integer.budget.Release(n)
		return nil, e
	}
	var id uint64
	for {
		old := upolyOrigins.Load()
		if old == math.MaxUint64 {
			z.Close()
			c.Integer.budget.Release(n)
			return nil, arena.ErrSlotLimit
		}
		if upolyOrigins.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	if store == nil {
		store, e = newUPolynomialCoefficientStore(c.Integer, id)
	} else {
		e = store.retain()
	}
	if e != nil {
		z.Close()
		c.Integer.budget.Release(n)
		return nil, e
	}
	m := &UPolynomialManager{origin: id, config: c, modular: z, header: n, store: store}
	m.self = m
	return m, nil
}
func (m *UPolynomialManager) check() error {
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.closed || m.closing || m.sourceFinalizing || m.config.Integer.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *UPolynomialManager) IntegerManager() *IntegerManager {
	if m == nil || m.self != m {
		return nil
	}
	return m.config.Integer
}
func (m *UPolynomialManager) checkpoint() error {
	m.callbackDepth++
	defer func() { m.callbackDepth-- }()
	if !m.config.Checkpoint() {
		return ErrUPolynomialCanceled
	}
	return m.config.Cooperation()
}
func (m *UPolynomialManager) descriptor(p UPolynomial) (*upolyDescriptor, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if p.origin != m.origin || p.slot == 0 {
		return nil, ErrUPolynomialOwner
	}
	i := p.slot - 1
	if uint64(i/32) >= uint64(len(m.pages)) {
		return nil, ErrUPolynomialOwner
	}
	d := &m.pages[i/32].rows[i%32]
	if !d.live || d.generation != p.generation {
		return nil, ErrUPolynomialOwner
	}
	return d, nil
}
func (m *UPolynomialManager) NewPolynomial() (UPolynomial, error) {
	if e := m.check(); e != nil {
		return UPolynomial{}, e
	}
	for i, page := range m.pages {
		for j := range page.rows {
			d := &page.rows[j]
			if !d.live && d.generation != math.MaxUint32 {
				if d.generation == 0 {
					d.generation = 1
				}
				d.live = true
				m.live++
				return UPolynomial{m.origin, uint32(i*32 + j + 1), d.generation}, nil
			}
		}
	}
	if uint64(len(m.pages)+1)*32 > math.MaxUint32 {
		return UPolynomial{}, arena.ErrSlotLimit
	}
	if len(m.pages) == cap(m.pages) {
		next, bytes, e := upolyDirectoryNext(uint64(cap(m.pages)), uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()), uint64(int(^uint(0)>>1)))
		if e != nil {
			return UPolynomial{}, e
		}
		n := int(next)
		if e := m.config.Integer.budget.Reserve(bytes); e != nil {
			return UPolynomial{}, e
		}
		a := make([]*upolyDescriptorPage, len(m.pages), n)
		copy(a, m.pages)
		m.config.Integer.budget.Release(uint64(cap(m.pages)) * uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()))
		m.pages = a
	}
	bytes := uint64(reflect.TypeOf(upolyDescriptorPage{}).Size())
	if e := m.config.Integer.budget.Reserve(bytes); e != nil {
		return UPolynomial{}, e
	}
	page := &upolyDescriptorPage{}
	m.pages = append(m.pages, page)
	page.rows[0] = upolyDescriptor{generation: 1, live: true}
	m.live++
	return UPolynomial{m.origin, uint32((len(m.pages)-1)*32 + 1), 1}, nil
}
func (m *UPolynomialManager) DelPolynomial(p *UPolynomial) error {
	if p == nil {
		return ErrUPolynomialOwner
	}
	d, e := m.descriptor(*p)
	if e != nil {
		return e
	}
	qh := &m.oneHalfRoot
	if qh.owner == m && qh.active && !qh.falseAcknowledged && qh.target.origin == m.origin && qh.target.backing == d.state.backing && qh.target.generation == d.state.generation && (qh.executing || qh.finishing || qh.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}
	qk := &m.knuth
	if qk.owner == m && qk.active && !qk.falseAcknowledged && qk.target.origin == m.origin && qk.target.backing == d.state.backing && qk.target.generation == d.state.generation && (qk.executing || qk.finishing || qk.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}
	qu := &m.rootUpperBound
	if qu.owner == m && qu.active && !qu.falseAcknowledged && qu.target.origin == m.origin && qu.target.backing == d.state.backing && qu.target.generation == d.state.generation && (qu.executing || qu.finishing || qu.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}

	qi := &m.integerEvaluation
	if qi.owner == m && qi.active && !qi.falseAcknowledged && qi.target.origin == m.origin && qi.target.backing == d.state.backing && qi.target.generation == d.state.generation && (qi.executing || qi.finishing || qi.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}

	qr := &m.rationalEvaluation
	if qr.owner == m && qr.active && !qr.falseAcknowledged && qr.target.origin == m.origin && qr.target.backing == d.state.backing && qr.target.generation == d.state.generation && (qr.executing || qr.finishing || qr.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}

	q := &m.binaryEvaluation
	if q.owner == m && q.active && !q.falseAcknowledged && q.target.origin == m.origin && q.target.backing == d.state.backing && q.target.generation == d.state.generation && (q.executing || q.finishing || q.sourceCause != nil) {
		return ErrUPolynomialCleanupPending
	}
	if e = m.reset(&d.state); e != nil {
		return e
	}
	m.freeBacking(&d.state)
	d.live = false
	if d.generation != math.MaxUint32 {
		d.generation++
	}
	m.live--
	*p = UPolynomial{}
	return nil
}
func (m *UPolynomialManager) Close() error { return m.closeWithHensel(nil) }
func (m *UPolynomialManager) closeWithHensel(r *upolyHenselPairInvocationReceipt) error {
	return m.closeWithHenselOwners(r, nil)
}
func (m *UPolynomialManager) closeWithHenselOwners(r *upolyHenselPairInvocationReceipt, c *upolyHenselCallerScope) error {
	return m.closeWithCallerOwners(r, c, nil)
}
func (m *UPolynomialManager) closeWithCallerOwners(r *upolyHenselPairInvocationReceipt, c *upolyHenselCallerScope, p *upolyFactorPrimePreparationInvocationReceipt) error {
	if m.oneHalfRootExecuting() || m.knuthExecuting() || m.rootUpperBoundExecuting() || m.integerEvaluationExecuting() || m.rationalEvaluationExecuting() || m.binaryEvaluationExecuting() {
		return ErrUPolynomialCleanupPending
	}
	if m != nil && m.self == m && m.algebraicNegCaller != nil && m.algebraicNegCaller.negationExecuting() {
		return ErrUPolynomialCleanupPending
	}
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.mignotteCallPending() && !m.henselDestructorAuthorized(r, true) && !m.henselCallerDestructorAuthorized(c, true) && !m.primePreparationDestructorAuthorized(p, true) {
		return ErrUPolynomialCleanupPending
	}
	if m.closed || m.config.Integer.closed {
		return arena.ErrClosed
	}
	if m.live != 0 || m.liveSequences != 0 || m.liveFactors != 0 {
		return ErrUPolynomialLive
	}
	if e := m.CleanupOneHalfRoot(); e != nil {
		return e
	}
	if e := m.CleanupKnuthBounds(); e != nil {
		return e
	}
	if e := m.CleanupRootUpperBound(); e != nil {
		return e
	}
	if e := m.CleanupIntegerEvaluation(); e != nil {
		return e
	}
	if e := m.CleanupRationalEvaluation(); e != nil {
		return e
	}
	if e := m.CleanupBinaryEvaluation(); e != nil {
		return e
	}
	if e := m.CleanupDisplay(); e != nil {
		return e
	}
	m.closing = true
	if m.sourceFinalizing && !m.sourceFinalized && m.sourceCause == nil {
		if e := m.sourceFinalizeWithCallerOwners(r, c, p); e != nil {
			return e
		}
	}
	// Owning cleanup acknowledges actual retained release cursors separately;
	// stopped literal source cause/stage/history are never repaired or advanced.
	// Derived destructor reset order, followed by implicit reverse members.
	derived := [5]*upolyVectorState{&m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push}
	for _, s := range derived {
		if e := m.reset(s); e != nil {
			return e
		}
	}
	for i := len(derived) - 1; i >= 0; i-- {
		m.freeBacking(derived[i])
	}
	core := [16]*upolyVectorState{&m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw}
	for _, s := range core {
		if e := m.reset(s); e != nil {
			return e
		}
	}
	for i := len(core) - 1; i >= 0; i-- {
		m.freeBacking(core[i])
	}
	if e := m.modular.Close(); e != nil {
		return e
	}
	b := m.config.Integer.budget
	b.Release(uint64(len(m.pages)) * uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()))
	b.Release(uint64(cap(m.pages)) * uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()))
	m.store.release()
	for _, binding := range m.bindings {
		binding.store.release()
	}
	b.Release(uint64(cap(m.bindings)) * uint64(reflect.TypeOf(upolyStoreBinding{}).Size()))
	m.bindings = nil
	for q := m.displayFrames; q != nil; {
		next := q.next
		b.Release(q.header)
		q.header = 0
		q.next = nil
		q = next
	}
	m.displayFrames = nil
	b.Release(m.oneHalfRootHistoryBytes)
	m.oneHalfRootHistoryBytes = 0
	m.oneHalfRootHistory = nil
	b.Release(m.knuthHistoryBytes)
	m.knuthHistoryBytes = 0
	m.knuthHistory = nil
	b.Release(m.rootUpperBoundHistoryBytes)
	m.rootUpperBoundHistoryBytes = 0
	m.rootUpperBoundHistory = nil
	b.Release(m.integerEvaluationHistoryBytes)
	m.integerEvaluationHistoryBytes = 0
	m.integerEvaluationHistory = nil
	b.Release(m.rationalEvaluationHistoryBytes)
	m.rationalEvaluationHistoryBytes = 0
	m.rationalEvaluationHistory = nil
	b.Release(m.binaryEvaluationHistoryBytes)
	m.binaryEvaluationHistoryBytes = 0
	m.binaryEvaluationHistory = nil
	b.Release(m.header)
	m.pages = nil

	m.closed = true
	return nil
}
