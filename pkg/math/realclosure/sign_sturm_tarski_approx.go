// Source: src/math/realclosure/realclosure.cpp:3743-3770, 3881-3920.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// signApproxAtBinaryBound follows the source Horner interval sequence for
// retained algebraic-root and rational coefficient intervals. Rational
// coefficient refinement follows an ambiguous result; algebraic refinement
// and exact fallback remain separate work.
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
	evaluate := func() (int32, bool, error) {
		last, e := coefficientInterval(coefficients[len(coefficients)-1])
		if e != nil {
			return 0, false, e
		}
		if e = result.Mul(last, point); e != nil {
			return 0, false, e
		}
		for i := len(coefficients) - 2; i >= 0; i-- {
			if coefficients[i] != nil {
				coefficient, e := coefficientInterval(coefficients[i])
				if e != nil {
					return 0, false, e
				}
				if e = result.Add(result, coefficient); e != nil {
					return 0, false, e
				}
			}
			if i > 0 {
				if e = result.Mul(result, point); e != nil {
					return 0, false, e
				}
			}
		}
		contains, e := result.ContainsZero()
		if e != nil || contains {
			return 0, false, e
		}
		positive, e := result.IsP()
		if e != nil {
			return 0, false, e
		}
		if positive {
			return 1, true, nil
		}
		return -1, true, nil
	}
	if sign, decided, e := evaluate(); e != nil || decided {
		return sign, e
	}
	// Source find_biggest_interval_magnitude chooses the initial precision.
	// Algebraic coefficient intervals stay at their current enclosure until
	// their root-specific refinement is ported.
	biggest := int32(-1 << 31)
	for _, value := range coefficients {
		if value == nil {
			continue
		}
		coefficient, e := coefficientInterval(value)
		if e != nil {
			return 0, e
		}
		m, e := s.sturmIntervalMagnitude(coefficient)
		if e != nil {
			return 0, e
		}
		if m > biggest {
			biggest = m
		}
	}
	precision := uint32(1)
	if biggest < 0 {
		precision = uint32(-int64(biggest))
	}
	for ; precision <= 128; precision++ {
		for _, value := range coefficients {
			if value == nil || !value.isRational {
				continue
			}
			if e := s.refineSturmRationalInterval(value, owned[value], precision); e != nil {
				return 0, e
			}
		}
		if sign, decided, e := evaluate(); e != nil || decided {
			return sign, e
		}
	}
	return 0, arena.ErrInvalidSlot
}
