// Source: src/math/subpaving/subpaving_t_def.h:967-1060, concrete config_mpq.
package subpaving

func (c *RationalContext) pushLeafFront(n *RationalNode) {
	n.leafPrev = nil
	n.leafNext = c.leafHead
	if c.leafHead != nil {
		c.leafHead.leafPrev = n
	} else {
		c.leafTail = n
	}
	c.leafHead = n
	n.queuedLeaf = true
}

func (c *RationalContext) removeLeaf(n *RationalNode) {
	if !n.queuedLeaf {
		return
	}
	if n.leafPrev != nil {
		n.leafPrev.leafNext = n.leafNext
	} else {
		c.leafHead = n.leafNext
	}
	if n.leafNext != nil {
		n.leafNext.leafPrev = n.leafPrev
	} else {
		c.leafTail = n.leafPrev
	}
	n.leafPrev, n.leafNext = nil, nil
	n.queuedLeaf = false
}

func (c *RationalContext) SelectLeafBFS() (*RationalNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.leafTail, nil
}

func (c *RationalContext) PopLeafBFS() (*RationalNode, error) {
	n, err := c.SelectLeafBFS()
	if err != nil || n == nil {
		return n, err
	}
	c.removeLeaf(n)
	return n, nil
}
