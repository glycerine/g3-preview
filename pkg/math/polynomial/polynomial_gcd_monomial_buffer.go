// Source: src/util/ref_buffer.h:32-108; src/util/buffer.h;
// src/math/polynomial/polynomial.cpp:3895-4015.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

// Actual ref_buffer<monomial,manager,16>. INC-before-push is represented by
// incoming until the actual payload write and size publication complete.
type polynomialGCDMonomialBuffer struct {
	inline                                                     [16]Monomial
	raw, pending                                               util.Block
	sourceRaw                                                  util.Block
	rawDestroyed                                               bool
	size, capacity, copied, cursor                             uint32
	incoming                                                   Monomial
	incomingStarted, dropping, freeStarted, constructed, ended bool
	sourceCause                                                error
}

func (b *polynomialGCDMonomialBuffer) construct() {
	*b = polynomialGCDMonomialBuffer{capacity: 16, constructed: true}
}
func (b *polynomialGCDMonomialBuffer) get(c *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	if polynomialSourceDebug && i >= b.size {
		return Monomial{}, ErrPolynomialAssertion
	}
	if b.rawDestroyed {
		return skeletonReadMonomial(c.memory, b.sourceRaw, uint64(i)*16)
	}
	if b.raw == (util.Block{}) {
		if i >= 16 {
			return Monomial{}, ErrPolynomialOwner
		}
		return b.inline[i], nil
	}
	return skeletonReadMonomial(c.memory, b.raw, uint64(i)*16)
}
func (b *polynomialGCDMonomialBuffer) push(c *MultivariatePolynomialCore, a Monomial) error {
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.incoming != (Monomial{}) || b.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	if e := c.monomials.IncRef(a); e != nil {
		return e
	}
	b.incoming = a
	if c.generatedActive && c.generatedMonomial == a {
		c.transferGeneratedMonomial()
	}
	if b.size == b.capacity {
		n := b.capacity * 2
		var e error
		b.pending, e = c.memory.Allocate(uint64(n) * 16)
		if e != nil {
			return e
		}
		for b.copied < b.size {
			a, e := b.get(c, b.copied)
			if e != nil {
				return e
			}
			if e = skeletonWriteMonomial(c.memory, b.pending, uint64(b.copied)*16, a); e != nil {
				return e
			}
			b.copied++
		}
		if b.raw != (util.Block{}) {
			b.freeStarted = true
			b.sourceRaw = b.raw
			if e = c.memory.Deallocate(&b.raw); e != nil {
				b.rawDestroyed = b.raw == (util.Block{})
				b.sourceCause = e
				return e
			}
		}
		b.sourceRaw = util.Block{}
		b.rawDestroyed = false
		b.raw = b.pending
		b.pending = util.Block{}
		b.copied = 0
		b.capacity = n
		b.freeStarted = false
	}
	if b.raw == (util.Block{}) {
		b.inline[b.size] = a
	} else {
		if e := skeletonWriteMonomial(c.memory, b.raw, uint64(b.size)*16, a); e != nil {
			return e
		}
	}
	b.size++
	b.incoming = Monomial{}
	return nil
}
func (b *polynomialGCDMonomialBuffer) reset(c *MultivariatePolynomialCore, owning bool) error {
	if b.sourceCause != nil && !owning {
		return b.sourceCause
	}
	for b.cursor < b.size {
		var a Monomial
		var e error
		if owning && b.rawDestroyed {
			if b.copied != b.size || b.pending == (util.Block{}) {
				return ErrPolynomialPending
			}
			a, e = skeletonReadMonomial(c.memory, b.pending, uint64(b.cursor)*16)
		} else {
			a, e = b.get(c, b.cursor)
		}
		if e != nil {
			return e
		}
		r := polynomialDivisionReference{cause: b.sourceCause}
		if e = r.drop(c, &a, &b.dropping, owning); e != nil {
			b.sourceCause = e
			return e
		}
		// Source keeps stale pointer bytes until logical reset. Cursor records
		// completed DEC so explicit owning continuation cannot repeat it.
		b.cursor++
	}
	b.size = 0
	b.cursor = 0
	return nil
}
func (b *polynomialGCDMonomialBuffer) close(c *MultivariatePolynomialCore, owning bool) error {
	return b.closePhase(c, owning, owning)
}

// Completed class destruction includes the published base buffer. Unjoined
// INC and failed expansion storage are separate source-invisible owning roots.
func (b *polynomialGCDMonomialBuffer) closePhase(c *MultivariatePolynomialCore, owning, abort bool) error {
	if !b.constructed {
		return nil
	}
	if !b.ended {
		if e := b.reset(c, owning); e != nil {
			return e
		}
		if b.raw != (util.Block{}) {
			b.freeStarted = true
			if e := c.memory.Deallocate(&b.raw); e != nil {
				b.sourceCause = e
				return e
			}
		}
		if !owning && b.sourceCause != nil {
			return b.sourceCause
		}
		b.ended = true
	}
	if abort {
		if b.pending != (util.Block{}) {
			if e := c.memory.Deallocate(&b.pending); e != nil {
				return e
			}
		}
		r := polynomialDivisionReference{cause: b.sourceCause}
		if e := r.drop(c, &b.incoming, &b.incomingStarted, true); e != nil {
			return e
		}
	}
	return nil
}
