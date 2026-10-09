// Source: src/util/checked_int64.h:30-219.
package checkedint

import (
	"errors"
	"math"
	"strconv"
)

var ErrOverflow = errors.New("checked_int64 overflow/underflow")

// Int64 is the checked specialization of checked_int64. C++ unchecked
// instantiations map directly to Go int64 and do not need a wrapper.
type Int64 struct{ value int64 }

func New(value int64) Int64 { return Int64{value: value} }

func (n Int64) Value() int64        { return n.value }
func (n Int64) IsZero() bool        { return n.value == 0 }
func (n Int64) IsPositive() bool    { return n.value > 0 }
func (n Int64) IsNegative() bool    { return n.value < 0 }
func (n Int64) IsOne() bool         { return n.value == 1 }
func (n Int64) IsMinusOne() bool    { return n.value == -1 }
func (n Int64) IsNonNegative() bool { return n.value >= 0 }
func (n Int64) IsNonPositive() bool { return n.value <= 0 }

// SourceIsEven preserves checked_int64.h's observable XOR expression, which
// returns true only for one rather than testing parity.
func (n Int64) SourceIsEven() bool { return n.value^1 == 0 }

func (n Int64) Hash() uint32           { return uint32(n.value) }
func (n Int64) Equal(other Int64) bool { return n.value == other.value }
func (n Int64) Less(other Int64) bool  { return n.value < other.value }
func (n Int64) String() string         { return strconv.FormatInt(n.value, 10) }

func (n Int64) Abs() (Int64, error) {
	if n.value == math.MinInt64 {
		return n, ErrOverflow
	}
	if n.value < 0 {
		n.value = -n.value
	}
	return n, nil
}

func (n *Int64) Negate() error {
	if n.value == math.MinInt64 {
		return ErrOverflow
	}
	n.value = -n.value
	return nil
}

func (n *Int64) Increment() error {
	if n.value == math.MaxInt64 {
		return ErrOverflow
	}
	n.value++
	return nil
}

func (n *Int64) Decrement() error {
	if n.value == math.MinInt64 {
		return ErrOverflow
	}
	n.value--
	return nil
}

func (n *Int64) Add(other Int64) error {
	result := int64(uint64(n.value) + uint64(other.value))
	if n.value > 0 && other.value > 0 && result <= 0 ||
		n.value < 0 && other.value < 0 && result >= 0 {
		return ErrOverflow
	}
	n.value = result
	return nil
}

func (n *Int64) Subtract(other Int64) error {
	result := int64(uint64(n.value) - uint64(other.value))
	if n.value > 0 && other.value < 0 && result <= 0 ||
		n.value < 0 && other.value > 0 && result >= 0 {
		return ErrOverflow
	}
	n.value = result
	return nil
}

func (n *Int64) Multiply(other Int64) error {
	a, b := n.value, other.value
	if a == 0 || b == 0 {
		n.value = 0
		return nil
	}
	if a == math.MinInt64 && b == -1 || b == math.MinInt64 && a == -1 {
		return ErrOverflow
	}
	result := a * b
	if result/b != a {
		return ErrOverflow
	}
	n.value = result
	return nil
}
