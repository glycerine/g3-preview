// Source: src/util/bit_util.cpp; src/util/bit_util.h. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2012-09-11.
package util

import (
	"errors"
	"math/bits"
	"testing"
)

func TestBitsFiniteScalarAndCarryOracle(t *testing.T) {
	for v := uint32(1); v <= 65535; v++ {
		p, e := MSBPosition(v)
		requireBits(t, e)
		if p != uint32(bits.Len32(v)-1) {
			t.Fatal("MSB", v, p)
		}
		l, e := LeadingZerosCore(v)
		requireBits(t, e)
		z, e := TrailingZerosCore(v)
		requireBits(t, e)
		if l != uint32(bits.LeadingZeros32(v)) || z != uint32(bits.TrailingZeros32(v)) {
			t.Fatal("scalar zeros", v, l, z)
		}
	}
	values := []uint64{0, 1, 0x7FFFFFFF, 0x80000000, 0xFFFFFFFF, 0x100000000, 0xFFFFFFFF00000000, 0xFFFFFFFFFFFFFFFF}
	for _, x := range values {
		for _, y := range values {
			a, b := []uint32{uint32(x), uint32(x >> 32)}, []uint32{uint32(y), uint32(y >> 32)}
			want := x + y
			c := make([]uint32, 2)
			ok, e := Add(2, a, b, c)
			requireBits(t, e)
			if ok != (want >= x) {
				t.Fatal("carry-out", x, y, ok)
			}
			checkWords(t, c, []uint32{uint32(want), uint32(want >> 32)})
			less, e := Less(2, a, b)
			requireBits(t, e)
			if less != (x < y) {
				t.Fatal("order", x, y)
			}
			// Source accepts exact output/input identity; no input snapshot is
			// allocated by the production routines.
			ok, e = Add(2, a, b, a)
			requireBits(t, e)
			if ok != (want >= x) {
				t.Fatal("aliased carry")
			}
			checkWords(t, a, c)
		}
		a := []uint32{uint32(x), uint32(x >> 32)}
		ok, e := Increment(2, a)
		requireBits(t, e)
		want := x + 1
		if ok != (x != ^uint64(0)) {
			t.Fatal("increment status", x)
		}
		checkWords(t, a, []uint32{uint32(want), uint32(want >> 32)})
		a = []uint32{uint32(x), uint32(x >> 32)}
		ok, e = Decrement(2, a)
		requireBits(t, e)
		want = x - 1
		if ok != (x != 0) {
			t.Fatal("decrement status", x)
		}
		checkWords(t, a, []uint32{uint32(want), uint32(want >> 32)})
	}
	for n := uint32(0); n <= 6; n++ {
		a := make([]uint32, n)
		l, e := LeadingZeros(n, a)
		requireBits(t, e)
		z, e := TrailingZeros(n, a)
		requireBits(t, e)
		if l != n*32 || z != n*32 {
			t.Fatal("zero-vector lengths", n, l, z)
		}
	}
}

func bitOracle(src []uint32, k uint32, dn int, left bool) []uint32 {
	dst := make([]uint32, dn)
	for i, w := range src {
		for b := 0; b < 32; b++ {
			if w&(uint32(1)<<b) == 0 {
				continue
			}
			position := i*32 + b
			if left {
				position += int(k)
			} else {
				position -= int(k)
			}
			if position >= 0 && position < dn*32 {
				dst[position/32] |= uint32(1) << (position % 32)
			}
		}
	}
	return dst
}

func TestBitsFiniteWholeShiftMatrix(t *testing.T) {
	for sn := 1; sn <= 4; sn++ {
		for dn := 1; dn <= 6; dn++ {
			for k := uint32(1); k <= 159; k++ {
				src := make([]uint32, sn)
				for i := range src {
					src[i] = uint32(0xD35160A7) ^ (uint32(i) * 0x9E3779B9)
				}
				left, right := make([]uint32, dn), make([]uint32, dn)
				requireBits(t, ShiftLeft(uint32(sn), src, k, uint32(dn), left))
				checkWords(t, left, bitOracle(src, k, dn, true))
				requireBits(t, ShiftRight(uint32(sn), src, k, uint32(dn), right))
				checkWords(t, right, bitOracle(src, k, dn, false))
				inPlace := append([]uint32(nil), src...)
				requireBits(t, ShiftRightSame(uint32(sn), inPlace, k, inPlace))
				checkWords(t, inPlace, bitOracle(src, k, sn, false))
			}
		}
	}
}

func TestBitsSourceOverlapAndReachedAccess(t *testing.T) {
	a := []uint32{1, 2, 3, 4}
	requireBits(t, Copy(3, a[:3], 3, a[1:]))
	checkWords(t, a, []uint32{1, 1, 1, 1})
	// The source writes dst[i] before reading src[j+1]. This supported pointer
	// overlap therefore observes the just-written source word, rather than a
	// hidden input clone or an arithmetic snapshot of the original words.
	a = []uint32{6, 17, 29, 41}
	requireBits(t, ShiftRight(4, a, 1, 3, a[1:]))
	checkWords(t, a[1:], []uint32{0x80000003, 0xC0000001, 0x60000000})
	z, e := IsZero(9, []uint32{1})
	requireBits(t, e)
	if z {
		t.Fatal("early nonzero")
	}
	n, e := TrailingZeros(9, []uint32{1})
	requireBits(t, e)
	if n != 0 {
		t.Fatal("unread trailing tail")
	}
	a = []uint32{0}
	ok, e := Increment(9, a)
	requireBits(t, e)
	if !ok || a[0] != 1 {
		t.Fatal("early carry")
	}
	c := []uint32{99}
	_, e = Add(2, []uint32{1}, []uint32{2}, c)
	if !errors.Is(e, ErrBitBounds) || c[0] != 3 {
		t.Fatal("late input guard must retain first publication", c, e)
	}
	zero := []uint32{0xAABBCCDD}
	requireBits(t, ShiftRight(1, nil, 32, 1, zero))
	if zero[0] != 0 {
		t.Fatal("large shift leaves source unread")
	}
}

func TestBitsSourceZeroShiftBoundary(t *testing.T) {
	a := []uint32{7}
	e := ShiftRightSame(1, a, 0, a)
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) {
			t.Fatal(e)
		}
	} else {
		requireBits(t, e)
		if a[0] != 7 {
			t.Fatal("release one-word zero shift")
		}
	}
	a = []uint32{7, 9}
	e = ShiftRightSame(2, a, 0, a)
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) {
			t.Fatal(e)
		}
	} else {
		if !errors.Is(e, ErrUndefinedShift) {
			t.Fatal(e)
		}
	}
	if a[0] != 7 || a[1] != 9 {
		t.Fatal("invalid shift changed words")
	}
}

func TestBitsSourceNonGNUScalarBranches(t *testing.T) {
	for v := uint32(1); v <= 65535; v++ {
		l, e := leadingZerosFallbackCore(v)
		requireBits(t, e)
		z, e := trailingZerosFallbackCore(v)
		requireBits(t, e)
		if l != uint32(bits.LeadingZeros32(v)) || z != uint32(bits.TrailingZeros32(v)) {
			t.Fatal("source fallback", v, l, z)
		}
	}
	for _, v := range []uint32{0x7FFFFFFF, 0x80000001, 0x80000002, 0xFFFFFFFC, 0xFFFFFFFF} {
		z, e := trailingZerosFallbackCore(v)
		requireBits(t, e)
		if z != uint32(bits.TrailingZeros32(v)) {
			t.Fatal("source signed conversion", v, z)
		}
	}
	// The fallback's unary minus of signed INT_MIN is undefined in C++.
	// Diagnose that reached operand; the primary GNU intrinsic remains defined.
	_, e := trailingZerosFallbackCore(0x80000000)
	if !errors.Is(e, ErrUndefinedShift) {
		t.Fatal("source signed overflow guard", e)
	}
	z, e := TrailingZerosCore(0x80000000)
	requireBits(t, e)
	if z != 31 {
		t.Fatal("GNU high bit", z)
	}
}

// A zero shift is undefined only when the source actually executes the
// 32-bit shift expression. Empty release loops have no such expression.
func TestBitsSourceLeftZeroShiftReachedOperand(t *testing.T) {
	dst := []uint32{99, 88}
	e := ShiftLeft(0, nil, 0, 2, dst)
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) {
			t.Fatal(e)
		}
		checkWords(t, dst, []uint32{99, 88})
	} else {
		requireBits(t, e)
		checkWords(t, dst, []uint32{0, 0})
		requireBits(t, ShiftLeft(0, nil, 0, 0, nil))
		if e = ShiftLeft(1, nil, 0, 1, dst); !errors.Is(e, ErrBitBounds) {
			t.Fatal("source reads operand before undefined shift", e)
		}
		dst[0] = 77
		if e = ShiftLeft(1, []uint32{7}, 0, 1, dst); !errors.Is(e, ErrUndefinedShift) || dst[0] != 77 {
			t.Fatal("reached undefined shift precedes destination write", e, dst)
		}
	}
}

func TestBitsSourceRightZeroShiftReachedOperand(t *testing.T) {
	dst := []uint32{99, 88}
	e := ShiftRightSame(2, []uint32{7}, 0, dst)
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) {
			t.Fatal(e)
		}
		checkWords(t, dst, []uint32{99, 88})
	} else {
		if !errors.Is(e, ErrBitBounds) {
			t.Fatal("read next operand before undefined shift", e)
		}
		checkWords(t, dst, []uint32{7, 88})
	}
}
