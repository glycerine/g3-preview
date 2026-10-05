// Source: src/math/polynomial/polynomial.cpp:4507-4528,7228-7229;
// src/util/scoped_numeral.h. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// These are the actual two lexical numeral fields and known creation receipts.
// Orphans retain only genuinely newly-created zero-coefficient results which
// the source cheap buffer never INCs. This is owning Go metadata, not source
// arithmetic storage or a source destructor which repairs native leaks.
type polynomialDerivativeScope struct {
	invocation                           uint64
	n, a                                 Int
	constructed                          uint8
	stage                                uint8
	active, executing, stopped           bool
	generated                            Monomial
	calleeGeneration                     uint64
	factoryGeneration                    uint64
	numericPending                       memory.ObjectRef
	allocatorPending                     [3]memory.Block
	orphans                              []Monomial
	orphanInvocations                    []uint64
	drain                                uint32
	bodyCause, sourceCause, cleanupCause error
}

func (s *polynomialDerivativeScope) Error() string {
	return "polynomial derivative retained lexical owner"
}
func (s *polynomialDerivativeScope) Unwrap() error {
	if s.bodyCause != nil {
		return s.bodyCause
	}
	return s.sourceCause
}

func (m *MonomialManager) derivativeCreated(a Monomial, x uint32, out *monomialFactoryOutcome) (Monomial, error) {
	return m.variableTransformCreated(a, x, 0, 2, out)
}
func (m *MultivariatePolynomialManager) finishDerivativeNumerals() error {
	s := &m.derivative
	if s.stopped {
		return s.sourceCause
	}
	for s.stage < 2 {
		a := &s.a
		mask := uint8(2)
		if s.stage == 1 {
			a = &s.n
			mask = 1
		}
		if s.constructed&mask != 0 {
			_, e := upolynomialSourceNumeralDel(m.core.integer, a)
			if e != nil {
				s.stopped = true
				s.sourceCause = e
				return e
			}
			s.constructed &^= mask
		}
		s.stage++
	}
	return nil
}
func (m *MultivariatePolynomialManager) prepareDerivativeRetirement() error {
	s := &m.derivative
	if len(s.orphans) == cap(s.orphans) {
		c := cap(s.orphans)*2 + 1
		stride := uint64(reflect.TypeOf(Monomial{}).Size()) + 8
		if e := m.core.integer.budget.Reserve(uint64(c) * stride); e != nil {
			return e
		}
		next := make([]Monomial, len(s.orphans), c)
		copy(next, s.orphans)
		tags := make([]uint64, len(s.orphanInvocations), c)
		copy(tags, s.orphanInvocations)

		m.core.integer.budget.Release(uint64(cap(s.orphans)) * stride)
		s.orphans = next
		s.orphanInvocations = tags
	}
	return nil
}
func (m *MultivariatePolynomialManager) retainDerivativeGenerated() {
	s := &m.derivative
	if s.generated != (Monomial{}) {
		slot := uint32(len(s.orphans))
		l := &m.core.monomials.locations[s.generated.slot-1]
		l.derivativeCohort = s
		l.derivativeSlot = slot
		s.orphans = append(s.orphans, s.generated)
		s.orphanInvocations = append(s.orphanInvocations, s.invocation)
		s.generated = Monomial{}
	}
}
func (m *MultivariatePolynomialManager) derivativeHasOrphans() bool {
	for _, a := range m.derivative.orphans {
		if a != (Monomial{}) {
			return true
		}
	}
	return false
}
func (m *MultivariatePolynomialManager) transferDerivativeCreated(a Monomial) {
	s := &m.derivative
	if s.generated == a {
		s.generated = Monomial{}
	}
	for i, b := range s.orphans {
		if a == b {
			s.orphans[i] = Monomial{}
			s.orphanInvocations[i] = 0
		}
	}
}
func (m *MultivariatePolynomialManager) Derivative(p Polynomial, x uint32) (Polynomial, error) {
	return m.derivativeWithReceipt(p, x, nil)
}
func (m *MultivariatePolynomialManager) derivativeWithReceipt(p Polynomial, x uint32, receipt *polynomialDerivativeInvocationReceipt) (result Polynomial, cause error) {
	if e := m.ready(); e != nil {
		return Polynomial{}, e
	}
	s := &m.derivative
	if s.active {
		return Polynomial{}, ErrPolynomialPending
	}
	c := m.core
	if modularSourceDebug {
		if x >= c.monomials.nextVar {
			return Polynomial{}, ErrPolynomialAssertion
		}
		size, e := c.cheap.ms.size(c)
		if e != nil {
			return Polynomial{}, e
		}
		if size != 0 {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	size, e := c.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if s.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	s.invocation++
	if receipt != nil {
		*receipt = polynomialDerivativeInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	}
	s.active = true
	s.executing = true
	c.derivativeOwner = s
	s.bodyCause = nil
	s.sourceCause = nil
	s.cleanupCause = nil
	s.stopped = false
	defer func() {
		s.executing = false
		s.bodyCause = cause
		if e := m.finishDerivativeNumerals(); e != nil {
			cause = s
			return
		}
		if cause == nil && s.generated == (Monomial{}) && s.calleeGeneration == 0 {
			s.active = false
		}
	}()
	for i := uint32(0); i < size; i++ {
		term, e := c.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		d, e := c.monomials.DegreeOf(term, x)
		if e != nil {
			return Polynomial{}, e
		}
		if d == 0 {
			continue
		}
		// Source constructs n, sets its degree, then constructs a.
		s.n = Int{}
		s.a = Int{}
		s.constructed = 1
		s.stage = 0
		beforePending := c.integer.pendingCell.object
		beforeRaw, e := c.integer.allocator.PendingRoots()
		if e != nil {
			return Polynomial{}, e
		}
		e = c.modular.SetUint32(&s.n, d)
		pendingError := m.recordDerivativeNumericPending(beforePending, beforeRaw)
		if e != nil {
			return Polynomial{}, e
		}
		if pendingError != nil {
			return Polynomial{}, pendingError
		}
		s.constructed |= 2
		coefficient, e := c.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		beforePending = c.integer.pendingCell.object
		beforeRaw, e = c.integer.allocator.PendingRoots()
		if e != nil {
			return Polynomial{}, e
		}
		e = c.modular.Mul(&coefficient, &s.n, &s.a)
		pendingError = m.recordDerivativeNumericPending(beforePending, beforeRaw)
		if e != nil {
			return Polynomial{}, e
		}
		if pendingError != nil {
			return Polynomial{}, pendingError
		}
		if e = m.prepareDerivativeRetirement(); e != nil {
			return Polynomial{}, e
		}
		outcome := monomialFactoryOutcome{}
		derivative, e := c.monomials.derivativeCreated(term, x, &outcome)
		if e != nil {
			if outcome.started {
				c.recordGeneratedCallee(outcome)
				s.calleeGeneration = outcome.generation
			}
			return Polynomial{}, e
		}
		if outcome.created {
			s.generated = derivative
		}
		consumed := false
		e = c.cheap.addConsumed(&s.a, derivative, true, &consumed)
		if consumed {
			m.transferDerivativeCreated(derivative)
		}
		if e != nil {
			return Polynomial{}, e
		}
		m.retainDerivativeGenerated()
		if e = m.finishDerivativeNumerals(); e != nil {
			return Polynomial{}, e
		}
	}
	return c.cheap.mk(false)
}

func (m *MultivariatePolynomialManager) recordDerivativeNumericPending(before memory.ObjectRef, beforeRaw [3]memory.Block) error {
	s := &m.derivative
	if s.executing && before == (memory.ObjectRef{}) && m.core.integer.pendingCell.object != (memory.ObjectRef{}) {
		s.numericPending = m.core.integer.pendingCell.object
	}
	if s.executing && beforeRaw == ([3]memory.Block{}) {
		roots, e := m.core.integer.allocator.PendingRoots()
		if e != nil {
			return e
		}
		if roots != ([3]memory.Block{}) {
			s.allocatorPending = roots
		}
	}
	return nil
}

// Owning abort: only actual fields and matching factory continuation identities.
// No source destructor retry, no caller directory scan, no source counter repair.
func (m *MultivariatePolynomialManager) CleanupDerivative() error {
	return m.cleanupDerivativeMode(true)
}
func (m *MultivariatePolynomialManager) cleanupDerivativeMode(abort bool) error {
	return m.cleanupDerivativeModeOwned(abort, 0, &m.derivative.drain)
}
func (m *MultivariatePolynomialManager) cleanupDerivativeModeOwned(abort bool, invocation uint64, cursor *uint32) error {
	if e := m.identity(); e != nil {
		return e
	}
	s := &m.derivative
	if !s.active && !m.derivativeHasOrphans() && cap(s.orphans) == 0 {
		return nil
	}
	if s.executing {
		return ErrPolynomialPending
	}
	for _, a := range [2]*Int{&s.a, &s.n} {
		if e := m.core.integer.Del(a); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	s.constructed = 0
	if !abort {
		return nil
	}
	if s.generated != (Monomial{}) {
		if e := m.core.monomials.CleanupValue(&s.generated); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	if e := m.cleanupDerivativeOrphanInvocation(invocation, cursor); e != nil {
		return e
	}
	if s.calleeGeneration != 0 {
		c := m.core
		creation := &c.monomials.creation
		if creation.generation == s.calleeGeneration && creation.parent == c {
			if e := c.monomials.CleanupCreation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		if c.generatedCalleeGeneration == s.calleeGeneration {
			c.generatedCalleePending = false
		}
		s.calleeGeneration = 0
	}
	if s.factoryGeneration != 0 {
		p := &m.core.pending
		if p.generation == s.factoryGeneration && p.derivativeParent == s {
			if e := m.core.cleanupPendingCreation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.factoryGeneration = 0
	}
	if s.numericPending != (memory.ObjectRef{}) {
		if m.core.integer.pendingCell.object == s.numericPending {
			if e := m.core.integer.cleanupPendingAllocation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.numericPending = memory.ObjectRef{}
	}
	if s.allocatorPending != ([3]memory.Block{}) {
		current, e := m.core.integer.allocator.PendingRoots()
		if e != nil {
			s.cleanupCause = e
			return e
		}
		if current == s.allocatorPending {
			e = m.core.integer.allocator.CleanupPending()
			after, inspect := m.core.integer.allocator.PendingRoots()
			for i := range s.allocatorPending {
				if after[i] == (memory.Block{}) {
					s.allocatorPending[i] = memory.Block{}
				}
			}
			if e != nil {
				s.cleanupCause = e
				return e
			}
			if inspect != nil {
				s.cleanupCause = inspect
				return inspect
			}
		}
		// A changed generation belongs to a later caller; never sweep it.
		s.allocatorPending = [3]memory.Block{}
	}

	s.active = false
	m.releaseDerivativeRetirementIfEmpty()
	s.cleanupCause = nil
	return nil
}

// The tag is published with the actual created root at its append and cleared
// by the actual successful INC. It carries no additional authoritative token.
func (m *MultivariatePolynomialManager) cleanupDerivativeOrphanInvocation(invocation uint64, cursor *uint32) error {
	s := &m.derivative
	for *cursor < uint32(len(s.orphans)) {
		i := *cursor
		if (invocation == 0 || s.orphanInvocations[i] == invocation) && s.orphans[i] != (Monomial{}) {
			if e := m.core.monomials.CleanupValue(&s.orphans[i]); e != nil {
				s.cleanupCause = e
				return e
			}
			s.orphanInvocations[i] = 0
		}
		*cursor++
	}
	return nil
}
func (m *MultivariatePolynomialManager) releaseDerivativeRetirementIfEmpty() {
	s := &m.derivative
	if s.active || m.derivativeHasOrphans() {
		return
	}
	m.core.integer.budget.Release(uint64(cap(s.orphans))*uint64(reflect.TypeOf(Monomial{}).Size()) + uint64(cap(s.orphanInvocations))*8)
	s.orphans = nil
	s.orphanInvocations = nil
	s.drain = 0
}

func (m *MultivariatePolynomialManager) derivativeInvocationHasOrphans(r *polynomialDerivativeInvocationReceipt) bool {
	if r == nil || r.manager != m || r.frame != &m.derivative || r.invocation == 0 {
		return false
	}
	for i, p := range m.derivative.orphans {
		if p != (Monomial{}) && m.derivative.orphanInvocations[i] == r.invocation {
			return true
		}
	}
	return false
}
