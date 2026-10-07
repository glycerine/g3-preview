// Source: src/util/bit_util.cpp; src/util/bit_util.h.
package util

import (
	"errors"
	"math"
	"math/bits"
)

var ErrBitBounds = errors.New("utilbits: reached invalid word access")
var ErrBitAssertion = errors.New("utilbits: source assertion")
var ErrUndefinedShift = errors.New("utilbits: source undefined shift")

// Counts retain source uint32 wrap. The caller owns the borrowed slices; these
// routines neither allocate nor clone overlapping ranges. Bounds are guarded at
// each actual read/write, preserving earlier source actions on a late refusal.
func read(a []uint32, i uint32) (uint32, error) {
	if uint64(i) >= uint64(len(a)) {
		return 0, ErrBitBounds
	}
	return a[i], nil
}
func write(a []uint32, i, v uint32) error {
	if uint64(i) >= uint64(len(a)) {
		return ErrBitBounds
	}
	a[i] = v
	return nil
}
func slowMSB(v uint32) (uint32, error) {
	if v == 0 {
		return 0, ErrBitAssertion
	}
	r := uint32(0)
	for v != 1 {
		v >>= 1
		r++
	}
	return r, nil
}
func MSBPosition(v uint32) (uint32, error) {
	if sourceDebug && v == 0 {
		return 0, ErrBitAssertion
	}
	original := v
	r := uint32(0)
	if v > 0xFFFF {
		r = 16
	}
	v >>= r
	shift := uint32(0)
	if v > 0xFF {
		shift = 8
	}
	v >>= shift
	r |= shift
	shift = 0
	if v > 0xF {
		shift = 4
	}
	v >>= shift
	r |= shift
	shift = 0
	if v > 0x3 {
		shift = 2
	}
	v >>= shift
	r |= shift
	r |= v >> 1
	if sourceDebug {
		expected, e := slowMSB(original)
		if e != nil {
			return 0, e
		}
		if r != expected {
			return 0, ErrBitAssertion
		}
	}
	return r, nil
}
func LeadingZerosCore(v uint32) (uint32, error) {
	if v == 0 {
		if sourceDebug {
			return 0, ErrBitAssertion
		}
		return 0, ErrUndefinedShift
	}
	return uint32(bits.LeadingZeros32(v)), nil
}
func TrailingZerosCore(v uint32) (uint32, error) {
	if v == 0 {
		if sourceDebug {
			return 0, ErrBitAssertion
		}
		return 0, ErrUndefinedShift
	}
	return uint32(bits.TrailingZeros32(v)), nil
}

// The non-GNU source branches remain separate concrete engines. The fixed
// reference profile casts uint32 to signed int32 using two's complement. Its
// INT_MIN unary negation is diagnosed at that actual undefined operand; this
// does not reject the same high-bit input in the primary GNU intrinsic route.
func leadingZerosFallbackCore(v uint32) (uint32, error) {
	if sourceDebug && v == 0 {
		return 0, ErrBitAssertion
	}
	p, e := MSBPosition(v)
	return 31 - p, e
}
func trailingZerosFallbackCore(v uint32) (uint32, error) {
	if sourceDebug && v == 0 {
		return 0, ErrBitAssertion
	}
	if v == 0x80000000 {
		return 0, ErrUndefinedShift
	}
	f := float32(v & uint32(-int32(v)))
	// Go fixes both source scalar types at32 bits. Float32bits preserves the
	// source memcpy bit reinterpretation without unsafe or an allocation.
	u := math.Float32bits(f)
	return (u >> 23) - 0x7f, nil
}
func LeadingZeros(n uint32, a []uint32) (uint32, error) {
	r := uint32(0)
	for i := n; i > 0; {
		i--
		v, e := read(a, i)
		if e != nil {
			return 0, e
		}
		if v == 0 {
			r += 32
		} else {
			z, e := LeadingZerosCore(v)
			return r + z, e
		}
	}
	return r, nil
}
func TrailingZeros(n uint32, a []uint32) (uint32, error) {
	r := uint32(0)
	for i := uint32(0); i < n; i++ {
		v, e := read(a, i)
		if e != nil {
			return 0, e
		}
		if v == 0 {
			r += 32
		} else {
			z, e := TrailingZerosCore(v)
			return r + z, e
		}
	}
	return r, nil
}
func Copy(sn uint32, src []uint32, dn uint32, dst []uint32) error {
	n := sn
	if dn < sn {
		n = dn
	}
	for i := uint32(0); i < n; i++ {
		v, e := read(src, i)
		if e != nil {
			return e
		}
		if e = write(dst, i, v); e != nil {
			return e
		}
	}
	for i := n; i < dn; i++ {
		if e := write(dst, i, 0); e != nil {
			return e
		}
	}
	return nil
}
func IsZero(n uint32, a []uint32) (bool, error) {
	for i := uint32(0); i < n; i++ {
		v, e := read(a, i)
		if e != nil {
			return false, e
		}
		if v != 0 {
			return false, nil
		}
	}
	return true, nil
}
func Reset(n uint32, a []uint32) error {
	for i := uint32(0); i < n; i++ {
		if e := write(a, i, 0); e != nil {
			return e
		}
	}
	return nil
}
func ShiftLeft(sn uint32, src []uint32, k, dn uint32, dst []uint32) error {
	if sourceDebug && (sn == 0 || dn == 0 || k == 0) {
		return ErrBitAssertion
	}
	w, b := k/32, k%32
	if w > 0 {
		j, i := sn, sn+w
		if i > dn {
			if j >= i-dn {
				j -= i - dn
			} else {
				j = 0
			}
			i = dn
		} else if i < dn {
			for r := i; r < dn; r++ {
				if e := write(dst, r, 0); e != nil {
					return e
				}
			}
		}
		for j > 0 {
			j--
			i--
			v, e := read(src, j)
			if e != nil {
				return e
			}
			if e = write(dst, i, v); e != nil {
				return e
			}
		}
		for i > 0 {
			i--
			if e := write(dst, i, 0); e != nil {
				return e
			}
		}
		if b > 0 {
			prev := uint32(0)
			for i := w; i < dn; i++ {
				v, e := read(dst, i)
				if e != nil {
					return e
				}
				next := v >> (32 - b)
				if e = write(dst, i, (v<<b)|prev); e != nil {
					return e
				}
				prev = next
			}
		}
	} else {
		prev := uint32(0)
		if sn > dn {
			sn = dn
		}
		for i := uint32(0); i < sn; i++ {
			v, e := read(src, i)
			if e != nil {
				return e
			}
			if b == 0 {
				return ErrUndefinedShift
			}
			next := v >> (32 - b)
			if e = write(dst, i, (v<<b)|prev); e != nil {
				return e
			}
			prev = next
		}
		if dn > sn {
			if e := write(dst, sn, prev); e != nil {
				return e
			}
			for i := sn + 1; i < dn; i++ {
				if e := write(dst, i, 0); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func rightWord(src []uint32, j uint32, dst []uint32, i, b uint32, carry bool) error {
	v, e := read(src, j)
	if e != nil {
		return e
	}
	if e = write(dst, i, v>>b); e != nil {
		return e
	}
	if carry {
		v, e = read(src, j+1)
		if e != nil {
			return e
		}
		if b == 0 {
			return ErrUndefinedShift
		}
		old, e := read(dst, i)
		if e != nil {
			return e
		}
		if e = write(dst, i, old|(v<<(32-b))); e != nil {
			return e
		}
	}
	return nil
}
func ShiftRightSame(n uint32, src []uint32, k uint32, dst []uint32) error {
	d := k / 32
	if d >= n {
		return Reset(n, dst)
	}
	b := k % 32
	newN := n - d
	if d > 0 {
		i, j := uint32(0), d
		if b != 0 {
			for ; i < newN-1; i, j = i+1, j+1 {
				if e := rightWord(src, j, dst, i, b, true); e != nil {
					return e
				}
			}
			if e := rightWord(src, j, dst, i, b, false); e != nil {
				return e
			}
		} else {
			for ; i < newN; i, j = i+1, j+1 {
				v, e := read(src, j)
				if e != nil {
					return e
				}
				if e = write(dst, i, v); e != nil {
					return e
				}
			}
		}
		for i := newN; i < n; i++ {
			if e := write(dst, i, 0); e != nil {
				return e
			}
		}
	} else {
		if sourceDebug && b == 0 {
			return ErrBitAssertion
		}
		i := uint32(0)
		for ; i < newN-1; i++ {
			if e := rightWord(src, i, dst, i, b, true); e != nil {
				return e
			}
		}
		if e := rightWord(src, i, dst, i, b, false); e != nil {
			return e
		}
	}
	return nil
}
func ShiftRight(sn uint32, src []uint32, k, dn uint32, dst []uint32) error {
	d := k / 32
	if d >= sn {
		return Reset(dn, dst)
	}
	b, newN := k%32, sn-d
	if d > 0 {
		i, j := uint32(0), d
		if b != 0 {
			n := newN
			if n > dn {
				n = dn
			}
			for ; i < n-1; i, j = i+1, j+1 {
				if e := rightWord(src, j, dst, i, b, true); e != nil {
					return e
				}
			}
			if e := rightWord(src, j, dst, i, b, newN > dn); e != nil {
				return e
			}
		} else {
			if newN > dn {
				newN = dn
			}
			for ; i < newN; i, j = i+1, j+1 {
				v, e := read(src, j)
				if e != nil {
					return e
				}
				if e = write(dst, i, v); e != nil {
					return e
				}
			}
		}
	} else {
		if sourceDebug && b == 0 {
			return ErrBitAssertion
		}
		n := newN
		if n > dn {
			n = dn
		}
		i := uint32(0)
		for ; i < n-1; i++ {
			if e := rightWord(src, i, dst, i, b, true); e != nil {
				return e
			}
		}
		if e := rightWord(src, i, dst, i, b, newN > dn); e != nil {
			return e
		}
	}
	for i := newN; i < dn; i++ {
		if e := write(dst, i, 0); e != nil {
			return e
		}
	}
	return nil
}
func HasOneAtFirstKBits(n uint32, a []uint32, k uint32) (bool, error) {
	if sourceDebug && n == 0 {
		return false, ErrBitAssertion
	}
	w := k / 32
	if w > n {
		w = n
	}
	for i := uint32(0); i < w; i++ {
		v, e := read(a, i)
		if e != nil {
			return false, e
		}
		if v != 0 {
			return true, nil
		}
	}
	if w < n {
		v, e := read(a, w)
		if e != nil {
			return false, e
		}
		return v&((uint32(1)<<(k%32))-1) != 0, nil
	}
	return false, nil
}
func Increment(n uint32, a []uint32) (bool, error) {
	for i := uint32(0); i < n; i++ {
		v, e := read(a, i)
		if e != nil {
			return false, e
		}
		v++
		if e = write(a, i, v); e != nil {
			return false, e
		}
		if v != 0 {
			return true, nil
		}
	}
	return false, nil
}
func Decrement(n uint32, a []uint32) (bool, error) {
	for i := uint32(0); i < n; i++ {
		v, e := read(a, i)
		if e != nil {
			return false, e
		}
		v--
		if e = write(a, i, v); e != nil {
			return false, e
		}
		if v != ^uint32(0) {
			return true, nil
		}
	}
	return false, nil
}
func Less(n uint32, a, b []uint32) (bool, error) {
	for i := n; i > 0; {
		i--
		x, e := read(a, i)
		if e != nil {
			return false, e
		}
		y, e := read(b, i)
		if e != nil {
			return false, e
		}
		if x < y {
			return true, nil
		}
		if x > y {
			return false, nil
		}
	}
	return false, nil
}
func Add(n uint32, a, b, c []uint32) (bool, error) {
	k := uint32(0)
	for j := uint32(0); j < n; j++ {
		x, e := read(a, j)
		if e != nil {
			return false, e
		}
		y, e := read(b, j)
		if e != nil {
			return false, e
		}
		r := x + y
		c1 := r < x
		v := r + k
		if e = write(c, j, v); e != nil {
			return false, e
		}
		c2 := v < r
		k = 0
		if c1 || c2 {
			k = 1
		}
	}
	return k == 0, nil
}
