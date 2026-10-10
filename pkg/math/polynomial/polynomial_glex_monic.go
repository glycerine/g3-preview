// Source: src/math/polynomial/polynomial.cpp:6843-6862,7388-7390;
// src/math/polynomial/polynomial.h:986.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type polynomialMonicScope struct {
	inv, newA               unumeral.Int
	numeric                 polynomialNumericRoots
	active, topOwner        bool
	cursor                  uint8
	generation              uint64
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialManager) finishGlexMonicSource() (err error) {
	s := &m.monic
	if !s.active {
		return nil
	}
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	if e := m.core.pscSourceStopped(); e != nil {
		s.cleanupCause = e
		return e
	}
	defer func() {
		if err != nil {
			s.cleanupCause = err
		}
	}()
	if s.cursor == 0 {
		if _, e := unumeral.SourceNumeralDel(m.core.integer, &s.newA); e != nil {
			return e
		}
		s.cursor = 1
	}
	if s.cursor == 1 {
		if _, e := unumeral.SourceNumeralDel(m.core.integer, &s.inv); e != nil {
			return e
		}
		s.cursor = 2
	}
	return nil
}
func (m *MultivariatePolynomialManager) CleanupGlexMonic() error {
	if m == nil || m.self != m || m.core == nil {
		return ErrPolynomialOwner
	}
	s := &m.monic
	if !s.active {
		return nil
	}
	c := m.core
	if e := c.deleteNumeral(&s.newA); e != nil {
		return e
	}
	if e := c.deleteNumeral(&s.inv); e != nil {
		return e
	}
	if e := s.numeric.cleanup(c.integer); e != nil {
		return e
	}
	if s.topOwner && s.generation == c.pscReceipts.generation {
		if e := c.CleanupPSC(); e != nil {
			return e
		}
	}
	*s = polynomialMonicScope{}
	return nil
}
func (m *MultivariatePolynomialManager) GlexMonic(p Polynomial) (result Polynomial, err error) {
	if e := m.ready(); e != nil {
		return Polynomial{}, e
	}
	c := m.core
	if polynomialSourceDebug {
		field, e := c.modular.Field()
		if e != nil {
			return Polynomial{}, e
		}
		if !field {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	zero, e := c.IsZero(p)
	if e != nil {
		return Polynomial{}, e
	}
	if zero {
		return p, nil
	}
	pos, e := c.gradedLexMaxPosition(p)
	if e != nil {
		return Polynomial{}, e
	}
	a, e := c.coefficientCore(p, pos)
	if e != nil {
		return Polynomial{}, e
	}
	one, e := c.modular.IsOne(&a)
	if e != nil {
		return Polynomial{}, e
	}
	if one {
		return p, nil
	}
	s := &m.monic
	if s.active {
		return Polynomial{}, ErrPolynomialPending
	}
	top := c.pscReceipts.depth == 0
	if e = c.pscEnter(); e != nil {
		return Polynomial{}, e
	}
	*s = polynomialMonicScope{active: true, topOwner: top, generation: c.pscReceipts.generation}
	defer func() {
		c.pscLeave()
		s.bodyCause = err
		if e := m.finishGlexMonicSource(); e != nil {
			err = errors.Join(err, e)
			return
		}
		if s.newA.NativeCellToken() == 0 && s.inv.NativeCellToken() == 0 && s.numeric.PendingObject().OwnerID() == 0 && !(s.topOwner && c.pscPending()) {
			hasRaw := false
			for _, p := range s.numeric.PendingAllocatorBlocks() {
				hasRaw = hasRaw || p.OwnerID() != 0
			}
			if !hasRaw {
				*s = polynomialMonicScope{}
			}
		}
	}()
	a, e = c.coefficientCore(p, pos)
	if e != nil {
		return Polynomial{}, e
	}
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return Polynomial{}, e
	}
	e = c.modular.Copy(&s.inv, &a)
	e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
	if e != nil {
		return Polynomial{}, e
	}
	before, e = polynomialNumericBefore(c.integer)
	if e != nil {
		return Polynomial{}, e
	}
	e = c.modular.Inv(&s.inv)
	e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
	if e != nil {
		return Polynomial{}, e
	}
	if e = c.cheap.reset(); e != nil {
		return Polynomial{}, e
	}
	n, e := c.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	for i := uint32(0); i < n; i++ {
		a, e := c.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, e = polynomialNumericBefore(c.integer)
		if e != nil {
			return Polynomial{}, e
		}
		e = c.modular.Copy(&s.newA, &a)
		e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		before, e = polynomialNumericBefore(c.integer)
		if e != nil {
			return Polynomial{}, e
		}
		e = c.modular.Mul(&s.newA, &s.inv, &s.newA)
		e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		mono, e := c.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, e = polynomialNumericBefore(c.integer)
		if e != nil {
			return Polynomial{}, e
		}
		e = c.cheap.add(&s.newA, mono, false)
		e = polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
	}
	before, e = polynomialNumericBefore(c.integer)
	if e != nil {
		return Polynomial{}, e
	}
	result, e = c.cheap.mk(false)
	return result, polynomialNewtonJoin(e, s.numeric.after(c.integer, before))
}
