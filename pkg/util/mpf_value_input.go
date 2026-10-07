// Source: src/util/mpf.cpp:69-106,114-181; src/util/mpf.h:76,78-79.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// SetInt32 is the literal machine-int overload. In particular it normalizes the
// uint32 word before removing the implicit leading bit; no rounding mode is
// invented for the source-disabled INT_MAX-to-single assertion.
func (m *ArbitraryFloatManager) SetInt32(out *ArbitraryFloat, ebits, sbits uint32, value int32) error {
	return m.SetInt32WithReceipt(out, ebits, sbits, value, nil)
}

func (m *ArbitraryFloatManager) SetInt32WithReceipt(out *ArbitraryFloat, ebits, sbits uint32, value int32, receipt *ArbitraryFloatValueInputReceipt) (cause error) {
	if e := m.valueGuard(out); e != nil {
		return e
	}
	context, e := m.beginValueInput(receipt)
	if e != nil {
		return e
	}
	defer func() { m.finishValueInput(cause) }()
	out.assignSign(false)
	out.assignEbits(ebits)
	out.assignSbits(sbits)
	if value == 0 {
		return m.MkZero(ebits, sbits, false, out)
	}
	uval := uint32(value)
	if value < 0 {
		out.assignSign(true)
		if value == (-1 << 31) {
			uval = 0x80000000
		} else {
			uval = uint32(-value)
		}
	}
	out.exponent = 31
	for uval&0x80000000 == 0 {
		uval <<= 1
		out.exponent--
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	if e := m.integer.setUint32WithAllocationContext(&out.significand, uval&0x7fffffff, context); e != nil {
		return e
	}
	if sbits > 31 {
		return m.integer.mul2kCoreWithAllocationContext(&out.significand, sbits-32, context)
	}
	return m.integer.MachineDiv2kInPlace(&out.significand, 32-sbits)
}

// SetDouble copies the native IEEE bits, publishes source fields, then clamps
// the exponent and shifts the stored fraction exactly as the source overload.
func (m *ArbitraryFloatManager) SetDouble(out *ArbitraryFloat, ebits, sbits uint32, value float64) error {
	return m.SetDoubleWithReceipt(out, ebits, sbits, value, nil)
}

func (m *ArbitraryFloatManager) SetDoubleWithReceipt(out *ArbitraryFloat, ebits, sbits uint32, value float64, receipt *ArbitraryFloatValueInputReceipt) (cause error) {
	if e := m.valueGuard(out); e != nil {
		return e
	}
	context, e := m.beginValueInput(receipt)
	if e != nil {
		return e
	}
	defer func() { m.finishValueInput(cause) }()
	raw := math.Float64bits(value)
	sign := raw>>63 != 0
	exp := int64((raw&0x7ff0000000000000)>>52) - 1023
	sig := raw & 0x000fffffffffffff
	if modularSourceDebug && (exp < -1023 || exp > 1024) {
		return ErrIntegerAssertion
	}
	out.assignEbits(ebits)
	out.assignSbits(sbits)
	out.assignSign(sign)
	if e := m.assignNativeInputExponent(out, ebits, exp); e != nil {
		return e
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	if e := m.integer.setUint64WithAllocationContext(&out.significand, sig, context); e != nil {
		return e
	}
	if sbits < 53 {
		return m.integer.MachineDiv2kInPlace(&out.significand, 53-sbits)
	}
	if sbits > 53 {
		return m.integer.mul2kCoreWithAllocationContext(&out.significand, sbits-53, context)
	}
	return nil
}

func (m *ArbitraryFloatManager) SetFloat(out *ArbitraryFloat, ebits, sbits uint32, value float32) error {
	return m.SetFloatWithReceipt(out, ebits, sbits, value, nil)
}

func (m *ArbitraryFloatManager) SetFloatWithReceipt(out *ArbitraryFloat, ebits, sbits uint32, value float32, receipt *ArbitraryFloatValueInputReceipt) (cause error) {
	if e := m.valueGuard(out); e != nil {
		return e
	}
	context, e := m.beginValueInput(receipt)
	if e != nil {
		return e
	}
	defer func() { m.finishValueInput(cause) }()
	raw := math.Float32bits(value)
	sign := raw>>31 != 0
	exp := int64(int32((raw&0x7f800000)>>23) - 127)
	sig := raw & 0x007fffff
	if modularSourceDebug && (exp < -127 || exp > 128) {
		return ErrIntegerAssertion
	}
	out.assignEbits(ebits)
	out.assignSbits(sbits)
	out.assignSign(sign)
	if e := m.assignNativeInputExponent(out, ebits, exp); e != nil {
		return e
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	if e := m.integer.setUint32WithAllocationContext(&out.significand, sig, context); e != nil {
		return e
	}
	if sbits < 24 {
		return m.integer.MachineDiv2kInPlace(&out.significand, 24-sbits)
	}
	if sbits > 24 {
		return m.integer.mul2kCoreWithAllocationContext(&out.significand, sbits-24, context)
	}
	return nil
}

// The source predicates use the original uint32 argument at this expression,
// before any significand SET. ebits=0/64+ reach an undefined signed-shift or
// subtraction domain; the Go guard preserves already-published format/sign.
func (m *ArbitraryFloatManager) assignNativeInputExponent(out *ArbitraryFloat, ebits uint32, exp int64) error {
	if ebits == 0 || ebits >= 64 {
		return arena.ErrBounds
	}
	upper := int64(1) << (ebits - 1)
	if exp <= -(upper - 1) {
		bottom, e := m.bottomExponent(ebits)
		if e != nil {
			return e
		}
		out.exponent = bottom
	} else if exp >= upper {
		top, e := m.topExponent(ebits)
		if e != nil {
			return e
		}
		out.exponent = top
	} else {
		out.exponent = exp
	}
	return nil
}
