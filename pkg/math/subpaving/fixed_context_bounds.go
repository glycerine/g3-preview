// Source: src/math/subpaving/subpaving_t_def.h:528-589,1176-1315,
// concrete config_mpfx bound normalization and relevance.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func (c *FixedContext) checkNode(n *FixedNode, variable uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c || variable >= uint32(len(c.isInt)) {
		return arena.ErrInvalidSlot
	}
	return n.open()
}

// NormalizeBound updates a scratch numeral and openness as in the source's
// integer-bound normalization before relevance testing.
func (c *FixedContext) NormalizeBound(variable uint32, value *numeral.FixedPoint, lower bool, open *bool) error {
	if err := c.open(); err != nil {
		return err
	}
	if variable >= uint32(len(c.isInt)) || open == nil {
		return arena.ErrInvalidSlot
	}
	m := c.numbers
	if _, err := m.IsZero(value); err != nil {
		return err
	}
	if !c.isInt[variable] {
		return nil
	}
	integer, err := m.IsInt(value)
	if err != nil {
		return err
	}
	if !integer {
		*open = false
	}
	if lower {
		err = m.Ceil(value)
	} else {
		err = m.Floor(value)
	}
	if err != nil {
		return err
	}
	if *open {
		*open = false
		if lower {
			if err = m.SetRounding(false); err != nil {
				return err
			}
			return m.Inc(value)
		}
		if err = m.SetRounding(true); err != nil {
			return err
		}
		return m.Dec(value)
	}
	return nil
}

// RelevantNewBound follows the source conflict-first, zero-epsilon, threshold
// and epsilon-gap branches. Arithmetic exceptions mark the context failed and
// refuse the candidate, matching the source catch block.
func (c *FixedContext) RelevantNewBound(n *FixedNode, variable uint32, k *numeral.FixedPoint, lower, open bool) (relevant bool, err error) {
	if err = c.checkNode(n, variable); err != nil {
		return false, err
	}
	m := c.numbers
	if _, err = m.IsZero(k); err != nil {
		return false, err
	}
	defer func() {
		if errors.Is(err, numeral.ErrFixedPointOverflow) || errors.Is(err, numeral.ErrFixedPointDivZero) || errors.Is(err, numeral.ErrFixedPointDomain) {
			c.arithmeticFailed = true
			relevant, err = false, nil
		}
	}()
	currentLower, currentUpper := n.lower[variable], n.upper[variable]
	if lower {
		if currentUpper != nil {
			cmp, e := m.Compare(k, currentUpper.value)
			if e != nil {
				return false, e
			}
			if cmp > 0 || cmp == 0 && (open || currentUpper.open) {
				return true, nil
			}
		}
		if c.zeroEpsilon && currentLower != nil {
			cmp, e := m.Compare(k, currentLower.value)
			if e != nil {
				return false, e
			}
			if cmp < 0 || cmp == 0 && (currentLower.open || !open) {
				return false, nil
			}
		}
		if currentUpper == nil {
			cmp, e := m.Compare(c.maxBound, k)
			if e != nil {
				return false, e
			}
			if cmp < 0 {
				return false, nil
			}
		}
	} else {
		if currentLower != nil {
			cmp, e := m.Compare(currentLower.value, k)
			if e != nil {
				return false, e
			}
			if cmp > 0 || cmp == 0 && (open || currentLower.open) {
				return true, nil
			}
		}
		if c.zeroEpsilon && currentUpper != nil {
			cmp, e := m.Compare(currentUpper.value, k)
			if e != nil {
				return false, e
			}
			if cmp < 0 || cmp == 0 && (currentUpper.open || !open) {
				return false, nil
			}
		}
		if currentLower == nil {
			cmp, e := m.Compare(k, c.minusMaxBound)
			if e != nil {
				return false, e
			}
			if cmp < 0 {
				return false, nil
			}
		}
	}
	if c.zeroEpsilon || lower && currentLower == nil || !lower && currentUpper == nil {
		return true, nil
	}
	minimum, absolute, e := fixedScratchPair(m)
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, minimum.Close(), absolute.Close()) }()
	delta, e := m.New()
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, delta.Close()) }()
	current := currentUpper
	other := currentLower
	if lower {
		current, other = currentLower, currentUpper
	}
	if err = m.Copy(absolute, current.value); err != nil {
		return false, err
	}
	if err = m.Abs(absolute); err != nil {
		return false, err
	}
	if other != nil {
		if err = m.Sub(currentUpper.value, currentLower.value, minimum); err != nil {
			return false, err
		}
		cmp, e := m.Compare(absolute, minimum)
		if e != nil {
			return false, e
		}
		if cmp < 0 {
			if err = m.Copy(minimum, absolute); err != nil {
				return false, err
			}
		}
	} else {
		if err = m.Copy(minimum, absolute); err != nil {
			return false, err
		}
	}
	if err = m.SetInt64(delta, 1); err != nil {
		return false, err
	}
	cmp, e := m.Compare(minimum, delta)
	if e != nil {
		return false, e
	}
	if cmp < 0 {
		if err = m.Copy(delta, minimum); err != nil {
			return false, err
		}
	}
	if err = m.Mul(delta, c.epsilon, delta); err != nil {
		return false, err
	}
	if lower {
		if err = m.Add(currentLower.value, delta, delta); err != nil {
			return false, err
		}
		cmp, e = m.Compare(k, delta)
		if e != nil {
			return false, e
		}
		return cmp > 0, nil
	}
	if err = m.Sub(currentUpper.value, delta, delta); err != nil {
		return false, err
	}
	cmp, e = m.Compare(k, delta)
	if e != nil {
		return false, e
	}
	return cmp < 0, nil
}

// PropagateBound creates the normalized bound, queues it, then checks the
// node's new interval for conflict. Relevance testing is the caller's choice,
// as source unit clauses publish directly.
func (c *FixedContext) PropagateBound(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open bool) (err error) {
	return c.propagateBoundReason(n, variable, value, lower, open, fixedBoundReason{})
}

func (c *FixedContext) propagateBoundReason(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open bool, reason fixedBoundReason) error {
	return c.publishBound(n, variable, value, lower, open, true, reason)
}

// DecideBound creates the source's branch decision without putting it on the
// propagation queue. AddRecentBounds queues surviving child decisions later.
func (c *FixedContext) DecideBound(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open bool) error {
	if n == nil || n.parent == nil {
		return arena.ErrInvalidSlot
	}
	return c.publishBound(n, variable, value, lower, open, false, fixedBoundReason{kind: fixedReasonAxiom})
}

func (c *FixedContext) publishBound(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open, enqueue bool, reason fixedBoundReason) (err error) {
	if err = c.checkNode(n, variable); err != nil {
		return err
	}
	if n.inconsistent {
		return arena.ErrInvalidSlot
	}
	m := c.numbers
	copyValue, err := m.New()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, copyValue.Close()) }()
	if err = m.Copy(copyValue, value); err != nil {
		return err
	}
	if err = c.NormalizeBound(variable, copyValue, lower, &open); err != nil {
		return err
	}
	if err = n.PushBound(variable, copyValue, lower, open); err != nil {
		return err
	}
	b := n.trail
	b.timestamp = c.timestamp
	b.decision = !enqueue
	b.reason = reason
	if enqueue {
		c.queue = append(c.queue, b)
	}
	conflict, e := n.ConflictingBounds(variable)
	if e != nil {
		return e
	}
	if conflict {
		n.inconsistent, n.conflictVar = true, variable
	}
	c.timestamp++
	if c.timestamp == math.MaxUint64 {
		return ErrFixedContextTimestamp
	}
	return nil
}

// PropagateCandidate is the source definition-propagation sequence: normalize
// a scratch candidate, check its relevance, then publish it when selected.
func (c *FixedContext) PropagateCandidate(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open bool) (published bool, err error) {
	return c.propagateCandidateReason(n, variable, value, lower, open, fixedBoundReason{})
}

func (c *FixedContext) propagateCandidateReason(n *FixedNode, variable uint32, value *numeral.FixedPoint, lower, open bool, reason fixedBoundReason) (published bool, err error) {
	if err = c.checkNode(n, variable); err != nil {
		return false, err
	}
	scratch, err := c.numbers.New()
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, scratch.Close()) }()
	if err = c.numbers.Copy(scratch, value); err != nil {
		return false, err
	}
	if err = c.NormalizeBound(variable, scratch, lower, &open); err != nil {
		return false, err
	}
	published, err = c.RelevantNewBound(n, variable, scratch, lower, open)
	if err != nil || !published {
		return published, err
	}
	return true, c.propagateBoundReason(n, variable, scratch, lower, open, reason)
}
