// Source: src/util/mpq.cpp set(mpq &, char const *), all source branches.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

var ErrMixedNotation = errors.New("mpnum: mixing rational/scientific notation")
var ErrExponentTooLarge = errors.New("mpnum: exponent is too big")

type ParseErrorReason uint8

const (
	ParseMixedNotation ParseErrorReason = iota + 1
	ParseExponentTooLarge
	ParseZeroDenominator
)

// ParseError classifies the three actual source parser exceptions. Allocation,
// owner and nonterminating-Power failures retain their distinct error types.
type ParseError struct{ Reason ParseErrorReason }

func (e *ParseError) Unwrap() error {
	switch e.Reason {
	case ParseMixedNotation:
		return ErrMixedNotation
	case ParseExponentTooLarge:
		return ErrExponentTooLarge
	default:
		return ErrDivideByZero
	}
}
func (e *ParseError) Error() string { return e.Unwrap().Error() }

// SetString explicitly initializes/reinitializes owned integer fields, including
// a zero denominator. Ordinary arithmetic still rejects that raw state. The
// source parser publishes numerator reset and each digit/component action into
// the caller; errors retain those completed actions. Scientific input without
// a dot keeps the actual old denominator until its later source arithmetic.
func (m *RationalManager) SetString(out *Rational, text string) error {
	if m == nil {
		return umemory.ErrRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if out == nil {
		return arena.ErrInvalidSlot
	}
	// Ordinary checks reject stale/failed references before Reset. Cleanup-only
	// acceptance must not authorize arithmetic on a failed owning descriptor.
	if e := m.Z.check(&out.Num); e != nil {
		return e
	}
	if e := m.Z.check(&out.Den); e != nil {
		return e
	}
	if e := m.Z.Reset(&out.Num); e != nil {
		return e
	}
	ten := NewInt(10)
	var tmp Int
	defer m.Z.Del(&tmp)
	i := 0
	for numeralByte(text, i) == ' ' {
		i++
	}
	negative := numeralByte(text, i) == '-'
	for {
		b := numeralByte(text, i)
		if b == 0 || b == '/' || b == '.' || b == 'e' || b == 'E' {
			break
		}
		if b >= '0' && b <= '9' {
			digit := NewInt(int32(b - '0'))
			if e := m.Z.Mul(&out.Num, &ten, &tmp); e != nil {
				return e
			}
			if e := m.Z.Add(&tmp, &digit, &out.Num); e != nil {
				return e
			}
		}
		i++
	}
	marker := numeralByte(text, i)
	if marker == '/' || marker == '.' || marker == 'e' || marker == 'E' {
		if e := m.parseRationalSuffix(out, text, i, &tmp, &ten); e != nil {
			return e
		}
	} else {
		if e := m.resetDenominator(out); e != nil {
			return e
		}
	}
	if negative {
		if e := m.Z.Neg(&out.Num); e != nil {
			return e
		}
	}
	if e := m.normalize(out); e != nil {
		return e
	}
	return nil
}

// A separate lexical scope keeps source suffix temporaries alive exactly until
// the suffix branch exits, before the outer sign adjustment/normalization.
func (m *RationalManager) parseRationalSuffix(work *Rational, text string, i int, tmp, ten *Int) error {
	marker := numeralByte(text, i)

	isRat := marker == '/'
	tmp2 := NewInt(1)
	defer m.Z.Del(&tmp2)
	hasDen := false
	if marker == '/' || marker == '.' {
		hasDen = true
		i++
		if e := m.Z.Reset(&work.Den); e != nil {
			return e
		}
		for {
			b := numeralByte(text, i)
			if b == 0 || b == 'e' || b == 'E' {
				break
			}
			if b >= '0' && b <= '9' {
				digit := NewInt(int32(b - '0'))
				if e := m.Z.Mul(&work.Den, ten, tmp); e != nil {
					return e
				}
				if e := m.Z.Add(tmp, &digit, &work.Den); e != nil {
					return e
				}
				if !isRat {
					if e := m.Z.Mul(&tmp2, ten, &tmp2); e != nil {
						return e
					}
				}
			}
			i++
		}
	}
	var exponent uint64
	expNegative := false
	if b := numeralByte(text, i); b == 'e' || b == 'E' {
		if isRat {
			return &ParseError{Reason: ParseMixedNotation}
		}
		i++
		if numeralByte(text, i) == '-' {
			expNegative = true
			i++
		}
		for ; numeralByte(text, i) != 0; i++ {
			b := numeralByte(text, i)
			if b >= '0' && b <= '9' {
				exponent = 10*exponent + uint64(b-'0')
			} else if b == '/' {
				return &ParseError{Reason: ParseMixedNotation}
			}
		}
	}
	if !isRat {
		if exponent > uint64(^uint32(0)) {
			return &ParseError{Reason: ParseExponentTooLarge}
		}
		b := NewRational()
		defer m.drop(&b)
		if hasDen {
			if e := m.SetParts(&b, &work.Den, &tmp2); e != nil {
				return e
			}
			if e := m.Z.SetInt64(&work.Den, 1); e != nil {
				return e
			}
			if e := m.Add(work, &b, work); e != nil {
				return e
			}
		}
		if exponent > 0 {
			exp, tenRat := NewRational(), NewRational()
			defer m.drop(&exp)
			defer m.drop(&tenRat)
			if e := m.SetInt64(&tenRat, 10); e != nil {
				return e
			}
			if e := m.Power(&tenRat, uint32(exponent), &exp); e != nil {
				return e
			}
			if expNegative {
				if e := m.Div(work, &exp, work); e != nil {
					return e
				}
			} else {
				if e := m.Mul(work, &exp, work); e != nil {
					return e
				}
			}
		}
	} else if work.Den.Val == 0 {
		return &ParseError{Reason: ParseZeroDenominator}
	}
	return nil
}
