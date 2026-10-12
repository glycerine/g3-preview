// Source: src/math/subpaving/subpaving_t_def.h:1170-1175,1664-1802,
// concrete config_mpfx definition watchers and bound queue.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrFixedContextCanceled = errors.New("subpaving: propagation resource canceled")

func (c *FixedContext) definitionVisited(variable uint32) (uint64, bool) {
	if p := c.polynomials[variable]; p != nil {
		return p.visited, true
	}
	if m := c.monomials[variable]; m != nil {
		return m.visited, true
	}
	return 0, false
}

func (c *FixedContext) propagateDefinition(variable uint32, n *FixedNode) error {
	c.numVisited++
	if c.monomials[variable] != nil {
		return c.PropagateMonomial(variable, n)
	}
	if c.polynomials[variable] != nil {
		return c.PropagatePolynomial(variable, n)
	}
	return arena.ErrInvalidSlot
}

// PropagateAllDefinitions is the source initialization sweep, before the
// bound queue is drained. Definitions retain variable-creation order.
func (c *FixedContext) PropagateAllDefinitions(n *FixedNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	for x := range c.isInt {
		if n.inconsistent {
			break
		}
		variable := uint32(x)
		if _, ok := c.definitionVisited(variable); ok {
			if err := c.propagateDefinition(variable, n); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *FixedContext) boundOnTrail(n *FixedNode, b *FixedBound) bool {
	for current := n.trail; current != nil; current = current.prev {
		if current == b {
			return true
		}
	}
	return false
}

func (c *FixedContext) processQueuedBound(n *FixedNode, b *FixedBound) error {
	if b.variable >= uint32(len(c.watchers)) || !c.boundOnTrail(n, b) {
		return arena.ErrInvalidSlot
	}
	visit := func(variable uint32, catchArithmetic bool) error {
		visited, ok := c.definitionVisited(variable)
		if !ok || b.timestamp < visited {
			return nil
		}
		if err := c.propagateDefinition(variable, n); err != nil {
			if catchArithmetic && (errors.Is(err, numeral.ErrFixedPointOverflow) || errors.Is(err, numeral.ErrFixedPointDivZero) || errors.Is(err, numeral.ErrFixedPointDomain)) {
				c.arithmeticFailed = true
				return nil
			}
			return err
		}
		return nil
	}
	for _, variable := range c.watchers[b.variable] {
		if n.inconsistent {
			return nil
		}
		if err := visit(variable, true); err != nil {
			return err
		}
	}
	if n.inconsistent {
		return nil
	}
	if _, ok := c.definitionVisited(b.variable); ok {
		return visit(b.variable, false)
	}
	return nil
}

// PropagateQueued processes the live queue, including bounds added by the
// current definition. A canceled or failed checkpoint retains the unprocessed
// suffix so a later call can resume it.
func (c *FixedContext) PropagateQueued(n *FixedNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	for c.queueHead < len(c.queue) && !n.inconsistent {
		if !c.limit.Inc() {
			return ErrFixedContextCanceled
		}
		b := c.queue[c.queueHead]
		c.queueHead++
		if err := c.processQueuedBound(n, b); err != nil {
			return err
		}
	}
	c.queue = nil
	c.queueHead = 0
	return nil
}
