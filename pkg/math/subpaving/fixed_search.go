// Source: src/math/subpaving/subpaving_t_def.h:24-65,1806-1843;
// concrete config_mpfx breadth-first search loop.
package subpaving

import "github.com/glycerine/g3/pkg/arena"

func fixedNodeDepth(n *FixedNode) uint32 {
	var depth uint32
	for n.parent != nil {
		depth++
		n = n.parent
	}
	return depth
}

// Search initializes the root and processes the oldest leaf first. A stopped
// propagation retains its selected child and queue for the next invocation.
// Children created by this loop remain context-owned until Close.
func (c *FixedContext) Search() error {
	if err := c.InitializeRoot(); err != nil {
		return err
	}
	if !c.searchStarted {
		if c.root.children != 0 {
			return arena.ErrInvalidSlot
		}
		c.leafQueue = []*FixedNode{c.root}
		c.searchStarted = true
	}
	for c.searchNode != nil || len(c.leafQueue) != 0 {
		if c.searchNode == nil {
			if err := sourceCheckpoint(c.limit, nil, nil, c.maxMemory, ErrFixedContextCanceled, c.cooperation); err != nil {
				return err
			}
			if len(c.queue) != 0 || c.queueHead != 0 {
				return arena.ErrInvalidSlot
			}
			if uint64(1+len(c.searchOwned)) > uint64(c.maxNodes) {
				return nil
			}
			c.searchNode = c.leafQueue[0]
			c.leafQueue[0] = nil
			c.leafQueue = c.leafQueue[1:]
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
		if n.inconsistent || fixedNodeDepth(n) >= c.maxDepth || len(c.isInt) == 0 {
			c.searchNode = nil
			continue
		}
		variable, found, err := c.SelectBranchVariable(n, true)
		if err != nil {
			return err
		}
		if found {
			left, right, err := c.SplitMidpoint(n, variable, true, 1)
			for _, child := range []*FixedNode{left, right} {
				if child != nil {
					c.searchOwned = append(c.searchOwned, child)
					c.leafQueue = append(c.leafQueue, child)
				}
			}
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
