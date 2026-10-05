// Source: src/math/polynomial/polynomial.cpp:2217-2255,2280-2299.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type polynomialCheapAppendScope struct {
	invocation                                    uint64
	active, stopped, foreignMixed                 bool
	coefficients, monomials                       uint32
	pending                                       Monomial
	coefficientCursor, monomialCursor             uint32
	numeric                                       polynomialNumericRoots
	factoryInvocation                             uint64
	transferredCoefficients, transferredMonomials uint32
	completedTransfer                             bool
	movingCoefficient                             Int
}
type polynomialCheapAppendInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialCheapAppendScope
	invocation uint64
}

func (b *polynomialSOM) matchesAppendInvocation(r *polynomialCheapAppendInvocationReceipt) bool {
	return r != nil && r.manager == b.owner && r.frame == &b.appendScope && r.invocation != 0 && r.invocation == b.appendScope.invocation && b.appendScope.active
}
func (b *polynomialSOM) acquireAppendInvocation(r *polynomialCheapAppendInvocationReceipt, n uint32) error {
	if !b.cheap || b != &b.owner.cheap {
		return ErrPolynomialOwner
	}
	s := &b.appendScope
	as, e := b.as.size(b.owner)
	if e != nil {
		return e
	}
	if s.active {
		if !b.matchesAppendInvocation(r) || s.stopped || s.foreignMixed || as != s.coefficients || n != s.monomials {
			return ErrPolynomialPending
		}
		return nil
	}
	if n != 0 || as != 0 || b.hasPendingReference || s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	*s = polynomialCheapAppendScope{active: true, invocation: s.invocation + 1}
	*r = polynomialCheapAppendInvocationReceipt{manager: b.owner, frame: s, invocation: s.invocation}
	return nil
}
func (b *polynomialSOM) addWithAppendReceipt(a *Int, term Monomial, take bool, r *polynomialCheapAppendInvocationReceipt) error {
	return b.addConsumedReceipts(a, term, take, nil, nil, r)
}

// This partial prerequisite can retire a matched unmixed append cohort.
// Foreign mixed vectors stop before any row or numeric token is consumed.
// Ordered retirement through a later factory transfer remains open.
func (b *polynomialSOM) cleanupAppendInvocation(r *polynomialCheapAppendInvocationReceipt, abort bool) error {
	if !abort || !b.matchesAppendInvocation(r) {
		return nil
	}
	s := &b.appendScope
	m := b.owner
	as, e := b.as.size(m)
	if e != nil {
		return e
	}
	n, e := b.ms.size(m)
	if e != nil {
		return e
	}
	if s.foreignMixed || b.resetActive || b.removeActive {
		return ErrPolynomialPending
	}
	if s.completedTransfer {
		if as != 0 || n != 0 || s.transferredCoefficients != s.coefficients || s.transferredMonomials != s.monomials {
			return ErrPolynomialPending
		}
	} else if as > s.coefficients || n != s.monomials || b.transferred != s.transferredMonomials {
		return ErrPolynomialPending
	}
	if b.hasPendingReference && (s.pending == (Monomial{}) || b.pendingMonomial != s.pending) {
		return ErrPolynomialPending
	}
	if e = m.deleteNumeral(&s.movingCoefficient); e != nil {
		return e
	}
	if s.coefficientCursor < s.transferredCoefficients {
		s.coefficientCursor = s.transferredCoefficients
	}
	if s.monomialCursor < s.transferredMonomials {
		s.monomialCursor = s.transferredMonomials
	}
	for s.coefficientCursor < s.coefficients {
		i := s.coefficientCursor
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
		s.coefficientCursor++
	}
	if s.pending != (Monomial{}) {
		if e = m.monomials.DecRef(s.pending); e != nil {
			return e
		}
		s.pending = Monomial{}
		b.pendingMonomial = Monomial{}
		b.hasPendingReference = false
	}
	for s.monomialCursor < s.monomials {
		term, e := b.ms.monomial(m, s.monomialCursor)
		if e != nil {
			return e
		}
		if e = m.monomials.DecRef(term); e != nil {
			return e
		}
		s.monomialCursor++
	}
	if e = b.as.reset(m); e != nil {
		return e
	}
	if e = b.ms.reset(m); e != nil {
		return e
	}
	if e = s.numeric.cleanup(m.integer); e != nil {
		return e
	}
	*s = polynomialCheapAppendScope{invocation: s.invocation}
	return nil
}
