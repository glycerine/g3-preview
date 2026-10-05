// Source: src/math/polynomial/polynomial.cpp:5904-5940;
// src/math/polynomial/polynomial.h:873.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

type polynomialNormalizeScope struct {
	invocation                                   uint64
	g, a                                         Int
	active, gConstructed, aConstructed, topOwner bool
	generation                                   uint64
	cursor                                       uint8
	somInvocation                                uint64
	numeric                                      polynomialNumericRoots
	bodyCause, cleanupCause                      error
}

func (m *MultivariatePolynomialManager) beginNormalization() error {
	s := &m.normalization
	if s.active {
		return ErrPolynomialPending
	}
	if s.invocation == ^uint64(0) {
		return arena.ErrBounds
	}
	c := m.core
	top := c.pscReceipts.depth == 0
	if e := c.pscEnter(); e != nil {
		return e
	}
	*s = polynomialNormalizeScope{invocation: s.invocation + 1, active: true, topOwner: top, generation: c.pscReceipts.generation}
	return nil
}
func (m *MultivariatePolynomialManager) finishNormalizationSource() error {
	s := &m.normalization
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	b := &m.core.cheap
	if b.gcdActive && b.gcdInvocation == s.somInvocation && b.gcdNormalization == s && b.gcdCleanupCause != nil {
		s.cleanupCause = b.gcdCleanupCause
		return s.cleanupCause
	}
	if e := m.core.pscSourceStopped(); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.cursor == 0 {
		if s.aConstructed {
			if _, e := sourceNumeralDel(m.core.integer, &s.a); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.cursor = 1
	}
	if s.cursor == 1 {
		if s.gConstructed {
			if _, e := sourceNumeralDel(m.core.integer, &s.g); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.cursor = 2
	}
	pending := s.numeric.cell.object.OwnerID() != 0
	for _, raw := range s.numeric.allocator {
		pending = pending || raw.OwnerID() != 0
	}
	pending = pending || (s.topOwner && m.core.pscPending()) || b.gcdActive && b.gcdInvocation == s.somInvocation && b.gcdNormalization == s
	if !pending {
		*s = polynomialNormalizeScope{invocation: s.invocation}
	}
	return nil
}
func (m *MultivariatePolynomialManager) CleanupNormalization() error {
	if m == nil || m.self != m || m.core == nil {
		return ErrPolynomialOwner
	}
	if e := m.cleanupNormalizationPhase(false); e != nil {
		return e
	}
	return m.cleanupNormalizationPhase(true)
}
func (m *MultivariatePolynomialManager) cleanupNormalizationPhase(abort bool) error {
	s := &m.normalization
	if !s.active {
		return nil
	}
	c := m.core
	b := &c.cheap
	if b.gcdActive && b.gcdInvocation == s.somInvocation && b.gcdNormalization == s {
		if e := b.finishNormalizeNumeralPhase(true, abort); e != nil {
			return e
		}
	}
	if s.aConstructed {
		if e := c.deleteNumeral(&s.a); e != nil {
			return e
		}
	}
	if s.gConstructed {
		if e := c.deleteNumeral(&s.g); e != nil {
			return e
		}
	}
	if !abort {
		return nil
	}
	if e := s.numeric.cleanup(c.integer); e != nil {
		return e
	}
	if s.topOwner && s.generation == c.pscReceipts.generation {
		if e := c.CleanupPSC(); e != nil {
			return e
		}
	}
	*s = polynomialNormalizeScope{invocation: s.invocation}
	return nil
}
func (m *MultivariatePolynomialManager) Normalize(p Polynomial) (Polynomial, error) {
	return m.normalizeWithReceipt(p, nil)
}
func (m *MultivariatePolynomialManager) normalizeWithReceipt(p Polynomial, receipt *polynomialNormalizeInvocationReceipt) (result Polynomial, err error) {
	if e := m.ready(); e != nil {
		return Polynomial{}, e
	}
	c := m.core
	zero, e := c.IsZero(p)
	if e != nil || zero {
		return p, e
	}
	n, e := c.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	field, e := c.modular.Modular()
	if e != nil {
		return Polynomial{}, e
	}
	started := false
	defer func() {
		if !started {
			return
		}
		s := &m.normalization
		s.bodyCause = err
		cleanup := m.finishNormalizationSource()
		c.pscLeave()
		if cleanup != nil {
			err = errors.Join(err, cleanup)
		}
	}()
	if field {
		i := uint32(0)
		for ; i < n; i++ {
			a, e := c.coefficientCore(p, i)
			if e != nil {
				return Polynomial{}, e
			}
			normal, e := c.modular.IsPNormalized(&a)
			if e != nil {
				return Polynomial{}, e
			}
			if !normal {
				break
			}
		}
		if i < n {
			if e = c.cheap.reset(); e != nil {
				return Polynomial{}, e
			}
			if e = m.beginNormalization(); e != nil {
				return Polynomial{}, e
			}
			started = true
			if receipt != nil {
				*receipt = polynomialNormalizeInvocationReceipt{m, &m.normalization, m.normalization.invocation}
			}
			s := &m.normalization
			s.aConstructed = true
			for i := uint32(0); i < n; i++ {
				term, e := c.monomialCore(p, i)
				if e != nil {
					return Polynomial{}, e
				}
				a, e := c.coefficientCore(p, i)
				if e != nil {
					return Polynomial{}, e
				}
				before, e := polynomialNumericBefore(c.integer)
				if e != nil {
					return Polynomial{}, e
				}
				e = c.modular.Copy(&s.a, &a)
				e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
				if e != nil {
					return Polynomial{}, e
				}
				if e = c.cheap.add(&s.a, term, true); e != nil {
					return Polynomial{}, e
				}
			}
			if e = c.cheap.normalizeWithOwner(s); e != nil {
				return Polynomial{}, e
			}
			return c.cheap.mk(false)
		}
	}
	if e = m.beginNormalization(); e != nil {
		return Polynomial{}, e
	}
	started = true
	if receipt != nil {
		*receipt = polynomialNormalizeInvocationReceipt{m, &m.normalization, m.normalization.invocation}
	}
	s := &m.normalization
	s.gConstructed = true
	l, e := c.polynomialLocation(p)
	if e != nil {
		return Polynomial{}, e
	}
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return Polynomial{}, e
	}
	e = c.modular.GCDManyIndexed(integerArrayView{allocator: c.monomials.allocator, object: l.raw, count: n, offset: 12, stride: 12}, &s.g)
	e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
	if e != nil {
		return Polynomial{}, e
	}
	one, e := c.modular.IsOne(&s.g)
	if e != nil {
		return Polynomial{}, e
	}
	if one {
		return p, nil
	}
	if e = c.cheap.reset(); e != nil {
		return Polynomial{}, e
	}
	s.aConstructed = true
	for i := uint32(0); i < n; i++ {
		term, e := c.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		a, e := c.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, e := polynomialNumericBefore(c.integer)
		if e != nil {
			return Polynomial{}, e
		}
		e = c.modular.Div(&a, &s.g, &s.a)
		e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		if e = c.cheap.add(&s.a, term, true); e != nil {
			return Polynomial{}, e
		}
	}
	return c.cheap.mk(false)
}
