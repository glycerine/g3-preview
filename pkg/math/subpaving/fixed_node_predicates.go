// Source: src/math/subpaving/subpaving_t_def.h:1321-1343, concrete config_mpfx.
package subpaving

func (n *FixedNode) IsUnbounded(variable uint32) (bool, error) {
	lower, err := n.bound(variable, true)
	if err != nil {
		return false, err
	}
	upper, err := n.bound(variable, false)
	if err != nil {
		return false, err
	}
	return lower == nil && upper == nil, nil
}

func (n *FixedNode) IsZero(variable uint32) (bool, error) {
	lower, err := n.bound(variable, true)
	if err != nil {
		return false, err
	}
	upper, err := n.bound(variable, false)
	if err != nil {
		return false, err
	}
	if lower == nil || upper == nil || lower.open || upper.open {
		return false, nil
	}
	lz, err := n.numbers.IsZero(lower.value)
	if err != nil {
		return false, err
	}
	uz, err := n.numbers.IsZero(upper.value)
	if err != nil {
		return false, err
	}
	return lz && uz, nil
}

func (n *FixedNode) IsUpperZero(variable uint32) (bool, error) {
	upper, err := n.bound(variable, false)
	if err != nil {
		return false, err
	}
	if upper == nil || upper.open {
		return false, nil
	}
	return n.numbers.IsZero(upper.value)
}

func (n *FixedNode) ConflictingBounds(variable uint32) (bool, error) {
	lower, err := n.bound(variable, true)
	if err != nil {
		return false, err
	}
	upper, err := n.bound(variable, false)
	if err != nil {
		return false, err
	}
	if lower == nil || upper == nil {
		return false, nil
	}
	cmp, err := n.numbers.Compare(upper.value, lower.value)
	if err != nil {
		return false, err
	}
	return cmp < 0 || cmp == 0 && (lower.open || upper.open), nil
}
