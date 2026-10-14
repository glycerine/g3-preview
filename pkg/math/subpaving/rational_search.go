// Source: src/math/subpaving/subpaving_t_def.h:1665-1682,1806-1843,
// concrete config_mpq recent-bound propagation and breadth-first search.
package subpaving

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// AddRecentBounds queues the newest bound for each affected direction since
// the parent trail. Decision bounds keep the current timestamp until selected.
func (c *RationalContext) AddRecentBounds(n *RationalNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c || !n.registered || n.parent == nil || len(c.queue) != 0 || c.queueHead != 0 || n.conflictVar != math.MaxUint32 {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	stop := n.parent.trail
	for b := n.trail; b != stop; b = b.previous {
		if b == nil {
			return arena.ErrInvalidSlot
		}
		if b.lower && n.lower[b.variable] != b || !b.lower && n.upper[b.variable] != b {
			continue
		}
		b.timestamp = c.timestamp
		c.queue = append(c.queue, b)
	}
	return nil
}

// Search resumes a selected child after cancellation without replaying its
// recent-bound queue. Children remain owned by the context until Close.
func (c *RationalContext) Search() error {
	if err := c.InitializeRoot(); err != nil {
		return err
	}
	for c.searchNode != nil || c.leafHead != nil {
		if c.searchNode == nil {
			if err := sourceCheckpoint(c.limit, c.runtime, c.process, c.maxMemory, ErrRationalContextCanceled, c.cooperation); err != nil {
				return err
			}
			if len(c.queue) != 0 || c.queueHead != 0 {
				return arena.ErrInvalidSlot
			}
			if uint64(len(c.nodes)) > uint64(c.maxNodes) {
				return nil
			}
			n, err := c.PopLeafBFS()
			if err != nil || n == nil {
				return err
			}
			c.searchNode, c.searchPhase = n, 0
		}
		n := c.searchNode
		if c.searchPhase == 0 {
			if n != c.root {
				if err := c.AddRecentBounds(n); err != nil {
					return err
				}
			}
			c.searchPhase = 1
		}
		if c.searchPhase == 1 {
			if n != c.root {
				if err := c.PropagateQueued(n); err != nil {
					return err
				}
			}
			c.searchPhase = 2
		}
		if n.conflictVar != math.MaxUint32 || n.depth >= c.maxDepth || len(c.variables) == 0 {
			c.searchNode = nil
			continue
		}
		variable, found, err := c.SelectBranchVariable(n, true)
		if err != nil {
			return err
		}
		if found {
			_, _, err := c.SplitMidpoint(n, variable, true, 1)
			if err != nil {
				c.searchNode = nil
				return err
			}
			c.numSplits++
		}
		c.searchNode = nil
	}
	return nil
}
