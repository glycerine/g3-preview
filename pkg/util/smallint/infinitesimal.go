package smallint

// Source: src/util/inf_s_integer.h:25-347; src/util/inf_s_integer.cpp:22-24.

import (
	"errors"
	"strconv"
)

var ErrAssertion = errors.New("g3: inf_s_integer source assertion failed")

// Infinitesimal represents whole + epsilon*e and orders values
// lexicographically by those two int32 coefficients.
type Infinitesimal struct {
	whole   Integer
	epsilon Integer
}

func NewInfinitesimal(n int32) Infinitesimal { return Infinitesimal{whole: New(n)} }
func NewPair(whole, epsilon Integer) Infinitesimal {
	return Infinitesimal{whole: whole, epsilon: epsilon}
}
func NewOffset(whole Integer, positive bool) Infinitesimal {
	epsilon := Integer(-1)
	if positive {
		epsilon = 1
	}
	return NewPair(whole, epsilon)
}
func NewFraction(n, d int32) (Infinitesimal, error) {
	if sourceDebug && d != 1 {
		return Infinitesimal{}, ErrAssertion
	}
	return NewInfinitesimal(n), nil
}

func (n Infinitesimal) Whole() Integer      { return n.whole }
func (n Infinitesimal) Epsilon() Integer    { return n.epsilon }
func (n Infinitesimal) Hash() uint32        { return uint32(n.whole ^ (n.epsilon + 1)) }
func (n Infinitesimal) IsInteger() bool     { return n.epsilon == 0 }
func (n Infinitesimal) IsZero() bool        { return n.whole == 0 && n.epsilon == 0 }
func (n Infinitesimal) IsOne() bool         { return n.whole == 1 && n.epsilon == 0 }
func (n Infinitesimal) IsMinusOne() bool    { return n.whole == -1 && n.epsilon == 0 }
func (n Infinitesimal) IsNegative() bool    { return n.whole < 0 || n.whole == 0 && n.epsilon < 0 }
func (n Infinitesimal) IsPositive() bool    { return n.whole > 0 || n.whole == 0 && n.epsilon > 0 }
func (n Infinitesimal) IsNonnegative() bool { return n.whole > 0 || n.whole == 0 && n.epsilon >= 0 }
func (n Infinitesimal) IsNonpositive() bool { return n.whole < 0 || n.whole == 0 && n.epsilon <= 0 }
func (n Infinitesimal) Add(other Infinitesimal) Infinitesimal {
	return NewPair(n.whole+other.whole, n.epsilon+other.epsilon)
}
func (n Infinitesimal) Sub(other Infinitesimal) Infinitesimal {
	return NewPair(n.whole-other.whole, n.epsilon-other.epsilon)
}
func (n Infinitesimal) AddInteger(other Integer) Infinitesimal {
	return NewPair(n.whole+other, n.epsilon)
}
func (n Infinitesimal) SubInteger(other Integer) Infinitesimal {
	return NewPair(n.whole-other, n.epsilon)
}
func (n Infinitesimal) Scale(factor Integer) Infinitesimal {
	return NewPair(n.whole*factor, n.epsilon*factor)
}
func (n Infinitesimal) Quo(divisor Integer) Infinitesimal {
	return NewPair(n.whole/divisor, n.epsilon/divisor)
}
func (n Infinitesimal) AddMul(c Integer, k Infinitesimal) Infinitesimal {
	return NewPair(n.whole+c*k.whole, n.epsilon+c*k.epsilon)
}
func (n Infinitesimal) SubMul(c Integer, k Infinitesimal) Infinitesimal {
	return NewPair(n.whole-c*k.whole, n.epsilon-c*k.epsilon)
}
func (n Infinitesimal) Neg() Infinitesimal { return NewPair(-n.whole, -n.epsilon) }
func (n Infinitesimal) Abs() Infinitesimal {
	if n.IsNegative() {
		return n.Neg()
	}
	return n
}
func (n Infinitesimal) Compare(other Infinitesimal) int {
	if c := n.whole.Compare(other.whole); c != 0 {
		return c
	}
	return n.epsilon.Compare(other.epsilon)
}
func (n Infinitesimal) CompareInteger(other Integer) int {
	return n.Compare(NewPair(other, 0))
}
func (n Infinitesimal) Floor() Integer {
	if n.epsilon < 0 {
		return n.whole - 1
	}
	return n.whole
}
func (n Infinitesimal) Ceil() Integer {
	if n.epsilon > 0 {
		return n.whole + 1
	}
	return n.whole
}
func (n Infinitesimal) Numerator() (Infinitesimal, error) {
	if sourceDebug && n.epsilon != 0 {
		return Infinitesimal{}, ErrAssertion
	}
	return NewPair(n.whole, 0), nil
}
func (n Infinitesimal) Denominator() (Infinitesimal, error) {
	if sourceDebug && n.epsilon != 0 {
		return Infinitesimal{}, ErrAssertion
	}
	return NewInfinitesimal(1), nil
}
func (n Infinitesimal) String() string {
	whole := strconv.FormatInt(int64(n.whole), 10)
	if n.epsilon == 0 {
		return whole
	}
	separator := " +e*"
	if n.epsilon < 0 {
		separator = " -e*"
	}
	return "(" + whole + separator + strconv.FormatInt(int64(n.epsilon), 10) + ")"
}
