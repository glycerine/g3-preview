// Source: src/math/subpaving/subpaving_t_def.h:1755-1802, concrete
// config_hwf root creation and propagation order.
package subpaving

import "math"

// InitializeRoot asserts units, visits definitions in variable order, then
// drains the resulting bound queue. Completed stages survive cancellation.
func (c *HardwareContext) InitializeRoot() error {
	if err := c.open(); err != nil {
		return err
	}
	if c.root == nil {
		if _, err := c.CreateRootNode(); err != nil {
			return err
		}
	}
	n := c.root
	if err := n.open(); err != nil {
		return err
	}
	if c.initStage == 0 {
		if err := c.AssertUnitClauses(n); err != nil {
			return err
		}
		c.initStage = 1
	}
	if c.initStage == 1 {
		for c.initDefHead < uint32(len(c.isInt)) && n.conflictVar == math.MaxUint32 {
			variable := c.initDefHead
			if _, ok := c.definitionVisited(variable); ok {
				if err := c.propagateDefinition(variable, n); err != nil {
					return err
				}
			}
			c.initDefHead++
		}
		c.initStage = 2
	}
	if c.initStage == 2 {
		if err := c.PropagateQueued(n); err != nil {
			return err
		}
		c.initStage = 3
	}
	return nil
}
