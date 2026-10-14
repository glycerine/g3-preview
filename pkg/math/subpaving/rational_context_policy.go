// Source: src/math/subpaving/subpaving_t_def.h:473-494, concrete config_mpq
// bound relevance numerals. Other context search parameters remain open.
package subpaving

// UpdateBoundPolicy sets epsilon to 1/denominator (or zero) and the maximum
// bound to 10^maxPower in source statement order. A failed numeral operation
// leaves its completed prefix in this owning context for Close.
func (c *RationalContext) UpdateBoundPolicy(denominator, maxPower uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	m := c.numbers
	if denominator != 0 {
		if err := m.SetInt32(&c.epsilon, int32(denominator)); err != nil {
			return err
		}
		if err := m.InvInPlace(&c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = false
	} else {
		if err := m.Reset(&c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = true
	}
	if err := m.SetInt32(&c.maxBound, 10); err != nil {
		return err
	}
	if err := m.Power(&c.maxBound, maxPower, &c.maxBound); err != nil {
		return err
	}
	if err := m.Copy(&c.minusMaxBound, &c.maxBound); err != nil {
		return err
	}
	return m.Neg(&c.minusMaxBound)
}
