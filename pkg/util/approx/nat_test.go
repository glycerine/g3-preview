package approx

// Source: src/util/approx_nat.h:26-43; src/util/approx_nat.cpp:22-61.

import (
	"math/rand/v2"
	"testing"
)

func TestNatBoundariesComparisonsAndString(t *testing.T) {
	values := []uint32{0, 1, 2, natLimit - 1, natLimit, natLimit + 1, 1 << 31, ^uint32(0)}
	for _, a := range values {
		x := NewNat(a)
		want := a
		if a > natLimit {
			want = huge
		}
		if x.Value() != want || x.IsHuge() != (a > natLimit) {
			t.Fatal(a, x)
		}
		for _, b := range values {
			y := NewNat(b)
			if x.LessUint32(b) != (a <= natLimit && a < b) {
				t.Fatal("uint32 comparison", a, b)
			}
			if x.Less(y) != (a <= natLimit && b <= natLimit && a < b) {
				t.Fatal("Nat comparison", a, b)
			}
		}
	}
	if got := NewNat(63).String(); got != "63" {
		t.Fatal(got)
	}
	if got := NewNat(natLimit + 1).String(); got != "[huge]" {
		t.Fatal(got)
	}
}

func TestNatArithmetic(t *testing.T) {
	check := func(a, b uint32) {
		t.Helper()
		for _, multiply := range []bool{false, true} {
			x := NewNat(a)
			value := uint64(a) + uint64(b)
			if multiply {
				value = uint64(a) * uint64(b)
			}
			want := uint32(value)
			if uint64(a) > uint64(natLimit) || (!multiply && b > natLimit) || value > uint64(natLimit) {
				want = huge
			}
			if multiply {
				x.MulUint32(b)
			} else {
				x.AddUint32(b)
			}
			if x.Value() != want {
				t.Fatal(a, b, multiply, x.Value(), want)
			}
		}
	}
	values := []uint32{0, 1, 2, 32767, 32768, 65535, 65536, natLimit - 1, natLimit, natLimit + 1, 1 << 31, ^uint32(0)}
	for _, a := range values {
		for _, b := range values {
			check(a, b)
		}
	}
	random := rand.New(rand.NewChaCha8([32]byte{17}))
	for range 10000 {
		check(random.Uint32(), random.Uint32())
		check(random.Uint32()>>2, random.Uint32()>>2)
	}

	finite, sentinel := NewNat(0), NewNat(natLimit+1)
	finite.Mul(sentinel)
	if finite.IsHuge() {
		t.Fatal("finite zero times huge must remain zero")
	}
	sentinel.MulUint32(0)
	if !sentinel.IsHuge() {
		t.Fatal("huge times zero must remain huge")
	}
	finite.Add(sentinel)
	if !finite.IsHuge() {
		t.Fatal("finite plus huge must be huge")
	}
}
