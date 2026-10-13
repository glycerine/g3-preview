// Source: src/math/subpaving/subpaving_t_def.h:1694-1750, concrete config_mpf
// bound queue and insertion-order watched clause dispatch.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type arbitraryWatch struct {
	definition uint32
	clause     *ArbitraryClause
}

// PropagateQueued checkpoints before each bound and processes clause and
// definition watches in their original interleaved registration order.
func (c *ArbitraryContext) PropagateQueued(n *ArbitraryNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if err := c.validNode(n); err != nil {
		return err
	}
	for c.queueHead < len(c.queue) && n.conflictVar == math.MaxUint32 {
		if err := sourceCheckpoint(c.limit, c.process, c.maxMemory, ErrArbitraryContextCanceled); err != nil {
			return err
		}
		b := c.queue[c.queueHead]
		if b == nil || b.context != c || b.variable >= uint32(len(c.variables)) || !c.boundOnTrail(n, b) {
			return arena.ErrInvalidSlot
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
						if errors.Is(err, numeral.ErrArbitraryFloatNonRegular) {
							c.arithmeticFailed = true
							continue
						}
						return err
					}
				}
			} else {
				p := c.polynomials[watch.definition]
				m := c.monomials[watch.definition]
				if p != nil && b.timestamp >= p.visited {
					if err := c.PropagatePolynomial(watch.definition, n); err != nil {
						if errors.Is(err, numeral.ErrArbitraryFloatNonRegular) {
							c.arithmeticFailed = true
							continue
						}
						return err
					}
				} else if m != nil && b.timestamp >= m.visited {
					if err := c.PropagateMonomial(watch.definition, n); err != nil {
						if errors.Is(err, numeral.ErrArbitraryFloatNonRegular) {
							c.arithmeticFailed = true
							continue
						}
						return err
					}
				} else if p == nil && m == nil {
					return arena.ErrInvalidSlot
				}
			}
		}
		if n.conflictVar == math.MaxUint32 {
			if p := c.polynomials[b.variable]; p != nil && b.timestamp >= p.visited {
				if err := c.PropagatePolynomial(b.variable, n); err != nil {
					return err
				}
			} else if m := c.monomials[b.variable]; m != nil && b.timestamp >= m.visited {
				if err := c.PropagateMonomial(b.variable, n); err != nil {
					return err
				}
			}
		}
	}
	c.queue = nil
	c.queueHead = 0
	return nil
}
