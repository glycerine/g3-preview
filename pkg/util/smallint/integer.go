// Source: src/util/s_integer.h:25-145; src/util/s_integer.cpp:22-70.
package smallint

import (
	"errors"
	"strconv"
)

// Integer is Z3's signed machine-integer numeral. The selected source uses a
// 32-bit C++ int, so arithmetic intentionally has int32 wraparound in Go.
type Integer int32

var ErrPowerNonterminating = errors.New("g3: s_integer power mask wraps into a nonterminating loop")

func New(n int32) Integer             { return Integer(n) }
func FromInt64(n int64) Integer       { return Integer(int32(n)) }
func FromUint64(n uint64) Integer     { return Integer(int32(n)) }
func (n Integer) Int32() int32        { return int32(n) }
func (n Integer) Int64() int64        { return int64(n) }
func (n Integer) Uint32() uint32      { return uint32(n) }
func (n Integer) Uint64() uint64      { return uint64(int64(n)) }
func (n Integer) Hash() uint32        { return uint32(n) }
func (n Integer) String() string      { return strconv.FormatInt(int64(n), 10) }
func (n Integer) IsZero() bool        { return n == 0 }
func (n Integer) IsOne() bool         { return n == 1 }
func (n Integer) IsMinusOne() bool    { return n == -1 }
func (n Integer) IsNegative() bool    { return n < 0 }
func (n Integer) IsPositive() bool    { return n > 0 }
func (n Integer) IsNonnegative() bool { return n >= 0 }
func (n Integer) IsNonpositive() bool { return n <= 0 }
func (n Integer) IsEven() bool        { return n&1 == 0 }

func (n Integer) Add(other Integer) Integer { return n + other }
func (n Integer) Sub(other Integer) Integer { return n - other }
func (n Integer) Mul(other Integer) Integer { return n * other }
func (n Integer) Quo(other Integer) Integer { return n / other }
func (n Integer) Rem(other Integer) Integer { return n % other }

// Mod preserves the source operation: a negative remainder is adjusted by the
// divisor. Source callers use a positive divisor.
func (n Integer) Mod(other Integer) Integer {
	r := n % other
	if r < 0 {
		r += other
	}
	return r
}

func (n Integer) Neg() Integer { return -n }
func (n Integer) Abs() Integer {
	if n < 0 {
		return -n
	}
	return n
}
func (n Integer) AddMul(c, k Integer) Integer { return n + c*k }
func (n Integer) SubMul(c, k Integer) Integer { return n - c*k }

func (n Integer) Compare(other Integer) int {
	if n < other {
		return -1
	}
	if n > other {
		return 1
	}
	return 0
}

// Expt preserves the source signed-exponent loop; negative exponents return 1.
func (n Integer) Expt(exponent int32) Integer {
	result := Integer(1)
	for i := int32(0); i < exponent; i++ {
		result *= n
	}
	return result
}

// Power follows the source unsigned mask loop. Exponents with bit 31 set make
// the source mask wrap to zero and loop forever; report that reached condition.
func Power(n Integer, exponent uint32) (Integer, error) {
	mask := uint32(1)
	result, power := Integer(1), n
	for mask <= exponent {
		if mask == 0 {
			return result, ErrPowerNonterminating
		}
		if mask&exponent != 0 {
			result *= power
		}
		power *= power
		mask <<= 1
	}
	return result, nil
}

// GCD preserves the source domain: both inputs must be nonzero and their
// absolute values must be representable in int32.
func GCD(a, b Integer) Integer {
	a, b = a.Abs(), b.Abs()
	if a < b {
		a, b = b, a
	}
	for {
		r := a % b
		if r == 0 {
			return b
		}
		a, b = b, r
	}
}

func LCM(a, b Integer) Integer { return (a / GCD(a, b)) * b }
