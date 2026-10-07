// Source: src/math/polynomial/polynomial.cpp:7439-7476; src/util/buffer.h;
// src/util/scoped_numeral_buffer.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

// Actual lexical buffers are embedded in the charged class object. The cold
// orphan backing records only newly returned zero-reference monomials; it is a
// Go owning-abort adaptation, not a C++ allocation or pointer-buffer ownership.
type polynomialGlobalConversionScope struct {
	ms                                   skeletonMonomialBuffer
	as                                   util.ScopedIntegerModularBuffer128
	orphans                              []Monomial
	result                               Polynomial
	monomialRaw                          [3]util.Block
	factoryGeneration, calleeGeneration  uint64
	active, executing, ownsResult        bool
	bodyCause, sourceCause, cleanupCause error
}

func ConvertPolynomial(sm *MultivariatePolynomialManager, p Polynomial, tm *MultivariatePolynomialManager, x, maxD uint32) (out Polynomial, cause error) {
	if e := sm.ready(); e != nil {
		return Polynomial{}, e
	}
	if e := tm.ready(); e != nil {
		return Polynomial{}, e
	}
	if e := tm.beginConversion(); e != nil {
		return Polynomial{}, e
	}
	s := &tm.conversion
	defer func() {
		cause = tm.finishConversion(cause)
		if cause != nil {
			out = Polynomial{}
			return
		}
		s.result = Polynomial{}
		s.ownsResult = false
		s.active = false
	}()
	n, e := sm.core.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if sm == tm {
		return p, nil
	}
	for i := uint32(0); i < n; i++ {
		term, e := sm.core.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		if x != ^uint32(0) {
			degree, e := sm.core.monomials.DegreeOf(term, x)
			if e != nil {
				return Polynomial{}, e
			}
			if degree > maxD {
				continue
			}
		}
		if sm.core.monomials != tm.core.monomials {
			term, e = tm.conversionMonomial(sm.core.monomials, term)
			if e != nil {
				return Polynomial{}, e
			}
		}
		if e = s.ms.push(tm.core, term); e != nil {
			return Polynomial{}, e
		}
		zero := util.Int{}
		if e = s.as.Push(&zero); e != nil {
			return Polynomial{}, e
		}
		coefficient, e := sm.core.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		if e = tm.conversionSet(s.as.Count()-1, sm.core.integer, &coefficient); e != nil {
			return Polynomial{}, e
		}
	}
	av, e := s.as.View()
	if e != nil {
		return Polynomial{}, e
	}
	mv := monomialArrayView{inline: &s.ms.initial, count: s.ms.size}
	if s.ms.raw != (util.Block{}) {
		mv = monomialArrayView{runtime: tm.core.memory, raw: s.ms.raw, stride: 16, count: s.ms.size}
	}
	if e = tm.core.som.reset(); e != nil {
		return Polynomial{}, e
	}
	return tm.core.mkTermsSource(s.as.Count(), polynomialUnivariateCoefficients{array: av, indexed: true}, mv)
}
func (m *MultivariatePolynomialManager) beginConversion() error {
	s := &m.conversion
	if s.active || m.conversionHasOrphans() {
		return ErrPolynomialPending
	}
	old := s.orphans
	*s = polynomialGlobalConversionScope{orphans: old[:0], active: true, executing: true}
	s.ms.capacity = 128            // first source constructor
	s.as.Construct(m.core.modular) // second source constructor
	m.core.conversionOwner = s
	return nil
}
func (m *MultivariatePolynomialManager) conversionHasOrphans() bool {
	for _, a := range m.conversion.orphans {
		if a != (Monomial{}) {
			return true
		}
	}
	return false
}
func (m *MultivariatePolynomialManager) conversionSet(i uint32, source *util.IntegerManager, v *util.Int) (err error) {
	b := &m.conversion.as
	a, e := b.Physical(i)
	if e != nil {
		return e
	}
	before, e := util.CaptureNumericAllocationRoots(m.core.integer)
	if e != nil {
		return e
	}
	err = m.core.modular.CopyFromOwner(&a, source, v)
	err = polynomialNewtonJoin(err, polynomialNewtonJoin(b.AllocationRoots().After(m.core.integer, before), b.Store(i, a)))
	if err != nil && b.BodyCause() == nil {
		b.RecordBodyCause(err)
	}
	return err
}
func (m *MultivariatePolynomialManager) conversionMonomial(source *MonomialManager, a Monomial) (Monomial, error) {
	s := &m.conversion
	c := m.core
	if len(s.orphans) == cap(s.orphans) {
		n := cap(s.orphans)*2 + 1
		bytes := uint64(n) * uint64(reflect.TypeOf(Monomial{}).Size())
		if e := m.originalBudget.Reserve(bytes); e != nil {
			return Monomial{}, e
		}
		v := make([]Monomial, len(s.orphans), n)
		copy(v, s.orphans)
		m.originalBudget.Release(uint64(cap(s.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size()))
		s.orphans = v
	}
	before, e := c.monomials.allocator.PendingRoots()
	if e != nil {
		return Monomial{}, e
	}
	outcome := monomialFactoryOutcome{}
	term, e := c.monomials.convertCreated(source, a, &outcome)
	if outcome.started && c.monomials.creation.generation == outcome.generation {
		s.calleeGeneration = outcome.generation
		c.monomials.creation.parent = c
	}
	after, probe := c.monomials.allocator.PendingRoots()
	if before == ([3]util.Block{}) {
		s.monomialRaw = after
	}
	if e != nil || probe != nil {
		return Monomial{}, polynomialNewtonJoin(e, probe)
	}
	if outcome.created {
		i := len(s.orphans)
		s.orphans = append(s.orphans, term)
		l, e := c.monomials.location(term)
		if e != nil {
			return Monomial{}, e
		}
		l.conversionCohort = s
		l.conversionSlot = uint32(i)
	}
	return term, nil
}
func (m *MultivariatePolynomialManager) finishConversion(body error) error {
	s := &m.conversion
	s.executing = false
	if s.bodyCause == nil {
		s.bodyCause = body
	}
	if s.sourceCause != nil {
		return polynomialNewtonJoin(body, s.sourceCause)
	}
	e := s.as.SourceClose()
	if e == nil && s.ms.raw != (util.Block{}) {
		e = m.core.memory.Deallocate(&s.ms.raw)
	}
	if e != nil {
		s.sourceCause = e
	}
	return polynomialNewtonJoin(body, e)
}
func (m *MultivariatePolynomialManager) CleanupConversion() error {
	if e := m.identity(); e != nil {
		return e
	}
	s := &m.conversion
	c := m.core
	if s.executing {
		return ErrPolynomialPending
	}
	fail := func(e error) error { s.cleanupCause = e; return e }
	if s.factoryGeneration != 0 {
		if c.pending.generation == s.factoryGeneration && c.pending.conversionParent == s {
			if e := c.cleanupPendingCreation(); e != nil {
				return fail(e)
			}
		}
		s.factoryGeneration = 0
	}
	if s.calleeGeneration != 0 {
		if c.monomials.creation.generation == s.calleeGeneration && c.monomials.creation.parent == c {
			if e := c.monomials.CleanupCreation(); e != nil {
				return fail(e)
			}
		}
		s.calleeGeneration = 0
	}
	if s.ownsResult && s.result != (Polynomial{}) {
		if e := c.CleanupValue(&s.result); e != nil {
			return fail(e)
		}
		s.ownsResult = false
	}
	if e := s.as.Cleanup(); e != nil {
		return fail(e)
	}
	for _, r := range []*util.Block{&s.ms.raw, &s.ms.pending} {
		if *r != (util.Block{}) {
			if e := c.memory.Deallocate(r); e != nil {
				return fail(e)
			}
		}
	}
	for i := range s.orphans {
		if s.orphans[i] != (Monomial{}) {
			if e := c.monomials.CleanupValue(&s.orphans[i]); e != nil {
				return fail(e)
			}
		}
	}
	if s.monomialRaw != ([3]util.Block{}) {
		now, e := c.monomials.allocator.PendingRoots()
		if e != nil {
			return fail(e)
		}
		if now == s.monomialRaw {
			e = c.monomials.allocator.CleanupPending()
			after, x := c.monomials.allocator.PendingRoots()
			for i := range s.monomialRaw {
				if after[i] == (util.Block{}) {
					s.monomialRaw[i] = util.Block{}
				}
			}
			if e != nil || x != nil {
				return fail(polynomialNewtonJoin(e, x))
			}
		}
		s.monomialRaw = [3]util.Block{}
	}
	m.originalBudget.Release(uint64(cap(s.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size()))
	s.orphans = nil
	s.active = false
	s.cleanupCause = nil
	return nil
}
