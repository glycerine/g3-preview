// Source: src/math/subpaving/subpaving_t_def.h:24-65,1806-1843;
// concrete config_mpff breadth-first search loop.
package subpaving

import "github.com/glycerine/g3/pkg/arena"

func (c *FastContext) removeLeaf(n *FastNode) {
	if !n.queuedLeaf {
		return
	}
	for i, leaf := range c.leafQueue {
		if leaf == n {
			copy(c.leafQueue[i:], c.leafQueue[i+1:])
			last := len(c.leafQueue) - 1
			c.leafQueue[last] = nil
			c.leafQueue = c.leafQueue[:last]
			break
		}
	}
	n.queuedLeaf = false
}

func (c *FastContext) SelectLeafBFS() (*FastNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if len(c.leafQueue) == 0 {
		return nil, nil
	}
	return c.leafQueue[0], nil
}

func (c *FastContext) PopLeafBFS() (*FastNode, error) {
	n, err := c.SelectLeafBFS()
	if err != nil || n == nil {
		return n, err
	}
	c.removeLeaf(n)
	return n, nil
}

// Search initializes the root and processes the oldest leaf first. A canceled
// propagation retains its selected child and pending queue for resumption.
// Children created here are owned by the context until Close.
func (c *FastContext) Search() error {
	if err := c.InitializeRoot(); err != nil {
		return err
	}
	for c.searchNode != nil || len(c.leafQueue) != 0 {
		if c.searchNode == nil {
			if err := sourceCheckpoint(c.limit, c.runtime, c.process, c.maxMemory, ErrFastContextCanceled); err != nil {
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
		if n.inconsistent || n.depth >= c.maxDepth || len(c.isInt) == 0 {
			c.searchNode = nil
			continue
		}
		variable, found, err := c.SelectBranchVariable(n, true)
		if err != nil {
			return err
		}
		if found {
			left, right, err := c.SplitMidpoint(n, variable, true, 1)
			if left != nil {
				c.searchOwned = append(c.searchOwned, left)
			}
			if right != nil {
				c.searchOwned = append(c.searchOwned, right)
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
