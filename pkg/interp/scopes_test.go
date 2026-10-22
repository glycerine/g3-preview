package interp

import "testing"

func TestInterpolationScopesLinearRanges(t *testing.T) {
	s, err := NewScopes(nil)
	interpMust(t, err)
	r := EmptyScopeRange()
	interpMust(t, s.RangeAdd(2, &r))
	interpMust(t, s.RangeAdd(5, &r))
	if r != (ScopeRange{Lo: 2, Hi: 5}) || r.IsEmpty() || r.IsFull() || r.Min() != 2 || r.Max() != 5 {
		t.Fatal(r)
	}
	for node, want := range map[int]bool{1: false, 2: true, 4: true, 5: true, 6: false} {
		got, e := s.InRange(node, r)
		interpMust(t, e)
		if got != want {
			t.Fatalf("InRange(%d, %v) = %v", node, r, got)
		}
	}
	glb, err := s.RangeGLB(r, ScopeRange{Lo: 4, Hi: 8})
	interpMust(t, err)
	if glb != (ScopeRange{Lo: 4, Hi: 5}) {
		t.Fatal(glb)
	}
	lub, err := s.RangeLUB(r, ScopeRange{Lo: -1, Hi: 3})
	interpMust(t, err)
	if lub != (ScopeRange{Lo: -1, Hi: 5}) {
		t.Fatal(lub)
	}
	contained, err := s.RangeContained(glb, r)
	interpMust(t, err)
	if !contained {
		t.Fatal("intersection should be contained")
	}
	near, err := s.RangeNear(r, ScopeRange{Lo: 7, Hi: 9})
	interpMust(t, err)
	if near != 5 {
		t.Fatal(near)
	}
}

func TestInterpolationScopesTreeRanges(t *testing.T) {
	// 0 and 1 are children of 2; 2 and 3 are children of 4.
	s, err := NewScopes([]int{2, 2, 4, 4, ScopeTop})
	interpMust(t, err)
	if !s.TreeMode() {
		t.Fatal("tree mode")
	}
	r := EmptyScopeRange()
	interpMust(t, s.RangeAdd(0, &r))
	interpMust(t, s.RangeAdd(1, &r))
	if r != (ScopeRange{Lo: ScopeBottom, Hi: 2}) {
		t.Fatal(r)
	}
	for node, want := range map[int]bool{0: true, 1: true, 2: true, 3: false, 4: false} {
		got, e := s.InRange(node, r)
		interpMust(t, e)
		if got != want {
			t.Fatalf("InRange(%d, %v) = %v", node, r, got)
		}
	}
	down := DownwardScopeRange(4)
	if !FullScopeRange().IsFull() || EmptyScopeRange().IsFull() {
		t.Fatal("sentinel ranges")
	}
	for node := 0; node <= 4; node++ {
		inside, e := s.InRange(node, down)
		interpMust(t, e)
		if !inside {
			t.Fatalf("node %d outside root downward range", node)
		}
	}
	intersects, err := s.RangesIntersect(r, DownwardScopeRange(3))
	interpMust(t, err)
	if !intersects {
		t.Fatal("source compact ranges retain their shared bottom sentinel")
	}
	near, err := s.RangeNear(DownwardScopeRange(2), DownwardScopeRange(3))
	interpMust(t, err)
	if near != ScopeBottom {
		t.Fatal(near)
	}
}

func TestInterpolationScopesCopyAndInvalidParentGuards(t *testing.T) {
	values := map[ScopeRange]int{DownwardScopeRange(1): 7}
	if values[DownwardScopeRange(1)] != 7 {
		t.Fatal("native comparable scope-range map key")
	}
	parents := []int{1, ScopeTop}
	s, err := NewScopes(parents)
	interpMust(t, err)
	parents[0] = 0
	inside, err := s.InRange(0, DownwardScopeRange(1))
	interpMust(t, err)
	if !inside {
		t.Fatal("parent relation was not copied")
	}
	for _, invalid := range [][]int{{0}, {2, ScopeTop}, {-1, ScopeTop}} {
		if _, e := NewScopes(invalid); e == nil {
			t.Fatalf("accepted invalid parents %v", invalid)
		}
	}
	var nilScopes *Scopes
	if _, e := nilScopes.InRange(0, EmptyScopeRange()); e == nil {
		t.Fatal("nil scopes accepted")
	}
	if e := s.RangeAdd(0, nil); e == nil {
		t.Fatal("nil range accepted")
	}
}
