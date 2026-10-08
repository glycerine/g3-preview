// Source: src/util/checked_int64.h:30-219.
package util

import (
	"errors"
	"math"
)

// CheckedInt64 represents the two original CHECK template profiles without Go
// generics. Constructors select the profile; a scalar copy preserves it.
// The Go zero value selects the unchecked profile. Unchecked overflow has Go
// wrapping semantics; native C++ signed overflow is not a portable promise.
type CheckedInt64 struct {
	value int64
	check bool
}

// CheckedInt64OverflowException preserves the nested z3_exception subtype.
type CheckedInt64OverflowException struct{}

func (CheckedInt64OverflowException) Message() string      { return "checked_int64 overflow/underflow" }
func (e CheckedInt64OverflowException) Error() string      { return e.Message() }
func (CheckedInt64OverflowException) ErrorCode() ErrorCode { return ErrorOK }
func (CheckedInt64OverflowException) HasErrorCode() bool   { return false }

var ErrCheckedInt64Overflow = CheckedInt64OverflowException{}
var ErrCheckedInt64Profile = errors.New("g3: checked_int64 profile mismatch")

func NewCheckedInt64(v int64) CheckedInt64         { return CheckedInt64{v, true} }
func NewUncheckedInt64(v int64) CheckedInt64       { return CheckedInt64{v, false} }
func CheckedInt64Zero(check bool) CheckedInt64     { return CheckedInt64{0, check} }
func CheckedInt64One(check bool) CheckedInt64      { return CheckedInt64{1, check} }
func CheckedInt64MinusOne(check bool) CheckedInt64 { return CheckedInt64{-1, check} }
func (n CheckedInt64) Int64() int64                { return n.value }
func (n CheckedInt64) Checks() bool                { return n.check }
func (n CheckedInt64) IsZero() bool                { return n.value == 0 }
func (n CheckedInt64) IsPos() bool                 { return n.value > 0 }
func (n CheckedInt64) IsNeg() bool                 { return n.value < 0 }
func (n CheckedInt64) IsOne() bool                 { return n.value == 1 }
func (n CheckedInt64) IsMinusOne() bool            { return n.value == -1 }
func (n CheckedInt64) IsNonNeg() bool              { return n.value >= 0 }
func (n CheckedInt64) IsNonPos() bool              { return n.value <= 0 }
func (n CheckedInt64) IsEven() bool                { return n.value^1 == 0 } // original source quirk.
func (n CheckedInt64) Hash() uint32                { return uint32(n.value) }
func (n CheckedInt64) Eq(other CheckedInt64) bool  { return n.value == other.value }
func (n CheckedInt64) Neq(other CheckedInt64) bool { return !n.Eq(other) }
func (n CheckedInt64) Lt(other CheckedInt64) bool  { return n.value < other.value }
func (n CheckedInt64) Gt(other CheckedInt64) bool  { return other.Lt(n) }
func (n CheckedInt64) Le(other CheckedInt64) bool  { return !n.Gt(other) }
func (n CheckedInt64) Ge(other CheckedInt64) bool  { return !n.Lt(other) }

type CheckedInt64HashProc struct{}

func (CheckedInt64HashProc) Hash(n CheckedInt64) uint32 { return n.Hash() }

type CheckedInt64EqProc struct{}

func (CheckedInt64EqProc) Equal(a, b CheckedInt64) bool     { return a.Eq(b) }
func (n CheckedInt64) Display(out *NumericTextStream) error { return out.AppendInt64(n.value) }
func (n CheckedInt64) Abs() (CheckedInt64, error) {
	if n.value >= 0 {
		return n, nil
	}
	if n.check && n.value == math.MinInt64 {
		return CheckedInt64{}, ErrCheckedInt64Overflow
	}
	n.value = -n.value
	return n, nil
}
func (n *CheckedInt64) Neg() error {
	if n.check && n.value == math.MinInt64 {
		return ErrCheckedInt64Overflow
	}
	n.value = -n.value
	return nil
}
func (n *CheckedInt64) Inc() error {
	if n.check && n.value == math.MaxInt64 {
		return ErrCheckedInt64Overflow
	}
	n.value++
	return nil
}
func (n *CheckedInt64) Dec() error {
	if n.check && n.value == math.MinInt64 {
		return ErrCheckedInt64Overflow
	}
	n.value--
	return nil
}
func (n *CheckedInt64) PostInc() (CheckedInt64, error) { old := *n; e := n.Inc(); return old, e }
func (n *CheckedInt64) PostDec() (CheckedInt64, error) { old := *n; e := n.Dec(); return old, e }
func (n *CheckedInt64) AddAssign(other CheckedInt64) error {
	if n.check != other.check {
		return ErrCheckedInt64Profile
	}
	if n.check {
		result := int64(uint64(n.value) + uint64(other.value))
		if n.value > 0 && other.value > 0 && result <= 0 || n.value < 0 && other.value < 0 && result >= 0 {
			return ErrCheckedInt64Overflow
		}
		n.value = result
	} else {
		n.value += other.value
	}
	return nil
}
func (n *CheckedInt64) SubAssign(other CheckedInt64) error {
	if n.check != other.check {
		return ErrCheckedInt64Profile
	}
	if n.check {
		result := int64(uint64(n.value) - uint64(other.value))
		if n.value > 0 && other.value < 0 && result <= 0 || n.value < 0 && other.value > 0 && result >= 0 {
			return ErrCheckedInt64Overflow
		}
		n.value = result
	} else {
		n.value -= other.value
	}
	return nil
}
func CheckedInt64Neg(n CheckedInt64) (CheckedInt64, error)    { e := n.Neg(); return n, e }
func CheckedInt64Abs(n CheckedInt64) (CheckedInt64, error)    { return n.Abs() }
func CheckedInt64Add(a, b CheckedInt64) (CheckedInt64, error) { e := a.AddAssign(b); return a, e }
func CheckedInt64Sub(a, b CheckedInt64) (CheckedInt64, error) { e := a.SubAssign(b); return a, e }

// CheckedInt64MulScope is explicit caller-provided storage for the source
// rational expression's actual owned operands/result and stopped failure.
// It borrows the service corresponding to C++'s global rational manager.
// Its control storage is caller-owned; rational backing uses the service budget.
type CheckedInt64MulScope struct {
	self                 *CheckedInt64MulScope
	service              *RationalService
	left, right, product OwnedRational
	cause                error
	active               bool
}

// MulAssign preserves the strict INT_MIN < operand <= INT_MAX fast path.
// Fast/unchecked multiplication erases both service and scope. The checked
// fallback constructs rational(int64) operands and invokes rational operator*,
// including its original copy/compound/copy behavior rather than a shortcut.
func (n *CheckedInt64) MulAssign(other CheckedInt64, s *RationalService, local *CheckedInt64MulScope) error {
	if n.check != other.check {
		return ErrCheckedInt64Profile
	}
	if !n.check || math.MinInt32 < n.value && n.value <= math.MaxInt32 && math.MinInt32 < other.value && other.value <= math.MaxInt32 {
		n.value *= other.value
		return nil
	}
	if local == nil {
		return ErrParamsPrecondition
	}
	if local.self != nil {
		if local.self == local && local.cause != nil {
			return local.cause
		}
		return ErrParamsPrecondition
	}
	if e := s.lock(); e != nil {
		return e
	}
	s.mu.Unlock()
	local.self = local
	local.service = s
	local.active = true
	return local.multiply(n, other)
}
func (local *CheckedInt64MulScope) multiply(dst *CheckedInt64, other CheckedInt64) (e error) {
	defer func() {
		if e != nil {
			local.cause = e
		}
	}()
	s := local.service
	local.left, e = s.NewValue()
	if e != nil {
		return e
	}
	if e = s.SetInt64(&local.left, dst.value); e != nil {
		return e
	}
	local.right, e = s.NewValue()
	if e != nil {
		return e
	}
	if e = s.SetInt64(&local.right, other.value); e != nil {
		return e
	}
	local.product, e = s.Mul(&local.left, &local.right)
	if e != nil {
		return e
	}
	// End of the initializing full expression: right then left source operands
	// are destroyed before inspecting the named result r.
	if e = s.Del(&local.right); e != nil {
		return e
	}
	if e = s.Del(&local.left); e != nil {
		return e
	}
	fits, e := s.IsInt64(&local.product)
	if e != nil {
		return e
	}
	if !fits {
		// A reached source overflow exception still destroys the completed r.
		if e = s.Del(&local.product); e != nil {
			return e
		}
		local.active = false
		return ErrCheckedInt64Overflow
	}
	value, e := s.Int64(&local.product)
	if e != nil {
		return e
	}
	dst.value = value // original assignment precedes the named r's destructor.
	if e = s.Del(&local.product); e != nil {
		return e
	}
	local.active = false
	return nil
}
func CheckedInt64Mul(a, b CheckedInt64, s *RationalService, local *CheckedInt64MulScope) (CheckedInt64, error) {
	e := a.MulAssign(b, s, local)
	return a, e
}

// Cleanup joins younger numerical/facade locals before owning the retained
// expression operands. It does not retry multiplication or scalar assignment.
func (local *CheckedInt64MulScope) Cleanup() error {
	if local == nil {
		return ErrParamsPrecondition
	}
	if local.self == nil {
		return nil
	}
	if local.self != local {
		return ErrParamsPrecondition
	}
	if !local.active {
		return nil
	}
	s := local.service
	if e := s.CleanupLexical(); e != nil {
		return e
	}
	if e := s.Manager.CleanupRawArithmetic(); e != nil {
		return e
	}
	if e := s.Manager.CleanupLexical(); e != nil {
		return e
	}
	for _, v := range [...]*OwnedRational{&local.product, &local.right, &local.left} {
		if v.ownerID != 0 {
			if e := s.Del(v); e != nil {
				return e
			}
		}
	}
	local.active = false
	return nil
}
