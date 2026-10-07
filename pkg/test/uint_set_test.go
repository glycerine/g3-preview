// Assertion ports from src/test/uint_set.cpp, registered in src/test/main.cpp.
package test

import (
	_ "embed"
	"encoding/json"
	"github.com/glycerine/g3/pkg/util"
	rand "math/rand/v2"

	"testing"
)

//go:embed test_vectors/uint_set_prints.json
var uintSetPrintsJSON []byte

func TestUintSetSourceTst1(t *testing.T) {
	r := rand.New(rand.NewPCG(72, 93))
	for run := 0; run < 100; run++ {
		for _, n := range []int{1 + r.IntN(31), 1 + r.IntN(100)} {
			var s util.UintSet
			ref := make([]bool, n)
			count := uint32(0)
			operations := r.IntN(1000)
			for step := 0; step < operations; step++ {
				op := r.IntN(3)
				i := r.IntN(n)
				if op < 2 {
					if !ref[i] {
						count++
					}
					ref[i] = true
					s.Insert(uint32(i))
				} else {
					if ref[i] {
						count--
					}
					ref[i] = false
					s.Remove(uint32(i))
				}
				if s.NumElems() != count || s.Empty() != (count == 0) {
					t.Fatal(run, n, step, "count/empty")
				}
				for j, w := range ref {
					if s.Contains(uint32(j)) != w {
						t.Fatal(run, n, step, j)
					}
				}
			}
		}
	}
}
func TestUintSetSourceTst2(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 8))
	for run := 0; run < 1000; run++ {
		for _, n := range []uint32{1, 10, 31, 32, 100} {
			var s util.UintSet
			if !s.Empty() {
				t.Fatal("empty constructor")
			}
			v := uint32(r.IntN(int(n)))
			s.Insert(v)
			if s.Empty() || s.NumElems() != 1 {
				t.Fatal("singleton")
			}
			for i := 0; i < 100; i++ {
				v2 := uint32(r.IntN(int(n)))
				if v != v2 && s.Contains(v2) {
					t.Fatal("not a member")
				}
			}
			s.Remove(v)
			if s.NumElems() != 0 || !s.Empty() {
				t.Fatal("empty after remove")
			}
		}
	}
}
func TestUintSetSourceTst3(t *testing.T) {
	for _, n := range []uint32{12, 100} {
		var a, b util.UintSet
		if !a.Equal(&b) {
			t.Fatal("initial equality")
		}
		a.Insert(3)
		if a.NumElems() != 1 || b.NumElems() != 0 || a.Equal(&b) {
			t.Fatal("first insert")
		}
		b.Insert(5)
		if b.NumElems() != 1 || a.Equal(&b) || a.SubsetOf(&b) {
			t.Fatal("distinct singletons")
		}
		b.Union(&a)
		if !a.SubsetOf(&b) || b.NumElems() != 2 || a.Equal(&b) {
			t.Fatal("union first")
		}
		a.Union(&b)
		if !a.SubsetOf(&b) || !b.SubsetOf(&a) || a.NumElems() != 2 || b.NumElems() != 2 || !a.Equal(&b) {
			t.Fatal("union second")
		}
		a.Insert(9)
		if a.NumElems() != 3 || b.NumElems() != 2 {
			t.Fatal("third insert")
		}
		a.Insert(9)
		if a.NumElems() != 3 || b.NumElems() != 2 || !b.SubsetOf(&a) || a.SubsetOf(&b) || a.Equal(&b) {
			t.Fatal("duplicate insert")
		}
		c := a.Clone()
		if !a.Equal(c) || !a.SubsetOf(c) || !c.SubsetOf(&a) || b.Equal(c) {
			t.Fatal("copy a")
		}
		d := b.Clone()
		if !b.Equal(d) || !b.SubsetOf(d) || !d.SubsetOf(&b) || b.Equal(c) {
			t.Fatal("copy b")
		}
		for i := uint32(0); i < n; i++ {
			var singleton util.UintSet
			singleton.Insert(i)
			if a.Contains(i) != singleton.SubsetOf(&a) {
				t.Fatal(n, i)
			}
		}
	}
}
func TestUintSetSourceTst4(t *testing.T) {
	var s util.UintSet
	s.Insert(32)
	if !s.Contains(32) || s.Contains(31) || s.Contains(0) {
		t.Fatal("word boundary insert")
	}
	s.Remove(32)
	if s.Contains(32) || s.Contains(31) || s.Contains(0) {
		t.Fatal("word boundary remove")
	}
}
func TestUintSetSourceTst5PrintedExpectations(t *testing.T) {
	var fixtures struct {
		Hashes   []uint32
		Displays []string
	}
	if e := json.Unmarshal(uintSetPrintsJSON, &fixtures); e != nil {
		t.Fatal(e)
	}
	var s util.UintSet
	for step := 0; step < 5; step++ {
		if step > 0 {
			s.Insert([]uint32{1, 2, 44, 4}[step-1])
		}
		if s.Hash() != fixtures.Hashes[step] || s.String() != fixtures.Displays[step] {
			t.Fatal(step, s.Hash(), s.String())
		}
	}
	a := s.Clone()
	s.Insert(5)
	if a.Contains(5) {
		t.Fatal("value-key copies must be independent")
	}
}
func TestUintSetSourcePaddingCapacityAndCopies(t *testing.T) {
	var a, b util.UintSet
	b.Insert(96)
	if b.MaxElem() != 128 || b.CapacityWords() != 5 {
		t.Fatal("word extent/capacity", b.MaxElem(), b.CapacityWords())
	}
	a.Union(&b)
	if a.MaxElem() != 160 || a.CapacityWords() != 5 || !a.Equal(&b) || a.Hash() != b.Hash() {
		t.Fatal("source_size+1 union padding", a.MaxElem())
	}
	b.Intersect(&util.UintSet{})
	if b.MaxElem() != 0 || b.CapacityWords() != 5 {
		t.Fatal("intersection retains capacity")
	}
	clone := b.Clone()
	if clone.CapacityWords() != 0 || clone.MaxElem() != 0 {
		t.Fatal("copy ctor only pushes logical words")
	}
	var dst util.UintSet
	dst.Assign(&b)
	if dst.CapacityWords() != 5 || !dst.Empty() {
		t.Fatal("assignment copies source capacity")
	}
	dst.Insert(2)
	if b.Contains(2) {
		t.Fatal("assignment aliases")
	}
	a.Reset()
	if a.MaxElem() != 0 || a.CapacityWords() != 5 {
		t.Fatal("reset retains capacity")
	}
	a.Insert(1)
	if a.NumElems() != 1 || a.Contains(96) {
		t.Fatal("reset/growth initializes words")
	}
	a.Swap(&dst)
	if !a.Contains(2) || !dst.Contains(1) {
		t.Fatal("swap")
	}
	a.Close()
	if a.CapacityWords() != 0 || a.MaxElem() != 0 {
		t.Fatal("destructor replacement")
	}
}
func TestUintSetIteratorMethods(t *testing.T) {
	var s util.UintSet
	for _, v := range []uint32{130, 0, 31, 32, 65, 97} {
		s.Insert(v)
	}
	want := []uint32{0, 31, 32, 65, 97, 130}
	it, end := s.Begin(), s.End()
	for i, w := range want {
		if it.Equal(end) || it.Value() != w {
			t.Fatal(i, it.Value(), w)
		}
		old := it.PostNext()
		if old.Value() != w {
			t.Fatal("postincrement")
		}
	}
	if !it.Equal(end) || !it.AtEnd() {
		t.Fatal("iterator end")
	}
	first := s.Begin()
	it.Assign(first)
	if it.Value() != 0 {
		t.Fatal("iterator assignment")
	}
	it.Next()
	if it.Value() != 31 {
		t.Fatal("preincrement")
	}
	var other util.UintSet
	other.Insert(31)
	oi := other.Begin()
	if !it.Equal(oi) {
		t.Fatal("source iterator equality compares index only")
	}
	var empty util.UintSet
	if !empty.Begin().Equal(empty.End()) {
		t.Fatal("empty iteration")
	}
	if !(util.UintSetEqual{}).Equal(&s, &s) || (util.UintSetHasher{}).Hash(&s) != s.Hash() {
		t.Fatal("source functor wrappers")
	}
}
func TestUintSetWeightedHashWrap(t *testing.T) {
	var s util.UintSet
	s.Insert(31)
	s.Insert(63)
	if s.Hash() != 2147483648 {
		t.Fatal("weighted hash modulo uint32", s.Hash())
	}
	s.Remove(63)
	if s.Hash() != 2147483648 {
		t.Fatal("trailing zero word hash")
	}
}
func TestUintSetIndependentBoundedOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(772, 993))
	var a, b util.UintSet
	var aa, bb [192]bool
	for step := 0; step < 3000; step++ {
		switch r.IntN(8) {
		case 0:
			i := r.IntN(len(aa))
			a.Insert(uint32(i))
			aa[i] = true
		case 1:
			i := r.IntN(len(aa))
			a.Remove(uint32(i))
			aa[i] = false
		case 2:
			i := r.IntN(len(bb))
			b.Insert(uint32(i))
			bb[i] = true
		case 3:
			a.Union(&b)
			for i := range aa {
				aa[i] = aa[i] || bb[i]
			}
		case 4:
			a.Intersect(&b)
			for i := range aa {
				aa[i] = aa[i] && bb[i]
			}
		case 5:
			a.Reset()
			clear(aa[:])
		case 6:
			a.Assign(&b)
			aa = bb
		case 7:
			a.Swap(&b)
			aa, bb = bb, aa
		}
		var count uint32
		equal, subset := true, true
		var ordered []uint32
		for i, v := range aa {
			if a.Contains(uint32(i)) != v {
				t.Fatal(step, i)
			}
			if v {
				count++
				ordered = append(ordered, uint32(i))
			}
			equal = equal && v == bb[i]
			subset = subset && (!v || bb[i])
		}
		if a.NumElems() != count || a.Empty() != (count == 0) || a.Equal(&b) != equal || a.SubsetOf(&b) != subset {
			t.Fatal(step, "relations")
		}
		it := a.Begin()
		for _, w := range ordered {
			if it.AtEnd() || it.Value() != w {
				t.Fatal(step, "iterator")
			}
			it.Next()
		}
		if !it.AtEnd() {
			t.Fatal(step, "iterator excess")
		}
	}
}
func TestUintSetWarmOperationsNoAllocation(t *testing.T) {
	var a, b util.UintSet
	a.Insert(255)
	b.Insert(129)
	if n := testing.AllocsPerRun(100, func() {
		a.Insert(3)
		a.Remove(3)
		a.NumElems()
		a.Hash()
		a.Union(&b)
		a.Intersect(&b)
		for it := a.Begin(); !it.AtEnd(); it.Next() {
			it.Value()
		}
	}); n != 0 {
		t.Fatal("warmed word operations allocated", n)
	}
}
func TestUintSetOverflowRejectedBeforeGrowth(t *testing.T) {
	var s util.UintSet
	s.Insert(3)
	before := s.CapacityWords()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("source endpoint overflow must be diagnosed")
			}
		}()
		s.Insert(^uint32(0))
	}()
	if s.NumElems() != 1 || !s.Contains(3) || s.CapacityWords() != before {
		t.Fatal("invalid extent changed set")
	}
}
