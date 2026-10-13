// Source: src/math/subpaving/subpaving_t_def.h:1694-1750, concrete config_mpf
// bound queue and insertion-order watched clause dispatch.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// A reached monomial watch remains queued for the later concrete path.
var ErrArbitraryDefinitionPending = errors.New("subpaving: arbitrary definition propagation pending")

type arbitraryWatch struct {
	definition uint32
	clause     *ArbitraryClause
}

// PropagateQueued checkpoints before each bound and processes clause watches
// in their original interleaved registration order. A reached monomial is
// refused before consuming that bound, preserving the queue for its port.
func (c *ArbitraryContext) PropagateQueued(n *ArbitraryNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n != c.root || n.context != c {
		return arena.ErrInvalidSlot
	}
	for c.queueHead < len(c.queue) && n.conflictVar == math.MaxUint32 {
		if !c.limit.Inc() {
			return ErrArbitraryContextCanceled
		}
		b := c.queue[c.queueHead]
		if b == nil || b.context != c || b.variable >= uint32(len(c.variables)) {
			return arena.ErrInvalidSlot
		}
		for _, watch := range c.watchOrder[b.variable] {
			if watch.clause == nil && c.polynomials[watch.definition] == nil {
				return ErrArbitraryDefinitionPending
			}
		}
		if c.monomials[b.variable] != nil {
			return ErrArbitraryDefinitionPending
		}
		c.queueHead++
		for _, watch := range c.watchOrder[b.variable] {
			if n.conflictVar != math.MaxUint32 {
				break
			}
			if watch.clause != nil {
				clause := watch.clause
				if b.timestamp >= clause.visited {
					if err := c.PropagateClause(clause, n); err != nil {
						return err
					}
				}
			} else {
				p := c.polynomials[watch.definition]
				if b.timestamp >= p.visited {
					if err := c.PropagatePolynomial(watch.definition, n); err != nil {
						return err
					}
				}
			}
		}
		if n.conflictVar == math.MaxUint32 {
			if p := c.polynomials[b.variable]; p != nil && b.timestamp >= p.visited {
				if err := c.PropagatePolynomial(b.variable, n); err != nil {
					return err
				}
			}
		}
	}
	c.queue = nil
	c.queueHead = 0
	return nil
}
