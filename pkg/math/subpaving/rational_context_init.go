// Source: src/math/subpaving/subpaving_t_def.h:1773-1808, concrete config_mpq
// root creation, unit assertion, definition sweep and bound propagation.
package subpaving

import "math"

// InitializeRoot resumes after a stopped unit or queue checkpoint without
// asserting an already-published unit or redispatching an older bound.
func (c *RationalContext) InitializeRoot() error {
	if err := c.open(); err != nil {
		return err
	}
	n, err := c.Root()
	if err != nil {
		return err
	}
	if c.initStage == 0 {
		if _, err := c.AssertRootUnits(); err != nil {
			return err
		}
		c.initStage = 1
	}
	if c.initStage == 1 {
		for c.initDefHead < uint32(len(c.variables)) && n.conflictVar == math.MaxUint32 {
			variable := c.initDefHead
			if c.polynomials[variable] != nil {
				if err := c.PropagatePolynomial(variable, n); err != nil {
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
