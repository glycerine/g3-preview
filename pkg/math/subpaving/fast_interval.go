// Source: src/math/subpaving/subpaving_t.h:239-316 and
// src/math/interval/interval_def.h:597-680, concrete config_mpff instance.
package subpaving

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// FastInterval owns the mutable fast-float endpoints used by the subpaving
// interval manager. Constant node/variable views belong to the context port.
type FastInterval struct {
	self                 *FastInterval
	numbers              *numeral.FastFloatManager
	lower, upper         *numeral.ScopedFastFloat
	lowerInf, upperInf   bool
	lowerOpen, upperOpen bool
	closed               bool
}

func NewFastInterval(m *numeral.FastFloatManager) (*FastInterval, error) {
	if m == nil {
		return nil, arena.ErrInvalidSlot
	}
	lower, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		return nil, err
	}
	upper, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		return nil, errors.Join(err, lower.Close())
	}
	v := &FastInterval{numbers: m, lower: lower, upper: upper, lowerInf: true, upperInf: true, lowerOpen: true, upperOpen: true}
	v.self = v
	return v, nil
}

func (v *FastInterval) endpoints() (*numeral.FastFloat, *numeral.FastFloat, error) {
	if v == nil || v.self != v || v.numbers == nil || v.lower == nil || v.upper == nil {
		return nil, nil, arena.ErrInvalidSlot
	}
	if v.closed {
		return nil, nil, arena.ErrClosed
	}
	lower, err := v.lower.Get()
	if err != nil {
		return nil, nil, err
	}
	upper, err := v.upper.Get()
	return lower, upper, err
}

func (v *FastInterval) lowerKind() interval.ExtendedKind {
	if v.lowerInf {
		return interval.MinusInfinity
	}
	return interval.Finite
}

func (v *FastInterval) upperKind() interval.ExtendedKind {
	if v.upperInf {
		return interval.PlusInfinity
	}
	return interval.Finite
}

func (v *FastInterval) SetFinite(lower, upper *numeral.FastFloat, lowerOpen, upperOpen bool) error {
	lo, hi, err := v.endpoints()
	if err != nil {
		return err
	}
	e := interval.FastExtended{Numbers: v.numbers}
	cmp, err := e.Compare(lower, interval.Finite, upper, interval.Finite)
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (lowerOpen || upperOpen) {
		return interval.ErrExtendedDomain
	}
	if err := v.numbers.Copy(lo, lower); err != nil {
		return err
	}
	if err := v.numbers.Copy(hi, upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = false, false
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return nil
}

func (v *FastInterval) ResetLower() error {
	lo, _, err := v.endpoints()
	if err != nil {
		return err
	}
	if err := v.numbers.Reset(lo); err != nil {
		return err
	}
	v.lowerInf, v.lowerOpen = true, true
	return nil
}

func (v *FastInterval) ResetUpper() error {
	_, hi, err := v.endpoints()
	if err != nil {
		return err
	}
	if err := v.numbers.Reset(hi); err != nil {
		return err
	}
	v.upperInf, v.upperOpen = true, true
	return nil
}

func (v *FastInterval) Reset() error {
	if err := v.ResetLower(); err != nil {
		return err
	}
	return v.ResetUpper()
}

func (v *FastInterval) SetFrom(src *FastInterval) error {
	lo, hi, err := v.endpoints()
	if err != nil {
		return err
	}
	slo, shi, err := src.endpoints()
	if err != nil {
		return err
	}
	if v.numbers != src.numbers {
		return arena.ErrInvalidSlot
	}
	if v == src {
		return nil
	}
	if !src.lowerInf {
		if err := v.numbers.Copy(lo, slo); err != nil {
			return err
		}
	}
	if !src.upperInf {
		if err := v.numbers.Copy(hi, shi); err != nil {
			return err
		}
	}
	v.lowerInf, v.upperInf = src.lowerInf, src.upperInf
	v.lowerOpen, v.upperOpen = src.lowerOpen, src.upperOpen
	return v.CheckInvariant()
}

func (v *FastInterval) CheckInvariant() error {
	lo, hi, err := v.endpoints()
	if err != nil {
		return err
	}
	cmp, err := (interval.FastExtended{Numbers: v.numbers}).Compare(lo, v.lowerKind(), hi, v.upperKind())
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (v.lowerOpen || v.upperOpen) {
		return interval.ErrExtendedDomain
	}
	return nil
}

func (v *FastInterval) ContainsZero() (bool, error) {
	lo, hi, err := v.endpoints()
	if err != nil {
		return false, err
	}
	e := interval.FastExtended{Numbers: v.numbers}
	neg, err := e.IsNeg(lo, v.lowerKind())
	if err != nil {
		return false, err
	}
	zero, err := e.IsZero(lo, v.lowerKind())
	if err != nil {
		return false, err
	}
	if !neg && (!zero || v.lowerOpen) {
		return false, nil
	}
	pos, err := e.IsPos(hi, v.upperKind())
	if err != nil {
		return false, err
	}
	zero, err = e.IsZero(hi, v.upperKind())
	if err != nil {
		return false, err
	}
	return pos || zero && !v.upperOpen, nil
}

func (v *FastInterval) Display(w io.Writer) error {
	lo, hi, err := v.endpoints()
	if err != nil {
		return err
	}
	left, right := "[", "]"
	if v.lowerOpen {
		left = "("
	}
	if v.upperOpen {
		right = ")"
	}
	if _, err := io.WriteString(w, left); err != nil {
		return err
	}
	e := interval.FastExtended{Numbers: v.numbers}
	if err := e.Display(w, lo, v.lowerKind()); err != nil {
		return err
	}
	if _, err := io.WriteString(w, ", "); err != nil {
		return err
	}
	if err := e.Display(w, hi, v.upperKind()); err != nil {
		return err
	}
	_, err = io.WriteString(w, right)
	return err
}

func (v *FastInterval) Close() error {
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
