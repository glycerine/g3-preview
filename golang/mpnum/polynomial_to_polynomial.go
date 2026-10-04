// Source: src/math/polynomial/polynomial.cpp:6832-6841,2582-2594;
// src/util/scoped_numeral_buffer.h; src/util/buffer.h.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// One actual per-invocation lexical buffer embedded in the reflected charged
// class header. Successful destruction permits reuse; stopped roots retain
// their exact source history until explicit owning abort.
type polynomialToPolynomialScope struct {
	buffer                               scopedIntegerModularBuffer128
	result                               Polynomial
	generated                            Monomial
	monomialRaw                          [3]memory.Block
	factoryGeneration, calleeGeneration  uint64
	active, executing, ownsResult        bool
	bodyCause, sourceCause, cleanupCause error
}

func (m *MultivariatePolynomialManager) ToPolynomial(sz uint32, as []Int, x uint32) (out Polynomial, cause error) {
	if e := m.ready(); e != nil {
		return Polynomial{}, e
	}
	c := m.core
	if sz == 0 {
		return c.zero, nil
	}
	if e := m.beginToPolynomial(); e != nil {
		return Polynomial{}, e
	}
	s := &m.toPolynomial
	defer func() {
		cause = m.finishToPolynomial(cause)
		if cause != nil {
			out = Polynomial{}
			return
		}
		s.result = Polynomial{}
		s.ownsResult = false
		s.active = false
	}()
	for i := uint32(0); i < sz; i++ {
		zero := Int{}
		if cause = s.buffer.push(&zero); cause != nil {
			return Polynomial{}, cause
		}
		if uint64(i) >= uint64(len(as)) {
			return Polynomial{}, arena.ErrBounds
		}
		if cause = s.buffer.set(i, &as[i]); cause != nil {
			return Polynomial{}, cause
		}
	}
	view, e := s.buffer.view()
	if e != nil {
		return Polynomial{}, e
	}
	p, e := c.mkUnivariateSource(x, sz-1, polynomialUnivariateCoefficients{array: view, indexed: true})
	if e != nil {
		return Polynomial{}, e
	}
	return p, nil
}
func (m *MultivariatePolynomialManager) beginToPolynomial() error {
	s := &m.toPolynomial
	if s.active {
		return ErrPolynomialPending
	}
	*s = polynomialToPolynomialScope{active: true, executing: true}
	m.core.toPolynomialOwner = s
	s.buffer.construct(m.core.modular)
	return nil
}
func (m *MultivariatePolynomialManager) finishToPolynomial(body error) error {
	s := &m.toPolynomial
	s.executing = false
	if s.bodyCause == nil {
		s.bodyCause = body
	}
	e := s.buffer.sourceClose()
	if e != nil && s.sourceCause == nil {
		s.sourceCause = e
	}
	return polynomialNewtonJoin(body, e)
}
func (m *MultivariatePolynomialManager) CleanupToPolynomial() error {
	if e := m.identity(); e != nil {
		return e
	}
	s := &m.toPolynomial
	if !s.active {
		return nil
	}
	if s.executing {
		return ErrPolynomialPending
	}
	c := m.core
	fail := func(e error) error { s.cleanupCause = e; return e }
	// Failed callee factories are younger than the lexical coefficient buffer.
	if s.factoryGeneration != 0 {
		if c.pending.generation == s.factoryGeneration && c.pending.toPolynomialParent == s {
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
		if c.generatedCalleeGeneration == s.calleeGeneration {
			c.generatedCalleePending = false
		}
		s.calleeGeneration = 0
	}
	if s.generated != (Monomial{}) {
		if c.generatedActive && c.generatedMonomial == s.generated {
			if e := c.cleanupGeneratedMonomial(); e != nil {
				return fail(e)
			}
		}
		s.generated = Monomial{}
	}
	if s.ownsResult && s.result != (Polynomial{}) {
		if e := c.CleanupValue(&s.result); e != nil {
			return fail(e)
		}
		s.ownsResult = false
	}
	if e := s.buffer.cleanup(); e != nil {
		return fail(e)
	}
	if s.monomialRaw != ([3]memory.Block{}) {
		now, e := c.monomials.allocator.PendingRoots()
		if e != nil {
			return fail(e)
		}
		if now == s.monomialRaw {
			e = c.monomials.allocator.CleanupPending()
			after, x := c.monomials.allocator.PendingRoots()
			for i := range s.monomialRaw {
				if after[i] == (memory.Block{}) {
					s.monomialRaw[i] = memory.Block{}
				}
			}
			if e != nil || x != nil {
				return fail(polynomialNewtonJoin(e, x))
			}
		}
		s.monomialRaw = [3]memory.Block{}
	}
	s.active = false
	s.cleanupCause = nil
	return nil
}
