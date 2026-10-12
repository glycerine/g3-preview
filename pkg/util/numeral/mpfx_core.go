// Source: src/util/mpfx.h and src/util/mpfx.cpp:28-83,90-100,133-249,
// 316-520. Concrete fixed-point numeral reached by subpaving's mpfx context.
package numeral

import (
	"errors"
	"math/big"
)

var (
	ErrFixedPointOwner    = errors.New("mpfx: invalid numeral owner")
	ErrFixedPointClosed   = errors.New("mpfx: manager or numeral closed")
	ErrFixedPointOverflow = errors.New("mpfx: integer part overflow")
	ErrFixedPointDivZero  = errors.New("mpfx: division by zero")
	ErrFixedPointDomain   = errors.New("mpfx: undefined or nonterminating operation")
)

// FixedPointManager stores the source's fixed number of integer/fractional
// 32-bit words as a signed, scaled Go integer. Rounding initially points down.
type FixedPointManager struct {
	self                  *FixedPointManager
	intWords, fracWords   uint32
	fracBits              uint
	scale, magnitudeLimit big.Int
	roundUp               bool
	live                  uint32
	closed                bool
}

// FixedPoint is an owned concrete mpfx numeral. Its signed value is
// scaled / 2^(32*fracWords); zero has no negative representation.
type FixedPoint struct {
	self    *FixedPoint
	manager *FixedPointManager
	scaled  big.Int
	closed  bool
}

func NewFixedPointManager(intWords, fracWords uint32) (*FixedPointManager, error) {
	if intWords == 0 || fracWords == 0 || uint64(intWords)+uint64(fracWords) > 1<<20 {
		return nil, ErrFixedPointOwner
	}
	m := &FixedPointManager{intWords: intWords, fracWords: fracWords, fracBits: uint(fracWords) * 32}
	m.self = m
	m.scale.Lsh(big.NewInt(1), m.fracBits)
	m.magnitudeLimit.Lsh(big.NewInt(1), uint(intWords+fracWords)*32)
	return m, nil
}

func (m *FixedPointManager) check(v *FixedPoint) error {
	if m == nil || m.self != m || v == nil || v.self != v || v.manager != m {
		return ErrFixedPointOwner
	}
	if m.closed || v.closed {
		return ErrFixedPointClosed
	}
	return nil
}

func (m *FixedPointManager) pair(a, b *FixedPoint) error {
	if err := m.check(a); err != nil {
		return err
	}
	return m.check(b)
}

func (m *FixedPointManager) New() (*FixedPoint, error) {
	if m == nil || m.self != m {
		return nil, ErrFixedPointOwner
	}
	if m.closed {
		return nil, ErrFixedPointClosed
	}
	v := &FixedPoint{manager: m}
	v.self = v
	m.live++
	return v, nil
}

func (v *FixedPoint) Close() error {
	if v == nil || v.self != v || v.manager == nil || v.manager.self != v.manager {
		return ErrFixedPointOwner
	}
	if v.closed {
		return nil
	}
	v.scaled.SetInt64(0)
	v.closed = true
	v.manager.live--
	return nil
}

func (m *FixedPointManager) Close() error {
	if m == nil || m.self != m {
		return ErrFixedPointOwner
	}
	if m.live != 0 {
		return ErrFixedPointOwner
	}
	m.closed = true
	return nil
}

func (m *FixedPointManager) SetRounding(up bool) error {
	if m == nil || m.self != m {
		return ErrFixedPointOwner
	}
	if m.closed {
		return ErrFixedPointClosed
	}
	m.roundUp = up
	return nil
}

func (m *FixedPointManager) publish(dst *FixedPoint, scaled *big.Int) error {
	var abs big.Int
	abs.Abs(scaled)
	if abs.Cmp(&m.magnitudeLimit) >= 0 {
		return ErrFixedPointOverflow
	}
	dst.scaled.Set(scaled)
	return nil
}

func (m *FixedPointManager) Reset(dst *FixedPoint) error {
	if err := m.check(dst); err != nil {
		return err
	}
	dst.scaled.SetInt64(0)
	return nil
}

func (m *FixedPointManager) SetInt64(dst *FixedPoint, n int64) error {
	if err := m.check(dst); err != nil {
		return err
	}
	var scaled big.Int
	scaled.Mul(big.NewInt(n), &m.scale)
	return m.publish(dst, &scaled)
}

func (m *FixedPointManager) SetUint64(dst *FixedPoint, n uint64) error {
	if err := m.check(dst); err != nil {
		return err
	}
	var scaled, val big.Int
	val.SetUint64(n)
	scaled.Mul(&val, &m.scale)
	return m.publish(dst, &scaled)
}

func (m *FixedPointManager) Copy(dst, src *FixedPoint) error {
	if err := m.pair(dst, src); err != nil {
		return err
	}
	dst.scaled.Set(&src.scaled)
	return nil
}

// roundMagnitude is the source sign/mode rule shared by fixed-point
// multiplication and division: an inexact quotient grows in magnitude when
// its sign differs from the selected plus-infinity direction.
func (m *FixedPointManager) roundMagnitude(num, den *big.Int) (*big.Int, error) {
	if den.Sign() == 0 {
		return nil, ErrFixedPointDivZero
	}
	negative := num.Sign() != den.Sign() && num.Sign() != 0
	var an, ad, q, rem big.Int
	an.Abs(num)
	ad.Abs(den)
	q.QuoRem(&an, &ad, &rem)
	if rem.Sign() != 0 && negative != m.roundUp {
		q.Add(&q, big.NewInt(1))
	}
	if negative {
		q.Neg(&q)
	}
	return &q, nil
}

func (m *FixedPointManager) SetRatio(dst *FixedPoint, num int64, den uint64) error {
	if err := m.check(dst); err != nil {
		return err
	}
	var numerator, denominator big.Int
	numerator.Mul(big.NewInt(num), &m.scale)
	denominator.SetUint64(den)
	var absNum, scaledDen big.Int
	absNum.Abs(&numerator)
	scaledDen.Mul(&denominator, &m.scale)
	if absNum.Cmp(&m.magnitudeLimit) >= 0 || scaledDen.Cmp(&m.magnitudeLimit) >= 0 {
		return ErrFixedPointOverflow // source constructs both integer operands first
	}
	q, err := m.roundMagnitude(&numerator, &denominator)
	if err != nil {
		return err
	}
	return m.publish(dst, q)
}

func (m *FixedPointManager) Compare(a, b *FixedPoint) (int, error) {
	if err := m.pair(a, b); err != nil {
		return 0, err
	}
	return a.scaled.Cmp(&b.scaled), nil
}

func (m *FixedPointManager) IsZero(a *FixedPoint) (bool, error) {
	if err := m.check(a); err != nil {
		return false, err
	}
	return a.scaled.Sign() == 0, nil
}

func (m *FixedPointManager) IsInt(a *FixedPoint) (bool, error) {
	if err := m.check(a); err != nil {
		return false, err
	}
	var rem big.Int
	rem.Rem(&a.scaled, &m.scale)
	return rem.Sign() == 0, nil
}

func (m *FixedPointManager) Neg(a *FixedPoint) error {
	if err := m.check(a); err != nil {
		return err
	}
	a.scaled.Neg(&a.scaled)
	return nil
}

func (m *FixedPointManager) Abs(a *FixedPoint) error {
	if err := m.check(a); err != nil {
		return err
	}
	a.scaled.Abs(&a.scaled)
	return nil
}

func (m *FixedPointManager) Add(a, b, out *FixedPoint) error {
	if err := m.pair(a, b); err != nil {
		return err
	}
	if err := m.check(out); err != nil {
		return err
	}
	var v big.Int
	v.Add(&a.scaled, &b.scaled)
	return m.publish(out, &v)
}

func (m *FixedPointManager) Sub(a, b, out *FixedPoint) error {
	if err := m.pair(a, b); err != nil {
		return err
	}
	if err := m.check(out); err != nil {
		return err
	}
	var v big.Int
	v.Sub(&a.scaled, &b.scaled)
	return m.publish(out, &v)
}

func (m *FixedPointManager) Mul(a, b, out *FixedPoint) error {
	if err := m.pair(a, b); err != nil {
		return err
	}
	if err := m.check(out); err != nil {
		return err
	}
	var product big.Int
	product.Mul(&a.scaled, &b.scaled)
	q, err := m.roundMagnitude(&product, &m.scale)
	if err != nil {
		return err
	}
	return m.publish(out, q)
}

func (m *FixedPointManager) Div(a, b, out *FixedPoint) error {
	if err := m.pair(a, b); err != nil {
		return err
	}
	if err := m.check(out); err != nil {
		return err
	}
	var numerator big.Int
	numerator.Mul(&a.scaled, &m.scale)
	q, err := m.roundMagnitude(&numerator, &b.scaled)
	if err != nil {
		return err
	}
	return m.publish(out, q)
}

// Rational returns an independent exact value for testing and concrete
// subpaving boundary conversions.
func (m *FixedPointManager) Rational(a *FixedPoint) (*big.Rat, error) {
	if err := m.check(a); err != nil {
		return nil, err
	}
	return new(big.Rat).SetFrac(new(big.Int).Set(&a.scaled), new(big.Int).Set(&m.scale)), nil
}
