// Source: src/math/polynomial/polynomial.cpp:1495-1509,5954-6068,7284-7286;
// src/util/scoped_numeral.h; src/util/obj_ref.h. Licensed under the MIT license.
// Author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

// Four actual scoped numerals, one RAW numeral, and the actual monomial_ref.
// The created cohort is charged Go abort metadata for returned zero-RC locals;
// it is not a source buffer or a source destructor repairing abandoned locals.
type polynomialSquareRootScope struct {
	invocation                           uint64
	orphanInvocations                    []uint64
	values                               [4]util.Int
	two                                  util.Int
	mask                                 uint8
	reference                            polynomialSquareRootReference
	referenceCompleted                   bool
	active, executing, stopped           bool
	stage                                uint8
	orphans                              []Monomial
	drain                                uint32
	perfectSquare                        util.IntegerPerfectSquareInvocationReceipt
	factoryGeneration, calleeGeneration  uint64
	numericPending                       util.ObjectRef
	integerRaw, monomialRaw              [3]util.Block
	freshResult                          Polynomial
	bodyCause, sourceCause, cleanupCause error
}

func (s *polynomialSquareRootScope) constructNumeral(i uint8) {
	s.values[i] = util.Int{}
	s.mask |= 1 << i
}

type polynomialSquareRootReference struct {
	value, incoming                       Monomial
	consumed, incomingConsumed, assigning bool
	cause                                 error
}

func (r *polynomialSquareRootReference) drop(c *MultivariatePolynomialCore, p *Monomial, consumed *bool, owning bool) error {
	if *p == (Monomial{}) {
		return nil
	}
	if *consumed {
		if !owning {
			return r.cause
		}
		if e := c.monomials.CleanupValue(p); e != nil {
			return e
		}
		*consumed = false
		return nil
	}
	out := monomialDecrementOutcome{}
	e := c.monomials.decRefCoreOutcome(*p, &out)
	if out.written {
		if out.last && e != nil {
			*consumed = true
		} else {
			*p = Monomial{}
		}
	}
	if e != nil {
		r.cause = e
		return e
	}
	*p = Monomial{}
	return nil
}
func (r *polynomialSquareRootReference) assign(c *MultivariatePolynomialCore, p Monomial) error {
	if r.assigning {
		return ErrPolynomialPending
	}
	if e := c.monomials.IncRef(p); e != nil {
		return e
	}
	r.incoming = p
	r.assigning = true
	if e := r.drop(c, &r.value, &r.consumed, false); e != nil {
		return e
	}
	r.value = r.incoming
	r.incoming = Monomial{}
	r.assigning = false
	return nil
}
func (m *MultivariatePolynomialCore) gradedLexMinPosition(p Polynomial) (uint32, error) {
	n, e := m.readWord(p, 8)
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return ^uint32(0), nil
	}
	min := uint32(0)
	for i := uint32(1); i < n; i++ {
		a, e := m.monomialCore(p, i)
		if e != nil {
			return 0, e
		}
		b, e := m.monomialCore(p, min)
		if e != nil {
			return 0, e
		}
		cmp, e := m.monomials.GradedLex(a, b)
		if e != nil {
			return 0, e
		}
		if cmp < 0 {
			min = i
		}
	}
	return min, nil
}
func (m *MultivariatePolynomialManager) reserveSquareRootCreated() error {
	s := &m.squareRoot
	if len(s.orphans) < cap(s.orphans) {
		return nil
	}
	n := cap(s.orphans)*2 + 1
	stride := uint64(reflect.TypeOf(Monomial{}).Size()) + 8
	if e := m.core.integer.Budget().Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]Monomial, len(s.orphans), n)
	copy(next, s.orphans)
	tags := make([]uint64, len(s.orphanInvocations), n)
	copy(tags, s.orphanInvocations)
	m.core.integer.Budget().Release(uint64(cap(s.orphans)) * stride)
	s.orphans = next
	s.orphanInvocations = tags
	return nil
}
func (m *MultivariatePolynomialManager) squareRootMonomial(op uint8, a, b Monomial) (Monomial, bool, error) {
	if e := m.reserveSquareRootCreated(); e != nil {
		return Monomial{}, false, e
	}
	s := &m.squareRoot
	c := m.core
	before, e := c.monomials.allocator.PendingRoots()
	if e != nil {
		return Monomial{}, false, e
	}
	outcome := monomialFactoryOutcome{}
	var p Monomial
	ok := true
	switch op {
	case 0:
		p, ok, e = c.monomials.sqrtCreated(a, &outcome)
	case 1:
		p, ok, e = c.monomials.divideCreated(a, b, &outcome)
	case 2:
		p, e = c.monomials.mulCreated(a, b, &outcome)
	}
	if e != nil && outcome.started {
		c.recordGeneratedCallee(outcome)
		s.calleeGeneration = outcome.generation
	}
	if e == nil && outcome.created {
		idx := uint32(len(s.orphans))
		s.orphans = append(s.orphans, p)
		s.orphanInvocations = append(s.orphanInvocations, s.invocation)
		l := &c.monomials.locations[p.slot-1]
		l.squareRootCohort = s
		l.squareRootSlot = idx
	}
	if before == ([3]util.Block{}) {
		after, x := c.monomials.allocator.PendingRoots()
		if after != ([3]util.Block{}) {
			s.monomialRaw = after
		}
		if x != nil {
			return p, ok, errors.Join(e, x)
		}
	}
	return p, ok, e
}

// Receipt capture surrounds only the actual VM call, not retained global state.
func (m *MultivariatePolynomialManager) squareRootNumeral(op uint8, a, b, out *util.Int) (bool, error) {
	c := m.core
	z := c.integer
	s := &m.squareRoot
	pending := z.PendingAllocationObject()
	raw, e := z.SourceAllocator().PendingRoots()
	if e != nil {
		return false, e
	}
	ok := true
	switch op {
	case 0:
		ok, e = c.modular.IsPerfectSquareWithReceipt(a, out, &s.perfectSquare)
	case 1:
		e = c.modular.SetInt32(out, 2)
	case 2:
		e = c.modular.Mul(a, b, out)
	case 3:
		e = c.modular.Neg(out)
	case 4:
		e = c.modular.Div(a, b, out)
	}
	if pending == (util.ObjectRef{}) && z.PendingAllocationObject() != (util.ObjectRef{}) {
		s.numericPending = z.PendingAllocationObject()
	}
	if raw == ([3]util.Block{}) {
		after, x := z.SourceAllocator().PendingRoots()
		if after != ([3]util.Block{}) {
			s.integerRaw = after
		}
		if x != nil {
			return ok, errors.Join(e, x)
		}
	}
	return ok, e
}

// The actual persistent buffer keeps its own coefficient fields. Only a newly
// started subordinate physical allocation is an invocation's abort receipt.
func (m *MultivariatePolynomialManager) squareRootAdd(b *polynomialSOM, a *util.Int, t Monomial) error {
	z := m.core.integer
	s := &m.squareRoot
	pending := z.PendingAllocationObject()
	raw, e := z.SourceAllocator().PendingRoots()
	if e != nil {
		return e
	}
	e = b.add(a, t, false)
	if pending == (util.ObjectRef{}) && z.PendingAllocationObject() != (util.ObjectRef{}) {
		s.numericPending = z.PendingAllocationObject()
	}
	if raw == ([3]util.Block{}) {
		after, x := z.SourceAllocator().PendingRoots()
		if after != ([3]util.Block{}) {
			s.integerRaw = after
		}
		if x != nil {
			return errors.Join(e, x)
		}
	}
	return e
}
func (m *MultivariatePolynomialManager) squareRootTerm(p Polynomial, i uint32) (bool, error) {
	t, e := m.core.monomialCore(p, i)
	if e != nil {
		return false, e
	}
	ok, e := m.core.monomials.IsSquare(t)
	if e != nil || !ok {
		return ok, e
	}
	a, e := m.core.coefficientCore(p, i)
	if e != nil {
		return false, e
	}
	return m.squareRootNumeral(0, &a, nil, &m.squareRoot.values[0])
}
func (m *MultivariatePolynomialManager) finishSquareRoot(cause error) error {
	s := &m.squareRoot
	s.executing = false
	if s.bodyCause == nil {
		s.bodyCause = cause
	}
	z := m.core.integer
	if _, _, stopped := s.perfectSquare.State(z); stopped {
		s.stopped = true
		if s.sourceCause == nil {
			s.sourceCause = s.perfectSquare.SourceCause()
		}
	}
	if s.stopped {
		return errors.Join(cause, s.sourceCause)
	}
	if s.referenceCompleted {
		if e := s.reference.drop(m.core, &s.reference.value, &s.reference.consumed, false); e != nil {
			s.stopped = true
			s.sourceCause = e
			return errors.Join(cause, e)
		}
		s.referenceCompleted = false
	}
	for s.stage < 4 {
		i := uint8(3) - s.stage
		if s.mask&(1<<i) != 0 {
			if _, e := upolynomialSourceNumeralDel(z, &s.values[i]); e != nil {
				s.stopped = true
				s.sourceCause = e
				return errors.Join(cause, e)
			}
			s.mask &^= 1 << i
		}
		s.stage++
	}
	if cause == nil && s.freshResult == (Polynomial{}) && s.factoryGeneration == 0 && s.calleeGeneration == 0 && s.numericPending == (util.ObjectRef{}) && s.integerRaw == ([3]util.Block{}) && s.monomialRaw == ([3]util.Block{}) {
		s.active = false
	}
	return cause
}
func (m *MultivariatePolynomialManager) beginSquareRoot() error {
	return m.beginSquareRootWithReceipt(nil)
}
func (m *MultivariatePolynomialManager) beginSquareRootWithReceipt(receipt *polynomialSqrtInvocationReceipt) error {
	s := &m.squareRoot
	if s.active {
		return ErrPolynomialPending
	}
	if s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	nonce := s.invocation + 1
	orphans, tags, drain := s.orphans, s.orphanInvocations, s.drain
	live := false
	for _, a := range orphans {
		if a != (Monomial{}) {
			live = true
			break
		}
	}
	if !live {
		orphans = orphans[:0]
		tags = tags[:0]
		drain = 0
	}
	*s = polynomialSquareRootScope{invocation: nonce, active: true, executing: true, orphans: orphans, orphanInvocations: tags, drain: drain, mask: 1}
	m.core.squareRootOwner = s
	if receipt != nil {
		*receipt = polynomialSqrtInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	}
	return nil
}
func (m *MultivariatePolynomialManager) Sqrt(p Polynomial, r *PolynomialRef) (bool, error) {
	return m.sqrtWithReceipt(p, r, nil)
}
func (m *MultivariatePolynomialManager) sqrtWithReceipt(p Polynomial, r *PolynomialRef, receipt *polynomialSqrtInvocationReceipt) (ok bool, cause error) {
	if e := m.ready(); e != nil {
		return false, e
	}
	if r == nil || r.self != r || r.owner != m.core || r.closed || r.closing {
		return false, ErrPolynomialOwner
	}
	c := m.core
	n, e := c.readWord(p, 8)
	if e != nil {
		return false, e
	}
	if n == 0 {
		return true, r.Assign(p)
	}
	s := &m.squareRoot
	if s.active {
		return false, ErrPolynomialPending
	}
	if e = m.beginSquareRootWithReceipt(receipt); e != nil {
		return false, e
	}
	defer func() { cause = m.finishSquareRoot(cause) }()
	min, e := c.gradedLexMinPosition(p)
	if e != nil {
		return false, e
	}
	ok, e = m.squareRootTerm(p, min)
	if e != nil || !ok {
		return false, e
	}
	max, e := c.gradedLexMaxPosition(p)
	if e != nil {
		return false, e
	}
	ok, e = m.squareRootTerm(p, max)
	if e != nil || !ok {
		return false, e
	}
	term, e := c.monomialCore(p, max)
	if e != nil {
		return false, e
	}
	m1, ok, e := m.squareRootMonomial(0, term, Monomial{})
	if e != nil {
		return false, e
	}
	if polynomialSourceDebug && !ok {
		return false, ErrPolynomialAssertion
	}
	R, C := &c.som, &c.som2
	if e = R.reset(); e != nil {
		return false, e
	}
	if e = C.reset(); e != nil {
		return false, e
	}
	if _, e = m.squareRootNumeral(1, nil, nil, &s.two); e != nil {
		return false, e
	}
	s.constructNumeral(1)
	if _, e = m.squareRootNumeral(2, &s.values[0], &s.two, &s.values[1]); e != nil {
		return false, e
	}
	if e = m.squareRootAdd(R, &s.values[0], m1); e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		if i == max {
			continue
		}
		a, e := c.coefficientCore(p, i)
		if e != nil {
			return false, e
		}
		t, e := c.monomialCore(p, i)
		if e != nil {
			return false, e
		}
		if e = m.squareRootAdd(C, &a, t); e != nil {
			return false, e
		}
	}
	s.constructNumeral(2)
	s.constructNumeral(3)
	s.referenceCompleted = true
	for {
		s.executing = false
		e = c.checkpoint()
		s.executing = true
		if e != nil {
			return false, e
		}
		pos, e := C.gradedLexMaxPosition()
		if e != nil {
			return false, e
		}
		if pos == ^uint32(0) {
			if e = C.reset(); e != nil {
				return false, e
			}
			out, e := R.mk(false)
			if e != nil {
				return false, e
			}
			if e = r.Assign(out); e != nil {
				return false, e
			}
			if s.freshResult == out {
				s.freshResult = Polynomial{}
			}
			return true, nil
		}
		t, e := C.ms.monomial(c, pos)
		if e != nil {
			return false, e
		}
		mi, divides, e := m.squareRootMonomial(1, t, m1)
		if e != nil {
			return false, e
		}
		if !divides {
			if e = R.reset(); e != nil {
				return false, e
			}
			if e = C.reset(); e != nil {
				return false, e
			}
			return false, nil
		}
		coefficient, e := C.as.coefficient(c, pos)
		if e != nil {
			return false, e
		}
		divides, e = c.modular.Divides(&s.values[1], &coefficient)
		if e != nil {
			return false, e
		}
		if !divides {
			return false, nil
		}
		if _, e = m.squareRootNumeral(4, &coefficient, &s.values[1], &s.values[2]); e != nil {
			return false, e
		}
		count, e := R.as.size(c)
		if e != nil {
			return false, e
		}
		for j := uint32(0); j < count; j++ {
			a, e := R.as.coefficient(c, j)
			if e != nil {
				return false, e
			}
			zero, e := c.modular.IsZero(&a)
			if e != nil {
				return false, e
			}
			if zero {
				continue
			}
			if _, e = m.squareRootNumeral(2, &a, &s.values[2], &s.values[3]); e != nil {
				return false, e
			}
			if _, e = m.squareRootNumeral(2, &s.values[3], &s.two, &s.values[3]); e != nil {
				return false, e
			}
			if _, e = m.squareRootNumeral(3, nil, nil, &s.values[3]); e != nil {
				return false, e
			}
			t, e := R.ms.monomial(c, j)
			if e != nil {
				return false, e
			}
			product, _, e := m.squareRootMonomial(2, t, mi)
			if e != nil {
				return false, e
			}
			if e = s.reference.assign(c, product); e != nil {
				return false, e
			}
			if e = m.squareRootAdd(C, &s.values[3], s.reference.value); e != nil {
				return false, e
			}
		}
		if _, e = m.squareRootNumeral(2, &s.values[2], &s.values[2], &s.values[3]); e != nil {
			return false, e
		}
		if _, e = m.squareRootNumeral(3, nil, nil, &s.values[3]); e != nil {
			return false, e
		}
		product, _, e := m.squareRootMonomial(2, mi, mi)
		if e != nil {
			return false, e
		}
		if e = s.reference.assign(c, product); e != nil {
			return false, e
		}
		if e = m.squareRootAdd(C, &s.values[3], s.reference.value); e != nil {
			return false, e
		}
		if e = m.squareRootAdd(R, &s.values[2], mi); e != nil {
			return false, e
		}
	}
}

// Explicit owning abort, independent of literal stopped source destruction.
func (m *MultivariatePolynomialManager) CleanupSqrt() error {
	return m.cleanupSqrtModeOwned(true, 0, &m.squareRoot.drain)
}
func (m *MultivariatePolynomialManager) cleanupSqrtModeOwned(abort bool, invocation uint64, cursor *uint32) error {
	if e := m.identity(); e != nil {
		return e
	}
	s := &m.squareRoot
	if !s.active && len(s.orphans) == 0 {
		return nil
	}
	if m.core == nil {
		return nil
	}
	c := m.core
	if s.executing {
		return ErrPolynomialPending
	}
	active, endEntered, _ := s.perfectSquare.State(c.integer)
	if active && (abort || endEntered) {
		if e := c.integer.CleanupPerfectSquareInvocation(&s.perfectSquare, true); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	active, _, _ = s.perfectSquare.State(c.integer)
	if abort || !active {
		s.perfectSquare = util.IntegerPerfectSquareInvocationReceipt{}
	}
	if e := s.reference.drop(c, &s.reference.value, &s.reference.consumed, true); e != nil {
		s.cleanupCause = e
		return e
	}
	if abort {
		if e := s.reference.drop(c, &s.reference.incoming, &s.reference.incomingConsumed, true); e != nil {
			s.cleanupCause = e
			return e
		}
		s.reference.assigning = false
	}
	s.referenceCompleted = false
	for i := 3; i >= 0; i-- {
		if e := c.integer.Del(&s.values[i]); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	s.mask = 0
	if !abort {
		return nil
	}
	if e := c.integer.Del(&s.two); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.freshResult != (Polynomial{}) {
		if e := c.CleanupValue(&s.freshResult); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	if s.factoryGeneration != 0 {
		if c.pending.generation == s.factoryGeneration && c.pending.squareRootParent == s {
			if e := c.cleanupPendingCreation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.factoryGeneration = 0
	}
	if s.calleeGeneration != 0 {
		if c.monomials.creation.generation == s.calleeGeneration && c.monomials.creation.parent == c {
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
	if e := m.cleanupSqrtOrphanInvocation(invocation, cursor); e != nil {
		return e
	}
	if s.numericPending != (util.ObjectRef{}) {
		if c.integer.PendingAllocationObject() == s.numericPending {
			if e := c.integer.CleanupPendingAllocation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.numericPending = util.ObjectRef{}
	}
	for i, allocator := range [2]*util.SmallAllocator{c.integer.SourceAllocator(), c.monomials.allocator} {
		receipt := &s.integerRaw
		if i == 1 {
			receipt = &s.monomialRaw
		}
		if *receipt == ([3]util.Block{}) {
			continue
		}
		now, e := allocator.PendingRoots()
		if e != nil {
			return e
		}
		if now == *receipt {
			e = allocator.CleanupPending()
			after, x := allocator.PendingRoots()
			for j := range receipt {
				if after[j] == (util.Block{}) {
					receipt[j] = util.Block{}
				}
			}
			if e != nil {
				s.cleanupCause = e
				return e
			}
			if x != nil {
				return x
			}
		}
		*receipt = [3]util.Block{}
	}
	s.active = false
	s.cleanupCause = nil
	return nil
}
func (m *MultivariatePolynomialManager) closeSquareRootMetadata() error {
	if e := m.CleanupSqrt(); e != nil {
		return e
	}
	s := &m.squareRoot
	m.originalBudget.Release(uint64(cap(s.orphans))*uint64(reflect.TypeOf(Monomial{}).Size()) + uint64(cap(s.orphanInvocations))*8)
	s.orphans = nil
	s.orphanInvocations = nil
	s.drain = 0
	return nil
}
