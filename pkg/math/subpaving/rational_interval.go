// Source: src/math/subpaving/subpaving_t.h:239-316 and
// src/math/interval/interval_def.h:597-709, concrete config_mpq instance.
package subpaving

import (
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// RationalInterval owns the mutable endpoint numerals used by context_mpq.
// The constant node/variable view will be connected with the rational node
// owner; this type currently exposes the mutable source interval operations.
type RationalInterval struct {
	self                 *RationalInterval
	numbers              *numeral.RationalManager
	lower, upper         numeral.Rational
	lowerInf, upperInf   bool
	lowerOpen, upperOpen bool
	closed               bool
}

func NewRationalInterval(m *numeral.RationalManager) (*RationalInterval, error) {
	if m == nil {
		return nil, arena.ErrInvalidSlot
	}
	v := &RationalInterval{
		numbers: m, lower: numeral.NewRational(), upper: numeral.NewRational(),
		lowerInf: true, upperInf: true, lowerOpen: true, upperOpen: true,
	}
	v.self = v
	return v, nil
}

func (v *RationalInterval) open() error {
	if v == nil || v.self != v || v.numbers == nil {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}

func (v *RationalInterval) lowerKind() interval.ExtendedKind {
	if v.lowerInf {
		return interval.MinusInfinity
	}
	return interval.Finite
}

func (v *RationalInterval) upperKind() interval.ExtendedKind {
	if v.upperInf {
		return interval.PlusInfinity
	}
	return interval.Finite
}

func (v *RationalInterval) pair(other *RationalInterval) error {
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

func (v *RationalInterval) SetFinite(lower, upper *numeral.Rational, lowerOpen, upperOpen bool) error {
	if err := v.open(); err != nil {
		return err
	}
	e := interval.RationalExtended{Numbers: v.numbers}
	equal, err := e.Equal(lower, interval.Finite, upper, interval.Finite)
	if err != nil {
		return err
	}
	if equal {
		if lowerOpen || upperOpen {
			return interval.ErrExtendedDomain
		}
	} else {
		less, err := e.Less(lower, interval.Finite, upper, interval.Finite)
		if err != nil {
			return err
		}
		if !less {
			return interval.ErrExtendedDomain
		}
	}
	if err := v.numbers.Copy(&v.lower, lower); err != nil {
		return err
	}
	if err := v.numbers.Copy(&v.upper, upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = false, false
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return nil
}

func (v *RationalInterval) ResetLower() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(&v.lower); err != nil {
		return err
	}
	v.lowerInf, v.lowerOpen = true, true
	return nil
}

func (v *RationalInterval) ResetUpper() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(&v.upper); err != nil {
		return err
	}
	v.upperInf, v.upperOpen = true, true
	return nil
}

func (v *RationalInterval) Reset() error {
	if err := v.ResetLower(); err != nil {
		return err
	}
	return v.ResetUpper()
}

func (v *RationalInterval) SetFrom(src *RationalInterval) error {
	if err := v.pair(src); err != nil {
		return err
	}
	if v == src {
		return nil
	}
	if !src.lowerInf {
		if err := v.numbers.Copy(&v.lower, &src.lower); err != nil {
			return err
		}
	}
	if !src.upperInf {
		if err := v.numbers.Copy(&v.upper, &src.upper); err != nil {
			return err
		}
	}
	v.lowerInf, v.upperInf = src.lowerInf, src.upperInf
	v.lowerOpen, v.upperOpen = src.lowerOpen, src.upperOpen
	return v.CheckInvariant()
}

func (v *RationalInterval) CheckInvariant() error {
	if err := v.open(); err != nil {
		return err
	}
	e := interval.RationalExtended{Numbers: v.numbers}
	equal, err := e.Equal(&v.lower, v.lowerKind(), &v.upper, v.upperKind())
	if err != nil {
		return err
	}
	if equal {
		if v.lowerOpen || v.upperOpen {
			return interval.ErrExtendedDomain
		}
		return nil
	}
	less, err := e.Less(&v.lower, v.lowerKind(), &v.upper, v.upperKind())
	if err != nil {
		return err
	}
	if !less {
		return interval.ErrExtendedDomain
	}
	return nil
}

func (v *RationalInterval) ContainsZero() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	e := interval.RationalExtended{Numbers: v.numbers}
	neg, err := e.IsNeg(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	zero, err := e.IsZero(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	if !neg && (!zero || v.lowerOpen) {
		return false, nil
	}
	pos, err := e.IsPos(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	zero, err = e.IsZero(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	return pos || zero && !v.upperOpen, nil
}

func (v *RationalInterval) Display(w io.Writer) error {
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
	if err := writeSubpavingIntervalText(w, left); err != nil {
		return err
	}
	e := interval.RationalExtended{Numbers: v.numbers}
	if err := e.Display(w, &v.lower, v.lowerKind()); err != nil {
		return err
	}
	if err := writeSubpavingIntervalText(w, ", "); err != nil {
		return err
	}
	if err := e.Display(w, &v.upper, v.upperKind()); err != nil {
		return err
	}
	return writeSubpavingIntervalText(w, right)
}

func (v *RationalInterval) Close() error {
	if v == nil || v.self != v {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return nil
	}
	if err := v.numbers.Del(&v.lower); err != nil {
		return err
	}
	if err := v.numbers.Del(&v.upper); err != nil {
		return err
	}
	v.closed = true
	return nil
}
