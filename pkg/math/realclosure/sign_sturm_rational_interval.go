// Source: src/math/realclosure/realclosure.cpp:900-907,2515-2545.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
)

// bindInitialPrecision follows the generated rcf_params getter: local value,
// module value, then the literal default. The module reference is borrowed
// from the enclosing parameter owner, which owns its acquisition and release.
func (s *realValueStore) bindInitialPrecision(pc *params.ParamsContext, p, module *params.ParamsRef) error {
	if s == nil || s.numbers == nil || pc == nil || p == nil {
		return arena.ErrInvalidSlot
	}
	if err := pc.Check(p); err != nil {
		return err
	}
	name := "initial_precision"
	value, err := pc.GetUIntFallbackName(p, &name, module, 24)
	if err != nil {
		return err
	}
	s.initialPrecision = value
	s.initialPrecisionBound = true
	return nil
}

func (s *realValueStore) rationalInitialPrecision() uint32 {
	if s.initialPrecisionBound {
		return s.initialPrecision
	}
	return 24
}

// rationalApproxInterval initializes one nonzero rational coefficient using
// the source initial precision. The returned interval is owned by the
// caller; cachedRationalApproxInterval transfers that ownership to the value.
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
			precise, e := bounds.LessOneDiv2k(&width, s.rationalInitialPrecision())
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

// Source rational_value::m_interval is initialized lazily and survives
// subsequent sign checks so coefficient refinement is not discarded.
func (s *realValueStore) cachedRationalApproxInterval(value *realValue, bounds *numeral.BinaryRationalManager) (*BinaryInterval, error) {
	if s == nil || value == nil || value.store != s || value.deleted || value.refs == 0 || !value.isRational || bounds == nil {
		return nil, arena.ErrInvalidSlot
	}
	if value.rationalInterval != nil {
		if value.rationalInterval.numbers != bounds {
			return nil, arena.ErrInvalidSlot
		}
		return value.rationalInterval, nil
	}
	interval, err := s.rationalApproxInterval(value, bounds)
	if err != nil {
		return nil, err
	}
	value.rationalInterval = interval
	return interval, nil
}
