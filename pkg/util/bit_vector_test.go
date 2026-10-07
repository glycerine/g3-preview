// Assertions translated from src/test/bit_vector.cpp.
package util

import (
	_ "embed"
	"encoding/json"
	rand "math/rand/v2"

	"testing"
)

//go:embed test_vectors/source_hash.json
var hashJSON []byte

func bv(n uint32, ones ...uint32) *Vector {
	v := new(Vector)
	v.Resize(n, false)
	for _, i := range ones {
		v.Set(i, true)
	}
	return v
}
func check(t *testing.T, v *Vector, n uint32, ones ...uint32) {
	t.Helper()
	want := make([]bool, n)
	for _, i := range ones {
		want[i] = true
	}
	if v.Size() != n {
		t.Fatalf("size %d want %d", v.Size(), n)
	}
	for i, w := range want {
		if v.Get(uint32(i)) != w {
			t.Fatalf("bit %d got %v want %v", i, v.Get(uint32(i)), w)
		}
	}
}
func TestSourceResizeCrash(t *testing.T) {
	a := new(Vector)
	a.Push(true)
	a.Resize(64, false)
	check(t, a, 64, 0)
	b := new(Vector)
	b.Push(false)
	b.Resize(64, true)
	for i := uint32(0); i < 64; i++ {
		if b.Get(i) != (i != 0) {
			t.Fatal(i)
		}
	}
}
func TestSourceResize30(t *testing.T) {
	b := new(Vector)
	b.Push(true)
	b.Push(false)
	b.Push(true)
	b.Resize(30, false)
	check(t, b, 30, 0, 2)
}
func TestSourceShift(t *testing.T) {
	ones := []uint32{105, 99, 98, 90, 80, 75, 33, 32, 31, 30, 10}
	b := bv(111, ones...)
	n := uint32(111)
	for _, k := range []uint32{16, 1, 32, 42, 16, 63} {
		b.ShiftRight(k)
		n += k
		for i := range ones {
			ones[i] += k
		}
		check(t, b, n, ones...)
	}
}
func TestSourceOr(t *testing.T) {
	a, b := bv(5, 4), bv(10, 8, 3, 2, 1)
	a.Or(b)
	if a.Size() != 10 || a.Equal(b) {
		t.Fatal("or size/equality")
	}
	a.Unset(4)
	if !a.Equal(b) {
		t.Fatal("equal after unset")
	}
	a.Unset(3)
	if a.Equal(b) {
		t.Fatal("unequal")
	}
	a, b = bv(2, 0), bv(5, 4)
	a.Or(b)
	if a.Equal(b) {
		t.Fatal("unequal")
	}
	b.Set(0, true)
	if !a.Equal(b) {
		t.Fatal("equal")
	}
	a, b = bv(10, 5), bv(10, 8, 3)
	b.Resize(5, false)
	a.Or(b)
	check(t, a, 10, 5, 3)
	a, b = bv(123, 122, 100, 80, 4), bv(126, 124, 100, 80, 4)
	if a.Equal(b) {
		t.Fatal("different sizes")
	}
	b.Resize(123, false)
	if a.Equal(b) {
		t.Fatal("different bits")
	}
	a.Resize(120, false)
	b.Resize(120, false)
	if !a.Equal(b) {
		t.Fatal("shrink equal")
	}
	a.Unset(80)
	a.Unset(100)
	if a.Equal(b) {
		t.Fatal("unset unequal")
	}
	a.Or(b)
	if !a.Equal(b) {
		t.Fatal("or equal")
	}
	a, b = bv(5, 4, 0), bv(10, 8, 1)
	a.Or(b)
	check(t, a, 10, 8, 4, 1, 0)
	a, b = bv(32, 1, 5), bv(32, 8, 0)
	a.Or(b)
	check(t, a, 32, 1, 5, 8, 0)
}
func TestSourceAnd(t *testing.T) {
	a, b := bv(5, 2, 4), bv(3, 2)
	a.And(b)
	check(t, a, 5, 2)
	a, b = bv(241, 240, 232, 127, 128, 8), bv(128, 127, 5)
	a.And(b)
	check(t, a, 241, 127)
}
func TestSourceReset(t *testing.T) {
	b := new(Vector)
	bit := true
	for sz := uint32(1); sz < 84; sz++ {
		b.Reset()
		b.Resize(sz, bit)
		for i := uint32(0); i < sz; i++ {
			if b.Get(i) != bit {
				t.Fatal(sz, i)
			}
		}
		for sz2 := sz; sz2 < sz+10; sz2++ {
			b.Resize(sz2, !bit)
			for i := sz; i < sz2; i++ {
				if b.Get(i) == bit {
					t.Fatal(sz, sz2, i)
				}
			}
		}
		bit = !bit
	}
}
func TestSourceEq(t *testing.T) {
	a, b, c := bv(32), bv(32), bv(32)
	a.Set(3, true)
	if a.Equal(b) || !b.Equal(c) {
		t.Fatal("eq")
	}
	c.Set(3, true)
	if !a.Equal(c) {
		t.Fatal("eq set")
	}
	b.Set(31, true)
	c.Set(31, true)
	c.Unset(3)
	if !b.Equal(c) {
		t.Fatal("eq top bit")
	}
}

// Source tst1's bool-vector oracle and six operation classes, bounded with a
// fixed independent PCG generator. Source tst1 is unreachable after its return.
func TestSourceBoolVectorOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(19, 23))
	b := new(Vector)
	var ref []bool
	for i := 0; i < 4000; i++ {
		op := r.IntN(6)
		switch {
		case op <= 1:
			v := r.IntN(2) != 0
			b.Push(v)
			ref = append(ref, v)
		case op <= 3:
			if len(ref) > 0 {
				j := r.IntN(len(ref))
				if b.Get(uint32(j)) != ref[j] {
					t.Fatal(i, j)
				}
				v := r.IntN(2) != 0
				b.Set(uint32(j), v)
				ref[j] = v
				if b.Get(uint32(j)) != v {
					t.Fatal(i, j)
				}
			}
		case op == 4:
			if len(ref) > 0 {
				j := r.IntN(len(ref))
				if b.Get(uint32(j)) != ref[j] {
					t.Fatal(i, j)
				}
			}
		default:
			for j, v := range ref {
				if b.Get(uint32(j)) != v {
					t.Fatal(i, j)
				}
			}
		}
		if b.Size() != uint32(len(ref)) {
			t.Fatal("size")
		}
	}
}
func TestOwnershipAndRemainingMethods(t *testing.T) {
	a := New(100)
	if !a.Empty() || a.CapacityWords() != 4 {
		t.Fatal("reserve constructor")
	}
	a.Resize(35, true)
	b := a.Clone()
	b.Unset(2)
	if !a.Get(2) {
		t.Fatal("copy aliases")
	}
	c := new(Vector)
	c.Assign(b)
	b.Unset(3)
	if !c.Get(3) {
		t.Fatal("assign aliases")
	}
	c.Shrink(32)
	c.Pop()
	if c.Size() != 31 || !c.Back() {
		t.Fatal("pop")
	}
	c.Reserve(12, false)
	if c.Size() != 31 {
		t.Fatal("reserve shrank")
	}
	c.Neg()
	check(t, c, 31, 2)
	c.Fill0()
	check(t, c, 31)
	a.Swap(c)
	check(t, a, 31)
	if c.Size() != 35 {
		t.Fatal("swap")
	}
	d := FromWords([]uint32{0x80000001, 3}, 34)
	if d.Word(0) != 0x80000001 || d.NumWords() != 2 || d.String() != "1110000000000000000000000000000001" {
		t.Fatal("words/display", d.String())
	}
	if !d.Contains(bv(34, 0, 31, 32)) {
		t.Fatal("contains")
	}
	if d.Contains(bv(34, 2)) {
		t.Fatal("contains false")
	}
}
func TestFastReset(t *testing.T) {
	var f FastReset
	f.Push(true)
	f.Push(false)
	f.Push(true)
	f.Set(0, true)
	f.Set(2, false)
	f.Reset()
	if f.Size() != 3 || f.Get(0) || f.Get(1) || f.Get(2) {
		t.Fatal("fast reset preserves size, clears indexed bits")
	}
	f.Set(1, true)
	f.Fill0()
	if f.Get(1) {
		t.Fatal("fill0")
	}
}

// Hash values recorded from actual C++ string_hash over {1,0,0,0,3,0,0,0}.
func TestSourceHashByteRounding(t *testing.T) {
	var want []uint32
	if err := json.Unmarshal(hashJSON, &want); err != nil {
		t.Fatal(err)
	}
	for n := uint32(0); n <= 64; n++ {
		b := FromWords([]uint32{1, 3}, n)
		if g := b.Hash(); g != want[n/8] {
			t.Fatalf("hash size %d=%d want %d", n, g, want[n/8])
		}
	}
}
func TestAndEmptyAndLonger(t *testing.T) {
	a := bv(40, 0, 31, 39)
	a.And(bv(100, 0, 39, 90))
	check(t, a, 40, 0, 39)
	a.And(bv(0))
	check(t, a, 40)
	empty := bv(0)
	empty.And(a)
	if !empty.Empty() {
		t.Fatal("empty and")
	}
	a.Or(empty)
	check(t, a, 40)
}
func TestGrowthCapacityAndStaleBits(t *testing.T) {
	v := new(Vector)
	for i := uint32(0); i < 129; i++ {
		v.Push(true)
		want := uint32(2)
		if i >= 64 {
			want = 3
		}
		if i >= 96 {
			want = 5
		}
		if v.CapacityWords() != want {
			t.Fatal(i, v.CapacityWords(), want)
		}
	}
	v.Shrink(1)
	v.Resize(129, false)
	check(t, v, 129, 0)
	v.Reset()
	if v.CapacityWords() != 5 {
		t.Fatal("reset capacity")
	}
}
func TestBoundedOperationOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(999, 1001))
	v := new(Vector)
	var ref []bool
	for step := 0; step < 5000; step++ {
		switch r.IntN(9) {
		case 0:
			val := r.IntN(2) == 0
			v.Push(val)
			ref = append(ref, val)
		case 1:
			if len(ref) > 0 {
				v.Pop()
				ref = ref[:len(ref)-1]
			}
		case 2:
			n := r.IntN(180)
			val := r.IntN(2) == 0
			v.Resize(uint32(n), val)
			if n <= len(ref) {
				ref = ref[:n]
			} else {
				for len(ref) < n {
					ref = append(ref, val)
				}
			}
		case 3:
			k := r.IntN(65)
			v.ShiftRight(uint32(k))
			nr := make([]bool, len(ref)+k)
			copy(nr[k:], ref)
			ref = nr
		case 4:
			v.Neg()
			for i := range ref {
				ref[i] = !ref[i]
			}
		case 5:
			v.Fill0()
			clear(ref)
		case 6:
			if len(ref) > 0 {
				n := r.IntN(len(ref) + 1)
				v.Shrink(uint32(n))
				ref = ref[:n]
			}
		case 7:
			v.Reset()
			ref = ref[:0]
		case 8:
			n := r.IntN(180)
			other := bv(uint32(n))
			or := r.IntN(2) == 0
			values := make([]bool, n)
			for i := range values {
				values[i] = r.IntN(2) == 0
				other.Set(uint32(i), values[i])
			}
			if or {
				v.Or(other)
				for len(ref) < n {
					ref = append(ref, false)
				}
				for i := range values {
					ref[i] = ref[i] || values[i]
				}
			} else {
				v.And(other)
				for i := range ref {
					ref[i] = i < len(values) && ref[i] && values[i]
				}
			}
		}
		if v.Size() != uint32(len(ref)) {
			t.Fatal(step, "size")
		}
		for i, val := range ref {
			if v.Get(uint32(i)) != val {
				t.Fatal(step, i)
			}
		}
	}
}
func TestWarmOperationsDoNotAllocate(t *testing.T) {
	b := bv(128, 1, 65, 127)
	c := bv(128, 2, 64, 126)
	if n := testing.AllocsPerRun(100, func() {
		b.Hash()
		b.Or(c)
		b.And(c)
		b.Neg()
		b.Reset()
		b.Resize(128, false)
	}); n != 0 {
		t.Fatalf("warmed bounded operations allocated %g", n)
	}
}
