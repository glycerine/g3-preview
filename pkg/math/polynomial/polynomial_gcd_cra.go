// Source: src/math/polynomial/polynomial.cpp:3675-3767.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util"
)

// The twelve actual scoped_numeral declarations, constructed at their source
// sites. The outer manager's reflected cold extent owns this concrete carrier.

type polynomialCRAScope struct {
	nums                    [12]util.Int
	constructed             uint8
	active, topOwner        bool
	generation              uint64
	numeric                 polynomialNumericRoots
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialManager) craSourceEnd() error {
	s := &m.cra
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	if e := m.core.pscSourceStopped(); e != nil {
		s.cleanupCause = e
		return e
	}
	for s.constructed != 0 {
		i := s.constructed - 1
		if _, e := util.SourceNumeralDel(m.core.integer, &s.nums[i]); e != nil {
			s.cleanupCause = e
			return e
		}
		s.constructed--
	}
	pending := s.numeric.PendingObject().OwnerID() != 0 || s.topOwner && m.core.pscPending()
	for _, raw := range s.numeric.PendingAllocatorBlocks() {
		pending = pending || raw.OwnerID() != 0
	}
	if !pending {
		*s = polynomialCRAScope{}
	}
	return nil
}
func (m *MultivariatePolynomialManager) cleanupCRACompleted() error {
	s := &m.cra
	for s.constructed != 0 {
		i := s.constructed - 1
		if e := m.core.deleteNumeral(&s.nums[i]); e != nil {
			return e
		}
		s.constructed--
	}
	return nil
}
func (m *MultivariatePolynomialManager) CleanupCRA() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.cra
	if !s.active {
		return nil
	}
	c := m.core
	if s.generation == c.pscReceipts.generation {
		if e := c.cleanupPSCCompleted(s.generation); e != nil {
			return e
		}
	}
	if e := m.cleanupCRACompleted(); e != nil {
		return e
	}
	if e := s.numeric.cleanup(c.integer); e != nil {
		return e
	}
	if s.topOwner && s.generation == c.pscReceipts.generation {
		if e := c.cleanupPSCAborts(s.generation); e != nil {
			return e
		}
	}
	*s = polynomialCRAScope{}
	return nil
}

// Concrete operation dispatch shares the real numeric engines; the receipt
// brackets only the invoked arithmetic leaf, never caller callbacks.
func (m *MultivariatePolynomialManager) craLeaf(op uint8, a, b, out *util.Int) error {
	c := m.core
	s := &m.cra
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return e
	}
	switch op {
	case 0:
		e = c.modular.Mul(a, b, out)
	case 1:
		e = c.modular.Add(a, b, out)
	case 2:
		e = c.modular.Sub(a, b, out)
	case 3:
		e = c.integer.Mod(a, b, out)
	case 4:
		e = c.modular.Div(a, b, out)
	case 5:
		e = c.modular.Copy(out, a)
	case 6:
		e = c.modular.Neg(a)
	}
	return errors.Join(e, s.numeric.after(c.integer, before))
}
func (m *MultivariatePolynomialManager) craAdd(a Polynomial, ai uint32, b Polynomial, bi uint32, term Monomial, monoFrom Polynomial, mi uint32) error {
	c := m.core
	s := &m.cra
	n := &s.nums
	av := util.NewInt(0)
	if a != (Polynomial{}) {
		var e error
		av, e = c.coefficientCore(a, ai)
		if e != nil {
			return e
		}
	}
	if e := m.craLeaf(0, &av, &n[3], &n[9]); e != nil {
		return e
	}
	bv := util.NewInt(0)
	if b != (Polynomial{}) {
		var e error
		bv, e = c.coefficientCore(b, bi)
		if e != nil {
			return e
		}
	}
	if e := m.craLeaf(0, &bv, &n[4], &n[10]); e != nil {
		return e
	}
	if e := m.craLeaf(1, &n[9], &n[10], &n[11]); e != nil {
		return e
	}
	if e := m.craLeaf(3, &n[11], &n[5], &n[8]); e != nil {
		return e
	}
	gt, e := c.modular.Greater(&n[8], &n[7])
	if e != nil {
		return e
	}
	if gt {
		if e = m.craLeaf(2, &n[8], &n[5], &n[8]); e != nil {
			return e
		}
	}
	if monoFrom != (Polynomial{}) {
		term, e = c.monomialCore(monoFrom, mi)
		if e != nil {
			return e
		}
	}
	return c.cheap.add(&n[8], term, false)
}
func (m *MultivariatePolynomialManager) gcdCRA(a Polynomial, b1 *util.Int, b Polynomial, b2 *util.Int, out *PolynomialRef) (err error) {
	if e := m.ready(); e != nil {
		return e
	}
	if out == nil || out.owner != m.core {
		return ErrPolynomialOwner
	}
	if e := out.open(); e != nil {
		return e
	}
	c := m.core
	if e := m.LexSort(a); e != nil {
		return e
	}
	if e := m.LexSort(b); e != nil {
		return e
	}
	if polynomialSourceDebug {
		size, e := c.cheap.ms.size(c)
		if e != nil {
			return e
		}
		if size != 0 {
			return ErrPolynomialAssertion
		}
		even, e := c.integer.IsEven(b1)
		if e != nil {
			return e
		}
		if even {
			return ErrPolynomialAssertion
		}
		even, e = c.integer.IsEven(b2)
		if e != nil {
			return e
		}
		if even {
			return ErrPolynomialAssertion
		}
	}
	s := &m.cra
	if s.active {
		return ErrPolynomialPending
	}
	top := c.pscReceipts.depth == 0
	if e := c.pscEnter(); e != nil {
		return e
	}
	*s = polynomialCRAScope{active: true, topOwner: top, generation: c.pscExecutingOwner(), constructed: 3}
	defer func() { s.bodyCause = err; err = errors.Join(err, m.craSourceEnd()); c.pscLeave() }()
	n := &s.nums
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return e
	}
	e = c.modular.ExtendedGCD(b1, b2, &n[0], &n[1], &n[2])
	e = errors.Join(e, s.numeric.after(c.integer, before))
	if e != nil {
		return e
	}
	if polynomialSourceDebug {
		one, e := c.modular.IsOne(&n[2])
		if e != nil {
			return e
		}
		if !one {
			return ErrPolynomialAssertion
		}
	}
	if e = m.craLeaf(3, &n[0], b2, &n[0]); e != nil {
		return e
	}
	if e = m.craLeaf(3, &n[1], b1, &n[1]); e != nil {
		return e
	}
	s.constructed = 5
	if e = m.craLeaf(0, b2, &n[1], &n[3]); e != nil {
		return e
	}
	if e = m.craLeaf(0, b1, &n[0], &n[4]); e != nil {
		return e
	}
	s.constructed = 6
	if e = m.craLeaf(0, b1, b2, &n[5]); e != nil {
		return e
	}
	s.constructed = 12
	two := util.NewInt(2)
	if e = m.craLeaf(4, &n[5], &two, &n[7]); e != nil {
		return e
	}
	if e = m.craLeaf(5, &n[7], nil, &n[6]); e != nil {
		return e
	}
	if e = m.craLeaf(6, &n[6], nil, nil); e != nil {
		return e
	}
	sa, e := c.readWord(a, 8)
	if e != nil {
		return e
	}
	sb, e := c.readWord(b, 8)
	if e != nil {
		return e
	}
	var ia, ib uint32
	for {
		if ia == sa {
			for ib < sb {
				if e = m.craAdd(Polynomial{}, 0, b, ib, Monomial{}, b, ib); e != nil {
					return e
				}
				ib++
			}
			break
		}
		if ib == sb {
			for ia < sa {
				if e = m.craAdd(a, ia, Polynomial{}, 0, Monomial{}, a, ia); e != nil {
					return e
				}
				ia++
			}
			break
		}
		ma, e := c.monomialCore(a, ia)
		if e != nil {
			return e
		}
		mb, e := c.monomialCore(b, ib)
		if e != nil {
			return e
		}
		order, e := c.monomials.Lex(ma, mb)
		if e != nil {
			return e
		}
		if order == 0 {
			e = m.craAdd(a, ia, b, ib, ma, Polynomial{}, 0)
			ia++
			ib++
		} else if order > 0 {
			e = m.craAdd(a, ia, Polynomial{}, 0, ma, Polynomial{}, 0)
			ia++
		} else {
			e = m.craAdd(Polynomial{}, 0, b, ib, mb, Polynomial{}, 0)
			ib++
		}
		if e != nil {
			return e
		}
	}
	if e = m.craLeaf(5, &n[5], nil, b2); e != nil {
		return e
	}
	result, e := c.cheap.mk(false)
	if e != nil {
		return e
	}
	return out.root.assign(c, result)
}
