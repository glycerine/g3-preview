// Source: src/util/inf_s_integer.h:25-347; src/util/inf_s_integer.cpp:22-24.
package util

import "strconv"

// InfSInteger is the source pair of signed32 integer and infinitesimal
// coefficients, ordered lexicographically. Its zero value is source zero.
// Go wrapping outside the representable C++ signed domain is not a claim
// about native signed-overflow behavior; division by zero panics.
type InfSInteger struct{ first, second int32 }
type InfSIntegerHashProc struct{}
type InfSIntegerEqProc struct{}

func (InfSIntegerHashProc) Hash(x InfSInteger) uint32 { return x.Hash() }
func (InfSIntegerEqProc) Equal(x, y InfSInteger) bool { return x.Equal(y) }
func NewInfSInteger(n int32) InfSInteger              { return InfSInteger{n, 0} }
func NewInfSIntegerFraction(n, d int32) (InfSInteger, error) {
	if sourceDebug && d != 1 {
		return InfSInteger{}, ErrIntegerAssertion
	}
	return NewInfSInteger(n), nil
}
func NewInfSIntegerFromSInteger(n SInteger) InfSInteger { return NewInfSInteger(n.GetInt()) }
func NewInfSIntegerOffset(n SInteger, positive bool) InfSInteger {
	epsilon := int32(-1)
	if positive {
		epsilon = 1
	}
	return InfSInteger{n.GetInt(), epsilon}
}
func NewInfSIntegerPair(n, epsilon SInteger) InfSInteger {
	return InfSInteger{n.GetInt(), epsilon.GetInt()}
}
func InfSIntegerFromRational(s *RationalService, r *OwnedRational) (InfSInteger, error) {
	n, e := SIntegerFromRational(s, r)
	return NewInfSIntegerFromSInteger(n), e
}
func InfSIntegerZero() InfSInteger     { return NewInfSInteger(0) }
func InfSIntegerOne() InfSInteger      { return NewInfSInteger(1) }
func InfSIntegerMinusOne() InfSInteger { return NewInfSInteger(-1) }
func (n InfSInteger) Hash() uint32     { return uint32(n.first ^ (n.second + 1)) }
func (n *InfSInteger) Swap(other *InfSInteger) {
	n.first, other.first = other.first, n.first
	n.second, other.second = other.second, n.second
}
func (n *InfSInteger) Reset()                    { n.first = 0; n.second = 0 }
func (n InfSInteger) IsInt() bool                { return n.second == 0 }
func (n InfSInteger) IsInt64() bool              { return n.second == 0 }
func (n InfSInteger) IsUint64() bool             { return n.second == 0 }
func (n InfSInteger) IsRational() bool           { return n.second == 0 }
func (n InfSInteger) GetInt64() int64            { return int64(n.first) }
func (n InfSInteger) GetUint64() uint64          { return uint64(int64(n.first)) }
func (n InfSInteger) GetRational() SInteger      { return NewSInteger(n.first) }
func (n InfSInteger) GetInfinitesimal() SInteger { return NewSInteger(n.second) }
func (n *InfSInteger) Assign(other InfSInteger) *InfSInteger {
	n.first = other.first
	n.second = other.second
	return n
}
func (n *InfSInteger) AssignSInteger(other SInteger) *InfSInteger {
	n.first = other.GetInt()
	n.second = 0
	return n
}
func (n *InfSInteger) AssignRational(s *RationalService, r *OwnedRational) error {
	value, e := SIntegerFromRational(s, r)
	if e != nil {
		return e
	}
	n.AssignSInteger(value)
	return nil
}
func (n InfSInteger) Numerator() (InfSInteger, error) {
	if sourceDebug && n.second != 0 {
		return InfSInteger{}, ErrIntegerAssertion
	}
	return NewInfSInteger(n.first), nil
}
func (n InfSInteger) Denominator() (InfSInteger, error) {
	if sourceDebug && n.second != 0 {
		return InfSInteger{}, ErrIntegerAssertion
	}
	return InfSIntegerOne(), nil
}
func (n *InfSInteger) AddAssign(other InfSInteger) *InfSInteger {
	n.first += other.first
	n.second += other.second
	return n
}
func (n *InfSInteger) SubAssign(other InfSInteger) *InfSInteger {
	n.first -= other.first
	n.second -= other.second
	return n
}
func (n *InfSInteger) AddAssignSInteger(other SInteger) *InfSInteger {
	n.first += other.GetInt()
	return n
}
func (n *InfSInteger) SubAssignSInteger(other SInteger) *InfSInteger {
	n.first -= other.GetInt()
	return n
}
func (n *InfSInteger) MulAssign(other SInteger) *InfSInteger {
	n.first *= other.GetInt()
	n.second *= other.GetInt()
	return n
}
func (n *InfSInteger) DivAssign(other SInteger) *InfSInteger {
	n.first /= other.GetInt()
	n.second /= other.GetInt()
	return n
}
func (n InfSInteger) Add(other InfSInteger) InfSInteger         { n.AddAssign(other); return n }
func (n InfSInteger) Sub(other InfSInteger) InfSInteger         { n.SubAssign(other); return n }
func (n InfSInteger) Div(other SInteger) InfSInteger            { n.DivAssign(other); return n }
func (n SInteger) MulInfSInteger(other InfSInteger) InfSInteger { other.MulAssign(n); return other }
func (n *InfSInteger) Inc() *InfSInteger                        { n.first++; return n }
func (n *InfSInteger) Dec() *InfSInteger                        { n.first--; return n }
func (n *InfSInteger) PostInc() InfSInteger                     { old := *n; n.Inc(); return old }
func (n *InfSInteger) PostDec() InfSInteger                     { old := *n; n.Dec(); return old }
func (n InfSInteger) Equal(other InfSInteger) bool {
	return n.first == other.first && n.second == other.second
}
func (n InfSInteger) NotEqual(other InfSInteger) bool { return !n.Equal(other) }
func (n InfSInteger) Less(other InfSInteger) bool {
	return n.first < other.first || n.first == other.first && n.second < other.second
}
func (n InfSInteger) Greater(other InfSInteger) bool      { return other.Less(n) }
func (n InfSInteger) LessEqual(other InfSInteger) bool    { return !n.Greater(other) }
func (n InfSInteger) GreaterEqual(other InfSInteger) bool { return !n.Less(other) }
func (n InfSInteger) EqualSInteger(other SInteger) bool {
	return n.first == other.GetInt() && n.second == 0
}
func (n InfSInteger) NotEqualSInteger(other SInteger) bool { return !n.EqualSInteger(other) }
func (n InfSInteger) LessSInteger(other SInteger) bool {
	return n.first < other.GetInt() || n.first == other.GetInt() && n.second < 0
}
func (n InfSInteger) GreaterSInteger(other SInteger) bool      { return other.LessInfSInteger(n) }
func (n InfSInteger) LessEqualSInteger(other SInteger) bool    { return !n.GreaterSInteger(other) }
func (n InfSInteger) GreaterEqualSInteger(other SInteger) bool { return !n.LessSInteger(other) }
func (n SInteger) EqualInfSInteger(other InfSInteger) bool     { return other.EqualSInteger(n) }
func (n SInteger) NotEqualInfSInteger(other InfSInteger) bool  { return !n.EqualInfSInteger(other) }
func (n SInteger) LessInfSInteger(other InfSInteger) bool {
	return n.GetInt() < other.first || n.GetInt() == other.first && other.second > 0
}
func (n SInteger) GreaterInfSInteger(other InfSInteger) bool      { return other.LessSInteger(n) }
func (n SInteger) LessEqualInfSInteger(other InfSInteger) bool    { return !n.GreaterInfSInteger(other) }
func (n SInteger) GreaterEqualInfSInteger(other InfSInteger) bool { return !n.LessInfSInteger(other) }
func (n *InfSInteger) Neg()                                       { n.first = -n.first; n.second = -n.second }
func (n InfSInteger) Negated() InfSInteger                        { n.Neg(); return n }
func (n InfSInteger) IsZero() bool                                { return n.first == 0 && n.second == 0 }
func (n InfSInteger) IsOne() bool                                 { return n.first == 1 && n.second == 0 }
func (n InfSInteger) IsMinusOne() bool                            { return n.first == -1 && n.second == 0 }
func (n InfSInteger) IsNeg() bool                                 { return n.first < 0 || n.first == 0 && n.second < 0 }
func (n InfSInteger) IsPos() bool                                 { return n.first > 0 || n.first == 0 && n.second > 0 }
func (n InfSInteger) IsNonneg() bool                              { return n.first > 0 || n.first == 0 && n.second >= 0 }
func (n InfSInteger) IsNonpos() bool                              { return n.first < 0 || n.first == 0 && n.second <= 0 }
func (n InfSInteger) Floor() SInteger {
	if n.second < 0 {
		return NewSInteger(n.first - 1)
	}
	return NewSInteger(n.first)
}
func (n InfSInteger) Ceil() SInteger {
	if n.second > 0 {
		return NewSInteger(n.first + 1)
	}
	return NewSInteger(n.first)
}
func (n *InfSInteger) Addmul(c SInteger, k InfSInteger) {
	n.first += c.GetInt() * k.first
	n.second += c.GetInt() * k.second
}
func (n *InfSInteger) Submul(c SInteger, k InfSInteger) {
	n.first -= c.GetInt() * k.first
	n.second -= c.GetInt() * k.second
}
func (n InfSInteger) Abs() InfSInteger {
	if n.IsNeg() {
		n.Neg()
	}
	return n
}
func (n InfSInteger) Display(out *NumericTextStream) error {
	if n.second == 0 {
		return out.AppendInt32(n.first)
	}
	if e := out.AppendCString("("); e != nil {
		return e
	}
	if e := out.AppendInt32(n.first); e != nil {
		return e
	}
	if n.second < 0 {
		if e := out.AppendCString(" -e*"); e != nil {
			return e
		}
	} else {
		if e := out.AppendCString(" +e*"); e != nil {
			return e
		}
	}
	if e := out.AppendInt32(n.second); e != nil {
		return e
	}
	return out.AppendCString(")")
}

// ToString supplies the undefined C++ declaration as a Go convenience, using
// decimal text and preserving the implemented stream body's coefficient sign.
func (n InfSInteger) ToString() string {
	first := strconv.FormatInt(int64(n.first), 10)
	if n.second == 0 {
		return first
	}
	separator := " +e*"
	if n.second < 0 {
		separator = " -e*"
	}
	return "(" + first + separator + strconv.FormatInt(int64(n.second), 10) + ")"
}
