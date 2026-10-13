// Source: src/math/subpaving/subpaving_t_def.h:1170-1175,1664-1802,
// concrete config_mpff definition watchers and bound queue.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrFastContextCanceled = errors.New("subpaving: propagation resource canceled")

func (c *FastContext) definitionVisited(variable uint32) (uint64, bool) {
	if p := c.polynomials[variable]; p != nil {
		return p.visited, true
	}
	return 0, false
}

func (c *FastContext) propagateDefinition(variable uint32, n *FastNode) error {
	c.numVisited++
	if c.polynomials[variable] != nil {
		return c.PropagatePolynomial(variable, n)
	}
	return arena.ErrInvalidSlot
}

// PropagateAllDefinitions is the source initialization sweep, before the
// bound queue is drained. Definitions retain variable-creation order.
func (c *FastContext) PropagateAllDefinitions(n *FastNode) error {
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

func (c *FastContext) boundOnTrail(n *FastNode, b *FastBound) bool {
	for current := n.trail; current != nil; current = current.prev {
		if current == b {
			return true
		}
	}
	return false
}

// AddRecentBounds requeues the current bounds added since the parent trail.
// Source traversal is newest first and gives every replayed bound the current
// context timestamp without advancing it.
func (c *FastContext) AddRecentBounds(n *FastNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c || n.parent == nil || len(c.queue) != 0 || c.queueHead != 0 {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	boundary := n.parent.trail
	var recent []*FastBound
	current := n.trail
	for current != boundary && current != nil {
		b := current
		if b.variable >= uint32(len(n.lower)) {
			return arena.ErrInvalidSlot
		}
		if b.lower && n.lower[b.variable] == b || !b.lower && n.upper[b.variable] == b {
			recent = append(recent, b)
		}
		current = current.prev
	}
	if current != boundary {
		return arena.ErrInvalidSlot
	}
	for _, b := range recent {
		b.timestamp = c.timestamp
		c.queue = append(c.queue, b)
	}
	return nil
}

func (c *FastContext) processQueuedBound(n *FastNode, b *FastBound) error {
	if b.variable >= uint32(len(c.watchers)) || !c.boundOnTrail(n, b) {
		return arena.ErrInvalidSlot
	}
	visit := func(variable uint32, catchArithmetic bool) error {
		visited, ok := c.definitionVisited(variable)
		if !ok || b.timestamp < visited {
			return nil
		}
		if err := c.propagateDefinition(variable, n); err != nil {
			if catchArithmetic && (errors.Is(err, numeral.ErrFastFloatOverflow) || errors.Is(err, numeral.ErrFastFloatDivisionByZero) || errors.Is(err, numeral.ErrFastFloatDomain)) {
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
func (c *FastContext) PropagateQueued(n *FastNode) error {
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
			return ErrFastContextCanceled
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
