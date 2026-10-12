// Source: src/math/realclosure/realclosure.cpp:54-135 and selected
// src/math/interval/interval.h/interval_def.h mpbq_config instantiation.
package realclosure

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// BinaryInterval owns its real-closure binary-rational endpoints and borrows
// their manager. The source default constructor starts at (-oo,+oo).
type BinaryInterval struct {
	self                 *BinaryInterval
	numbers              *numeral.BinaryRationalManager
	lower, upper         numeral.BinaryRational
	lowerInf, upperInf   bool
	lowerOpen, upperOpen bool
	closed               bool
}

func NewBinaryInterval(m *numeral.BinaryRationalManager) (*BinaryInterval, error) {
	if m == nil {
		return nil, arena.ErrInvalidSlot
	}
	v := &BinaryInterval{numbers: m, lowerInf: true, upperInf: true, lowerOpen: true, upperOpen: true}
	v.self = v
	return v, nil
}

func (v *BinaryInterval) open() error {
	if v == nil || v.self != v || v.numbers == nil {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}

func (v *BinaryInterval) lowerKind() interval.ExtendedKind {
	if v.lowerInf {
		return interval.MinusInfinity
	}
	return interval.Finite
}

func (v *BinaryInterval) upperKind() interval.ExtendedKind {
	if v.upperInf {
		return interval.PlusInfinity
	}
	return interval.Finite
}

func (v *BinaryInterval) SetFinite(lower, upper *numeral.BinaryRational, lowerOpen, upperOpen bool) error {
	if err := v.open(); err != nil {
		return err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	cmp, err := e.Compare(lower, interval.Finite, upper, interval.Finite)
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (lowerOpen || upperOpen) {
		return interval.ErrExtendedDomain
	}
	if err = v.numbers.Copy(&v.lower, lower); err != nil {
		return err
	}
	if err = v.numbers.Copy(&v.upper, upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = false, false
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return nil
}

func (v *BinaryInterval) ResetLower() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(&v.lower); err != nil {
		return err
	}
	v.lowerInf, v.lowerOpen = true, true
	return nil
}

func (v *BinaryInterval) ResetUpper() error {
	if err := v.open(); err != nil {
		return err
	}
	if err := v.numbers.Reset(&v.upper); err != nil {
		return err
	}
	v.upperInf, v.upperOpen = true, true
	return nil
}

func (v *BinaryInterval) Reset() error {
	if err := v.ResetLower(); err != nil {
		return err
	}
	return v.ResetUpper()
}

func (v *BinaryInterval) SetFrom(src *BinaryInterval) error {
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

func (v *BinaryInterval) pair(other *BinaryInterval) error {
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

func (v *BinaryInterval) CheckInvariant() error {
	if err := v.open(); err != nil {
		return err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	cmp, err := e.Compare(&v.lower, v.lowerKind(), &v.upper, v.upperKind())
	if err != nil {
		return err
	}
	if cmp > 0 || cmp == 0 && (v.lowerOpen || v.upperOpen) {
		return interval.ErrExtendedDomain
	}
	return nil
}

func (v *BinaryInterval) IsP() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	pos, err := e.IsPos(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	z, err := e.IsZero(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	return pos || z, nil
}

func (v *BinaryInterval) IsN() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	neg, err := e.IsNeg(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	z, err := e.IsZero(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	return neg || z, nil
}

func (v *BinaryInterval) ContainsZero() (bool, error) {
	if err := v.open(); err != nil {
		return false, err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	neg, err := e.IsNeg(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	z, err := e.IsZero(&v.lower, v.lowerKind())
	if err != nil {
		return false, err
	}
	if !neg && (!z || v.lowerOpen) {
		return false, nil
	}
	pos, err := e.IsPos(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	z, err = e.IsZero(&v.upper, v.upperKind())
	if err != nil {
		return false, err
	}
	return pos || z && !v.upperOpen, nil
}

func (v *BinaryInterval) Before(other *BinaryInterval) (bool, error) {
	if err := v.pair(other); err != nil {
		return false, err
	}
	if v.upperInf || other.lowerInf {
		return false, nil
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	cmp, err := e.Compare(&v.upper, interval.Finite, &other.lower, interval.Finite)
	if err != nil {
		return false, err
	}
	return cmp < 0 || cmp == 0 && v.upperOpen, nil
}

func (v *BinaryInterval) Neg(src *BinaryInterval) error {
	if err := v.pair(src); err != nil {
		return err
	}
	m := v.numbers
	lo, uo := src.lowerOpen, src.upperOpen
	if src.lowerInf {
		if src.upperInf {
			return v.Reset()
		}
		if err := m.Copy(&v.lower, &src.upper); err != nil {
			return err
		}
		if err := m.Neg(&v.lower); err != nil {
			return err
		}
		if err := m.Reset(&v.upper); err != nil {
			return err
		}
		v.lowerInf, v.upperInf = false, true
		v.lowerOpen, v.upperOpen = uo, true
	} else if src.upperInf {
		if err := m.Copy(&v.upper, &src.lower); err != nil {
			return err
		}
		if err := m.Neg(&v.upper); err != nil {
			return err
		}
		if err := m.Reset(&v.lower); err != nil {
			return err
		}
		v.lowerInf, v.upperInf = true, false
		v.lowerOpen, v.upperOpen = true, lo
	} else {
		if v == src {
			if err := m.Swap(&v.lower, &v.upper); err != nil {
				return err
			}
		} else {
			if err := m.Copy(&v.lower, &src.upper); err != nil {
				return err
			}
			if err := m.Copy(&v.upper, &src.lower); err != nil {
				return err
			}
		}
		if err := m.Neg(&v.lower); err != nil {
			return err
		}
		if err := m.Neg(&v.upper); err != nil {
			return err
		}
		v.lowerInf, v.upperInf = false, false
		v.lowerOpen, v.upperOpen = uo, lo
	}
	return v.CheckInvariant()
}

func (v *BinaryInterval) Add(a, b *BinaryInterval) error {
	if err := v.pair(a); err != nil {
		return err
	}
	if err := v.pair(b); err != nil {
		return err
	}
	e := interval.BinaryExtended{Numbers: v.numbers}
	lk, err := e.Add(&a.lower, a.lowerKind(), &b.lower, b.lowerKind(), &v.lower)
	if err != nil {
		return err
	}
	uk, err := e.Add(&a.upper, a.upperKind(), &b.upper, b.upperKind(), &v.upper)
	if err != nil {
		return err
	}
	v.lowerInf, v.upperInf = lk == interval.MinusInfinity, uk == interval.PlusInfinity
	v.lowerOpen, v.upperOpen = a.lowerOpen || b.lowerOpen, a.upperOpen || b.upperOpen
	return v.CheckInvariant()
}

func (v *BinaryInterval) Display(w io.Writer) error {
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
	ls, us := "-oo", "+oo"
	var err error
	if !v.lowerInf {
		ls, err = v.numbers.String(&v.lower)
		if err != nil {
			return err
		}
	}
	if !v.upperInf {
		us, err = v.numbers.String(&v.upper)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, left+ls+", "+us+right)
	return err
}

func (v *BinaryInterval) Close() error {
	if v == nil || v.self != v {
		return arena.ErrInvalidSlot
	}
	if v.closed {
		return nil
	}
	if err := errors.Join(v.numbers.Del(&v.lower), v.numbers.Del(&v.upper)); err != nil {
		return err
	}
	v.closed = true
	return nil
}
