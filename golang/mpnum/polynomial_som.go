// Source: src/math/polynomial/polynomial.cpp:1885-2156,2217-2300. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import "github.com/glycerine/g3/golang/arena"

type polynomialSOM struct {
	retirementOwner               *PolynomialSOMBuffer
	borrowedAppend                bool
	borrowedAppendIndex           uint32
	unreferencedConsumed          bool
	unreferencedIndex             uint32
	removeActive, removeStopped   bool
	removePrefix, removeSuffix    uint32
	owner                         *MultivariatePolynomialCore
	positions                     MonomialPositions
	as, ms                        polynomialVector
	gcd                           Int
	gcdActive                     bool
	gcdBodyCause, gcdCleanupCause error
	pendingMonomial               Monomial
	transferred, cleanupIndex     uint32
	cheap, hasPendingReference    bool
	resetActive                   bool
	resetIndex                    uint32
	resetStage                    uint8
}

func (b *polynomialSOM) init(m *MultivariatePolynomialCore, cheap bool) {
	b.owner = m
	b.cheap = cheap
	b.as.width = 12
	b.ms.width = 16
	b.positions.manager = m.monomials
	b.positions.self = &b.positions
}
func (b *polynomialSOM) add(a *Int, term Monomial, take bool) error {
	return b.addConsumed(a, term, take, nil)
}
func (b *polynomialSOM) addConsumed(a *Int, term Monomial, take bool, consumed *bool) error {
	if consumed != nil {
		*consumed = false
	}
	m := b.owner
	if b.cheap && modularSourceDebug {
		n, e := b.ms.size(m)
		if e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			x, e := b.ms.monomial(m, i)
			if e != nil {
				return e
			}
			if x == term {
				return ErrPolynomialAssertion
			}
		}
	}
	zero, e := m.modular.IsZero(a)
	if e != nil || zero {
		return e
	}
	pos := ^uint32(0)
	if !b.cheap {
		pos, e = b.positions.Get(term)
		if e != nil {
			return e
		}
	}
	if pos != ^uint32(0) {
		old, e := b.as.coefficient(m, pos)
		if e != nil {
			return e
		}
		e = m.compositionNumericCall(1, &old, a, nil, &old)
		w := b.as.putCoefficient(m, pos, old)
		if e != nil {
			return e
		}
		return w
	}
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	if b.retirementOwner != nil {
		if e = b.retirementOwner.prepareRetirement(n + 1); e != nil {
			return e
		}
	}

	if !b.cheap {
		if e = b.positions.Set(term, n); e != nil {
			return e
		}
	}
	if b.cheap {
		if e = b.as.appendCoefficient(m, Int{}); e != nil {
			return e
		}
		value, e := b.as.coefficient(m, n)
		if e != nil {
			return e
		}
		if take {
			e = m.modular.Swap(&value, a)
		} else {
			e = m.compositionNumericCall(0, a, nil, nil, &value)
		}
		if w := b.as.putCoefficient(m, n, value); w != nil {
			return w
		}
		if e != nil {
			return e
		}
	}
	if e = m.monomials.IncRef(term); e != nil {
		return e
	}
	if consumed != nil {
		*consumed = true
	}
	b.pendingMonomial = term
	b.hasPendingReference = true
	if m.generatedActive && m.generatedMonomial == term {
		m.transferGeneratedMonomial()
	}
	if e = b.ms.appendMonomial(m, term); e != nil {
		return e
	}
	b.hasPendingReference = false
	b.pendingMonomial = Monomial{}
	if !b.cheap {
		if e = b.as.appendCoefficient(m, Int{}); e != nil {
			return e
		}
		value, e := b.as.coefficient(m, n)
		if e != nil {
			return e
		}
		if take {
			e = m.modular.Swap(&value, a)
		} else {
			e = m.compositionNumericCall(0, a, nil, nil, &value)
		}
		if w := b.as.putCoefficient(m, n, value); w != nil {
			return w
		}
		return e
	}
	return nil
}

type polynomialSOMInput struct {
	polynomial Polynomial
	buffer     *PolynomialSOMBuffer
	fromBuffer bool
}

func (p polynomialSOMInput) size(m *MultivariatePolynomialCore) (uint32, error) {
	if !p.fromBuffer {
		return m.readWord(p.polynomial, 8)
	}
	if p.buffer == nil || p.buffer.self != p.buffer || p.buffer.storage != m {
		return 0, ErrPolynomialOwner
	}
	if p.buffer.closed || p.buffer.closing || p.buffer.sourceStarted {
		return 0, arena.ErrClosed
	}
	return p.buffer.buffer.ms.size(m)
}
func (p polynomialSOMInput) monomial(m *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	if !p.fromBuffer {
		return m.monomialCore(p.polynomial, i)
	}
	if modularSourceDebug {
		n, e := p.buffer.buffer.ms.size(m)
		if e != nil {
			return Monomial{}, e
		}
		if i >= n {
			return Monomial{}, ErrPolynomialAssertion
		}
	}
	return p.buffer.buffer.ms.monomial(m, i)
}
func (p polynomialSOMInput) coefficient(m *MultivariatePolynomialCore, i uint32) (Int, error) {
	if !p.fromBuffer {
		return m.coefficientCore(p.polynomial, i)
	}
	if modularSourceDebug {
		n, e := p.buffer.buffer.as.size(m)
		if e != nil {
			return Int{}, e
		}
		if i >= n {
			return Int{}, ErrPolynomialAssertion
		}
	}
	return p.buffer.buffer.as.coefficient(m, i)
}
func (b *polynomialSOM) addMul(a *Int, term Monomial, p Polynomial) error {
	return b.addSourceInput(a, term, polynomialSOMInput{polynomial: p}, true)
}
func (b *polynomialSOM) addSourceInput(a *Int, term Monomial, p polynomialSOMInput, multiply bool) error {
	m := b.owner
	if multiply {
		z, e := m.modular.IsZero(a)
		if e != nil || z {
			return e
		}
	}
	n, e := p.size(m)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		other, e := p.monomial(m, i)
		if e != nil {
			return e
		}
		prod := other
		if multiply {
			prod, e = m.generatedProduct(term, other)
			if e != nil {
				return e
			}
		}
		pos := ^uint32(0)
		if !b.cheap {
			pos, e = b.positions.Get(prod)
			if e != nil {
				return e
			}
		}
		if pos == ^uint32(0) {
			n, e := b.ms.size(m)
			if e != nil {
				return e
			}
			if b.cheap && modularSourceDebug {
				for j := uint32(0); j < n; j++ {
					x, e := b.ms.monomial(m, j)
					if e != nil {
						return e
					}
					if x == prod {
						return ErrPolynomialAssertion
					}
				}
			}
			if b.retirementOwner != nil {
				if e = b.retirementOwner.prepareRetirement(n + 1); e != nil {
					return e
				}
			}
			if !b.cheap {
				if e = b.positions.Set(prod, n); e != nil {
					return e
				}
				if e = b.ms.appendMonomial(m, prod); e != nil {
					return e
				}
				b.borrowedAppend = true
				b.borrowedAppendIndex = n
				if multiply && m.generatedActive {
					m.generatedBuffer = b
					m.generatedIndex = n
				}
			}
			if e = m.monomials.IncRef(prod); e != nil {
				return e
			}
			if !b.cheap {
				b.borrowedAppend = false
			}
			if b.cheap {
				b.pendingMonomial = prod
				b.hasPendingReference = true
				if multiply {
					m.transferGeneratedMonomial()
				}
				if e = b.ms.appendMonomial(m, prod); e != nil {
					return e
				}
				b.hasPendingReference = false
				b.pendingMonomial = Monomial{}
			}
			if !b.cheap {
				if multiply {
					m.transferGeneratedMonomial()
				}
			}
			if e = b.as.appendCoefficient(m, Int{}); e != nil {
				return e
			}
			out, e := b.as.coefficient(m, n)
			if e != nil {
				return e
			}
			source, e := p.coefficient(m, i)
			if e != nil {
				return e
			}
			if multiply {
				e = m.compositionNumericCall(2, a, &source, nil, &out)
			} else {
				e = m.compositionNumericCall(0, &source, nil, nil, &out)
			}
			if w := b.as.putCoefficient(m, n, out); w != nil {
				return w
			}
			if e != nil {
				return e
			}
		} else {
			if multiply {
				m.transferGeneratedMonomial()
			}
			out, e := b.as.coefficient(m, pos)
			if e != nil {
				return e
			}
			source, e := p.coefficient(m, i)
			if e != nil {
				return e
			}
			if multiply {
				e = m.compositionNumericCall(3, &out, a, &source, &out)
			} else {
				e = m.compositionNumericCall(1, &out, &source, nil, &out)
			}
			if w := b.as.putCoefficient(m, pos, out); w != nil {
				return w
			}
			if e != nil {
				return e
			}
		}
	}
	return nil
}
func (b *polynomialSOM) reset() error {
	if b.resetActive {
		return ErrPolynomialPending
	}
	m := b.owner
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	sz, e := b.as.size(m)
	if e != nil {
		return e
	}
	if modularSourceDebug && sz != n {
		return ErrPolynomialAssertion
	}
	b.resetActive = true
	for i := uint32(0); i < n; i++ {
		b.resetIndex = i
		b.resetStage = 0
		term, e := b.ms.monomial(m, i)
		if e != nil {
			return e
		}
		if !b.cheap {
			if e = b.positions.Reset(term); e != nil {
				return e
			}
		}
		b.resetStage = 1
		if i >= sz {
			return arena.ErrBounds
		}
		a, e := b.as.coefficient(m, i)
		if e != nil {
			return e
		}
		if b.cheap {
			e = m.deleteNumeral(&a)
		} else {
			e = m.modular.Reset(&a)
		}
		if w := b.as.putCoefficient(m, i, a); w != nil {
			return w
		}
		if e != nil {
			return e
		}
		b.resetStage = 3
		if e = m.monomials.DecRef(term); e != nil {
			return e
		}
		b.resetIndex = i + 1
		b.resetStage = 0
	}
	if e = b.as.reset(m); e != nil {
		return e
	}
	if e = b.ms.reset(m); e != nil {
		return e
	}
	b.resetActive = false
	b.resetIndex = 0
	b.resetStage = 0
	return nil
}
func (b *polynomialSOM) removeZeros(normalize bool) error {
	m := b.owner
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	sz, e := b.as.size(m)
	if e != nil {
		return e
	}
	if modularSourceDebug && sz != n {
		return ErrPolynomialAssertion
	}
	if b.removeActive {
		return ErrPolynomialPending
	}
	b.removeActive = true
	b.removeStopped = false
	b.removePrefix = 0
	b.removeSuffix = 0
	j := uint32(0)
	for i := uint32(0); i < n; i++ {
		b.removeSuffix = i
		b.removeStopped = false
		term, e := b.ms.monomial(m, i)
		if e != nil {
			return e
		}
		if e = b.positions.Reset(term); e != nil {
			return e
		}
		a, e := b.as.coefficient(m, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(&a)
		if e != nil {
			return e
		}
		if zero {
			e = m.modular.Reset(&a)
			if w := b.as.putCoefficient(m, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
			b.removeStopped = true
			if e = m.monomials.DecRef(term); e != nil {
				return e
			}
			b.removeStopped = false
			b.removeSuffix = i + 1
		} else {
			if i != j {
				if modularSourceDebug {
					old, e := b.ms.monomial(m, j)
					if e != nil {
						return e
					}
					if old == term {
						return ErrPolynomialAssertion
					}
				}
				if e = b.ms.putMonomial(m, j, term); e != nil {
					return e
				}
				// The source raw pointer copy does not increment refs. Ownership
				// follows that one real destination; the old source slot is a
				// borrowed alias until the source shrink removes it.
				b.removePrefix = j + 1
				b.removeSuffix = i + 1
				old, e := b.as.coefficient(m, j)
				if e != nil {
					return e
				}
				e = m.modular.Swap(&old, &a)
				if w := b.as.putCoefficient(m, j, old); w != nil {
					return w
				}
				if w := b.as.putCoefficient(m, i, a); w != nil {
					return w
				}
				if e != nil {
					return e
				}
			}
			j++
			b.removePrefix = j
			b.removeSuffix = i + 1
		}
	}
	if modularSourceDebug {
		for i := j; i < n; i++ {
			a, e := b.as.coefficient(m, i)
			if e != nil {
				return e
			}
			zero, e := m.modular.IsZero(&a)
			if e != nil {
				return e
			}
			if !zero {
				return ErrPolynomialAssertion
			}
		}
	}
	if n != 0 {
		if e = b.as.publishSize(m, j); e != nil {
			return e
		}
		if e = b.ms.publishSize(m, j); e != nil {
			return e
		}
	}
	b.removeActive = false
	b.removePrefix = 0
	b.removeSuffix = 0
	b.removeStopped = false
	if normalize {
		return b.normalize()
	}
	return nil
}
func (b *polynomialSOM) normalize() (result error) {
	m := b.owner
	n, e := b.as.size(m)
	if e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	// A fresh completed default scoped_numeral is constructed only after the
	// source empty early return. An interrupted previous local is not repaired.
	if b.gcdActive {
		return ErrPolynomialPending
	}
	b.gcd = Int{}
	b.gcdActive = true
	b.gcdBodyCause = nil
	b.gcdCleanupCause = nil
	defer func() {
		b.gcdBodyCause = result
		cleanup := m.deleteNumeral(&b.gcd)
		b.gcdCleanupCause = cleanup
		if cleanup == nil {
			b.gcdActive = false
		} else if result == nil {
			result = cleanup
		}
	}()
	if e = m.modular.GCDManyIndexed(integerArrayView{runtime: m.memory, raw: b.as.raw, count: n, offset: 8, stride: 12}, &b.gcd); e != nil {
		return e
	}
	one, e := m.modular.IsOne(&b.gcd)
	if e != nil {
		return e
	}
	if !one {
		if modularSourceDebug {
			pos, e := m.modular.IsPos(&b.gcd)
			if e != nil {
				return e
			}
			if !pos {
				return ErrPolynomialAssertion
			}
		}
		for i := uint32(0); i < n; i++ {
			a, e := b.as.coefficient(m, i)
			if e != nil {
				return e
			}
			e = m.compositionNumericCall(4, &a, &b.gcd, nil, &a)
			if w := b.as.putCoefficient(m, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
		}
	}
	return nil
}
func (b *polynomialSOM) mk(normalize bool) (Polynomial, error) {
	if !b.cheap {
		if e := b.removeZeros(normalize); e != nil {
			return Polynomial{}, e
		}
	}
	n, e := b.ms.size(b.owner)
	if e != nil {
		return Polynomial{}, e
	}
	p, e := b.owner.mkCell(n, nil, nil, b, nil)
	if e != nil {
		return Polynomial{}, e
	}
	if e = b.as.reset(b.owner); e != nil {
		return Polynomial{}, e
	}
	if e = b.ms.reset(b.owner); e != nil {
		return Polynomial{}, e
	}
	b.transferred = 0
	return p, nil
}
func (b *polynomialSOM) close() error {
	if b.owner == nil {
		return nil
	}
	m := b.owner
	if b.hasPendingReference {
		if e := m.monomials.DecRef(b.pendingMonomial); e != nil {
			return e
		}
		b.hasPendingReference = false
		b.pendingMonomial = Monomial{}
	}
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	as, e := b.as.size(m)
	if e != nil {
		return e
	}
	if b.as.interrupted && as > b.as.initialized {
		as = b.as.initialized
	}
	// An interrupted source reset has already consumed the preceding actual
	// monomial fields. Owning cleanup resumes known fields, never replays reset.
	if b.resetActive && b.cleanupIndex < b.resetIndex {
		b.cleanupIndex = b.resetIndex
	}
	for b.cleanupIndex < n || b.cleanupIndex < as {
		i := b.cleanupIndex
		if i < as {
			a, e := b.as.coefficient(m, i)
			if e != nil {
				return e
			}
			e = m.deleteNumeral(&a)
			if w := b.as.putCoefficient(m, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
		}
		if i < n && !(b.borrowedAppend && i == b.borrowedAppendIndex) && !(b.unreferencedConsumed && i == b.unreferencedIndex) && i >= b.transferred && (!b.removeActive || i < b.removePrefix || i >= b.removeSuffix) {
			x, e := b.ms.monomial(m, i)
			if e != nil {
				return e
			}
			if !b.cheap && !(b.resetActive && i == b.resetIndex && b.resetStage >= 1) && !(b.removeActive && b.removeStopped && i == b.removeSuffix) {
				if e = b.positions.Reset(x); e != nil {
					return e
				}
			}
			if (b.resetActive && i == b.resetIndex && b.resetStage == 3) || (b.removeActive && b.removeStopped && i == b.removeSuffix) {
				l, e := m.monomials.location(x)
				if e != nil {
					return e
				}
				if l.stage != 0 {
					if e = m.monomials.CleanupValue(&x); e != nil {
						return e
					}
				} else if e = m.monomials.DecRef(x); e != nil {
					return e
				}
			} else if e = m.monomials.DecRef(x); e != nil {
				return e
			}
		}
		b.cleanupIndex++
	}
	if e = m.deleteNumeral(&b.gcd); e != nil {
		b.gcdCleanupCause = e
		return e
	}
	b.gcdActive = false
	b.gcdCleanupCause = nil
	if e = b.ms.close(m); e != nil {
		return e
	}
	if e = b.as.close(m); e != nil {
		return e
	}
	if !b.cheap && !b.positions.closed {
		if e = b.positions.CloseChecked(); e != nil {
			return e
		}
	}
	return nil
}
