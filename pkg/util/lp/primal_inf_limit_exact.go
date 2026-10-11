// Source: src/util/lp/lp_primal_core_solver.h:546-607; precise mpq/impq branches.
package lp

import "github.com/glycerine/g3/pkg/util/numeral"

// limitExactBoundRatio owns the full (bound-x)/multiplier expression and theta
// assignment. Its caller has finished delegated base comparisons. The scalar
// copy used by numeric_pair division is distinct from the borrowed multiplier.
func (m *PairPrimalCoreSolver) limitExactBoundRatio(multiplier *numeral.Rational, x, bound, theta *RationalPair, unlimited *bool) (err error) {
	before, e := m.calls.begin(m.manager)
	if e != nil {
		return e
	}
	defer func() { err = m.calls.after(m.manager, before, err) }()
	m.work = NewIndexedRationalVector(m.manager, 6)
	m.pairWork = NewIndexedPairVector(m.manager, 2)
	tmp, dx, dy, scalar, qx, qy := &m.work.Data[0], &m.work.Data[1], &m.work.Data[2], &m.work.Data[3], &m.work.Data[4], &m.work.Data[5]
	if err = etaRationalDifference(m.manager, tmp, dx, &bound.X, &x.X); err != nil {
		return err
	}
	if err = etaRationalDifference(m.manager, tmp, dy, &bound.Y, &x.Y); err != nil {
		return err
	}
	difference, ratio := &m.pairWork.Data[0], &m.pairWork.Data[1]
	if err = m.manager.Copy(&difference.X, dx); err != nil {
		return err
	}
	if err = m.manager.Copy(&difference.Y, dy); err != nil {
		return err
	}
	if err = m.manager.Copy(scalar, multiplier); err != nil {
		return err
	}
	if err = etaRationalQuotient(m.manager, tmp, qx, &difference.X, scalar); err != nil {
		return err
	}
	if err = etaRationalQuotient(m.manager, tmp, qy, &difference.Y, scalar); err != nil {
		return err
	}
	if err = m.manager.Copy(&ratio.X, qx); err != nil {
		return err
	}
	if err = m.manager.Copy(&ratio.Y, qy); err != nil {
		return err
	}
	if err = m.limitThetaValue(ratio, theta, unlimited); err != nil {
		return err
	}
	if err = m.pairWork.Close(); err != nil {
		return err
	}
	m.pairWork = nil
	if err = m.work.Close(); err != nil {
		return err
	}
	m.work = nil
	return nil
}

func (m *PairPrimalCoreSolver) zeroInfTheta(theta *RationalPair, unlimited *bool) (err error) {
	before, e := m.calls.begin(m.manager)
	if e != nil {
		return e
	}
	defer func() { err = m.calls.after(m.manager, before, err) }()
	zero := NewRationalPair()
	if err = theta.Copy(m.manager, &zero); err != nil {
		return err
	}
	*unlimited = false
	return nil
}

// LimitInfOnBoundMNeg leaves theta untouched when x is already below the
// bound. Equality publishes zero; a value above the bound limits the step.
func (m *PairPrimalCoreSolver) LimitInfOnBoundMNeg(multiplier *numeral.Rational, x, bound, theta *RationalPair, unlimited *bool) (limited bool, err error) {
	if err = m.ready(); err != nil {
		return false, err
	}
	defer func() { m.stopped = err != nil }()
	if multiplier == nil || x == nil || bound == nil || theta == nil || unlimited == nil {
		return false, ErrBounds
	}
	if sourceDebugEnabled {
		negative, e := m.manager.IsNeg(multiplier)
		if e != nil {
			return false, e
		}
		if !negative {
			return false, ErrAssertion
		}
	}
	below, e := m.BelowBound(x, bound)
	if e != nil {
		return false, e
	}
	if below {
		return false, nil
	}
	above, e := m.AboveBound(x, bound)
	if e != nil {
		return false, e
	}
	if above {
		err = m.limitExactBoundRatio(multiplier, x, bound, theta, unlimited)
	} else {
		err = m.zeroInfTheta(theta, unlimited)
	}
	return true, err
}

func (m *PairPrimalCoreSolver) LimitInfOnBoundMPos(multiplier *numeral.Rational, x, bound, theta *RationalPair, unlimited *bool) (limited bool, err error) {
	if err = m.ready(); err != nil {
		return false, err
	}
	defer func() { m.stopped = err != nil }()
	if multiplier == nil || x == nil || bound == nil || theta == nil || unlimited == nil {
		return false, ErrBounds
	}
	if sourceDebugEnabled {
		positive, e := m.manager.IsPos(multiplier)
		if e != nil {
			return false, e
		}
		if !positive {
			return false, ErrAssertion
		}
	}
	above, e := m.AboveBound(x, bound)
	if e != nil {
		return false, e
	}
	if above {
		return false, nil
	}
	below, e := m.BelowBound(x, bound)
	if e != nil {
		return false, e
	}
	if below {
		err = m.limitExactBoundRatio(multiplier, x, bound, theta, unlimited)
	} else {
		err = m.zeroInfTheta(theta, unlimited)
	}
	return true, err
}

// LimitInfOnLowBoundMPos differs at equality: moving upward from an already
// feasible low bound does not constrain theta or change unlimited.
func (m *PairPrimalCoreSolver) LimitInfOnLowBoundMPos(multiplier *numeral.Rational, x, bound, theta *RationalPair, unlimited *bool) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer func() { m.stopped = err != nil }()
	if multiplier == nil || x == nil || bound == nil || theta == nil || unlimited == nil {
		return ErrBounds
	}
	if sourceDebugEnabled {
		positive, e := m.manager.IsPos(multiplier)
		if e != nil {
			return e
		}
		if !positive {
			return ErrAssertion
		}
	}
	below, e := m.BelowBound(x, bound)
	if e != nil {
		return e
	}
	if !below {
		return nil
	}
	return m.limitExactBoundRatio(multiplier, x, bound, theta, unlimited)
}
