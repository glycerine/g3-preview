// Source: src/math/polynomial/polynomial.cpp:1298-1517,2420-2500; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Polynomial is a checked scalar locator, never an additional owning copy.
type Polynomial struct {
	owner            uint64
	slot, generation uint32
}
type PolynomialInfo struct {
	RefCount, ID, Size uint32
	LexSorted          bool
}
type polynomialLocation struct {
	monomialDropConsumed, monomialDropLast                               bool
	monomialDropCause                                                    error
	callback                                                             *PolynomialDeletionHandler
	callbackCause                                                        error
	callbackStarted, callbackInvoked                                     bool
	raw                                                                  memory.ObjectRef
	generation, requested, initializedCoefficients, initializedMonomials uint32
	deleteIndex                                                          uint32
	deleteStage                                                          uint8
	live, published, released                                            bool
}

// The actual codec has a 12-byte scalar header followed by all numeral fields
// (val32/ref64), then all monomial handles (owner64/slot32/generation32).
// Derived native as/ms pointers are omitted, and translated scalar handle
// widths replace native pointer widths. The allocation stores actual values.
func polynomialCodecBytes(n uint32) uint32 { return 12 + n*28 }
func (m *MultivariatePolynomialCore) polynomialLocation(p Polynomial) (*polynomialLocation, error) {
	if p.owner != m.origin || p.slot == 0 || uint64(p.slot) > uint64(len(m.locations)) {
		return nil, ErrPolynomialOwner
	}
	l := &m.locations[p.slot-1]
	if !l.live || l.generation != p.generation {
		return nil, ErrPolynomialOwner
	}
	return l, nil
}
func (m *MultivariatePolynomialCore) readWord(p Polynomial, o uint64) (uint32, error) {
	l, e := m.polynomialLocation(p)
	if e != nil {
		return 0, e
	}
	return m.monomials.allocator.ReadUint32(l.raw, o)
}
func (m *MultivariatePolynomialCore) writeWord(p Polynomial, o uint64, v uint32) error {
	l, e := m.polynomialLocation(p)
	if e != nil {
		return e
	}
	return m.monomials.allocator.WriteUint32(l.raw, o, v)
}
func (m *MultivariatePolynomialCore) read64(p Polynomial, o uint64) (uint64, error) {
	lo, e := m.readWord(p, o)
	if e != nil {
		return 0, e
	}
	hi, e := m.readWord(p, o+4)
	return uint64(lo) | uint64(hi)<<32, e
}
func (m *MultivariatePolynomialCore) write64(p Polynomial, o, v uint64) error {
	if e := m.writeWord(p, o, uint32(v)); e != nil {
		return e
	}
	return m.writeWord(p, o+4, uint32(v>>32))
}
func (m *MultivariatePolynomialCore) coefficientCore(p Polynomial, i uint32) (Int, error) {
	n, e := m.readWord(p, 8)
	if e != nil {
		return Int{}, e
	}
	if modularSourceDebug && i >= n {
		return Int{}, ErrPolynomialAssertion
	}
	l, e := m.polynomialLocation(p)
	if e != nil {
		return Int{}, e
	}
	if i >= l.initializedCoefficients {
		return Int{}, ErrPolynomialPartial
	}
	o := uint64(12) + uint64(i)*12
	v, e := m.readWord(p, o)
	if e != nil {
		return Int{}, e
	}
	r, e := m.read64(p, o+4)
	return Int{val: int32(v), cell: numericCellRef(r)}, e
}
func (m *MultivariatePolynomialCore) writeCoefficient(p Polynomial, i uint32, a Int) error {
	o := uint64(12) + uint64(i)*12
	if e := m.writeWord(p, o, uint32(a.val)); e != nil {
		return e
	}
	return m.write64(p, o+4, uint64(a.cell))
}
func (m *MultivariatePolynomialCore) monomialCore(p Polynomial, i uint32) (Monomial, error) {
	n, e := m.readWord(p, 8)
	if e != nil {
		return Monomial{}, e
	}
	if modularSourceDebug && i >= n {
		return Monomial{}, ErrPolynomialAssertion
	}
	l, e := m.polynomialLocation(p)
	if e != nil {
		return Monomial{}, e
	}
	if i >= l.initializedMonomials {
		return Monomial{}, ErrPolynomialPartial
	}
	o := uint64(12) + uint64(n)*12 + uint64(i)*16
	r, e := m.read64(p, o)
	if e != nil {
		return Monomial{}, e
	}
	s, e := m.readWord(p, o+8)
	if e != nil {
		return Monomial{}, e
	}
	g, e := m.readWord(p, o+12)
	return Monomial{owner: r, slot: s, generation: g}, e
}
func (m *MultivariatePolynomialCore) writeMonomial(p Polynomial, i uint32, a Monomial) error {
	n, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	o := uint64(12) + uint64(n)*12 + uint64(i)*16
	if e = m.write64(p, o, a.owner); e != nil {
		return e
	}
	if e = m.writeWord(p, o+8, a.slot); e != nil {
		return e
	}
	return m.writeWord(p, o+12, a.generation)
}
func (m *MultivariatePolynomialCore) Info(p Polynomial) (PolynomialInfo, error) {
	if e := m.check(); e != nil {
		return PolynomialInfo{}, e
	}
	r, e := m.readWord(p, 0)
	if e != nil {
		return PolynomialInfo{}, e
	}
	f, e := m.readWord(p, 4)
	if e != nil {
		return PolynomialInfo{}, e
	}
	n, e := m.readWord(p, 8)
	return PolynomialInfo{RefCount: r, ID: f & 0x7fffffff, Size: n, LexSorted: f>>31 != 0}, e
}
func (m *MultivariatePolynomialCore) Coefficient(p Polynomial, i uint32) (Int, error) {
	if e := m.check(); e != nil {
		return Int{}, e
	}
	return m.coefficientCore(p, i)
}
func (m *MultivariatePolynomialCore) TermMonomial(p Polynomial, i uint32) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	return m.monomialCore(p, i)
}
func (m *MultivariatePolynomialCore) IncRef(p Polynomial) error {
	if e := m.check(); e != nil {
		return e
	}
	r, e := m.readWord(p, 0)
	if e != nil {
		return e
	}
	return m.writeWord(p, 0, r+1)
}
func (m *MultivariatePolynomialCore) DecRef(p Polynomial) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.decRefCore(p)
}

type polynomialDecrementOutcome struct{ consumed, last bool }

func (m *MultivariatePolynomialCore) decRefCore(p Polynomial) error {
	return m.decRefCoreOutcome(p, nil)
}
func (m *MultivariatePolynomialCore) decRefCoreOutcome(p Polynomial, out *polynomialDecrementOutcome) error {
	r, e := m.readWord(p, 0)
	if e != nil {
		return e
	}
	if modularSourceDebug && r == 0 {
		return ErrPolynomialAssertion
	}
	r--
	if e = m.writeWord(p, 0, r); e != nil {
		return e
	}
	if out != nil {
		out.consumed = true
		out.last = r == 0
	}
	if r == 0 {
		return m.deleteCore(p)
	}
	return nil
}
func (m *MultivariatePolynomialCore) swapTerms(p Polynomial, i, j uint32) error {
	if i == j {
		return nil
	}
	a, e := m.coefficientCore(p, i)
	if e != nil {
		return e
	}
	b, e := m.coefficientCore(p, j)
	if e != nil {
		return e
	}
	if e = m.writeCoefficient(p, i, b); e != nil {
		return e
	}
	if e = m.writeCoefficient(p, j, a); e != nil {
		return e
	}
	x, e := m.monomialCore(p, i)
	if e != nil {
		return e
	}
	y, e := m.monomialCore(p, j)
	if e != nil {
		return e
	}
	if e = m.writeMonomial(p, i, y); e != nil {
		return e
	}
	return m.writeMonomial(p, j, x)
}
func (m *MultivariatePolynomialCore) firstGreater(a, b Monomial) (bool, error) {
	x, e := m.monomials.sizeCore(a)
	if e != nil {
		return false, e
	}
	if x == 0 {
		return false, nil
	}
	y, e := m.monomials.sizeCore(b)
	if e != nil {
		return false, e
	}
	if y == 0 {
		return true, nil
	}
	av, e := m.monomials.MaxVariable(a)
	if e != nil {
		return false, e
	}
	bv, e := m.monomials.MaxVariable(b)
	if e != nil {
		return false, e
	}
	if av != bv {
		return av > bv, nil
	}
	ad, e := m.monomials.MaxVariableDegree(a)
	if e != nil {
		return false, e
	}
	bd, e := m.monomials.MaxVariableDegree(b)
	return ad > bd, e
}
func (m *MultivariatePolynomialCore) deleteCore(p Polynomial) error {
	return m.deleteCoreMode(p, false)
}
func (m *MultivariatePolynomialCore) deleteCoreMode(p Polynomial, owning bool) error {
	l, e := m.polynomialLocation(p)
	if e != nil {
		return e
	}
	if !owning && l.monomialDropCause != nil {
		return l.monomialDropCause
	}
	if l.deleteStage == 0 {
		if e = m.deletionCallbacks(p); e != nil {
			return e
		}
		l, e = m.polynomialLocation(p)
		if e != nil {
			return e
		}
		l.deleteStage = 1
	}
	if l.deleteStage == 1 {
		n, e := m.readWord(p, 8)
		if e != nil {
			return e
		}
		for l.deleteIndex < n {
			i := l.deleteIndex
			a, e := m.coefficientCore(p, i)
			if e != nil {
				return e
			}
			e = m.deleteNumeral(&a)
			if w := m.writeCoefficient(p, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
			b, e := m.monomialCore(p, i)
			if e != nil {
				return e
			}

			if owning && l.monomialDropConsumed && l.monomialDropLast {
				e = m.monomials.CleanupValue(&b)
				if w := m.writeMonomial(p, i, b); w != nil {
					return w
				}
				if e != nil {
					return e
				}
			} else if !l.monomialDropConsumed {
				outcome := monomialDecrementOutcome{}
				if e = m.monomials.check(); e == nil {
					e = m.monomials.decRefCoreOutcome(b, &outcome)
				}
				if outcome.written {
					l.monomialDropConsumed = true
					l.monomialDropLast = outcome.last
				}
				if e != nil {
					if l.monomialDropCause == nil {
						l.monomialDropCause = e
					}
					return e
				}
			}
			l.monomialDropConsumed = false
			l.monomialDropLast = false
			l.deleteIndex++
		}
		l.deleteStage = 2
	}
	if l.deleteStage == 2 {
		f, e := m.readWord(p, 4)
		if e != nil {
			return e
		}
		if !m.idsSourceDestroyed {
			if e = m.ids.Recycle(f & 0x7fffffff); e != nil {
				return e
			}
		}
		l.deleteStage = 3
	}
	if l.deleteStage == 3 && m.directorySourceDestroyed {
		// Failed-constructor owning abort cannot write the destroyed member.
		l.deleteStage = 4
	}
	if l.deleteStage == 3 {
		f, e := m.readWord(p, 4)
		if e != nil {
			return e
		}
		id := f & 0x7fffffff
		n, e := m.polynomials.size(m)
		if e != nil {
			return e
		}
		if id >= n {
			return arena.ErrBounds
		}
		if e = m.polynomials.putMonomial(m, id, Monomial{}); e != nil {
			return e
		}
		l.deleteStage = 4
	}
	if l.deleteStage == 4 {
		progress, e := m.monomials.allocator.DeallocateTracked(uint64(l.requested), l.raw)
		if progress.PhysicalReleased || progress.PooledLinked {
			l.released = true
			l.deleteStage = 5
		}
		if e != nil {
			return e
		}
	}
	l.live = false
	if l.published {
		m.live--
		l.published = false
	}
	return nil
}
