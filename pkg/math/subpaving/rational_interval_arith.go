// Source: src/math/interval/interval_def.h:704-837, concrete config_mpq.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// rationalIntervalScratch keeps both results separate from the inputs so the
// source interval operations also work when a destination aliases an input.
func rationalIntervalScratch() (numeral.Rational, numeral.Rational) {
	return numeral.NewRational(), numeral.NewRational()
}

func (v *RationalInterval) Neg(src *RationalInterval) (err error) {
	if err = v.pair(src); err != nil {
		return err
	}
	lower, upper := rationalIntervalScratch()
	defer func() { err = errors.Join(err, v.numbers.Del(&lower), v.numbers.Del(&upper)) }()
	if !src.upperInf {
		if err = v.numbers.Copy(&lower, &src.upper); err != nil {
			return err
		}
		if err = v.numbers.Neg(&lower); err != nil {
			return err
		}
	}
	if !src.lowerInf {
		if err = v.numbers.Copy(&upper, &src.lower); err != nil {
			return err
		}
		if err = v.numbers.Neg(&upper); err != nil {
			return err
		}
	}
	lowerInf, upperInf := src.upperInf, src.lowerInf
	lowerOpen, upperOpen := src.upperOpen, src.lowerOpen
	if err = v.numbers.Swap(&v.lower, &lower); err != nil {
		return err
	}
	if err = v.numbers.Swap(&v.upper, &upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = lowerInf, upperInf
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return v.CheckInvariant()
}

func (v *RationalInterval) Add(a, b *RationalInterval) (err error) {
	if err = v.pair(a); err != nil {
		return err
	}
	if err = v.pair(b); err != nil {
		return err
	}
	lower, upper := rationalIntervalScratch()
	defer func() { err = errors.Join(err, v.numbers.Del(&lower), v.numbers.Del(&upper)) }()
	e := interval.RationalExtended{Numbers: v.numbers}
	lk, err := e.Add(&a.lower, a.lowerKind(), &b.lower, b.lowerKind(), &lower)
	if err != nil {
		return err
	}
	uk, err := e.Add(&a.upper, a.upperKind(), &b.upper, b.upperKind(), &upper)
	if err != nil {
		return err
	}
	lowerOpen, upperOpen := a.lowerOpen || b.lowerOpen, a.upperOpen || b.upperOpen
	if err = v.numbers.Swap(&v.lower, &lower); err != nil {
		return err
	}
	if err = v.numbers.Swap(&v.upper, &upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = lk == interval.MinusInfinity, uk == interval.PlusInfinity
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return v.CheckInvariant()
}

func (v *RationalInterval) Sub(a, b *RationalInterval) (err error) {
	if err = v.pair(a); err != nil {
		return err
	}
	if err = v.pair(b); err != nil {
		return err
	}
	lower, upper := rationalIntervalScratch()
	defer func() { err = errors.Join(err, v.numbers.Del(&lower), v.numbers.Del(&upper)) }()
	e := interval.RationalExtended{Numbers: v.numbers}
	lk, err := e.Sub(&a.lower, a.lowerKind(), &b.upper, b.upperKind(), &lower)
	if err != nil {
		return err
	}
	uk, err := e.Sub(&a.upper, a.upperKind(), &b.lower, b.lowerKind(), &upper)
	if err != nil {
		return err
	}
	lowerOpen, upperOpen := a.lowerOpen || b.upperOpen, a.upperOpen || b.lowerOpen
	if err = v.numbers.Swap(&v.lower, &lower); err != nil {
		return err
	}
	if err = v.numbers.Swap(&v.upper, &upper); err != nil {
		return err
	}
	v.lowerInf, v.upperInf = lk == interval.MinusInfinity, uk == interval.PlusInfinity
	v.lowerOpen, v.upperOpen = lowerOpen, upperOpen
	return v.CheckInvariant()
}
