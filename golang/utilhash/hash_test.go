// Licensed under the MIT license.
package utilhash

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed test_vectors/source_hash_random.json
var sourceVectorsJSON []byte

type vectors struct {
	Uint   []struct{ X, Want uint32 }
	Uint64 []struct {
		X    uint64
		Want uint32
	}
	String, Composite, Random, RandomZero []uint32
	Combine, Mix, Pair                    uint32
	ShuffleZero                           []int32
}

func sourceVectors(t *testing.T) vectors {
	t.Helper()
	var v vectors
	if err := json.Unmarshal(sourceVectorsJSON, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// Constants recorded by compiling this checkout's hash.cpp/hash.h and util.h
// with GCC (32-bit unsigned, little-endian words, signed char). Tests need no C++.
func TestSourceHashVectors(t *testing.T) {
	vec := sourceVectors(t)
	for _, v := range vec.Uint {
		if g := Uint(v.X); g != v.Want {
			t.Fatalf("Uint(%d)=%d want %d", v.X, g, v.Want)
		}
	}
	for _, v := range vec.Uint64 {
		if g := Uint64(v.X); g != v.Want {
			t.Fatalf("Uint64(%d)=%d want %d", v.X, g, v.Want)
		}
	}
	if Combine(0xffffffff, 1234567) != vec.Combine || Mix(0xffffffff, 1234567, 9876543) != vec.Mix {
		t.Fatal("combine/mix source vectors")
	}
	if UintPair(0xffffffff, 1234567) != vec.Pair {
		t.Fatal("pair source vector")
	}
	want := vec.String
	var bytes [25]byte
	for i := range bytes {
		bytes[i] = byte(i*19 + 127)
	}
	for n, w := range want {
		if g := String(string(bytes[:n]), 123); g != w {
			t.Fatalf("String length %d=%d want %d", n, g, w)
		}
	}
	cw := vec.Composite
	for i, w := range cw {
		if g := Composite(17, uint32(i+1), func(i uint32) uint32 { return i*1234567 + 11 }); g != w {
			t.Fatalf("Composite n=%d got %d want %d", i+1, g, w)
		}
	}
}
func TestSourceRandomVectors(t *testing.T) {
	r := NewRandom(0xffffffff)
	for i, w := range sourceVectors(t).Random {
		if g := r.Next(); g != w {
			t.Fatalf("step %d=%d want %d", i, g, w)
		}
	}
	r.SetSeed(0xffffffff)
	if r.Bounded(7) != 0 {
		t.Fatal("bounded modulo")
	}
	if RandomMax != 32767 {
		t.Fatal("max")
	}
}
func TestShuffleTraversal(t *testing.T) {
	r := NewRandom(0)
	v := []int32{0, 1, 2, 3, 4}
	ShuffleInt32(v, r)
	want := sourceVectors(t).ShuffleZero
	for i := range v {
		if v[i] != want[i] {
			t.Fatalf("shuffle %v want %v", v, want)
		}
	}
}
func TestSourceZeroSeed(t *testing.T) {
	var r Random
	for i, w := range sourceVectors(t).RandomZero {
		if r.Next() != w {
			t.Fatal(i)
		}
	}
}
func TestCompositeVisitOrder(t *testing.T) {
	for n := uint32(1); n <= 10; n++ {
		var got []uint32
		Composite(17, n, func(i uint32) uint32 { got = append(got, i); return 0 })
		var want []uint32
		if n <= 3 {
			for i := uint32(0); i < n; i++ {
				want = append(want, i)
			}
		} else {
			for i := n; i > 0; i-- {
				want = append(want, i-1)
			}
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("n=%d visit %v want %v", n, got, want)
			}
		}
	}
}
func TestWordReaderSourceVectors(t *testing.T) {
	vec := sourceVectors(t)
	var words [7]uint32
	for i := 0; i < 25; i++ {
		words[i/4] |= uint32(byte(i*19+127)) << uint((i%4)*8)
	}
	for n, w := range vec.String {
		if h := WordReader(uint32(n), 123, func(i uint32) uint32 { return words[i] }); h != w {
			t.Fatal(n, h, w)
		}
	}
	if n := testing.AllocsPerRun(100, func() { WordReader(25, 123, func(i uint32) uint32 { return words[i] }) }); n != 0 {
		t.Fatal("word reader allocations", n)
	}
}
func TestConcreteShuffleSourceDrawOrder(t *testing.T) {
	want := sourceVectors(t).ShuffleZero
	r := NewRandom(0)
	values := []uint32{0, 1, 2, 3, 4}
	ShuffleUint32(values, r)
	for i, w := range want {
		if values[i] != uint32(w) {
			t.Fatalf("uint32 shuffle %v want %v", values, want)
		}
	}
	if n := r.Next(); n != 8855 {
		t.Fatalf("shuffle consumed wrong source draws: next=%d want 8855", n)
	}
	r.SetSeed(0)
	signed := []int32{-2, -1, 0, 1, 2}
	ShuffleInt32(signed, r)
	for i, w := range want {
		if signed[i] != w-2 {
			t.Fatalf("int32 shuffle %v at %d", signed, i)
		}
	}
	if n := r.Next(); n != 8855 {
		t.Fatal("signed shuffle draw order", n)
	}
	for n := 0; n <= 1; n++ {
		r.SetSeed(0)
		ShuffleUint32(values[:n], r)
		if g := r.Next(); g != 38 {
			t.Fatal("empty/singleton uint32 shuffle consumed draw", n, g)
		}
		r.SetSeed(0)
		ShuffleInt32(signed[:n], r)
		if g := r.Next(); g != 38 {
			t.Fatal("empty/singleton int32 shuffle consumed draw", n, g)
		}
	}
}
