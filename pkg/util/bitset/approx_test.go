package bitset

import "testing"

func TestApproxSetSource64ResiduesDisplayAndIterator(t *testing.T) {
	s := NewApproxSet(0, 63, 31, 64, 127)
	if s.Size() != 3 || s.String() != "{0, 31, 63}" || !s.MayContain(128) || !s.MayContain(95) || !s.MustNotContain(30) {
		t.Fatal("source modulo-64 projection", s, s.Size())
	}
	var visited []uint32
	s.ForEach(func(v uint32) bool { visited = append(visited, v); return true })
	if len(visited) != 3 || visited[0] != 0 || visited[1] != 31 || visited[2] != 63 {
		t.Fatal("source ascending iterator", visited)
	}
	visited = visited[:0]
	s.ForEach(func(v uint32) bool { visited = append(visited, v); return false })
	if len(visited) != 1 || visited[0] != 0 {
		t.Fatal("early stop", visited)
	}
	if (ApproxSet(0)).String() != "{}" || (ApproxSet(0)).Size() != 0 {
		t.Fatal("empty display or size")
	}
}

func TestApproxSetIndependentFiniteAlgebra(t *testing.T) {
	var sets [256]ApproxSet
	for mask := 0; mask < len(sets); mask++ {
		for bit := uint32(0); bit < 8; bit++ {
			if mask&(1<<bit) != 0 {
				sets[mask].Insert(bit + 64)
			}
		}
		if sets[mask].Size() != uint32(popcount8(mask)) {
			t.Fatal("size", mask)
		}
	}
	for a := range sets {
		for b := range sets {
			x, y := sets[a], sets[b]
			if x.Union(y) != sets[a|b] || x.Intersect(y) != sets[a&b] || x.ApproxSubset(y) != (a&^b == 0) || x.EmptyIntersection(y) != (a&b == 0) || x.MustNotSubset(y) != (a&^b != 0) || x.MustNotSubsume(y) != (a&^b != 0) || x.Equivalent(y) != (a == b) || x.MayEqual(y) != (a == b) || x.MustNotEqual(y) != (a != b) {
				t.Fatal("set algebra", a, b)
			}
			z := x
			z.UnionAssign(y)
			if z != sets[a|b] {
				t.Fatal("union assign", a, b)
			}
			z = x
			z.IntersectAssign(y)
			if z != sets[a&b] {
				t.Fatal("intersect assign", a, b)
			}
			z = x
			z.SubtractAssign(y)
			if z != sets[a&^b] {
				t.Fatal("subtract assign", a, b)
			}
		}
	}
	s := sets[255]
	s.Reset()
	if !s.Empty() {
		t.Fatal("reset")
	}
}

func popcount8(mask int) int {
	n := 0
	for i := 0; i < 8; i++ {
		if mask&(1<<i) != 0 {
			n++
		}
	}
	return n
}
