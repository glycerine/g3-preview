// Source: src/math/subpaving/subpaving_t_def.h:1773-1808, concrete config_mpq
// root creation, unit assertion and queued bound propagation.
package subpaving

// InitializeRoot resumes after a stopped unit or queue checkpoint without
// asserting an already-published unit or redispatching an older bound.
// Definition propagation is inserted between these stages when definitions
// are registered by this concrete context.
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
		if err := c.PropagateQueued(n); err != nil {
			return err
		}
		c.initStage = 2
	}
	return nil
}
