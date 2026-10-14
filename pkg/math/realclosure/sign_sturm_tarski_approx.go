// Source: src/math/realclosure/realclosure.cpp:3743-3770, 3881-3920.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// signApproxAtBinaryBound follows the source Horner interval sequence for
// retained algebraic-root and rational coefficient intervals. Refinement of
// an ambiguous result and exact fallback are separate work.
func (s *realValueStore) signApproxAtBinaryBound(coefficients []*realValue, bounds *numeral.BinaryRationalManager, bound *numeral.BinaryRational) (sign int32, err error) {
	for _, value := range coefficients {
		if value == nil {
			continue
		}
		if value.store != s || value.deleted || value.refs == 0 || !value.isRational && (value.functionInterval == nil || value.functionInterval.numbers != bounds) {
			return 0, arena.ErrInvalidSlot
		}
	}
	// Source rational values cache intervals in the value object. Keep one
	// concrete interval per rational coefficient for this invocation until
	// the result and point interval have been destroyed.
	owned := make(map[*realValue]*BinaryInterval)
	var ownedOrder []*BinaryInterval
	defer func() {
		for i := len(ownedOrder) - 1; i >= 0; i-- {
			err = errors.Join(err, s.releaseSturmInterval(ownedOrder[i]))
		}
	}()
	coefficientInterval := func(value *realValue) (*BinaryInterval, error) {
		if !value.isRational {
			return value.functionInterval, nil
		}
		if interval := owned[value]; interval != nil {
			return interval, nil
		}
		interval, e := s.rationalApproxInterval(value, bounds)
		if e != nil {
			return nil, e
		}
		owned[value] = interval
		ownedOrder = append(ownedOrder, interval)
		return interval, nil
	}
	point, err := NewBinaryInterval(bounds)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, s.releaseSturmInterval(point)) }()
	result, err := NewBinaryInterval(bounds)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, s.releaseSturmInterval(result)) }()
	if err = point.SetFinite(bound, bound, false, false); err != nil {
		return 0, err
	}
	last, err := coefficientInterval(coefficients[len(coefficients)-1])
	if err != nil {
		return 0, err
	}
	if err = result.Mul(last, point); err != nil {
		return 0, err
	}
	for i := len(coefficients) - 2; i >= 0; i-- {
		if coefficients[i] != nil {
			coefficient, e := coefficientInterval(coefficients[i])
			if e != nil {
				return 0, e
			}
			if err = result.Add(result, coefficient); err != nil {
				return 0, err
			}
		}
		if i > 0 {
			if err = result.Mul(result, point); err != nil {
				return 0, err
			}
		}
	}
	contains, err := result.ContainsZero()
	if err != nil {
		return 0, err
	}
	if contains {
		return 0, arena.ErrInvalidSlot
	}
	positive, err := result.IsP()
	if err != nil {
		return 0, err
	}
	if positive {
		return 1, nil
	}
	return -1, nil
}
