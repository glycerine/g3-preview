// Source: src/math/subpaving/subpaving_t_def.h:1753-1808, concrete config_mpf
// root creation, unit assertion, definition sweep, then bound queue.
package subpaving

import "math"

// InitializeRoot retains completed stages so a canceled unit or queue
// checkpoint resumes without replaying earlier bound publications.
func (c *ArbitraryContext) InitializeRoot() error {
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
			} else if c.monomials[variable] != nil {
				if err := c.PropagateMonomial(variable, n); err != nil {
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
