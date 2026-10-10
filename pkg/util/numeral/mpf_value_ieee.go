// Source: src/util/mpf.cpp:1712-1769; src/util/mpf.h:149-150.
package numeral

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// ToDouble follows the source uint64 significand read, signed shifts, exponent
// classification, and memcpy bit interpretation. Cached bounds keep their
// existing exact persistent Power owner; this body declares no source NUM.
func (m *ArbitraryFloatManager) ToDouble(x *ArbitraryFloat) (float64, error) {
	if e := m.valueGuard(x); e != nil {
		return 0, e
	}
	if modularSourceDebug && (x.EBits() > 11 || x.SBits() > 53) {
		return 0, ErrIntegerAssertion
	}
	sig, e := m.integer.Uint64(&x.significand)
	if e != nil {
		return 0, e
	}
	shift := uint32(53) - x.SBits()
	// C++11 CWG1457 permits a nonnegative signed left shift whose result
	// fits uint64, followed by the target two's-complement signed conversion.
	// A negative initial signed operand or an oversized unsigned result is
	// outside that defined source domain.
	if shift >= 64 || sig > uint64(1)<<63-1 || sig > ^uint64(0)>>shift {
		return 0, arena.ErrBounds
	}
	sig <<= shift
	top, e := m.hasTopExponent(x)
	if e != nil {
		return 0, e
	}
	var exp int64
	if top {
		exp = 1024
	} else {
		bottom, e := m.hasBottomExponent(x)
		if e != nil {
			return 0, e
		}
		if bottom {
			exp = -1023
		} else {
			exp = x.exponent
		}
	}
	// The source signed addition and exp<<52 must both be defined.
	if exp > int64(^uint64(0)>>1)-1023 {
		return 0, arena.ErrBounds
	}
	exp += 1023
	if exp < 0 || exp > 4095 {
		return 0, arena.ErrBounds
	}
	raw := uint64(exp)<<52 | sig
	if x.Sign() {
		raw |= 0x8000000000000000
	}
	return math.Float64frombits(raw), nil
}

// ToFloat retains the source uint32 conversions/addition/shift and final memcpy
// interpretation. In particular negative normal exponents convert to uint32
// before the unsigned bias addition.
func (m *ArbitraryFloatManager) ToFloat(x *ArbitraryFloat) (float32, error) {
	if e := m.valueGuard(x); e != nil {
		return 0, e
	}
	if modularSourceDebug && (x.EBits() > 8 || x.SBits() > 24) {
		return 0, ErrIntegerAssertion
	}
	q, e := m.integer.Uint64(&x.significand)
	if e != nil {
		return 0, e
	}
	if modularSourceDebug && q >= 1<<32 {
		return 0, ErrIntegerAssertion
	}
	sig := uint32(q)
	shift := uint32(24) - x.SBits()
	if shift >= 32 {
		return 0, arena.ErrBounds
	}
	sig <<= shift
	top, e := m.hasTopExponent(x)
	if e != nil {
		return 0, e
	}
	var exp uint32
	if top {
		exp = 128
	} else {
		bottom, e := m.hasBottomExponent(x)
		if e != nil {
			return 0, e
		}
		if bottom {
			exp = uint32(0xffffff81)
		} else {
			if modularSourceDebug && x.exponent >= 1<<32 {
				return 0, ErrIntegerAssertion
			}
			exp = uint32(x.exponent)
		}
	}
	exp += 127
	raw := exp<<23 | sig
	if x.Sign() {
		raw |= 0x80000000
	}
	return math.Float32frombits(raw), nil
}
