// Source: src/math/realclosure/realclosure.cpp:3743-3770, 3881-3920.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// signApproxAtBinaryBound follows the source Horner interval sequence for
// retained algebraic-root coefficient intervals. Rational coefficient
// conversion and the subsequent refinement/exact fallback are separate work.
func (s *realValueStore) signApproxAtBinaryBound(coefficients []*realValue, bounds *numeral.BinaryRationalManager, bound *numeral.BinaryRational) (sign int32, err error) {
	for _, value := range coefficients {
		if value == nil {
			continue
		}
		if value.store != s || value.deleted || value.refs == 0 || value.isRational || value.functionInterval == nil || value.functionInterval.numbers != bounds {
			return 0, arena.ErrInvalidSlot
		}
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
	if err = result.Mul(coefficients[len(coefficients)-1].functionInterval, point); err != nil {
		return 0, err
	}
	for i := len(coefficients) - 2; i >= 0; i-- {
		if coefficients[i] != nil {
			if err = result.Add(result, coefficients[i].functionInterval); err != nil {
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
