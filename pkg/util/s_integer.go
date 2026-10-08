// Source: src/util/s_integer.h:25-145; src/util/s_integer.cpp:22-70.
package util

import (
	"errors"
	"strconv"
)

var ErrSIntegerNonterminating = errors.New("g3: s_integer power mask wraps into a nonterminating loop")

// SInteger is the source signed 32-bit machine integer value. Go defines
// wrapping signed arithmetic outside the C++ representable domain; this does
// not claim portable C++ signed-overflow behavior. Division by zero panics.
type SInteger struct{ value int32 }
type SIntegerHashProc struct{}
type SIntegerEqProc struct{}

func (SIntegerHashProc) Hash(n SInteger) uint32 { return n.Hash() }
func (SIntegerEqProc) Equal(a, b SInteger) bool { return a.Equal(b) }
func NewSInteger(n int32) SInteger              { return SInteger{n} }
func NewSIntegerInt64(n int64) SInteger         { return SInteger{int32(n)} }
func NewSIntegerUint64(n uint64) SInteger       { return SInteger{int32(n)} }

// NewSIntegerString implements C-locale decimal strtol on the selected LP64
// profile, including prefix consumption and signed-long saturation before
// narrowing to int32. It neither allocates nor retains the input.
func NewSIntegerString(text string) SInteger {
	i := 0
	for i < len(text) {
		c := text[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\v' && c != '\f' {
			break
		}
		i++
	}
	negative := false
	if i < len(text) && (text[i] == '-' || text[i] == '+') {
		negative = text[i] == '-'
		i++
	}
	limit := uint64(1<<63 - 1)
	if negative {
		limit++
	}
	var value uint64
	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		digit := uint64(text[i] - '0')
		if value > (limit-digit)/10 {
			value = limit
		} else {
			value = value*10 + digit
		}
		i++
	}
	if negative {
		return NewSIntegerInt64(-int64(value))
	}
	return NewSIntegerInt64(int64(value))
}

func SIntegerFromRational(s *RationalService, r *OwnedRational) (SInteger, error) {
	n, e := s.Int64(r)
	return NewSIntegerInt64(n), e
}
func (n SInteger) ToRational(s *RationalService) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	// The source invokes rational(int), whose fresh signed32 NUM and DEN
	// remain inline even at INT_MIN/MAX. No numeric temporary or allocation
	// invocation is needed for this constructor.
	out := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	e := s.Manager.SetInt32(&out.Raw, n.value)
	return out, e
}
func SIntegerZero() SInteger             { return SInteger{0} }
func SIntegerOne() SInteger              { return SInteger{1} }
func SIntegerMinusOne() SInteger         { return SInteger{-1} }
func (n SInteger) Hash() uint32          { return uint32(n.value) }
func (n *SInteger) Swap(other *SInteger) { n.value, other.value = other.value, n.value }

// ToString supplies the decimal convenience whose C++ declaration has no
// definition in this source tree. Display follows the implemented stream body.
func (n SInteger) ToString() string                    { return strconv.FormatInt(int64(n.value), 10) }
func (n *SInteger) Reset()                             { n.value = 0 }
func (n SInteger) IsBig() bool                         { return false }
func (n SInteger) IsInt() bool                         { return true }
func (n SInteger) IsSInteger() bool                    { return true }
func (n SInteger) IsInt64() bool                       { return true }
func (n SInteger) IsUint64() bool                      { return true }
func (n SInteger) IsUnsigned() bool                    { return true }
func (n SInteger) IsRational() bool                    { return true }
func (n SInteger) GetInt() int32                       { return n.value }
func (n SInteger) GetInt64() int64                     { return int64(n.value) }
func (n SInteger) GetUint64() uint64                   { return uint64(int64(n.value)) }
func (n SInteger) GetUnsigned() uint32                 { return uint32(n.value) }
func (n SInteger) GetSInteger() SInteger               { return n }
func (n SInteger) GetRational() SInteger               { return n }
func (n SInteger) GetInfinitesimal() SInteger          { return SIntegerZero() }
func (n SInteger) Numerator() SInteger                 { return n }
func (n SInteger) Denominator() SInteger               { return SIntegerOne() }
func (n *SInteger) Assign(other SInteger) *SInteger    { n.value = other.value; return n }
func (n *SInteger) AddAssign(other SInteger) *SInteger { n.value += other.value; return n }
func (n *SInteger) SubAssign(other SInteger) *SInteger { n.value -= other.value; return n }
func (n *SInteger) MulAssign(other SInteger) *SInteger { n.value *= other.value; return n }
func (n *SInteger) DivAssign(other SInteger) *SInteger { n.value /= other.value; return n }
func (n *SInteger) RemAssign(other SInteger) *SInteger { n.value %= other.value; return n }
func (n SInteger) Add(other SInteger) SInteger         { n.AddAssign(other); return n }
func (n SInteger) Sub(other SInteger) SInteger         { n.SubAssign(other); return n }
func (n SInteger) Mul(other SInteger) SInteger         { n.MulAssign(other); return n }
func (n SInteger) Div(other SInteger) SInteger         { n.DivAssign(other); return n }
func (n SInteger) Rem(other SInteger) SInteger         { n.RemAssign(other); return n }
func (n SInteger) Mod(other SInteger) SInteger {
	n.RemAssign(other)
	if n.IsNeg() {
		n.AddAssign(other)
	}
	return n
}
func (n *SInteger) Inc() *SInteger                  { n.value++; return n }
func (n *SInteger) Dec() *SInteger                  { n.value--; return n }
func (n *SInteger) PostInc() SInteger               { old := *n; n.Inc(); return old }
func (n *SInteger) PostDec() SInteger               { old := *n; n.Dec(); return old }
func (n SInteger) Equal(other SInteger) bool        { return n.value == other.value }
func (n SInteger) NotEqual(other SInteger) bool     { return !n.Equal(other) }
func (n SInteger) Less(other SInteger) bool         { return n.value < other.value }
func (n SInteger) Greater(other SInteger) bool      { return other.Less(n) }
func (n SInteger) LessEqual(other SInteger) bool    { return !n.Greater(other) }
func (n SInteger) GreaterEqual(other SInteger) bool { return !n.Less(other) }
func (n *SInteger) Neg()                            { n.value = -n.value }
func (n SInteger) Negated() SInteger                { n.Neg(); return n }
func (n SInteger) IsZero() bool                     { return n.value == 0 }
func (n SInteger) IsOne() bool                      { return n.value == 1 }
func (n SInteger) IsMinusOne() bool                 { return n.value == -1 }
func (n SInteger) IsNeg() bool                      { return n.value < 0 }
func (n SInteger) IsPos() bool                      { return n.value > 0 }
func (n SInteger) IsNonneg() bool                   { return n.value >= 0 }
func (n SInteger) IsNonpos() bool                   { return n.value <= 0 }
func (n SInteger) IsEven() bool                     { return n.value&1 == 0 }
func (n SInteger) Floor() SInteger                  { return n }
func (n SInteger) Ceil() SInteger                   { return n }
func (n SInteger) Expt(exponent int32) SInteger {
	result := SIntegerOne()
	for i := int32(0); i < exponent; i++ {
		result.MulAssign(n)
	}
	return result
}
func (n *SInteger) Addmul(c, k SInteger) { n.value += c.value * k.value }
func (n *SInteger) Submul(c, k SInteger) { n.value -= c.value * k.value }
func (n SInteger) Abs() SInteger {
	if n.IsNeg() {
		n.Neg()
	}
	return n
}
func (n SInteger) Display(out *NumericTextStream) error { return out.AppendInt32(n.value) }

// SIntegerPower follows the source mask loop. Once the 32-bit mask wraps to
// zero the source never terminates; return its reached value and an explicit
// nontermination error instead of silently changing the exponent algorithm.
func SIntegerPower(n SInteger, exponent uint32) (SInteger, error) {
	mask := uint32(1)
	result, power := SIntegerOne(), n
	for mask <= exponent {
		if mask == 0 {
			return result, ErrSIntegerNonterminating
		}
		if mask&exponent != 0 {
			result.MulAssign(power)
		}
		power.MulAssign(power)
		mask <<= 1
	}
	return result, nil
}

// SIntegerGCD preserves the original Euclidean loop. Both inputs must be
// nonzero and absolute values must be representable in signed 32 bits.
func SIntegerGCD(a, b SInteger) SInteger {
	a, b = a.Abs(), b.Abs()
	if a.Less(b) {
		a.Swap(&b)
	}
	for {
		aux := a.Rem(b)
		if aux.IsZero() {
			return b
		}
		a, b = b, aux
	}
}
func SIntegerLCM(a, b SInteger) SInteger { g := SIntegerGCD(a, b); return a.Div(g).Mul(b) }
