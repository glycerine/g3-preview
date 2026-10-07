package sat

import (
	"reflect"
	"testing"
)

// Added finite oracle for sat::uint_set; source inline SASSERT obligations cover
// remove membership consistency and nonempty erase. Unlike util/uint_set.h,
// SAT's set preserves insertion order and removes by swapping the last member.
func TestSATSetOrderAndOwnership(t *testing.T) {
	var s satUintSet
	for _, v := range []uint32{4, 2, 9, 2, 1} {
		s.insert(v)
	}
	if !reflect.DeepEqual(s.values, []uint32{4, 2, 9, 1}) {
		t.Fatal(s.values)
	}
	s.remove(2)
	s.remove(77)
	if !reflect.DeepEqual(s.values, []uint32{4, 1, 9}) || s.contains(2) {
		t.Fatal(s.values)
	}
	if s.erase() != 9 || s.size() != 2 || s.empty() {
		t.Fatal("erase assertion")
	}
	var copy satUintSet
	copy.assign(&s)
	copy.remove(4)
	if !s.contains(4) || copy.contains(4) {
		t.Fatal("copy aliases storage")
	}
	var other satUintSet
	other.insert(1)
	other.insert(7)
	s.union(&other)
	if !reflect.DeepEqual(s.values, []uint32{4, 1, 7}) {
		t.Fatal(s.values)
	}
	s.intersect(&other)
	if !reflect.DeepEqual(s.values, []uint32{1, 7}) || s.contains(4) {
		t.Fatal(s.values)
	}
	s.reset()
	if !s.empty() || s.contains(1) {
		t.Fatal("reset")
	}
	s.insert(4)
	if s.contains(1) || !s.contains(4) {
		t.Fatal("stale membership on reserve")
	}
	s.finalize()
	if s.inSet != nil || s.values != nil {
		t.Fatal("finalize")
	}
}
func TestSATLiteralSets(t *testing.T) {
	p, q := MakeLiteral(0, false), MakeLiteral(1, false)
	s := satMakeLiteralSet([]Literal{p, p.Negated(), q, p})
	if s.size() != 3 || !s.contains(p) || !s.contains(p.Negated()) {
		t.Fatal("literal identity")
	}
	s.remove(p)
	if s.contains(p) || !s.contains(p.Negated()) {
		t.Fatal("sign conflated")
	}
	if s.pop() != p.Negated() {
		t.Fatal("swap-remove/pop order")
	}
	out := s.toVector()
	out[0] = p
	if s.toVector()[0] != q {
		t.Fatal("returned vector aliases")
	}
	var c satLiteralSet
	c.assign(&s)
	c.insert(p)
	s.intersect(&c)
	if s.size() != 1 {
		t.Fatal("intersection")
	}
	s.union(&c)
	if s.size() != 2 {
		t.Fatal("union")
	}
	s.assignVector([]Literal{p, p})
	if s.size() != 1 {
		t.Fatal("assign vector")
	}
	s.reset()
	if !s.empty() {
		t.Fatal("reset")
	}
	s.finalize()
}
func TestSATSetFiniteModel(t *testing.T) {
	var s satUintSet
	model := make([]bool, 64)
	state := uint32(1)
	for i := 0; i < 3000; i++ {
		state = state*1664525 + 1013904223
		v := state % 64
		switch (state >> 8) % 3 {
		case 0:
			s.insert(v)
			model[v] = true
		case 1:
			s.remove(v)
			model[v] = false
		case 2:
			if !s.empty() {
				v = s.erase()
				if !model[v] {
					t.Fatal("erased missing element")
				}
				model[v] = false
			}
		}
		count := uint32(0)
		for v, has := range model {
			if has {
				count++
			}
			if s.contains(uint32(v)) != has {
				t.Fatal("membership mismatch")
			}
		}
		if s.size() != count || s.empty() != (count == 0) {
			t.Fatal("source consistency")
		}
	}
}
func TestSATApproximation(t *testing.T) {
	var a, b satApproxSet
	a.insert(1)
	a.insert(33)
	b.insert(1)
	if !a.mayContain(65) || a.mustNotContain(1) || !a.mustNotContain(2) || a != b {
		t.Fatal("source low-bit collision")
	}
	b.insert(2)
	if a.mustNotSubset(b) || !b.mustNotSubset(a) || !a.approxSubset(b) {
		t.Fatal("subset masks")
	}
	c := a
	c.union(b)
	if c != b {
		t.Fatal("union")
	}
	c.subtract(a)
	if c.mayContain(1) || !c.mayContain(2) {
		t.Fatal("difference")
	}
	c.intersect(a)
	if !c.empty() || !a.emptyIntersection(c) {
		t.Fatal("intersection")
	}
	a.reset()
	if !a.empty() {
		t.Fatal("reset")
	}
}
