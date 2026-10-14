// Source: src/math/realclosure/realclosure.cpp:900-907,2515-2545.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// rationalApproxInterval initializes one nonzero rational coefficient using
// the source default initial precision. The returned interval is owned by the
// caller; the C++ value's persistent lazy cache is represented per invocation.
func (s *realValueStore) rationalApproxInterval(value *realValue, bounds *numeral.BinaryRationalManager) (interval *BinaryInterval, err error) {
	if s == nil || s.numbers == nil || bounds == nil || !s.rationalOperand(value) || value == nil {
		return nil, arena.ErrInvalidSlot
	}
	interval, err = NewBinaryInterval(bounds)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, s.releaseSturmInterval(interval))
			interval = nil
		}
	}()
	lower, upper, width := numeral.NewBinaryRational(), numeral.NewBinaryRational(), numeral.NewBinaryRational()
	defer func() {
		err = errors.Join(err,
			s.releaseSturmBinary(bounds, &width),
			s.releaseSturmBinary(bounds, &upper),
			s.releaseSturmBinary(bounds, &lower))
	}()
	exact, err := bounds.FromRational(s.numbers, &value.rational, &lower)
	if err != nil {
		return nil, err
	}
	if exact {
		err = interval.SetFinite(&lower, &lower, false, false)
		return interval, err
	}
	if err = bounds.Copy(&upper, &lower); err != nil {
		return nil, err
	}
	if err = bounds.Mul2(&upper); err != nil {
		return nil, err
	}
	negative, err := s.numbers.IsNeg(&value.rational)
	if err != nil {
		return nil, err
	}
	if negative {
		if err = bounds.Swap(&lower, &upper); err != nil {
			return nil, err
		}
	}
	for {
		if err = interval.SetFinite(&lower, &upper, true, true); err != nil {
			return nil, err
		}
		contains, e := interval.ContainsZero()
		if e != nil {
			return nil, e
		}
		refine := contains
		if !refine {
			if err = bounds.Sub(&upper, &lower, &width); err != nil {
				return nil, err
			}
			precise, e := bounds.LessOneDiv2k(&width, 24)
			if e != nil {
				return nil, e
			}
			refine = !precise
		}
		if !refine {
			refine, err = bounds.IsZero(&lower)
			if err != nil {
				return nil, err
			}
		}
		if !refine {
			refine, err = bounds.IsZero(&upper)
			if err != nil {
				return nil, err
			}
		}
		if !refine {
			return interval, nil
		}
		// Source checkpoint before each paired refinement still needs its
		// enclosing real-closure resource/cooperation owner.
		if err = bounds.RefineLower(s.numbers, &value.rational, &lower, &upper); err != nil {
			return nil, err
		}
		if err = bounds.RefineUpper(s.numbers, &value.rational, &lower, &upper); err != nil {
			return nil, err
		}
	}
}
