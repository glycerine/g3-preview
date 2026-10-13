// Source: src/math/subpaving/subpaving_t_def.h:1200-1317, concrete
// config_hwf candidate-bound relevance and arithmetic failure policy.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/numeral"
)

// RelevantNewBound applies the source's conflict-first, zero-epsilon,
// maximum-bound, and epsilon-gap tests in that order. The caller supplies an
// already normalized candidate, as do the retained propagation routines.
func (c *HardwareContext) RelevantNewBound(n *HardwareNode, variable uint32, candidate *numeral.HardwareFloat, lower, open bool) (relevant bool, err error) {
	if err = c.checkNode(n, variable); err != nil {
		return false, err
	}
	defer func() {
		if errors.Is(err, numeral.ErrHardwareFloatNonRegular) {
			c.arithmeticFailed = true
			relevant, err = false, nil
		}
	}()
	m := c.numbers
	var checked numeral.HardwareFloat
	if err = m.Set(&checked, candidate); err != nil {
		return false, err
	}
	currentLower, currentUpper := n.lower[variable], n.upper[variable]
	if lower {
		if currentUpper != nil && (m.Greater(candidate, &currentUpper.value) ||
			(open || currentUpper.open) && m.Equal(candidate, &currentUpper.value)) {
			return true, nil
		}
		if c.zeroEpsilon && currentLower != nil && (m.Less(candidate, &currentLower.value) ||
			(currentLower.open || !open) && m.Equal(candidate, &currentLower.value)) {
			return false, nil
		}
		if currentUpper == nil && m.Less(&c.maxBound, candidate) {
			return false, nil
		}
	} else {
		if currentLower != nil && (m.Greater(&currentLower.value, candidate) ||
			(open || currentLower.open) && m.Equal(candidate, &currentLower.value)) {
			return true, nil
		}
		if c.zeroEpsilon && currentUpper != nil && (m.Less(&currentUpper.value, candidate) ||
			(currentUpper.open || !open) && m.Equal(candidate, &currentUpper.value)) {
			return false, nil
		}
		if currentLower == nil && m.Less(candidate, &c.minusMaxBound) {
			return false, nil
		}
	}
	if c.zeroEpsilon || lower && currentLower == nil || !lower && currentUpper == nil {
		return true, nil
	}
	current := currentUpper
	if lower {
		current = currentLower
	}
	var absolute, minimum, delta numeral.HardwareFloat
	if err = m.Abs(&current.value, &absolute); err != nil {
		return false, err
	}
	if currentLower != nil && currentUpper != nil {
		if err = m.Sub(&currentUpper.value, &currentLower.value, &minimum); err != nil {
			return false, err
		}
		if m.Less(&absolute, &minimum) {
			if err = m.Set(&minimum, &absolute); err != nil {
				return false, err
			}
		}
	} else if err = m.Set(&minimum, &absolute); err != nil {
		return false, err
	}
	if err = m.SetInt32(&delta, 1); err != nil {
		return false, err
	}
	if m.Greater(&minimum, &delta) {
		if err = m.Set(&delta, &minimum); err != nil {
			return false, err
		}
	}
	if err = m.Mul(&delta, &c.epsilon, &delta); err != nil {
		return false, err
	}
	if lower {
		if err = m.Add(&currentLower.value, &delta, &delta); err != nil {
			return false, err
		}
		return m.Greater(candidate, &delta), nil
	}
	if err = m.Sub(&currentUpper.value, &delta, &delta); err != nil {
		return false, err
	}
	return m.Less(candidate, &delta), nil
}

// PropagateCandidate is the retained definition path: normalize a scalar
// candidate, test relevance, then publish only a selected bound.
func (c *HardwareContext) PropagateCandidate(n *HardwareNode, variable uint32, value *numeral.HardwareFloat, lower, open bool) (bool, error) {
	if err := c.checkNode(n, variable); err != nil {
		return false, err
	}
	var candidate numeral.HardwareFloat
	if err := c.numbers.Set(&candidate, value); err != nil {
		return false, err
	}
	if err := c.NormalizeBound(variable, &candidate, lower, &open); err != nil {
		return false, err
	}
	relevant, err := c.RelevantNewBound(n, variable, &candidate, lower, open)
	if err != nil || !relevant {
		return relevant, err
	}
	return true, c.PropagateBound(n, variable, &candidate, lower, open)
}
