// Concrete translations of util/scoped_numeral.h and scoped_numeral_vector.h.
package util

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrScopedCopy = errors.New("mpnum: shallow scoped owner copy")
var ErrScopedOwner = errors.New("mpnum: invalid scoped numeral owner")

// ScopedNumeralManager is the non-generic adapter for the Manager template in
// scoped_numeral.h. Every callback operates on the manager's concrete numeral
// value stored in an any slot.
type ScopedNumeralManager struct {
	New     func() any
	Delete  func(*any) error
	Set     func(*any, any) error
	SetInt  func(*any, int) error
	Reset   func(*any) error
	Add     func(*any, any, any) error
	Sub     func(*any, any, any) error
	Mul     func(*any, any, any) error
	Div     func(*any, any, any) error
	Rem     func(*any, any, any) error
	Neg     func(*any) error
	Abs     func(*any) error
	Compare func(any, any) (int, error)
	Sign    func(any) (int, error)
	Display func(any) (string, error)
}

func (manager *ScopedNumeralManager) valid() bool {
	return manager != nil && manager.New != nil && manager.Delete != nil && manager.Set != nil &&
		manager.SetInt != nil && manager.Reset != nil && manager.Add != nil && manager.Sub != nil &&
		manager.Mul != nil && manager.Div != nil && manager.Rem != nil && manager.Neg != nil &&
		manager.Abs != nil && manager.Compare != nil && manager.Sign != nil && manager.Display != nil
}

// ScopedNumeral is the non-generic Go form of _scoped_numeral. Concrete
// numerical packages may also expose typed wrappers such as ScopedInt.
type ScopedNumeral struct {
	self    *ScopedNumeral
	manager *ScopedNumeralManager
	num     any
	closed  bool
}

func NewScopedNumeral(manager *ScopedNumeralManager) (*ScopedNumeral, error) {
	if !manager.valid() {
		return nil, ErrScopedOwner
	}
	scope := &ScopedNumeral{manager: manager, num: manager.New()}
	scope.self = scope
	return scope, nil
}

func (scope *ScopedNumeral) requireOpen() error {
	if scope == nil || scope.self != scope {
		return ErrScopedCopy
	}
	if scope.closed {
		return arena.ErrClosed
	}
	return nil
}

func (scope *ScopedNumeral) Get() (*any, error) {
	if err := scope.requireOpen(); err != nil {
		return nil, err
	}
	return &scope.num, nil
}

func (scope *ScopedNumeral) Manager() (*ScopedNumeralManager, error) {
	if err := scope.requireOpen(); err != nil {
		return nil, err
	}
	return scope.manager, nil
}

func (scope *ScopedNumeral) Set(value any) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	return scope.manager.Set(&scope.num, value)
}

func (scope *ScopedNumeral) SetInt(value int) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	return scope.manager.SetInt(&scope.num, value)
}

func (scope *ScopedNumeral) Assign(other *ScopedNumeral) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	if err := other.requireOpen(); err != nil {
		return err
	}
	if scope == other {
		return nil
	}
	return scope.manager.Set(&scope.num, other.num)
}

func (scope *ScopedNumeral) Clone() (*ScopedNumeral, error) {
	if err := scope.requireOpen(); err != nil {
		return nil, err
	}
	clone, err := NewScopedNumeral(scope.manager)
	if err != nil {
		return nil, err
	}
	if err = clone.Set(scope.num); err != nil {
		_ = clone.Close()
		return nil, err
	}
	return clone, nil
}

func (scope *ScopedNumeral) Reset() error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	return scope.manager.Reset(&scope.num)
}

func (scope *ScopedNumeral) Swap(other *ScopedNumeral) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	if err := other.requireOpen(); err != nil {
		return err
	}
	scope.num, other.num = other.num, scope.num
	return nil
}

func (scope *ScopedNumeral) SwapValue(value *any) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	if value == nil {
		return ErrScopedOwner
	}
	scope.num, *value = *value, scope.num
	return nil
}

func (scope *ScopedNumeral) apply(value any, operation uint8) error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	switch operation {
	case 0:
		return scope.manager.Add(&scope.num, scope.num, value)
	case 1:
		return scope.manager.Sub(&scope.num, scope.num, value)
	case 2:
		return scope.manager.Mul(&scope.num, scope.num, value)
	case 3:
		return scope.manager.Div(&scope.num, scope.num, value)
	default:
		return scope.manager.Rem(&scope.num, scope.num, value)
	}
}

func (scope *ScopedNumeral) AddAssign(value any) error { return scope.apply(value, 0) }
func (scope *ScopedNumeral) SubAssign(value any) error { return scope.apply(value, 1) }
func (scope *ScopedNumeral) MulAssign(value any) error { return scope.apply(value, 2) }
func (scope *ScopedNumeral) DivAssign(value any) error { return scope.apply(value, 3) }
func (scope *ScopedNumeral) RemAssign(value any) error { return scope.apply(value, 4) }

func (scope *ScopedNumeral) result(value any, operation uint8) (*ScopedNumeral, error) {
	clone, err := scope.Clone()
	if err != nil {
		return nil, err
	}
	if err = clone.apply(value, operation); err != nil {
		_ = clone.Close()
		return nil, err
	}
	return clone, nil
}

func (scope *ScopedNumeral) Add(value any) (*ScopedNumeral, error) {
	return scope.result(value, 0)
}
func (scope *ScopedNumeral) Sub(value any) (*ScopedNumeral, error) {
	return scope.result(value, 1)
}
func (scope *ScopedNumeral) Mul(value any) (*ScopedNumeral, error) {
	return scope.result(value, 2)
}
func (scope *ScopedNumeral) Div(value any) (*ScopedNumeral, error) {
	return scope.result(value, 3)
}

func (scope *ScopedNumeral) Neg() error {
	if err := scope.requireOpen(); err != nil {
		return err
	}
	return scope.manager.Neg(&scope.num)
}

func (scope *ScopedNumeral) AbsCopy() (*ScopedNumeral, error) {
	clone, err := scope.Clone()
	if err != nil {
		return nil, err
	}
	if err = clone.manager.Abs(&clone.num); err != nil {
		_ = clone.Close()
		return nil, err
	}
	return clone, nil
}

func (scope *ScopedNumeral) Compare(value any) (int, error) {
	if err := scope.requireOpen(); err != nil {
		return 0, err
	}
	return scope.manager.Compare(scope.num, value)
}

func (scope *ScopedNumeral) Equal(value any) (bool, error) {
	comparison, err := scope.Compare(value)
	return comparison == 0, err
}
func (scope *ScopedNumeral) NotEqual(value any) (bool, error) {
	equal, err := scope.Equal(value)
	return !equal, err
}
func (scope *ScopedNumeral) Less(value any) (bool, error) {
	comparison, err := scope.Compare(value)
	return comparison < 0, err
}
func (scope *ScopedNumeral) Greater(value any) (bool, error) {
	comparison, err := scope.Compare(value)
	return comparison > 0, err
}
func (scope *ScopedNumeral) LessEqual(value any) (bool, error) {
	comparison, err := scope.Compare(value)
	return comparison <= 0, err
}
func (scope *ScopedNumeral) GreaterEqual(value any) (bool, error) {
	comparison, err := scope.Compare(value)
	return comparison >= 0, err
}

func (scope *ScopedNumeral) sign() (int, error) {
	if err := scope.requireOpen(); err != nil {
		return 0, err
	}
	return scope.manager.Sign(scope.num)
}

func (scope *ScopedNumeral) IsZero() (bool, error) { sign, err := scope.sign(); return sign == 0, err }
func (scope *ScopedNumeral) IsPos() (bool, error)  { sign, err := scope.sign(); return sign > 0, err }
func (scope *ScopedNumeral) IsNeg() (bool, error)  { sign, err := scope.sign(); return sign < 0, err }
func (scope *ScopedNumeral) IsNonpos() (bool, error) {
	sign, err := scope.sign()
	return sign <= 0, err
}
func (scope *ScopedNumeral) IsNonneg() (bool, error) {
	sign, err := scope.sign()
	return sign >= 0, err
}

func (scope *ScopedNumeral) Display() (string, error) {
	if err := scope.requireOpen(); err != nil {
		return "", err
	}
	return scope.manager.Display(scope.num)
}

func (scope *ScopedNumeral) Close() error {
	if scope == nil || scope.self != scope {
		return ErrScopedCopy
	}
	if scope.closed {
		return nil
	}
	if err := scope.manager.Delete(&scope.num); err != nil {
		return err
	}
	scope.num = nil
	scope.manager = nil
	scope.closed = true
	return nil
}

// ScopedInt owns one numeral and borrows its manager. Get returns the source's
// mutable numeral reference; it is valid only until Close and must not be copied
// as an owning Int. Clone is the explicit deep copy operation. All objects are
// owner-confined; close them before their manager. A self pointer rejects Go
// struct copies without unsafe or per-record interfaces.
type ScopedInt struct {
	manager *IntegerManager
	num     Int
	self    *ScopedInt
	closed  bool
}
type ScopedRational struct {
	manager *RationalManager
	num     Rational
	self    *ScopedRational
	closed  bool
}

func NewScopedInt(m *IntegerManager) (*ScopedInt, error) {
	if m == nil || m.Cells == nil {
		return nil, ErrScopedOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if e := m.BudgetField.Reserve(uint64(reflect.TypeOf(ScopedInt{}).Size())); e != nil {
		return nil, e
	}
	s := &ScopedInt{manager: m}
	s.self = s
	return s, nil
}
func NewScopedRational(m *RationalManager) (*ScopedRational, error) {
	if m == nil || m.Z == nil {
		return nil, ErrScopedOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if e := m.BudgetField.Reserve(uint64(reflect.TypeOf(ScopedRational{}).Size())); e != nil {
		return nil, e
	}
	s := &ScopedRational{manager: m, num: NewRational()}
	s.self = s
	return s, nil
}
func (s *ScopedInt) Get() (*Int, error) {
	if s == nil || s.self != s {
		return nil, ErrScopedCopy
	}
	if s.closed {
		return nil, arena.ErrClosed
	}
	if e := s.manager.check(&s.num); e != nil {
		return nil, e
	}
	return &s.num, nil
}
func (s *ScopedRational) Get() (*Rational, error) {
	if s == nil || s.self != s {
		return nil, ErrScopedCopy
	}
	if s.closed {
		return nil, arena.ErrClosed
	}
	if e := s.manager.check(&s.num); e != nil {
		return nil, e
	}
	return &s.num, nil
}
func (s *ScopedInt) Manager() (*IntegerManager, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	return s.manager, nil
}
func (s *ScopedRational) Manager() (*RationalManager, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	return s.manager, nil
}
func (s *ScopedInt) Set(v *Int) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Copy(&s.num, v)
}
func (s *ScopedRational) Set(v *Rational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Copy(&s.num, v)
}
func (s *ScopedInt) SetInt64(v int64) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.SetInt64(&s.num, v)
}
func (s *ScopedRational) SetInt64(v int64) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.SetInt64(&s.num, v)
}
func (s *ScopedInt) Assign(other *ScopedInt) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	v, e := other.Get()
	if e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.manager.Copy(&s.num, v)
}
func (s *ScopedRational) Assign(other *ScopedRational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	v, e := other.Get()
	if e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.manager.Copy(&s.num, v)
}
func (s *ScopedInt) Clone() (*ScopedInt, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	c, e := NewScopedInt(s.manager)
	if e != nil {
		return nil, e
	}
	if e = c.Set(&s.num); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedRational) Clone() (*ScopedRational, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	c, e := NewScopedRational(s.manager)
	if e != nil {
		return nil, e
	}
	if e = c.Set(&s.num); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedInt) Reset() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Reset(&s.num)
}
func (s *ScopedRational) Reset() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Reset(&s.num)
}
func (s *ScopedInt) SwapValue(v *Int) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Swap(&s.num, v)
}
func (s *ScopedRational) SwapValue(v *Rational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Swap(&s.num, v)
}
func (s *ScopedInt) Swap(other *ScopedInt) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	v, e := other.Get()
	if e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.SwapValue(v)
}
func (s *ScopedRational) Swap(other *ScopedRational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	v, e := other.Get()
	if e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.SwapValue(v)
}
func (s *ScopedInt) Neg() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Neg(&s.num)
}
func (s *ScopedRational) Neg() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Neg(&s.num)
}
func (s *ScopedInt) AbsCopy() (*ScopedInt, error) {
	c, e := s.Clone()
	if e != nil {
		return nil, e
	}
	if e = c.manager.Abs(&c.num); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedRational) AbsCopy() (*ScopedRational, error) {
	c, e := s.Clone()
	if e != nil {
		return nil, e
	}
	if e = c.manager.Abs(&c.num); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedInt) apply(v *Int, op uint8) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	switch op {
	case 0:
		return s.manager.Add(&s.num, v, &s.num)
	case 1:
		return s.manager.Sub(&s.num, v, &s.num)
	case 2:
		return s.manager.Mul(&s.num, v, &s.num)
	case 3:
		return s.manager.Div(&s.num, v, &s.num)
	default:
		return s.manager.Rem(&s.num, v, &s.num)
	}
}
func (s *ScopedRational) apply(v *Rational, op uint8) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	switch op {
	case 0:
		return s.manager.Add(&s.num, v, &s.num)
	case 1:
		return s.manager.Sub(&s.num, v, &s.num)
	case 2:
		return s.manager.Mul(&s.num, v, &s.num)
	case 3:
		return s.manager.Div(&s.num, v, &s.num)
	default:
		return s.manager.Rem(&s.num, v, &s.num)
	}
}
func (s *ScopedInt) AddAssign(v *Int) error           { return s.apply(v, 0) }
func (s *ScopedInt) SubAssign(v *Int) error           { return s.apply(v, 1) }
func (s *ScopedInt) MulAssign(v *Int) error           { return s.apply(v, 2) }
func (s *ScopedInt) DivAssign(v *Int) error           { return s.apply(v, 3) }
func (s *ScopedInt) RemAssign(v *Int) error           { return s.apply(v, 4) }
func (s *ScopedRational) AddAssign(v *Rational) error { return s.apply(v, 0) }
func (s *ScopedRational) SubAssign(v *Rational) error { return s.apply(v, 1) }
func (s *ScopedRational) MulAssign(v *Rational) error { return s.apply(v, 2) }
func (s *ScopedRational) DivAssign(v *Rational) error { return s.apply(v, 3) }
func (s *ScopedRational) RemAssign(v *Rational) error { return s.apply(v, 4) }
func (s *ScopedInt) result(v *Int, op uint8) (*ScopedInt, error) {
	c, e := s.Clone()
	if e != nil {
		return nil, e
	}
	if e = c.apply(v, op); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedRational) result(v *Rational, op uint8) (*ScopedRational, error) {
	c, e := s.Clone()
	if e != nil {
		return nil, e
	}
	if e = c.apply(v, op); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (s *ScopedInt) Add(v *Int) (*ScopedInt, error)                { return s.result(v, 0) }
func (s *ScopedInt) Sub(v *Int) (*ScopedInt, error)                { return s.result(v, 1) }
func (s *ScopedInt) Mul(v *Int) (*ScopedInt, error)                { return s.result(v, 2) }
func (s *ScopedInt) Div(v *Int) (*ScopedInt, error)                { return s.result(v, 3) }
func (s *ScopedRational) Add(v *Rational) (*ScopedRational, error) { return s.result(v, 0) }
func (s *ScopedRational) Sub(v *Rational) (*ScopedRational, error) { return s.result(v, 1) }
func (s *ScopedRational) Mul(v *Rational) (*ScopedRational, error) { return s.result(v, 2) }
func (s *ScopedRational) Div(v *Rational) (*ScopedRational, error) { return s.result(v, 3) }
func (s *ScopedInt) Compare(v *Int) (int, error) {
	if _, e := s.Get(); e != nil {
		return 0, e
	}
	return s.manager.Compare(&s.num, v)
}
func (s *ScopedRational) Compare(v *Rational) (int, error) {
	if _, e := s.Get(); e != nil {
		return 0, e
	}
	return s.manager.Compare(&s.num, v)
}
func (s *ScopedInt) Equal(v *Int) (bool, error) { c, e := s.Compare(v); return c == 0, e }
func (s *ScopedRational) Equal(v *Rational) (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.Equal(&s.num, v)
}
func (s *ScopedInt) NotEqual(v *Int) (bool, error)            { v1, e := s.Equal(v); return !v1, e }
func (s *ScopedRational) NotEqual(v *Rational) (bool, error)  { v1, e := s.Equal(v); return !v1, e }
func (s *ScopedInt) Less(v *Int) (bool, error)                { c, e := s.Compare(v); return c < 0, e }
func (s *ScopedInt) Greater(v *Int) (bool, error)             { c, e := s.Compare(v); return c > 0, e }
func (s *ScopedInt) LessEqual(v *Int) (bool, error)           { c, e := s.Compare(v); return c <= 0, e }
func (s *ScopedInt) GreaterEqual(v *Int) (bool, error)        { c, e := s.Compare(v); return c >= 0, e }
func (s *ScopedRational) Less(v *Rational) (bool, error)      { c, e := s.Compare(v); return c < 0, e }
func (s *ScopedRational) Greater(v *Rational) (bool, error)   { c, e := s.Compare(v); return c > 0, e }
func (s *ScopedRational) LessEqual(v *Rational) (bool, error) { c, e := s.Compare(v); return c <= 0, e }
func (s *ScopedRational) GreaterEqual(v *Rational) (bool, error) {
	c, e := s.Compare(v)
	return c >= 0, e
}
func (s *ScopedInt) sign() (int32, error) {
	if _, e := s.Get(); e != nil {
		return 0, e
	}
	return s.num.Val, nil
}
func (s *ScopedRational) sign() (int32, error) {
	if _, e := s.Get(); e != nil {
		return 0, e
	}
	return s.num.Num.Val, nil
}
func (s *ScopedInt) IsZero() (bool, error)        { v, e := s.sign(); return v == 0, e }
func (s *ScopedInt) IsPos() (bool, error)         { v, e := s.sign(); return v > 0, e }
func (s *ScopedInt) IsNeg() (bool, error)         { v, e := s.sign(); return v < 0, e }
func (s *ScopedInt) IsNonpos() (bool, error)      { v, e := s.sign(); return v <= 0, e }
func (s *ScopedInt) IsNonneg() (bool, error)      { v, e := s.sign(); return v >= 0, e }
func (s *ScopedRational) IsZero() (bool, error)   { v, e := s.sign(); return v == 0, e }
func (s *ScopedRational) IsPos() (bool, error)    { v, e := s.sign(); return v > 0, e }
func (s *ScopedRational) IsNeg() (bool, error)    { v, e := s.sign(); return v < 0, e }
func (s *ScopedRational) IsNonpos() (bool, error) { v, e := s.sign(); return v <= 0, e }
func (s *ScopedRational) IsNonneg() (bool, error) { v, e := s.sign(); return v >= 0, e }
func (s *ScopedInt) Display() (string, error) {
	if _, e := s.Get(); e != nil {
		return "", e
	}
	return s.manager.String(&s.num)
}
func (s *ScopedRational) Display() (string, error) {
	if _, e := s.Get(); e != nil {
		return "", e
	}
	return s.manager.String(&s.num)
}
func (s *ScopedInt) Close() error {
	if s == nil || s.self != s {
		return ErrScopedCopy
	}
	if s.closed {
		return nil
	}
	var e error
	if s.manager.closed {
		e = arena.ErrClosed
	} else {
		e = s.manager.Del(&s.num)
	}
	if e != nil && !s.manager.closed {
		return e
	}
	s.manager.BudgetField.Release(uint64(reflect.TypeOf(ScopedInt{}).Size()))
	s.num = Int{}
	s.closed = true
	return e
}
func (s *ScopedRational) Close() error {
	if s == nil || s.self != s {
		return ErrScopedCopy
	}
	if s.closed {
		return nil
	}
	var e error
	if s.manager.closed {
		e = arena.ErrClosed
	} else {
		e = s.manager.Del(&s.num)
	}
	if e != nil && !s.manager.closed {
		return e
	}
	s.manager.BudgetField.Release(uint64(reflect.TypeOf(ScopedRational{}).Size()))
	s.num = NewRational()
	s.closed = true
	return e
}

// Vectors remain source-noncopyable. No hot backing slice or element pointer
// escapes: Read makes a deep caller-owned numeral; Set performs manager copy.
type ScopedIntVector struct {
	manager *IntegerManager
	data    []Int
	self    *ScopedIntVector
	closed  bool
}
type ScopedRationalVector struct {
	manager *RationalManager
	data    []Rational
	self    *ScopedRationalVector
	closed  bool
}

func NewScopedIntVector(m *IntegerManager) (*ScopedIntVector, error) {
	if m == nil || m.Cells == nil {
		return nil, ErrScopedOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if e := m.BudgetField.Reserve(uint64(reflect.TypeOf(ScopedIntVector{}).Size())); e != nil {
		return nil, e
	}
	v := &ScopedIntVector{manager: m}
	v.self = v
	return v, nil
}
func NewScopedRationalVector(m *RationalManager) (*ScopedRationalVector, error) {
	if m == nil || m.Z == nil {
		return nil, ErrScopedOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if e := m.BudgetField.Reserve(uint64(reflect.TypeOf(ScopedRationalVector{}).Size())); e != nil {
		return nil, e
	}
	v := &ScopedRationalVector{manager: m}
	v.self = v
	return v, nil
}
func (v *ScopedIntVector) open() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed || v.manager.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *ScopedRationalVector) open() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed || v.manager.closed {
		return arena.ErrClosed
	}
	return nil
}
func scopedCapacity(old int, width uint64) (int, error) {
	n := uint64(2)
	if old > 0 {
		n = (3*uint64(old) + 1) / 2
	}
	if n > math.MaxUint32 || n <= uint64(old) || n > uint64(int(^uint(0)>>1))/width {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
func (v *ScopedIntVector) grow() error {
	width := uint64(reflect.TypeOf(Int{}).Size())
	n, e := scopedCapacity(cap(v.data), width)
	if e != nil {
		return e
	}
	if e = v.manager.BudgetField.Reserve(uint64(n) * width); e != nil {
		return e
	}
	next := make([]Int, len(v.data), n)
	copy(next, v.data)
	for i := range v.data {
		v.data[i].Cell = 0
	} // source moved-from m_val survives
	v.manager.BudgetField.Release(uint64(cap(v.data)) * width)
	v.data = next
	return nil
}
func (v *ScopedRationalVector) grow() error {
	width := uint64(reflect.TypeOf(Rational{}).Size())
	n, e := scopedCapacity(cap(v.data), width)
	if e != nil {
		return e
	}
	if e = v.manager.BudgetField.Reserve(uint64(n) * width); e != nil {
		return e
	}
	next := make([]Rational, len(v.data), n)
	copy(next, v.data)
	for i := range v.data {
		v.data[i].Num.Cell = 0
		v.data[i].Den.Cell = 0
	}
	v.manager.BudgetField.Release(uint64(cap(v.data)) * width)
	v.data = next
	return nil
}
func (v *ScopedIntVector) Size() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *ScopedRationalVector) Size() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *ScopedIntVector) Capacity() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return uint32(cap(v.data))
}
func (v *ScopedRationalVector) Capacity() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return uint32(cap(v.data))
}
func (v *ScopedIntVector) Manager() (*IntegerManager, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	return v.manager, nil
}
func (v *ScopedRationalVector) Manager() (*RationalManager, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	return v.manager, nil
}
func (v *ScopedIntVector) Read(i uint32, out *Int) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	return v.manager.Copy(out, &v.data[i])
}
func (v *ScopedRationalVector) Read(i uint32, out *Rational) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	return v.manager.Copy(out, &v.data[i])
}
func (v *ScopedIntVector) Set(i uint32, num *Int) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	return v.manager.Copy(&v.data[i], num)
}
func (v *ScopedRationalVector) Set(i uint32, num *Rational) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	return v.manager.Copy(&v.data[i], num)
}

// Source prefix: grow backing, publish a default element, then deep set. A
// failed copy retains the appended default element. Invalid operands are
// rejected before source actions, unlike raw source undefined references.
func (v *ScopedIntVector) Push(num *Int) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := v.manager.check(num); e != nil {
		return e
	}
	if len(v.data) == cap(v.data) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	v.data = append(v.data, Int{})
	return v.manager.Copy(&v.data[len(v.data)-1], num)
}
func (v *ScopedRationalVector) Push(num *Rational) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := v.manager.check(num); e != nil {
		return e
	}
	if len(v.data) == cap(v.data) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	v.data = append(v.data, NewRational())
	return v.manager.Copy(&v.data[len(v.data)-1], num)
}
func (v *ScopedIntVector) Shrink(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	for i := int(n); i < len(v.data); i++ {
		if e := v.manager.Del(&v.data[i]); e != nil {
			return e
		}
	}
	v.data = v.data[:n]
	return nil
}
func (v *ScopedRationalVector) Shrink(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	for i := int(n); i < len(v.data); i++ {
		if e := v.manager.Del(&v.data[i]); e != nil {
			return e
		}
	}
	v.data = v.data[:n]
	return nil
}
func (v *ScopedIntVector) Pop() error {
	if e := v.open(); e != nil {
		return e
	}
	if len(v.data) == 0 {
		return arena.ErrBounds
	}
	return v.Shrink(uint32(len(v.data) - 1))
}
func (v *ScopedRationalVector) Pop() error {
	if e := v.open(); e != nil {
		return e
	}
	if len(v.data) == 0 {
		return arena.ErrBounds
	}
	return v.Shrink(uint32(len(v.data) - 1))
}
func (v *ScopedIntVector) Resize(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) <= uint64(len(v.data)) {
		return v.Shrink(n)
	}
	for uint64(n) > uint64(cap(v.data)) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	old := len(v.data)
	v.data = v.data[:n]
	for i := old; i < len(v.data); i++ {
		v.data[i] = Int{}
	}
	return nil
}
func (v *ScopedRationalVector) Resize(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) <= uint64(len(v.data)) {
		return v.Shrink(n)
	}
	for uint64(n) > uint64(cap(v.data)) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	old := len(v.data)
	v.data = v.data[:n]
	for i := old; i < len(v.data); i++ {
		v.data[i] = NewRational()
	}
	return nil
}
func (v *ScopedIntVector) Reset() error      { return v.Shrink(0) }
func (v *ScopedRationalVector) Reset() error { return v.Shrink(0) }
func (v *ScopedIntVector) Close() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return nil
	}
	var e error
	if v.manager.closed {
		e = arena.ErrClosed
	} else {
		e = v.Reset()
	}
	if e != nil && !v.manager.closed {
		return e
	}
	v.manager.BudgetField.Release(uint64(reflect.TypeOf(ScopedIntVector{}).Size()) + uint64(cap(v.data))*uint64(reflect.TypeOf(Int{}).Size()))
	v.data = nil
	v.closed = true
	return e
}
func (v *ScopedRationalVector) Close() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return nil
	}
	var e error
	if v.manager.closed {
		e = arena.ErrClosed
	} else {
		e = v.Reset()
	}
	if e != nil && !v.manager.closed {
		return e
	}
	v.manager.BudgetField.Release(uint64(reflect.TypeOf(ScopedRationalVector{}).Size()) + uint64(cap(v.data))*uint64(reflect.TypeOf(Rational{}).Size()))
	v.data = nil
	v.closed = true
	return e
}
