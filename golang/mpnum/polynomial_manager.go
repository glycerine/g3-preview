// Source: src/math/polynomial/polynomial.cpp:2308-2370,2481-2633. Licensed under the MIT license.
// Selected multivariate cell/SOM prerequisite owner, not manager::imp.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilid"
	"math"
	"reflect"
	"sync/atomic"
)

var ErrPolynomialOwner = errors.New("mpnum: invalid multivariate polynomial owner")
var ErrPolynomialAssertion = errors.New("mpnum: multivariate polynomial source assertion")
var ErrPolynomialPartial = errors.New("mpnum: unconstructed polynomial field")
var ErrPolynomialPending = errors.New("mpnum: retained polynomial construction")
var ErrPolynomialLive = errors.New("mpnum: live caller polynomials")
var polynomialOrigins atomic.Uint64

type MultivariatePolynomialConfig struct {
	Integer     *IntegerManager
	Rationals   *RationalManager
	Monomials   *MonomialManager
	Checkpoint  func() bool
	Cooperation func() error
}
type polynomialCreation struct {
	appendParent       *polynomialCheapAppendScope
	factorsParent      *PolynomialFactors
	conversionParent   *polynomialGlobalConversionScope
	toPolynomialParent *polynomialToPolynomialScope
	compositionParent  *polynomialCompositionScope
	pscOwner           uint64
	substitutionOwned  bool
	translationOwned   bool
	generation         uint64
	squareRootParent   *polynomialSquareRootScope
	derivativeParent   *polynomialDerivativeScope
	raw                memory.ObjectRef
	value              Polynomial
	requested, id      uint32
	idConsumed         bool
	buffer             *polynomialSOM
}
type MultivariatePolynomialCore struct {
	collectingVariables                                                                        bool
	somNormalizationOwner                                                                      polynomialSOMNormalizeInvocationReceipt
	normalizationOwner                                                                         polynomialNormalizeInvocationReceipt
	constantCoefficientInvocation                                                              polynomialConstantCoefficientScope
	factorsOwner                                                                               *PolynomialFactors
	powerGeneration                                                                            uint64
	display                                                                                    polynomialDisplayState
	conversionOwner                                                                            *polynomialGlobalConversionScope
	toPolynomialOwner                                                                          *polynomialToPolynomialScope
	dividesLocal                                                                               polynomialDividesScope
	contentLocal                                                                               polynomialContentScope
	pscReceipts                                                                                polynomialPSCReceipts
	generatedPSC                                                                               uint64
	pscClassic                                                                                 polynomialPSCClassicScope
	pscLocal                                                                                   polynomialPSCScope
	pscNext                                                                                    polynomialPSCNextScope
	lazardLocal                                                                                polynomialLazardScope
	variableLocal                                                                              polynomialVariableScope
	pscBuffer                                                                                  polynomialPSCRefBuffer
	translateBuffers                                                                           PolynomialSOMVector
	translation                                                                                polynomialTranslationScope
	substitution                                                                               polynomialSubstitutionState
	tmp1                                                                                       monomialTemp
	tmp1SourceConsumed                                                                         bool
	pseudoDivision                                                                             polynomialPseudoDivisionScope
	exactDivision                                                                              polynomialExactDivisionScope
	queryPositions                                                                             MonomialPositions
	powerLocal                                                                                 polynomialPowerScope
	newtonLocal                                                                                *PolynomialNewtonInterpolator
	generatedCalleeGeneration                                                                  uint64
	generatedCalleePending                                                                     bool
	directoryInitialized                                                                       bool
	linearGenerated                                                                            []bool
	linearGeneratedCleanup                                                                     uint32
	generatedMonomial                                                                          Monomial
	generatedActive                                                                            bool
	generatedBuffer                                                                            *polynomialSOM
	generatedIndex                                                                             uint32
	constructorUnwindStage                                                                     uint8
	constructorModularIndex                                                                    uint8
	constructorUnwinding, idsSourceDestroyed, directorySourceDestroyed, modularSourceDestroyed bool
	sourceStage                                                                                uint8
	sourceStarted                                                                              bool
	sourceCause                                                                                error
	sourceDecDone, sourceDeleteNeeded                                                          [2]bool
	polynomialNegScope
	factoryInvocation                                            polynomialFactoryScope
	arithmeticInvocation                                         uint64
	coefficientInvocation                                        polynomialCoefficientScope
	scalarDivision                                               polynomialScalarDivScope
	deletionHead                                                 *PolynomialDeletionHandler
	callbackDepth                                                uint32
	callbackValue                                                Polynomial
	linearConstant                                               Int
	linearConstantActive                                         bool
	overloadResult                                               Polynomial
	constantScope                                                polynomialConstantScope
	singletonMonomial                                            Monomial
	singletonReference                                           bool
	singletonGeneration, constantGeneration                      uint64
	self                                                         *MultivariatePolynomialCore
	config                                                       MultivariatePolynomialConfig
	integer                                                      *IntegerManager
	modular                                                      *IntegerModularManager
	monomials                                                    *MonomialManager
	memory                                                       *memory.Runtime
	ids                                                          utilid.Generator
	polynomials                                                  polynomialVector
	locations                                                    []polynomialLocation
	squareRootOwner                                              *polynomialSquareRootScope
	derivativeOwner                                              *polynomialDerivativeScope
	compositionOwner                                             *polynomialCompositionScope
	creationGeneration                                           uint64
	pending                                                      polynomialCreation
	som, som2, cheap, cheap2                                     polynomialSOM
	zero, unit                                                   Polynomial
	zeroNumeral                                                  Int
	minusOne, constantCopy                                       Int
	rationalTerms, linearTerms                                   polynomialVector
	linearMonomials                                              polynomialVector
	origin, header                                               uint64
	live                                                         uint32
	closeStage                                                   uint8
	completed, closing, closed, monomialRetained, idsInitialized bool
	originalCause, cleanupCause                                  error
}

func NewMultivariatePolynomialCore(c MultivariatePolynomialConfig) (*MultivariatePolynomialCore, error) {
	m, e := preparePolynomialCore(c, true)
	if e != nil {
		return m, e
	}
	fail := func(e error) (*MultivariatePolynomialCore, error) {
		m.originalCause = e
		m.cleanupCause = m.unwindConstructorMembers()
		return m, m
	}
	if e = m.constructPolynomialCoreDefaults(); e != nil {
		return fail(e)
	}
	if e = m.monomials.tempConstruct(&m.tmp1); e != nil {
		return fail(e)
	}
	if e = m.initializePolynomialCoreBody(); e != nil {
		return fail(e)
	}
	return m, nil
}

// Preparation is a concrete member stage, not a completed polynomial::imp.
// The full source hierarchy must insert its distinct UP member before defaults.
func preparePolynomialCore(c MultivariatePolynomialConfig, requireMonomials bool) (*MultivariatePolynomialCore, error) {
	if c.Integer == nil || (requireMonomials && c.Monomials == nil) || c.Checkpoint == nil || c.Cooperation == nil {
		return nil, arena.ErrConfig
	}
	if e := polynomialIntegerReady(c.Integer); e != nil {
		return nil, e
	}
	if c.Monomials != nil {
		if e := c.Monomials.check(); e != nil {
			return nil, e
		}
		if c.Monomials.memory != c.Integer.memory || c.Monomials.config.Budget != c.Integer.budget {
			return nil, arena.ErrConfig
		}
	}
	if c.Rationals != nil {
		if e := c.Rationals.checkOwnerReady(); e != nil {
			return nil, e
		}
		if c.Rationals.z != c.Integer {
			return nil, arena.ErrConfig
		}
	}
	h := uint64(reflect.TypeOf(MultivariatePolynomialCore{}).Size())
	if e := c.Integer.budget.Reserve(h); e != nil {
		return nil, e
	}
	m := &MultivariatePolynomialCore{config: c, integer: c.Integer, monomials: c.Monomials, memory: c.Integer.memory, header: h}
	m.self = m
	m.initSubstitutionMembers()
	m.initTranslationMember()
	for {
		old := polynomialOrigins.Load()
		if old == math.MaxUint64 {
			c.Integer.budget.Release(h)
			return nil, arena.ErrSlotLimit
		}
		if polynomialOrigins.CompareAndSwap(old, old+1) {
			m.origin = old + 1
			break
		}
	}
	var e error
	m.modular, e = NewIntegerModularManager(c.Integer)
	if e != nil {
		m.originalCause = e
		m.cleanupCause = m.unwindConstructorMembers()
		return m, m
	}
	return m, nil
}
func (m *MultivariatePolynomialCore) constructPolynomialCoreDefaults() error {
	m.polynomials.width = 16
	m.directoryInitialized = true
	if e := m.ids.Init(0, m.memory); e != nil {
		return e
	}
	m.idsInitialized = true
	return nil
}
func (m *MultivariatePolynomialCore) initializePolynomialCoreBody() error {
	var e error
	if e = m.monomials.IncManagerRef(); e != nil {
		return e
	}
	m.monomialRetained = true
	m.polynomials.width = 16
	m.rationalTerms.width = 12
	m.linearTerms.width = 12
	m.linearMonomials.width = 16
	m.som.init(m, false)
	m.som2.init(m, false)
	m.cheap.init(m, true)
	m.cheap2.init(m, true)
	m.zero, e = m.mkCell(0, nil, nil, nil, nil)
	if e != nil {
		return e
	}
	if e = m.modular.SetInt64(&m.zeroNumeral, 0); e != nil {
		return e
	}
	if e = m.incRefCore(m.zero); e != nil {
		return e
	}
	one := NewInt(1)
	m.unit, e = m.mkConstCore(&one)
	if e != nil {
		return e
	}
	if e = m.incRefCore(m.unit); e != nil {
		return e
	}
	m.completed = true
	return nil
}
func (m *MultivariatePolynomialCore) Error() string {
	if m.originalCause != nil {
		return m.originalCause.Error()
	}
	if m.cleanupCause != nil {
		return m.cleanupCause.Error()
	}
	return ErrPolynomialPending.Error()
}
func (m *MultivariatePolynomialCore) Unwrap() error { return m.originalCause }
func (m *MultivariatePolynomialCore) check() error {
	if m != nil && m.self == m && m.collectingVariables {
		return ErrPolynomialPending
	}
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if m.closed || m.closing || m.sourceStarted || m.originalCause != nil {
		return arena.ErrClosed
	}
	if e := polynomialIntegerReady(m.integer); e != nil {
		return e
	}
	return m.monomials.check()
}
func (m *MultivariatePolynomialCore) incRefCore(p Polynomial) error {
	r, e := m.readWord(p, 0)
	if e != nil {
		return e
	}
	return m.writeWord(p, 0, r+1)
}
func (m *MultivariatePolynomialCore) MkCoreTake(as []Int, ms []Monomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if uint64(len(as)) > math.MaxUint32 {
		return Polynomial{}, arena.ErrBounds
	}
	return m.mkCell(uint32(len(as)), as, ms, nil, nil)
}
func (m *MultivariatePolynomialCore) mkConstCore(a *Int) (Polynomial, error) {
	return m.mkConstCoreWithArithmeticReceipt(a, nil)
}
func (m *MultivariatePolynomialCore) mkConstCoreWithArithmeticReceipt(a *Int, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if m.singletonReference {
		return Polynomial{}, ErrPolynomialPending
	}
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	if e = m.monomials.IncRef(u); e != nil {
		return Polynomial{}, e
	}
	m.singletonMonomial = u
	m.singletonReference = true
	m.singletonGeneration++
	if receipt != nil && m.matchesArithmeticInvocation(receipt) && receipt.frame.executing && m.callbackDepth == 0 {
		receipt.frame.singleton = m.singletonMonomial
		receipt.frame.singletonGeneration = m.singletonGeneration
	}
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 {
		f.singleton = m.singletonMonomial
		f.singletonGeneration = m.singletonGeneration
	}
	if s := m.compositionOwner; s != nil && s.executing {
		s.singleton = m.singletonMonomial
		s.singletonGeneration = m.singletonGeneration
	}
	if m.pscExecutingOwner() != 0 {
		m.pscReceipts.singleton = u
	}
	if receipt != nil {
		return m.mkCellWithReceipt(1, nil, []Monomial{u}, nil, a, &receipt.frame.factory)
	}
	return m.mkCell(1, nil, []Monomial{u}, nil, a)
}
func (m *MultivariatePolynomialCore) MkConstantTake(a *Int) (Polynomial, error) {
	return m.mkConstantTakeWithArithmeticReceipt(a, nil)
}
func (m *MultivariatePolynomialCore) mkConstantTakeWithArithmeticReceipt(a *Int, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if a == nil {
		return Polynomial{}, arena.ErrConfig
	}
	z, e := m.modular.IsZero(a)
	if e != nil {
		return Polynomial{}, e
	}
	if z {
		return m.zero, nil
	}
	one, e := m.modular.IsOne(a)
	if e != nil {
		return Polynomial{}, e
	}
	if one {
		return m.unit, nil
	}
	return m.mkConstCoreWithArithmeticReceipt(a, receipt)
}
func (m *MultivariatePolynomialCore) MkVariable(v, k uint32) (Polynomial, error) {
	return m.mkVariableWithReceipt(v, k, nil)
}
func (m *MultivariatePolynomialCore) mkVariableWithReceipt(v, k uint32, receipt *polynomialVariableInvocationReceipt) (result Polynomial, err error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && v >= m.monomials.nextVar {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if m.singletonReference {
		return Polynomial{}, ErrPolynomialPending
	}
	as := [1]Int{NewInt(1)}
	var scope *polynomialVariableScope
	if receipt != nil {
		scope = &m.variableLocal
		if scope.active || scope.invocation == ^uint64(0) || m.callbackDepth != 0 {
			return Polynomial{}, ErrPolynomialPending
		}
		*scope = polynomialVariableScope{active: true, executing: true, invocation: scope.invocation + 1}
		*receipt = polynomialVariableInvocationReceipt{m, scope, scope.invocation}
		defer func() { scope.executing = false; scope.bodyCause = err }()
	}
	var outcome monomialFactoryOutcome
	b, e := m.generatedVariableWithOutcome(v, k, &outcome)
	if scope != nil {
		if outcome.started {
			scope.allocator = outcome.allocatorInvocation
		}
		if e != nil && outcome.started {
			scope.calleeGeneration = outcome.generation
		}
		if e == nil && outcome.created {
			scope.generated = b
		}
	}
	if e != nil {
		return Polynomial{}, e
	}
	if e = m.monomials.IncRef(b); e != nil {
		return Polynomial{}, e
	}
	m.singletonMonomial = b
	m.singletonReference = true
	m.singletonGeneration++
	if scope != nil {
		scope.generated = Monomial{}
		scope.singleton = b
		scope.singletonGeneration = m.singletonGeneration
	}
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 {
		f.singleton = m.singletonMonomial
		f.singletonGeneration = m.singletonGeneration
	}
	if s := m.compositionOwner; s != nil && s.executing {
		s.singleton = m.singletonMonomial
		s.singletonGeneration = m.singletonGeneration
	}
	if m.pscExecutingOwner() != 0 {
		m.pscReceipts.singleton = b
	}
	m.transferGeneratedMonomial()
	if scope == nil {
		return m.mkCell(1, as[:], []Monomial{b}, nil, nil)
	}
	result, err = m.mkCellWithReceipts(1, as[:], []Monomial{b}, nil, nil, &scope.factory, nil)
	if err != nil {
		return result, err
	}
	if err = m.transferFactoryResult(&scope.factory, result); err != nil {
		return result, err
	}
	scope.active = false
	return result, nil
}
func (m *MultivariatePolynomialCore) MkTerms(as []Int, ms []Monomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	for i := range as {
		if i >= len(ms) {
			return Polynomial{}, arena.ErrBounds
		}
		if e := m.som.add(&as[i], ms[i], false); e != nil {
			return Polynomial{}, e
		}
	}
	return m.som.mk(false)
}
func (m *MultivariatePolynomialCore) CloseChecked() error {
	if m != nil && m.self == m && m.collectingVariables {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.normalizationPending() {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && (m.variableLocal.active || m.pscClassic.active || m.pscClassic.executing) {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.negativeActive && m.negativeExecuting {
		return ErrPolynomialPending
	}
	if m.arithmeticExecuting() {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && (m.coefficientInvocation.active || m.constantCoefficientInvocation.active) {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.displayPending() {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.dividesLocal.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.contentLocal.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.pscPending() {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.translation.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.substitution.scope.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.pseudoDivision.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.exactDivision.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.powerLocal.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.newtonLocal != nil {
		return ErrPolynomialPending
	}
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if m.originalCause != nil && m.constructorUnwinding {
		if e := m.unwindConstructorMembers(); e != nil {
			m.cleanupCause = e
			return e
		}
		m.cleanupCause = nil
	}
	if !m.closing {
		// Named cached cells can also carry real caller-owned references. The
		// owning adapter refuses before source actions, without a directory
		// sweep or changing the literal source destructor's later assertion.
		for i, p := range [2]Polynomial{m.zero, m.unit} {
			if p != (Polynomial{}) && int(m.sourceStage) <= i {
				l, e := m.polynomialLocation(p)
				if e != nil {
					return e
				}
				if l.deleteStage == 0 {
					r, e := m.readWord(p, 0)
					if e != nil {
						return e
					}
					if r > 1 {
						return ErrPolynomialLive
					}
				}
			}
		}
		private, e := m.factoryCallerGuard()
		if e != nil {
			return e
		}
		if m.zero != (Polynomial{}) && m.sourceStage == 0 {
			private++
		}
		if m.unit != (Polynomial{}) && m.sourceStage <= 1 {
			private++
		}
		if m.constantScope.active && m.constantScope.result != (Polynomial{}) && m.constantScope.result != m.zero && m.constantScope.result != m.unit {
			private++
		}
		if m.overloadResult != (Polynomial{}) && m.overloadResult != m.zero && m.overloadResult != m.unit {
			private++
		}
		if m.negativeResult != (Polynomial{}) && m.negativeResult != m.zero && m.negativeResult != m.unit {
			private++
		}
		if m.live > private {
			return ErrPolynomialLive
		}
		m.closing = true
	}
	if e := m.closeDisplayMetadata(); e != nil {
		return e
	}
	if e := m.cleanupCreation(); e != nil {
		m.cleanupCause = e
		return e
	}
	if m.overloadResult != (Polynomial{}) {
		if e := m.deleteCore(m.overloadResult); e != nil {
			return e
		}
		m.overloadResult = Polynomial{}
	}
	if m.linearConstantActive {
		if e := m.deleteNumeral(&m.linearConstant); e != nil {
			return e
		}
		m.linearConstantActive = false
	}
	if e := m.cleanupConstantScope(); e != nil {
		m.cleanupCause = e
		return e
	}
	if e := m.cleanupNegPhase(true); e != nil {
		m.cleanupCause = e
		return e
	}
	if e := m.cleanupScalarDivPhase(true); e != nil {
		m.cleanupCause = e
		return e
	}
	privates := [2]*Polynomial{&m.zero, &m.unit}
	for m.closeStage < 2 {
		p := privates[m.closeStage]
		if m.sourceStage > m.closeStage {
			*p = Polynomial{}
			m.closeStage++
			continue
		}
		if *p != (Polynomial{}) {
			l, e := m.polynomialLocation(*p)
			if e != nil {
				return e
			}
			if l.callbackCause != nil {
				// The source DEC already reached zero before its handler
				// refused. Explicit owning retry aborts this retained action;
				// neither replay the callback nor repeat the source DEC.
				if e = m.CleanupValue(p); e != nil {
					m.cleanupCause = e
					return e
				}
			} else if l.deleteStage == 0 {
				if e = m.decRefCore(*p); e != nil {
					m.cleanupCause = e
					return e
				}
			} else {
				if e = m.deleteCore(*p); e != nil {
					return e
				}
			}
			*p = Polynomial{}
		}
		m.closeStage++
	}
	buffers := [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2}
	for m.closeStage < 6 {
		if e := buffers[m.closeStage-2].close(); e != nil {
			return e
		}
		m.closeStage++
	}
	if m.closeStage == 6 {
		// The real modular child may have refused its cold header before
		// construction. No numeral-manager action exists in that prefix.
		if m.modular != nil {
			if e := m.deleteNumeral(&m.zeroNumeral); e != nil {
				return e
			}
		}
		m.closeStage++
	}
	for _, v := range [4]*polynomialVector{&m.rationalTerms, &m.linearTerms, &m.linearMonomials, &m.polynomials} {
		if e := v.close(m); e != nil {
			return e
		}
	}
	if m.idsInitialized || m.idsSourceDestroyed {
		if e := m.ids.CloseChecked(); e != nil {
			return e
		}
		m.idsInitialized = false
	}
	if e := m.translateBuffers.CloseChecked(); e != nil {
		return e
	}
	if m.modular != nil && !m.modular.closed {
		for _, a := range [3]*Int{&m.minusOne, &m.negativeCoefficient, &m.constantCopy} {
			if e := m.deleteNumeral(a); e != nil {
				return e
			}
		}
		if m.modularSourceDestroyed {
			// Actual source fields were destroyed during failed-constructor
			// unwind. This refunds only the retained Go adapter header.
			m.integer.budget.Release(m.modular.header)
			m.modular.header = 0
			m.modular.closed = true
		} else if e := m.modular.Close(); e != nil {
			return e
		}
	}
	if e := m.closeSubstitutionMembers(); e != nil {
		return e
	}
	if m.queryPositions.self != nil {
		if e := m.queryPositions.CloseChecked(); e != nil {
			return e
		}
	}
	if m.monomialRetained {
		before := m.monomials.managerRefs
		e := m.monomials.DecManagerRef()
		if m.monomials.managerRefs != before {
			m.monomialRetained = false
		}
		if e != nil {
			return e
		}
	}
	m.integer.budget.Release(uint64(cap(m.linearGenerated)))
	m.linearGenerated = nil
	m.integer.budget.Release(uint64(cap(m.locations)) * uint64(reflect.TypeOf(polynomialLocation{}).Size()))
	m.locations = nil
	m.integer.budget.Release(m.header)
	m.header = 0
	m.closed = true
	return nil
}

func polynomialIntegerReady(m *IntegerManager) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceFinalizing || m.constructionFailed {
		return arena.ErrClosed
	}
	return m.memory.CheckOpen()
}

func (m *MultivariatePolynomialCore) SetZPUint64(p uint64) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.modular.SetZPUint64(p)
}
func (m *MultivariatePolynomialCore) SetZ() error {
	if e := m.check(); e != nil {
		return e
	}
	return m.modular.SetZ()
}
