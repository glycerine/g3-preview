// Source: src/math/polynomial/polynomial.cpp:2481-2495; src/util/vector.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package polynomial

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Concrete scalar vector codec: coefficients use 12 bytes and monomial or
// polynomial locators use 16. Headers and payload inhabit one real Runtime
// allocation, with no shadow typed backing or fabricated allocation events.
type polynomialVector struct {
	raw, pending, old                                            util.Block
	initialized, oldInitialized, oldSize, moved, oldCleanupIndex uint32
	retired                                                      []util.Int
	retirementStart, retiredActive                               uint32
	width                                                        uint32
	interrupted, partial                                         bool
}

func (v *polynomialVector) size(m *MultivariatePolynomialCore) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return m.memory.ReadUint32(v.raw, 4)
}
func (v *polynomialVector) capacity(m *MultivariatePolynomialCore) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return m.memory.ReadUint32(v.raw, 0)
}
func (v *polynomialVector) expand(m *MultivariatePolynomialCore) error {
	if v.interrupted || v.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	old, e := v.capacity(m)
	if e != nil {
		return e
	}
	n := uint32(2)
	if v.raw != (util.Block{}) {
		n = (3*old + 1) >> 1
	}
	bytes := v.width*n + 8
	oldBytes := v.width*old + 8
	if v.raw != (util.Block{}) && (old == 0 || n <= old || bytes <= oldBytes) {
		return util.Failure{Kind: util.FailureVectorOverflow}
	}
	if e := m.prepareNormalizationConverterHistoryAtAllocation(v, nil, nil); e != nil {
		return e
	}
	if v.raw == (util.Block{}) {
		v.pending, e = m.memory.Allocate(uint64(bytes))
		if e != nil {
			return e
		}
		if e = m.memory.WriteUint32(v.pending, 0, n); e != nil {
			return e
		}
		if e = m.memory.WriteUint32(v.pending, 4, 0); e != nil {
			return e
		}
		v.raw = v.pending
		v.pending = util.Block{}
		return nil
	}
	if v.width == 12 {
		return v.expandNumerals(m, n, bytes)
	}
	before := v.raw
	if e = m.memory.Reallocate(&v.raw, uint64(bytes)); e != nil {
		if v.raw != before {
			v.interrupted = true
			v.partial = m.memory.SizeTBits() == 32 && uint32(bytes+4) < 4
		}
		return e
	}
	return m.memory.WriteUint32(v.raw, 0, n)
}
func (v *polynomialVector) appendSlot(m *MultivariatePolynomialCore) (uint32, error) {
	n, e := v.size(m)
	if e != nil {
		return 0, e
	}
	c, e := v.capacity(m)
	if e != nil {
		return 0, e
	}
	if n >= c {
		if e = v.expand(m); e != nil {
			return 0, e
		}
	}
	return n, nil
}
func (v *polynomialVector) publishSize(m *MultivariatePolynomialCore, n uint32) error {
	return m.memory.WriteUint32(v.raw, 4, n)
}
func (v *polynomialVector) reset(m *MultivariatePolynomialCore) error {
	if v.raw == (util.Block{}) {
		return nil
	}
	return v.publishSize(m, 0)
}
func (v *polynomialVector) read64(m *MultivariatePolynomialCore, o uint64) (uint64, error) {
	a, e := m.memory.ReadUint32(v.raw, o)
	if e != nil {
		return 0, e
	}
	b, e := m.memory.ReadUint32(v.raw, o+4)
	return uint64(a) | uint64(b)<<32, e
}
func (v *polynomialVector) write64(m *MultivariatePolynomialCore, o, x uint64) error {
	if e := m.memory.WriteUint32(v.raw, o, uint32(x)); e != nil {
		return e
	}
	return m.memory.WriteUint32(v.raw, o+4, uint32(x>>32))
}
func (v *polynomialVector) coefficient(m *MultivariatePolynomialCore, i uint32) (util.Int, error) {
	o := uint64(8) + uint64(i)*12
	a, e := m.memory.ReadUint32(v.raw, o)
	if e != nil {
		return util.Int{}, e
	}
	r, e := v.read64(m, o+4)
	return util.IntFromNativeWords([3]uint32{uint32(a), uint32(r), uint32(uint64(r) >> 32)}), e
}
func (v *polynomialVector) putCoefficient(m *MultivariatePolynomialCore, i uint32, a util.Int) error {
	o := uint64(8) + uint64(i)*12
	if e := m.memory.WriteUint32(v.raw, o, uint32(a.NativeScalar())); e != nil {
		return e
	}
	return v.write64(m, o+4, uint64(a.NativeCellToken()))
}
func (v *polynomialVector) monomial(m *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	o := uint64(8) + uint64(i)*16
	a, e := v.read64(m, o)
	if e != nil {
		return Monomial{}, e
	}
	s, e := m.memory.ReadUint32(v.raw, o+8)
	if e != nil {
		return Monomial{}, e
	}
	g, e := m.memory.ReadUint32(v.raw, o+12)
	return Monomial{owner: a, slot: s, generation: g}, e
}
func (v *polynomialVector) putMonomial(m *MultivariatePolynomialCore, i uint32, a Monomial) error {
	o := uint64(8) + uint64(i)*16
	if e := v.write64(m, o, a.owner); e != nil {
		return e
	}
	if e := m.memory.WriteUint32(v.raw, o+8, a.slot); e != nil {
		return e
	}
	return m.memory.WriteUint32(v.raw, o+12, a.generation)
}
func (v *polynomialVector) appendCoefficient(m *MultivariatePolynomialCore, a util.Int) error {
	return v.appendCoefficientWithAppendScope(m, a, nil, false)
}
func (v *polynomialVector) appendCoefficientWithAppendScope(m *MultivariatePolynomialCore, a util.Int, scope *polynomialCheapAppendScope, owned bool) error {
	i, e := v.appendSlot(m)
	if e != nil {
		return e
	}
	if i < v.initialized {
		old, e := v.coefficient(m, i)
		if e != nil {
			return e
		}
		if old.NativeCellToken() != 0 {
			if e = v.retainNumeral(m, old); e != nil {
				return e
			}
		}
	}
	if e = v.putCoefficient(m, i, a); e != nil {
		return e
	}
	v.retiredActive = uint32(len(v.retired))
	if i >= v.initialized {
		v.initialized = i + 1
	}
	// The actual zero NUM slot exists before vector-size publication and the
	// subsequent SET. A stopped SET still belongs to this constructed row.
	if scope != nil {
		scope.buffer.coefficientTransferred[i] = 0
		if owned {
			scope.coefficients = i + 1
			scope.buffer.coefficientOwners[i] = scope.invocation
			scope.coefficientClaims++
		} else {
			scope.foreignMixed = true
			scope.buffer.coefficientOwners[i] = 0
			if scope.retirePhase == 1 || scope.retirePhase == 2 {
				scope.retireAs = i + 1
			}
		}
	}
	return v.publishSize(m, i+1)
}
func (v *polynomialVector) appendMonomial(m *MultivariatePolynomialCore, a Monomial) error {
	return v.appendMonomialWithAppendScope(m, a, nil, false)
}
func (v *polynomialVector) appendMonomialWithAppendScope(m *MultivariatePolynomialCore, a Monomial, scope *polynomialCheapAppendScope, owned bool) error {
	i, e := v.appendSlot(m)
	if e != nil {
		return e
	}
	if e = v.putMonomial(m, i, a); e != nil {
		return e
	}
	if scope != nil {
		scope.buffer.monomialTransferred[i] = 0
		if owned {
			scope.buffer.monomialOwners[i] = scope.invocation
			scope.monomialClaims++
			scope.monomials = i + 1
			scope.pending = Monomial{}
		} else {
			scope.buffer.monomialOwners[i] = 0
			if scope.retirePhase == 1 || scope.retirePhase == 2 {
				scope.retireMs = i + 1
			}
			scope.foreignMixed = true
		}
	}
	return v.publishSize(m, i+1)
}
func (v *polynomialVector) close(m *MultivariatePolynomialCore) error {
	if v.width == 12 {
		if e := v.cleanupNumerals(m); e != nil {
			return e
		}
	}
	if v.pending != (util.Block{}) {
		if e := m.memory.Deallocate(&v.pending); e != nil {
			return e
		}
	}
	if v.raw != (util.Block{}) {
		var e error
		if v.partial {
			e = m.memory.CleanupPartial(&v.raw)
		} else {
			e = m.memory.Deallocate(&v.raw)
		}
		if e != nil {
			return e
		}
	}
	v.interrupted = false
	v.partial = false
	return nil
}
func (m *MultivariatePolynomialCore) prepareLocation() (uint32, error) {
	for i := range m.locations {
		l := &m.locations[i]
		if !l.live && l.generation != math.MaxUint32 {
			l.generation++
			return uint32(i + 1), nil
		}
	}
	if uint64(len(m.locations)) >= math.MaxUint32 {
		return 0, arena.ErrSlotLimit
	}
	n := len(m.locations) + 1
	if n > cap(m.locations) {
		old := cap(m.locations)
		c := old + old/2 + 1
		stride := uint64(reflect.TypeOf(polynomialLocation{}).Size())
		if uint64(c) > uint64(^uint(0)>>1)/stride {
			return 0, arena.ErrBounds
		}
		bytes := uint64(c) * stride
		if e := m.integer.Budget().Reserve(bytes); e != nil {
			return 0, e
		}
		a := make([]polynomialLocation, n, c)
		copy(a, m.locations)
		m.locations = a
		m.integer.Budget().Release(uint64(old) * stride)
	} else {
		m.locations = m.locations[:n]
	}
	m.locations[n-1].generation = 1
	return uint32(n), nil
}
func (m *MultivariatePolynomialCore) publishDirectory(p Polynomial, id uint32) error {
	n, e := m.polynomials.size(m)
	if e != nil {
		return e
	}
	for n <= id {
		if e = m.polynomials.appendMonomial(m, Monomial{}); e != nil {
			return e
		}
		n++
	}
	old, e := m.polynomials.monomial(m, id)
	if e != nil {
		return e
	}
	if polynomialSourceDebug && old != (Monomial{}) {
		return ErrPolynomialAssertion
	}
	return m.polynomials.putMonomial(m, id, Monomial{owner: p.owner, slot: p.slot, generation: p.generation})
}
func (m *MultivariatePolynomialCore) mkCell(n uint32, as []util.Int, ms []Monomial, buffer *polynomialSOM, single *util.Int) (Polynomial, error) {
	return m.mkCellWithReceipt(n, as, ms, buffer, single, nil)
}
func (m *MultivariatePolynomialCore) mkCellWithReceipt(n uint32, as []util.Int, ms []Monomial, buffer *polynomialSOM, single *util.Int, receipt *polynomialFactoryInvocationReceipt) (Polynomial, error) {
	return m.mkCellWithReceipts(n, as, ms, buffer, single, receipt, nil)
}
func (m *MultivariatePolynomialCore) mkCellWithReceipts(n uint32, as []util.Int, ms []Monomial, buffer *polynomialSOM, single *util.Int, receipt *polynomialFactoryInvocationReceipt, appendReceipt *polynomialCheapAppendInvocationReceipt) (Polynomial, error) {
	if m.pending.raw != (util.ObjectRef{}) || m.pending.value != (Polynomial{}) {
		return Polynomial{}, ErrPolynomialPending
	}
	if receipt != nil && m.factoryInvocation.active {
		return Polynomial{}, ErrPolynomialPending
	}
	var appendScope *polynomialCheapAppendScope
	if appendReceipt != nil && appendReceipt.invocation != 0 {
		if buffer == nil || !buffer.matchesAppendInvocation(appendReceipt) {
			return Polynomial{}, ErrPolynomialPending
		}
		appendScope = &buffer.appendScope
		size, e := buffer.as.size(m)
		if e != nil {
			return Polynomial{}, e
		}
		if appendScope.foreignMixed || appendScope.stopped || appendScope.completedTransfer || appendScope.monomials != n || appendScope.coefficients != size || buffer.hasPendingReference {
			return Polynomial{}, ErrPolynomialPending
		}
	}
	// Actual constructor start receipt, after the foreign pending guard.
	if m.creationGeneration == ^uint64(0) {
		return Polynomial{}, arena.ErrBounds
	}
	m.creationGeneration++
	m.pending = polynomialCreation{generation: m.creationGeneration}
	if appendScope != nil {
		m.pending.appendParent = appendScope
		appendScope.factoryInvocation = m.creationGeneration
	} else if buffer != nil && buffer.appendScope.active {
		// Actual unreceipted constructor transfer is foreign to the old append
		// cohort, even when a later vector happens to have identical sizes.
		buffer.appendScope.foreignMixed = true
	}
	if receipt != nil {
		m.factoryInvocation = polynomialFactoryScope{active: true, executing: true, invocation: m.creationGeneration, buffer: buffer}
		*receipt = polynomialFactoryInvocationReceipt{manager: m, frame: &m.factoryInvocation, invocation: m.creationGeneration}
		defer func() { m.factoryInvocation.executing = false }()
	}
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 {
		m.pending.factorsParent = f
		f.factoryGeneration = m.creationGeneration
	}
	if owner := m.squareRootOwner; owner != nil && owner.executing && m.callbackDepth == 0 {
		m.pending.squareRootParent = owner
		owner.factoryGeneration = m.creationGeneration
	}
	if s := m.conversionOwner; s != nil && s.executing && m.callbackDepth == 0 {
		m.pending.conversionParent = s
		s.factoryGeneration = m.creationGeneration
	}
	if m.compositionOwner != nil && m.compositionOwner.executing {
		m.pending.compositionParent = m.compositionOwner
		m.compositionOwner.factoryGeneration = m.creationGeneration
	}
	if m.derivativeOwner != nil && m.derivativeOwner.executing {
		m.pending.derivativeParent = m.derivativeOwner
		m.derivativeOwner.factoryGeneration = m.creationGeneration
	}
	m.pending.substitutionOwned = m.substitution.scope.executing
	m.pending.pscOwner = m.pscExecutingOwner()
	m.pending.translationOwned = m.translation.executing
	requested := polynomialCodecBytes(n)
	observer := m.pending.compositionParent
	squareObserver := m.pending.squareRootParent
	var before [3]util.Block
	if observer != nil || squareObserver != nil || m.pending.conversionParent != nil || m.pending.factorsParent != nil {
		var e error
		before, e = m.monomials.allocator.PendingRoots()
		if e != nil {
			return Polynomial{}, e
		}
	}
	var raw util.ObjectRef
	var e error
	if outcome := m.factoryAllocationOutcome(receipt); outcome != nil {
		var started bool
		if e := m.prepareNormalizationConverterHistoryAtAllocation(nil, receipt, nil); e != nil {
			return Polynomial{}, e
		}
		raw, started, e = m.monomials.allocator.AllocateWithOutcomeStarted(uint64(requested), outcome)
		if started {
			m.factoryInvocation.allocationStarted = true
			m.factoryInvocation.allocationCause = e
			m.factoryInvocation.allocator[0] = outcome.Pending
		}
	} else {
		raw, e = m.monomials.allocator.Allocate(uint64(requested))
	}
	if raw != (util.ObjectRef{}) {
		m.pending.raw = raw
		m.pending.requested = requested
	}
	if (observer != nil || squareObserver != nil) && before == ([3]util.Block{}) {
		after, probe := m.monomials.allocator.PendingRoots()
		if after != ([3]util.Block{}) {
			if observer != nil {
				observer.monomialRaw = after
			}
			if squareObserver != nil {
				squareObserver.monomialRaw = after
			}
		}
		if probe != nil {
			return Polynomial{}, errors.Join(e, probe)
		}
	}
	if owner := m.pending.conversionParent; owner != nil && before == ([3]util.Block{}) {
		after, probe := m.monomials.allocator.PendingRoots()
		if after != ([3]util.Block{}) {
			owner.monomialRaw = after
		}
		if probe != nil {
			return Polynomial{}, errors.Join(e, probe)
		}
	}
	if f := m.pending.factorsParent; f != nil && before == ([3]util.Block{}) {
		after, x := m.monomials.allocator.PendingRoots()
		if after != ([3]util.Block{}) {
			f.monomialRaw = after
		}
		if x != nil {
			return Polynomial{}, errors.Join(e, x)
		}
	}
	if e != nil {
		return Polynomial{}, e
	}
	id, e := m.ids.Next()
	if e != nil {
		return Polynomial{}, e
	}
	m.pending.id = id
	m.pending.idConsumed = true
	slot, e := m.prepareLocation()
	if e != nil {
		return Polynomial{}, e
	}
	l := &m.locations[slot-1]
	g := l.generation
	// Actual constructor layout fixes m_ms independently of later mutable SIZE.
	// Retain the allocation count before the uint32 request-size wrap.
	*l = polynomialLocation{raw: raw, requested: requested, generation: g, live: true, monomialBase: 12 + uint64(n)*12}
	p := Polynomial{owner: m.origin, slot: slot, generation: g}
	m.pending.value = p
	m.pending.raw = util.ObjectRef{}
	m.pending.buffer = buffer
	if e = m.writeWord(p, 0, 0); e != nil {
		return Polynomial{}, e
	}
	if e = m.writeWord(p, 4, id&0x7fffffff); e != nil {
		return Polynomial{}, e
	}
	if e = m.writeWord(p, 8, n); e != nil {
		return Polynomial{}, e
	}
	max := uint32(0)
	for i := uint32(0); i < n; i++ {
		if e = m.writeCoefficient(p, i, util.Int{}); e != nil {
			return Polynomial{}, e
		}
		l.initializedCoefficients = i + 1
		var a util.Int
		movingAppend := false
		if buffer != nil {
			a, e = buffer.as.coefficient(m, i)
			if e != nil {
				return Polynomial{}, e
			}
			if e = buffer.as.putCoefficient(m, i, util.Int{}); e != nil {
				return Polynomial{}, e
			}
			if buffer.appendScope.active && uint64(i) < uint64(len(buffer.coefficientOwners)) && buffer.coefficientOwners[i] == buffer.appendScope.invocation {
				buffer.appendScope.movingCoefficient = a
				movingAppend = true
				buffer.coefficientOwners[i] = 0
				buffer.coefficientTransferred[i] = buffer.appendScope.invocation
				buffer.appendScope.coefficientClaims--
			}
		} else if single != nil {
			a = *single
			*single = util.Int{}
		} else {
			if uint64(i) >= uint64(len(as)) {
				return Polynomial{}, arena.ErrBounds
			}
			a = as[i]
			as[i] = util.Int{}
		}
		if e = m.writeCoefficient(p, i, a); e != nil {
			return Polynomial{}, e
		}
		if movingAppend {
			buffer.appendScope.movingCoefficient = util.Int{}
		}
		if appendScope != nil {
			appendScope.transferredCoefficients = i + 1
		}
		var term Monomial
		if buffer != nil {
			term, e = buffer.ms.monomial(m, i)
			if e != nil {
				return Polynomial{}, e
			}
		} else {
			if uint64(i) >= uint64(len(ms)) {
				return Polynomial{}, arena.ErrBounds
			}
			term = ms[i]
		}
		if polynomialSourceDebug {
			r, e := m.monomialRefCount(term)
			if e != nil {
				return Polynomial{}, e
			}
			if r == 0 {
				return Polynomial{}, ErrPolynomialAssertion
			}
		}
		if e = m.writeMonomial(p, i, term); e != nil {
			return Polynomial{}, e
		}
		l.initializedMonomials = i + 1
		if buffer == nil && i == 0 && m.singletonReference && term == m.singletonMonomial {
			if m.pscReceipts.singleton == term {
				m.pscReceipts.singleton = Monomial{}
			}
			if f := m.factorsOwner; f != nil && f.singleton == term && f.singletonGeneration == m.singletonGeneration {
				f.singleton = Monomial{}
			}
			if s := &m.variableLocal; s.active && s.singleton == term && s.singletonGeneration == m.singletonGeneration {
				s.singleton = Monomial{}
			}
			m.singletonReference = false
			m.singletonMonomial = Monomial{}
		}
		if buffer != nil && buffer.appendScope.active && uint64(i) < uint64(len(buffer.monomialOwners)) && buffer.monomialOwners[i] == buffer.appendScope.invocation {
			buffer.monomialOwners[i] = 0
			buffer.monomialTransferred[i] = buffer.appendScope.invocation
			buffer.appendScope.monomialClaims--
		}
		if buffer != nil {
			buffer.transferred = i + 1
			if appendScope != nil {
				appendScope.transferredMonomials = i + 1
			}
		}
		if i > 0 {
			best, e := m.monomialCore(p, max)
			if e != nil {
				return Polynomial{}, e
			}
			greater, e := m.firstGreater(term, best)
			if e != nil {
				return Polynomial{}, e
			}
			if greater {
				max = i
			}
		}
	}
	if e = m.swapTerms(p, 0, max); e != nil {
		return Polynomial{}, e
	}
	if e = m.publishDirectory(p, id); e != nil {
		return Polynomial{}, e
	}
	l.published = true
	if owner := m.pending.conversionParent; owner != nil && owner.factoryGeneration == m.pending.generation {
		owner.factoryGeneration = 0
		owner.result = p
		owner.ownsResult = true
	}
	if m.translation.executing {
		m.translation.freshResult = p
	}
	if m.pending.derivativeParent != nil && m.pending.derivativeParent.factoryGeneration == m.pending.generation {
		m.pending.derivativeParent.factoryGeneration = 0
	}
	if owner := m.pending.compositionParent; owner != nil && owner.factoryGeneration == m.pending.generation {
		owner.factoryGeneration = 0
		owner.freshResult = p
	}
	if owner := m.pending.squareRootParent; owner != nil && owner.factoryGeneration == m.pending.generation {
		owner.factoryGeneration = 0
		owner.freshResult = p
	}
	if f := m.pending.factorsParent; f != nil && f.factoryGeneration == m.pending.generation {
		f.factoryGeneration = 0
		f.freshResult = p
	}
	if receipt != nil {
		m.factoryInvocation.result = p
	}
	m.pending = polynomialCreation{}
	m.live++
	return p, nil
}
