// Source: src/test/bits.cpp; src/util/bit_util.h; src/util/bit_util.cpp.
package util

import (
	"math/big"
	"testing"
)

func requireBits(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func checkWords(t *testing.T, actual, expected []uint32) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatal("word count")
	}
	for i := range actual {
		if actual[i] != expected[i] {
			t.Fatalf("word %d: %#x != %#x", i, actual[i], expected[i])
		}
	}
}

// This retains every assertion and call order in tst_shl. Console/TRACE output
// is omitted; the words and the full sample count remain observable checks.
func sourceShiftLeft(t *testing.T, src []uint32, k uint32, expected []uint32) {
	t.Helper()
	n, dn := uint32(len(src)), uint32(len(expected))
	actual := make([]uint32, dn)
	for i := range actual {
		actual[i] = 0xAAAAAAAA
	}
	for size := uint32(1); size <= dn; size++ {
		requireBits(t, ShiftLeft(n, src, k, size, actual))
		low, err := HasOneAtFirstKBits(size, actual, k)
		requireBits(t, err)
		if low {
			t.Fatal("source ENSURE: shifted low bits")
		}
		checkWords(t, actual[:size], expected[:size])
		if size == n {
			nz1, err := LeadingZeros(size, src)
			requireBits(t, err)
			zero, err := IsZero(size, src)
			requireBits(t, err)
			if nz1 >= k && !zero {
				nz2, err := LeadingZeros(size, actual)
				requireBits(t, err)
				if nz1-k != nz2 {
					t.Fatal("source UNREACHABLE: leading zeros")
				}
			}
		}
		if size >= n+k/32+1 {
			inverse := make([]uint32, size)
			for i := range inverse {
				inverse[i] = 0xAAAAAAAA
			}
			requireBits(t, ShiftRightSame(size, actual, k, inverse))
			checkWords(t, inverse[:n], src)
		}
	}
	requireBits(t, Copy(n, src, dn, actual))
	for i := uint32(0); i < k; i++ {
		requireBits(t, ShiftLeft(dn, actual, 1, dn, actual))
	}
	checkWords(t, actual, expected)
	if n <= dn {
		requireBits(t, ShiftLeft(n, src, k, n, src))
		checkWords(t, src, actual[:n])
	}
}

func TestBitsRegisteredFixedSourceAssertions(t *testing.T) {
	zero := []uint32{0, 0}
	dst := []uint32{0xAAAAAAAA, 0xAAAAAAAA}
	requireBits(t, ShiftRightSame(2, zero, 1, dst))
	checkWords(t, dst, zero)
	cases := []struct {
		src []uint32
		k   uint32
		dst []uint32
	}{
		{[]uint32{0, 2}, 10, []uint32{0, 2 << 10}},
		{[]uint32{2, 0}, 42, []uint32{0, 2 << 10}},
		{[]uint32{0, 0}, 1, []uint32{0, 0, 0}},
		{[]uint32{0x80000009, 5}, 1, []uint32{18, 11}},
		{[]uint32{0x80000009, 0x80000005}, 1, []uint32{18, 11}},
		{[]uint32{0x80000009, 0x80000005}, 1, []uint32{18, 11, 1}},
		{[]uint32{0x80000009, 0x80000005}, 33, []uint32{0, 18, 11}},
		{[]uint32{0x80000009, 0x80000005}, 33, []uint32{0, 18, 11, 1}},
		{[]uint32{0xFFFFFFFF, 0xFFFFFFFF}, 4, []uint32{0xFFFFFFF0, 0xFFFFFFFF}},
	}
	for _, c := range cases {
		sourceShiftLeft(t, c.src, c.k, c.dst)
	}
}

// The source uses host libc rand(). This hermetic 31-bit draw adapter keeps its
// draw order, modulo choices, zero-to-one corrections and all100000 trials.
// math/big is test-only and independent of the ported production algorithms.
func TestBitsRegisteredFullRandomCampaign(t *testing.T) {
	state := uint32(1)
	draw := func() uint32 { state = state*1103515245 + 12345; return (state >> 1) & 0x7FFFFFFF }
	base := new(big.Int).Lsh(big.NewInt(1), 32)
	for trial := 0; trial < 100000; trial++ {
		n := draw() % 4
		if n == 0 {
			n = 1
		}
		k := draw() % 100
		if k == 0 {
			k = 1
		}
		src := make([]uint32, n)
		for i := range src {
			src[i] = draw()
		}
		value := new(big.Int)
		for i := len(src) - 1; i >= 0; i-- {
			value.Lsh(value, 32)
			value.Add(value, new(big.Int).SetUint64(uint64(src[i])))
		}
		value.Lsh(value, uint(k))
		var dst []uint32
		for value.Sign() != 0 {
			digit := new(big.Int).Mod(value, base)
			if !digit.IsUint64() || digit.Uint64() >= uint64(^uint32(0)) {
				t.Fatal("source ENSURE: unsigned digit bound")
			}
			dst = append(dst, uint32(digit.Uint64()))
			value.Div(value, base)
		}
		for len(dst) < len(src) {
			dst = append(dst, 0)
		}
		dst = append(dst, 0)
		for i := uint32(0); i < k/32; i++ {
			dst = append(dst, 0)
		}
		sourceShiftLeft(t, src, k, dst)
	}
}

func TestBitsWholeUtilityArithmeticAndBitPositions(t *testing.T) {
	for bit := uint32(0); bit < 32; bit++ {
		v := uint32(1) << bit
		p, e := MSBPosition(v)
		requireBits(t, e)
		if p != bit {
			t.Fatal("MSB")
		}
		p, e = LeadingZerosCore(v)
		requireBits(t, e)
		if p != 31-bit {
			t.Fatal("leading")
		}
		p, e = TrailingZerosCore(v)
		requireBits(t, e)
		if p != bit {
			t.Fatal("trailing")
		}
	}
	a := []uint32{^uint32(0), 0}
	b := []uint32{1, 0}
	c := make([]uint32, 2)
	ok, e := Add(2, a, b, c)
	requireBits(t, e)
	if !ok {
		t.Fatal("carry")
	}
	checkWords(t, c, []uint32{0, 1})
	ok, e = Less(2, a, c)
	requireBits(t, e)
	if !ok {
		t.Fatal("less")
	}
	ok, e = Increment(2, a)
	requireBits(t, e)
	if !ok {
		t.Fatal("increment")
	}
	checkWords(t, a, c)
	ok, e = Decrement(2, a)
	requireBits(t, e)
	if !ok {
		t.Fatal("decrement")
	}
	checkWords(t, a, []uint32{^uint32(0), 0})
	requireBits(t, Reset(2, a))
	checkWords(t, a, []uint32{0, 0})
	ok, e = Decrement(2, a)
	requireBits(t, e)
	if ok {
		t.Fatal("underflow")
	}
	ok, e = Increment(2, a)
	requireBits(t, e)
	if ok {
		t.Fatal("overflow")
	}
	n, e := TrailingZeros(2, []uint32{0, 8})
	requireBits(t, e)
	if n != 35 {
		t.Fatal("trailing vector")
	}
	d := make([]uint32, 3)
	requireBits(t, ShiftRight(2, []uint32{0, 2}, 33, 3, d))
	checkWords(t, d, []uint32{1, 0, 0})
}
