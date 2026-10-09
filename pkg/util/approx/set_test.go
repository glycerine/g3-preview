// Source: src/util/approx_set.h:43-158,162-233; src/util/approx_set.cpp:22-51.
package approx

import "testing"

func TestUintSetCollisionsTraversalAndString(t *testing.T) {
	s := UintSetFrom([]uint32{0, 2, 31, 32, 63, 64, 127, ^uint32(0)})
	want := []uint32{0, 2, 31, 32, 63}
	var got []uint32
	s.ForEach(func(value uint32) bool {
		got = append(got, value)
		return true
	})
	if len(got) != len(want) || s.Size() != uint32(len(want)) {
		t.Fatal(got, s.Size())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatal(got, want)
		}
	}
	if text := s.String(); text != "{0, 2, 31, 32, 63}" {
		t.Fatal(text)
	}
	visited := 0
	s.ForEach(func(uint32) bool { visited++; return false })
	if visited != 1 {
		t.Fatal(visited)
	}
}

func TestUintSetAlgebra(t *testing.T) {
	values := []uint32{0, 2, 9, 31, 32, 63, 64, ^uint32(0)}
	var sets [256]UintSet
	var oracle [256][64]bool
	for mask := range 256 {
		var input []uint32
		for i, value := range values {
			if mask&(1<<i) != 0 {
				input = append(input, value)
				oracle[mask][value%64] = true
			}
		}
		sets[mask] = UintSetFrom(input)
		copy := sets[mask]
		copy.Reset()
		if !copy.Empty() {
			t.Fatal(mask)
		}
	}
	for a := range 256 {
		for b := range 256 {
			union := sets[a].Union(sets[b])
			intersection := sets[a].Intersection(sets[b])
			difference := sets[a]
			difference.Subtract(sets[b])
			unionAssigned := sets[a]
			unionAssigned.UnionWith(sets[b])
			intersectionAssigned := sets[a]
			intersectionAssigned.IntersectWith(sets[b])
			if unionAssigned != union || intersectionAssigned != intersection {
				t.Fatal(a, b)
			}
			subset, equal, disjoint := true, true, true
			for residue := range 64 {
				av, bv := oracle[a][residue], oracle[b][residue]
				value := uint32(residue)
				if union.MayContain(value) != (av || bv) || intersection.MayContain(value) != (av && bv) || difference.MayContain(value) != (av && !bv) || sets[a].MustNotContain(value) == av {
					t.Fatal(a, b, residue)
				}
				if av && !bv {
					subset = false
				}
				if av != bv {
					equal = false
				}
				if av && bv {
					disjoint = false
				}
			}
			if sets[a].ApproxSubset(sets[b]) != subset || sets[a].MustNotSubset(sets[b]) == subset || sets[a].MustNotSubsume(sets[b]) == subset || sets[a].Equivalent(sets[b]) != equal || sets[a].MayEqual(sets[b]) != equal || sets[a].MustNotEqual(sets[b]) == equal || sets[a].EmptyIntersection(sets[b]) != disjoint {
				t.Fatal(a, b)
			}
		}
	}
}
