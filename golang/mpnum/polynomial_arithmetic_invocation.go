// Source: src/math/polynomial/polynomial.cpp:1885-2156,2217-2300,2665-2744;
// src/util/obj_ref.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// A concrete owning continuation records acquisition by this actual source
// buffer. It is embedded in its charged core header; no lexical Runtime ticket
// or duplicate numeral array is constructed for this metadata.
type polynomialArithmeticScope struct {
	coefficientOwner                                  *polynomialCoefficientScope
	coefficientInvocation                             uint64
	coefficientReady                                  bool
	buffer                                            *polynomialSOM
	invocation                                        uint64
	active, executing, claimed, foreignMixed, stopped bool
	numeric                                           polynomialNumericRoots
	normalize                                         integerPNormalizeInvocationReceipt
	addMul                                            integerAddMulInvocationReceipt
	factory                                           polynomialFactoryInvocationReceipt
	append                                            polynomialCheapAppendInvocationReceipt
	constantGeneration                                uint64
	singleton                                         Monomial
	singletonGeneration                               uint64
	transport                                         Int
	transportSlot                                     uint32
	transportBound                                    bool
	positionTerm                                      Monomial
	positionPublished                                 bool
	calleeGeneration                                  uint64
	generated                                         Monomial
	monomialAllocator                                 monomialAllocatorInvocationReceipt
	bodyCause, sourceCause                            error
}
type polynomialArithmeticInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialArithmeticScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesArithmeticInvocation(r *polynomialArithmeticInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame != nil && m.ownsArithmeticFrame(r.frame) && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}

func (m *MultivariatePolynomialCore) ownsArithmeticFrame(s *polynomialArithmeticScope) bool {
	if s == nil {
		return false
	}
	if m.ownsArithmeticBuffer(s.buffer) && s == &s.buffer.arithmeticScope {
		return true
	}
	return m != nil && m.self == m && s == &m.constantScope.arithmeticScope && s.buffer == nil && s.constantGeneration != 0 && s.constantGeneration == m.constantGeneration
}

// Direct numeric engines validate the concrete passed invocation. An executing
// high-level operation alone cannot acquire roots produced by an external hook.
func (m *IntegerManager) arithmeticAllocationRoots(r *polynomialArithmeticInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.integer != m || !r.manager.matchesArithmeticInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}

func (m *MultivariatePolynomialCore) ownsArithmeticBuffer(b *polynomialSOM) bool {
	return m != nil && m.self == m && b != nil && b.owner == m && (b == &m.som || b == &m.som2 || b == &m.cheap || b == &m.cheap2)
}

// Acquisition follows the reached source reset/empty-buffer preflight. It
// creates no source allocation and gives no authority over older source rows.
func (b *polynomialSOM) acquireArithmeticInvocation(r *polynomialArithmeticInvocationReceipt) error {
	return b.acquireArithmeticInvocationWithCoefficient(r, nil)
}
func (b *polynomialSOM) acquireArithmeticInvocationWithCoefficient(r *polynomialArithmeticInvocationReceipt, parent *polynomialCoefficientInvocationReceipt) error {
	if b == nil || b.owner == nil || !b.owner.ownsArithmeticBuffer(b) || r == nil {
		return ErrPolynomialOwner
	}
	m := b.owner
	if parent != nil && (!m.matchesCoefficientInvocation(parent) || !parent.frame.executing) {
		return ErrPolynomialOwner
	}
	if e := m.check(); e != nil {
		return e
	}
	if b.arithmeticScope.active || b.appendScope.active || b.resetActive || b.removeActive || b.hasPendingReference || b.borrowedAppend || b.as.interrupted || b.ms.interrupted || b.as.pending != (memory.Block{}) || b.ms.pending != (memory.Block{}) || m.arithmeticInvocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	a, e := b.as.size(m)
	if e != nil {
		return e
	}
	if n != 0 || a != 0 {
		return ErrPolynomialPending
	}
	m.arithmeticInvocation++
	b.arithmeticScope = polynomialArithmeticScope{buffer: b, invocation: m.arithmeticInvocation, active: true, claimed: true}
	if parent != nil {
		b.arithmeticScope.coefficientOwner = parent.frame
		b.arithmeticScope.coefficientInvocation = parent.invocation
		parent.frame.bufferActive = true
	}
	*r = polynomialArithmeticInvocationReceipt{manager: m, frame: &b.arithmeticScope, invocation: m.arithmeticInvocation}
	return nil
}

// The caller passes this real buffer invocation at each reached numeric call.
// Allocation authority is recorded by the existing Integer acquisition leaves.
func (b *polynomialSOM) arithmeticNumericCall(op uint8, a, c, d, out *Int, r *polynomialArithmeticInvocationReceipt) error {
	if r == nil {
		return b.owner.compositionNumericCall(op, a, c, d, out)
	}
	m := b.owner.modular
	switch op {
	case 0:
		return m.copyWithArithmeticReceipt(out, a, r)
	case 1:
		return m.addWithArithmeticReceipt(a, c, out, r)
	case 2:
		return m.mulWithArithmeticReceipt(a, c, out, r)
	case 3:
		return m.addMulWithArithmeticReceipt(a, c, d, out, r)
	}
	return ErrPolynomialAssertion
}

// The actual product engine returns its factory branch outcome. Only this
// concrete executing buffer invocation receives newly acquired authority.
func (b *polynomialSOM) arithmeticProduct(a, other Monomial, receipt *polynomialArithmeticInvocationReceipt) (Monomial, error) {
	m := b.owner
	if receipt == nil {
		return m.generatedProduct(a, other)
	}
	if !m.matchesArithmeticInvocation(receipt) || receipt.frame.buffer != b || !receipt.frame.executing || m.callbackDepth != 0 {
		return Monomial{}, ErrPolynomialOwner
	}
	var outcome monomialFactoryOutcome
	product, e := m.generatedProductWithOutcome(a, other, &outcome)
	if outcome.started {
		receipt.frame.monomialAllocator = outcome.allocatorInvocation
	}
	if e != nil && outcome.started {
		receipt.frame.calleeGeneration = outcome.generation
	}
	if e == nil && outcome.created {
		receipt.frame.generated = product
		// The actual returned creation token names this charged location.
		// The cold tag routes later successful INC/owning consumption only;
		// neither a table scan nor a refcount grants this authority.
		location := &m.monomials.locations[product.slot-1]
		location.arithmeticCohort = receipt.frame
		location.arithmeticInvocation = receipt.invocation
	}
	return product, e
}

// Readonly history and active owning authority are separate. A later real
// invocation invalidates an old receipt; owning acknowledgment retains cause.
func (m *MultivariatePolynomialCore) arithmeticInvocationSourceCause(r *polynomialArithmeticInvocationReceipt) error {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame == nil || !m.ownsArithmeticFrame(r.frame) || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.sourceCause
}

func (m *MultivariatePolynomialCore) cleanupArithmeticInvocation(r *polynomialArithmeticInvocationReceipt, abort bool) error {
	if !m.matchesArithmeticInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	// Only actually completed younger source declarations can be acknowledged
	// before the caller's older locals. Unjoined RAW body roots remain for true.
	if e := m.modular.cleanupPNormalizeInvocation(&s.normalize, false); e != nil {
		return e
	}
	if e := m.integer.cleanupAddMulInvocation(&s.addMul, false); e != nil {
		return e
	}
	if s == &m.constantScope.arithmeticScope {
		return m.cleanupConstantArithmeticInvocation(r, abort)
	}
	if !abort {
		return nil
	}
	// True is called after completed caller locals; exact failed constructors
	// and numeric leaves may now be aborted without replaying a source action.
	if s.calleeGeneration != 0 && m.monomials.creation.generation == s.calleeGeneration && m.monomials.creation.parent == m {
		if e := m.monomials.CleanupCreation(); e != nil {
			return e
		}
	}
	if e := m.monomials.cleanupMonomialAllocatorInvocation(&s.monomialAllocator, true); e != nil {
		return e
	}
	if s.transportBound {
		if e := m.integer.Del(&s.transport); e != nil {
			return e
		}
		s.transportBound = false
	}
	if e := s.numeric.cleanup(m.integer); e != nil {
		return e
	}
	if e := m.modular.cleanupPNormalizeInvocation(&s.normalize, true); e != nil {
		return e
	}
	if e := m.integer.cleanupAddMulInvocation(&s.addMul, true); e != nil {
		return e
	}
	if e := m.cleanupFactoryInvocation(&s.factory, true); e != nil {
		return e
	}
	if s.positionPublished {
		if e := s.buffer.positions.Reset(s.positionTerm); e != nil {
			return e
		}
		s.positionPublished = false
		s.positionTerm = Monomial{}
	}
	if e := s.buffer.cleanupAppendInvocation(&s.append, true); e != nil {
		return e
	}
	if s.generated != (Monomial{}) {
		if m.generatedActive && m.generatedMonomial == s.generated {
			if e := m.cleanupGeneratedMonomial(); e != nil {
				return e
			}
			s.generated = Monomial{}
		} else if e := m.monomials.CleanupValue(&s.generated); e != nil {
			return e
		}
	}
	s.active = false
	return nil
}

// The source returns this constructor result directly. Acknowledge the exact
// completed buffer transfer before exposing the token; failed acknowledgment
// leaves the actual child result owned for explicit abort.
func (b *polynomialSOM) mkWithArithmeticReceipt(normalize bool, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if receipt == nil {
		return b.mk(normalize)
	}
	m := b.owner
	if !m.matchesArithmeticInvocation(receipt) || receipt.frame.buffer != b || receipt.frame.executing {
		return Polynomial{}, ErrPolynomialOwner
	}
	s := receipt.frame
	s.executing = true
	p, e := b.mkWithAppendReceipt(normalize, &s.append, &s.factory)
	s.executing = false
	if e != nil {
		s.bodyCause = e
		return Polynomial{}, e
	}
	if e = m.cleanupArithmeticInvocation(receipt, false); e != nil {
		return Polynomial{}, e
	}
	if m.matchesFactoryInvocation(&s.factory) {
		if e = b.acknowledgeFactoryAppendTransfer(s.factory.invocation); e != nil {
			return Polynomial{}, e
		}
		if e = m.transferFactoryResult(&s.factory, p); e != nil {
			return Polynomial{}, e
		}
	}
	s.active = false
	return p, nil
}

// Actual INC/owning token consumption acknowledges only the recorded creation
// of this exact invocation. Recycled slots and later scope reuse cannot match.
func (l *monomialLocation) acknowledgeArithmeticProduct(a Monomial) {
	if d := l.arithmeticCohort; d != nil && d.buffer != nil && d.buffer.owner != nil && d.buffer.owner.ownsArithmeticFrame(d) && d.invocation == l.arithmeticInvocation && d.generated == a {
		d.generated = Monomial{}
	}
	l.arithmeticCohort = nil
	l.arithmeticInvocation = 0
}

// An executing allocation has not yet returned its concrete lower outcome.
// Owning callbacks must not retire the embedded caller before that publication.
func (m *MultivariatePolynomialCore) arithmeticExecuting() bool {
	if m == nil || m.self != m {
		return false
	}
	// A direct ExactDiv NUM is executing before any R.add arithmetic child.
	// Whole-owner destruction must refuse at that real caller frontier too.
	if s := &m.exactDivision; s.active && s.executing {
		return true
	}
	if s := &m.constantScope.arithmeticScope; s.active && s.executing {
		return true
	}
	for _, b := range [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		if b.arithmeticScope.active && b.arithmeticScope.executing {
			return true
		}
	}
	return false
}

// A literal source reset consumes the published member rows before a later
// invocation begins. Its actual captured frame/nonce may acknowledge metadata
// only after that action completes; it never aborts an unjoined numeric child.
func (b *polynomialSOM) acknowledgeArithmeticSourceReset(r *polynomialArithmeticInvocationReceipt) {
	m := b.owner
	if !m.matchesArithmeticInvocation(r) || r.frame.buffer != b || r.frame.executing {
		return
	}
	s := r.frame
	if s.sourceCause != nil || s.transportBound || s.positionPublished || s.singleton != (Monomial{}) || s.generated != (Monomial{}) || s.calleeGeneration != 0 || (m.monomials.matchesMonomialAllocatorInvocation(&s.monomialAllocator) && s.monomialAllocator.outcome.Pending != (memory.Block{})) || s.numeric.cell.object.OwnerID() != 0 || s.numeric.allocator != ([3]memory.Block{}) || m.matchesFactoryInvocation(&s.factory) || m.modular.matchesPNormalizeInvocation(&s.normalize) || m.integer.matchesAddMulInvocation(&s.addMul) {
		return
	}
	a := &b.appendScope
	if a.active {
		if !b.matchesAppendInvocation(&s.append) || a.pending != (Monomial{}) || a.movingCoefficient != (Int{}) || a.numeric.cell.object.OwnerID() != 0 || a.numeric.allocator != ([3]memory.Block{}) || a.coefficientClaims != 0 || a.monomialClaims != 0 || a.retirePhase != 0 {
			return
		}
		// Claim counters here refuse unsafe acknowledgement; the authority is the
		// captured exact invocation and the just-completed literal reset actions.
		*a = polynomialCheapAppendScope{invocation: a.invocation}
	}
	s.active = false
}
