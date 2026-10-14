// Source: src/math/subpaving/subpaving_t_def.h:575-587,1741-1752,
// concrete config_mpq bound queue and watched-clause dispatch.
package subpaving

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// PropagateQueued processes the exact bound queue prefix. A canceled
// checkpoint leaves the next bound queued; a conflict clears the suffix.
// Definitions will also dispatch at this boundary when registered.
func (c *RationalContext) PropagateQueued(n *RationalNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c || !n.registered {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	for c.queueHead < len(c.queue) && n.conflictVar == math.MaxUint32 {
		if err := sourceCheckpoint(c.limit, c.runtime, c.process, c.maxMemory, ErrRationalContextCanceled, nil); err != nil {
			return err
		}
		b := c.queue[c.queueHead]
		found := false
		for trail := n.trail; trail != nil; trail = trail.previous {
			if trail == b {
				found = true
				break
			}
		}
		if !found {
			return arena.ErrInvalidSlot
		}
		c.queueHead++
		for _, clause := range c.clauseWatchers[b.variable] {
			if n.conflictVar != math.MaxUint32 {
				break
			}
			if b.timestamp >= clause.visited {
				if err := c.PropagateClause(clause, n); err != nil {
					return err
				}
			}
		}
	}
	c.queue, c.queueHead = nil, 0
	return nil
}
