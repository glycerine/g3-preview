// Source: src/math/polynomial/polynomial.cpp:3344-3359,3384-3428.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

type polynomialContentScope struct {
	ai                        Int
	active, aiEnded, topOwner bool
	generation                uint64
	numeric                   polynomialNumericRoots
	bodyCause, cleanupCause   error
}

func (m *MultivariatePolynomialCore) finishIntegerContent(owning bool) error {
	return m.finishIntegerContentPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) finishIntegerContentPhase(owning, abort bool) error {
	s := &m.contentLocal
	if !s.active {
		return nil
	}
	if !owning && s.cleanupCause != nil {
		return s.cleanupCause
	}

	if !owning && s.aiEnded {
		return nil
	}
	var e error
	if owning {
		e = m.deleteNumeral(&s.ai)
	} else {
		_, e = sourceNumeralDel(m.integer, &s.ai)
	}
	if e != nil {
		s.cleanupCause = e
		return e
	}
	s.aiEnded = true
	if owning && abort {
		if e = s.numeric.cleanup(m.integer); e != nil {
			return e
		}
		*s = polynomialContentScope{}
		return nil
	}
	pending := s.numeric.cell.object.OwnerID() != 0
	for _, p := range s.numeric.allocator {
		pending = pending || p.OwnerID() != 0
	}
	pending = pending || (s.topOwner && m.pscPending())
	if !pending {
		*s = polynomialContentScope{}
	}

	return nil
}
func (m *MultivariatePolynomialCore) IntegerContent(p Polynomial, i *Int, c *PolynomialRef) error {
	if e := m.check(); e != nil {
		return e
	}
	if i == nil || c == nil || c.owner != m {
		return ErrPolynomialOwner
	}
	if e := c.open(); e != nil {
		return e
	}
	return m.integerContent(p, i, &c.root)
}
func (m *MultivariatePolynomialCore) integerContent(p Polynomial, i *Int, c *polynomialReference) (err error) {
	zero, e := m.IsZero(p)
	if e != nil {
		return e
	}
	if zero {
		if e = m.modular.Reset(i); e != nil {
			return e
		}
		return c.assign(m, p)
	}
	constant, e := m.IsConstant(p)
	if e != nil {
		return e
	}
	if constant {
		a, e := m.coefficientCore(p, 0)
		if e != nil {
			return e
		}
		e = m.modular.Copy(i, &a)
		if e != nil {
			return e
		}
		return c.assign(m, m.unit)
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	l, e := m.polynomialLocation(p)
	if e != nil {
		return e
	}
	view := integerArrayView{allocator: m.monomials.allocator, object: l.raw, count: n, offset: 12, stride: 12}
	e = m.modular.GCDManyIndexed(view, i)
	if e != nil {
		return e
	}
	one, e := m.modular.IsOne(i)
	if e != nil {
		return e
	}
	if one {
		return c.assign(m, p)
	}
	if e = m.cheap.reset(); e != nil {
		return e
	}
	s := &m.contentLocal
	if s.active {
		return ErrPolynomialPending
	}
	top := m.pscReceipts.depth == 0
	if e = m.pscEnter(); e != nil {
		return e
	}
	*s = polynomialContentScope{active: true, topOwner: top, generation: m.pscExecutingOwner()}
	defer func() {
		m.pscLeave()
		s.bodyCause = err
		if e := m.finishIntegerContent(false); e != nil {
			err = errors.Join(err, e)
		}
	}()
	for j := uint32(0); j < n; j++ {
		term, e := m.monomialCore(p, j)
		if e != nil {
			return e
		}
		a, e := m.coefficientCore(p, j)
		if e != nil {
			return e
		}
		before, x := polynomialNumericBefore(m.integer)
		if x != nil {
			return x
		}
		e = m.modular.Div(&a, i, &s.ai)
		e = polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
		if e != nil {
			return e
		}
		before, x = polynomialNumericBefore(m.integer)
		if x != nil {
			return x
		}
		e = m.cheap.add(&s.ai, term, true)
		e = polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
		if e != nil {
			return e
		}
	}
	before, x := polynomialNumericBefore(m.integer)
	if x != nil {
		return x
	}
	r, e := m.cheap.mk(false)
	e = polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
	if e != nil {
		return e
	}
	return c.assign(m, r)
}

// IntegerScalarContent is the source numeral-only overload. Unlike ic(p,a,c),
// its first SET preserves a lone negative coefficient and its early-one test
// occurs before reading the next coefficient.
func (full *MultivariatePolynomialManager) IntegerScalarContent(p Polynomial, out *Int) error {
	if e := full.ready(); e != nil {
		return e
	}
	if out == nil {
		return ErrPolynomialOwner
	}
	return full.core.integerScalarContent(p, out)
}
func (m *MultivariatePolynomialCore) integerScalarContent(p Polynomial, out *Int) error {
	zero, e := m.IsZero(p)
	if e != nil {
		return e
	}
	if zero {
		return m.modular.Reset(out)
	}
	constant, e := m.IsConstant(p)
	if e != nil {
		return e
	}
	if constant {
		a, e := m.coefficientCore(p, 0)
		if e != nil {
			return e
		}
		return m.modular.Copy(out, &a)
	}
	a, e := m.coefficientCore(p, 0)
	if e != nil {
		return e
	}
	if e = m.modular.Copy(out, &a); e != nil {
		return e
	}
	size, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	for i := uint32(1); i < size; i++ {
		one, e := m.modular.IsOne(out)
		if e != nil {
			return e
		}
		if one {
			return nil
		}
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return e
		}
		if e = m.modular.GCD(out, &a, out); e != nil {
			return e
		}
	}
	return nil
}

// CleanupIntegerContent is explicit owning continuation of this actual local.
// Caller output SETs before local construction are not claimed by this scope.
func (m *MultivariatePolynomialCore) CleanupIntegerContent() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.contentLocal
	top, g := s.topOwner, s.generation
	if e := m.finishIntegerContent(true); e != nil {
		return e
	}
	if top && g != 0 && m.pscReceipts.generation == g {
		return m.CleanupPSC()
	}
	return nil
}
