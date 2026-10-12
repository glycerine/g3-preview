// Source: src/math/subpaving/subpaving_t.h:239-316 and selected
// src/math/interval/interval_def.h:597-837, concrete config_mpfx instance.
package subpaving

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// FixedInterval is the mutable, numeral-owning subpaving scratch interval.
// Constant node/variable views borrow bounds and are represented separately.
type FixedInterval struct {
	self                 *FixedInterval
	numbers              *numeral.FixedPointManager
	lower, upper         *numeral.FixedPoint
	lowerInf, upperInf   bool
	lowerOpen, upperOpen bool
	closed               bool
}

func NewFixedInterval(m *numeral.FixedPointManager) (*FixedInterval, error) {
	if m == nil {
		return nil, arena.ErrInvalidSlot
	}
	lower, err := m.New()
	if err != nil {
		return nil, err
	}
	upper, err := m.New()
	if err != nil {
		return nil, errors.Join(err, lower.Close())
	}
	v := &FixedInterval{numbers: m, lower: lower, upper: upper, lowerInf: true, upperInf: true, lowerOpen: true, upperOpen: true}
	v.self = v
	return v, nil
}

func (v *FixedInterval) open() error {
	if v == nil || v.self != v || v.numbers == nil || v.lower == nil || v.upper == nil {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}

func (v *FixedInterval) pair(other *FixedInterval) error {
	if err := v.open(); err != nil {
		return err
	}
	if err := other.open(); err != nil {
		return err
	}
	if v.numbers != other.numbers {
		return arena.ErrInvalidSlot
	}
	return nil
}

func (v *FixedInterval) lowerKind() interval.ExtendedKind {
	if v.lowerInf {
		return interval.MinusInfinity
	}
	return interval.Finite
}

func (v *FixedInterval) upperKind() interval.ExtendedKind {
	if v.upperInf {
		return interval.PlusInfinity
	}
	return interval.Finite
}

func (v *FixedInterval) SetFinite(lower, upper *numeral.FixedPoint, lowerOpen, upperOpen bool) error {
	if err := v.open(); err != nil {
		return err
	}
	cmp, err := v.numbers.Compare(lower, upper)
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (lowerOpen || upperOpen) {
		return interval.ErrExtendedDomain
	}
	if err := v.numbers.Copy(v.lower, lower); err != nil {
		return err
	}
	if err := v.numbers.Copy(v.upper, upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = false, false
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return nil
}

func (v *FixedInterval) ResetLower() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(v.lower); err != nil {
		return err
	}
	v.lowerInf, v.lowerOpen = true, true
	return nil
}

func (v *FixedInterval) ResetUpper() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(v.upper); err != nil {
		return err
	}
	v.upperInf, v.upperOpen = true, true
	return nil
}

func (v *FixedInterval) Reset() error {
	if err := v.ResetLower(); err != nil {
		return err
	}
	return v.ResetUpper()
}

func (v *FixedInterval) SetFrom(src *FixedInterval) error {
	if err := v.pair(src); err != nil {
		return err
	}
	if v == src {
		return nil
	}
	if !src.lowerInf {
		if err := v.numbers.Copy(v.lower, src.lower); err != nil {
			return err
		}
	}
	if !src.upperInf {
		if err := v.numbers.Copy(v.upper, src.upper); err != nil {
			return err
		}
	}
	v.lowerInf, v.upperInf = src.lowerInf, src.upperInf
	v.lowerOpen, v.upperOpen = src.lowerOpen, src.upperOpen
	return v.CheckInvariant()
}

func (v *FixedInterval) CheckInvariant() error {
	if err := v.open(); err != nil {
		return err
	}
	e := interval.FixedExtended{Numbers: v.numbers}
	cmp, err := e.Compare(v.lower, v.lowerKind(), v.upper, v.upperKind())
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (v.lowerOpen || v.upperOpen) {
		return interval.ErrExtendedDomain
	}
	return nil
}

func (v *FixedInterval) ContainsZero() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	e := interval.FixedExtended{Numbers: v.numbers}
	neg, err := e.IsNeg(v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	zero, err := e.IsZero(v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	if !neg && (!zero || v.lowerOpen) {
		return false, nil
	}
	pos, err := e.IsPos(v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	zero, err = e.IsZero(v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	return pos || zero && !v.upperOpen, nil
}

func (v *FixedInterval) LowerIsNeg() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	return (interval.FixedExtended{Numbers: v.numbers}).IsNeg(v.lower, v.lowerKind())
}

func (v *FixedInterval) Display(w io.Writer) error {
	if err := v.open(); err != nil {
		return err
	}
	left, right := "[", "]"
	if v.lowerOpen {
		left = "("
	}
	if v.upperOpen {
		right = ")"
	}
	lo, hi := "-oo", "+oo"
	var err error
	if !v.lowerInf {
		lo, err = v.numbers.String(v.lower)
		if err != nil {
			return err
		}
	}
	if !v.upperInf {
		hi, err = v.numbers.String(v.upper)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, left+lo+", "+hi+right)
	return err
}

func (v *FixedInterval) Close() error {
	if v == nil || v.self != v {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return nil
	}
	if err := errors.Join(v.lower.Close(), v.upper.Close()); err != nil {
		return err
	}
	v.closed = true
	return nil
}
