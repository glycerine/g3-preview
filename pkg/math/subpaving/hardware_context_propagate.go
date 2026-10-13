// Source: src/math/subpaving/subpaving_t_def.h:1170-1175,1683-1802,
// concrete config_hwf polynomial watchers and bound queue.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func (c *HardwareContext) definitionVisited(variable uint32) (uint64, bool) {
	if p := c.polynomials[variable]; p != nil {
		return p.visited, true
	}
	return 0, false
}

func (c *HardwareContext) propagateDefinition(variable uint32, n *HardwareNode) error {
	if c.polynomials[variable] == nil {
		return arena.ErrInvalidSlot
	}
	c.numVisited++
	return c.PropagatePolynomial(variable, n)
}

// PropagateAllDefinitions visits definitions in variable-creation order.
func (c *HardwareContext) PropagateAllDefinitions(n *HardwareNode) error {
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
		if n.conflictVar != math.MaxUint32 {
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

func (c *HardwareContext) boundOnTrail(n *HardwareNode, b *HardwareBound) bool {
	for current := n.trail; current != nil; current = current.prev {
		if current == b {
			return true
		}
	}
	return false
}

func (c *HardwareContext) processQueuedBound(n *HardwareNode, b *HardwareBound) error {
	if b == nil || b.variable >= uint32(len(c.watchers)) || !c.boundOnTrail(n, b) {
		return arena.ErrInvalidSlot
	}
	visit := func(variable uint32, catchArithmetic bool) error {
		visited, ok := c.definitionVisited(variable)
		if !ok || b.timestamp < visited {
			return nil
		}
		if err := c.propagateDefinition(variable, n); err != nil {
			if catchArithmetic && errors.Is(err, numeral.ErrHardwareFloatNonRegular) {
				c.arithmeticFailed = true
				return nil
			}
			return err
		}
		return nil
	}
	for _, watched := range c.watchers[b.variable] {
		if n.conflictVar != math.MaxUint32 {
			return nil
		}
		if err := visit(watched, true); err != nil {
			return err
		}
	}
	if n.conflictVar != math.MaxUint32 {
		return nil
	}
	if _, ok := c.definitionVisited(b.variable); ok {
		return visit(b.variable, false)
	}
	return nil
}

// PropagateQueued drains the live queue, including bounds added by watcher
// visits. Checkpoint cancellation leaves the unprocessed suffix for resumption.
func (c *HardwareContext) PropagateQueued(n *HardwareNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	for c.queueHead < len(c.queue) && n.conflictVar == math.MaxUint32 {
		if !c.limit.Inc() {
			return ErrHardwareContextCanceled
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
