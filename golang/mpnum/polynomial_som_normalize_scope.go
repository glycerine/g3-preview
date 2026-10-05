// Source: src/math/polynomial/polynomial.cpp:1863-1876.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "github.com/glycerine/g3/golang/arena"

// Ordinary lexical destruction never acknowledges a stopped source pointer.
// The owning branch is reached only by explicit caller cleanup/owner close.
func (b *polynomialSOM) finishNormalizeNumeral(owning bool) error {
	return b.finishNormalizeNumeralPhase(owning, owning)
}
func (b *polynomialSOM) finishNormalizeNumeralPhase(owning, abort bool) error {
	if !b.gcdActive {
		return nil
	}
	if !owning && b.gcdCleanupCause != nil {
		return b.gcdCleanupCause
	}
	if !b.gcdEnded || owning {
		var e error
		if owning {
			e = b.owner.deleteNumeral(&b.gcd)
		} else {
			_, e = sourceNumeralDel(b.owner.integer, &b.gcd)
		}
		if e != nil {
			b.gcdCleanupCause = e
			return e
		}
		b.gcdEnded = true
	}
	if owning && abort {
		if e := b.gcdNumeric.cleanup(b.owner.integer); e != nil {
			return e
		}
	}
	pending := b.gcdNumeric.cell.object.OwnerID() != 0
	for _, raw := range b.gcdNumeric.allocator {
		pending = pending || raw.OwnerID() != 0
	}
	if !pending {
		b.gcdActive = false
		b.gcdGeneration = 0
		b.gcdSubstitution = nil
		b.gcdNormalization = nil
		b.gcdCleanupCause = nil
		b.gcdNumeric = polynomialNumericRoots{}
	}
	return nil
}

func (b *polynomialSOM) beginNormalizeNumeral(parent *polynomialNormalizeScope) error {
	m := b.owner
	if b.gcdActive {
		return ErrPolynomialPending
	}
	if b.gcdInvocation == ^uint64(0) {
		return arena.ErrBounds
	}
	b.gcdInvocation++
	b.gcdNormalization = parent
	if parent != nil {
		parent.somInvocation = b.gcdInvocation
	}
	b.gcdSubstitution = nil
	if b == &m.som && m.substitution.scope.executing && m.callbackDepth == 0 {
		b.gcdSubstitution = &m.substitution.scope
		m.substitution.scope.somInvocation = b.gcdInvocation
	}
	b.gcd = Int{}
	b.gcdActive = true
	b.gcdBodyCause = nil
	b.gcdCleanupCause = nil
	b.gcdEnded = false
	b.gcdGeneration = m.pscExecutingOwner()
	return nil
}
