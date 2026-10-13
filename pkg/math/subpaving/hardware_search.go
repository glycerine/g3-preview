// Source: src/math/subpaving/subpaving_t_def.h:1806-1843, concrete
// config_hwf default breadth-first search loop.
package subpaving

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Search initializes the root, then processes breadth-first leaves using the
// default midpoint splitter and round-robin variable selector. A canceled
// child queue keeps its selected node and unprocessed suffix for resumption.
func (c *HardwareContext) Search() error {
	if err := c.InitializeRoot(); err != nil {
		return err
	}
	for c.searchNode != nil || c.leafHead != nil {
		if c.searchNode == nil {
			if !c.limit.Inc() {
				return ErrHardwareContextCanceled
			}
			if len(c.queue) != 0 || c.queueHead != 0 {
				return arena.ErrInvalidSlot
			}
			if c.activeNodes > c.maxNodes {
				return nil
			}
			n, err := c.PopLeafBFS()
			if err != nil || n == nil {
				return err
			}
			c.searchNode = n
			c.searchPhase = 0
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
		if n.conflictVar != math.MaxUint32 || n.depth >= c.maxDepth {
			c.searchNode = nil
			continue
		}
		variable, found, err := c.SelectBranchVariable(n, true)
		if err != nil {
			return err
		}
		if found {
			_, _, err = c.SplitMidpoint(n, variable, true, 1)
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
