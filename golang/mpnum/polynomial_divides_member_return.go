// Source: src/math/polynomial/polynomial.cpp:5154-5194.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// Divides returns a boolean after its NUM/ref suffix. Its R rows remain in
// the actual persistent SOM member. This handoff transfers only the tags of
// that exact invocation; it performs no source RESET, SIZE write, DEL or DEC.
func (b *polynomialSOM) transferArithmeticRowsToMember(r *polynomialArithmeticInvocationReceipt) error {
	if b == nil || b.owner == nil || !b.owner.matchesArithmeticInvocation(r) || r.frame.buffer != b {
		return nil
	}
	m, s := b.owner, r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.cleanupArithmeticInvocation(r, false); e != nil {
		return e
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if s.transportBound || s.positionPublished || s.singleton != (Monomial{}) || s.generated != (Monomial{}) || s.calleeGeneration != 0 || (m.monomials.matchesMonomialAllocatorInvocation(&s.monomialAllocator) && s.monomialAllocator.outcome.Pending != (memory.Block{})) || s.numeric.cell.object.OwnerID() != 0 || s.numeric.allocator != ([3]memory.Block{}) || m.matchesFactoryInvocation(&s.factory) || m.modular.matchesPNormalizeInvocation(&s.normalize) || m.integer.matchesAddMulInvocation(&s.addMul) {
		return ErrPolynomialPending
	}
	a := &b.appendScope
	if a.active {
		if !b.matchesAppendInvocation(&s.append) || a.pending != (Monomial{}) || a.movingCoefficient != (Int{}) || a.numeric.cell.object.OwnerID() != 0 || a.numeric.allocator != ([3]memory.Block{}) || a.retirePhase != 0 || a.stopped {
			return ErrPolynomialPending
		}
		// The per-slot acquisition tags are the authority. Claim counters are
		// checked only before changing metadata, to refuse a malformed adapter.
		var coefficients, monomials uint32
		for _, tag := range b.coefficientOwners {
			if tag == a.invocation {
				coefficients++
			}
		}
		for _, tag := range b.monomialOwners {
			if tag == a.invocation {
				monomials++
			}
		}
		if coefficients != a.coefficientClaims || monomials != a.monomialClaims {
			return ErrPolynomialPending
		}
		for i, tag := range b.coefficientOwners {
			if tag == a.invocation {
				b.coefficientOwners[i] = 0
			}
		}
		for i, tag := range b.monomialOwners {
			if tag == a.invocation {
				b.monomialOwners[i] = 0
			}
		}
		for i, tag := range b.coefficientTransferred {
			if tag == a.invocation {
				b.coefficientTransferred[i] = 0
			}
		}
		for i, tag := range b.monomialTransferred {
			if tag == a.invocation {
				b.monomialTransferred[i] = 0
			}
		}
		a.coefficientClaims, a.monomialClaims = 0, 0
		a.active = false
	}
	s.active = false
	return nil
}
