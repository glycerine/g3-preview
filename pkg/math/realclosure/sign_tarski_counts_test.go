package realclosure

import "testing"

func TestRealClosureSourceTarskiSignCountBranches(t *testing.T) {
	cases := []struct {
		name                      string
		roots, q, q2              int
		equal, positive, negative int
		queriedSquare             bool
	}{
		{"all-positive", 3, 3, 3, 0, 3, 0, true},
		{"all-negative", 3, -3, 3, 0, 0, 3, true},
		{"one-zero-positive", 3, 2, 999, 1, 2, 0, false},
		{"one-zero-negative", 3, -2, 999, 1, 0, 2, false},
		{"all-three-signs", 3, 0, 2, 1, 1, 1, true},
		{"two-zeros-positive", 3, 1, 1, 2, 1, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			equal, positive, negative, queried, err := countSignsFromTarski(tc.roots, tc.q, tc.q2)
			if err != nil || equal != tc.equal || positive != tc.positive || negative != tc.negative || queried != tc.queriedSquare {
				t.Fatalf("counts=(%d,%d,%d), queried=%v, err=%v", equal, positive, negative, queried, err)
			}
		})
	}
	for _, query := range [][3]int{{0, 0, 0}, {3, 4, 3}, {3, 0, -1}, {3, 0, 1}, {3, 1, 0}} {
		if _, _, _, _, err := countSignsFromTarski(query[0], query[1], query[2]); err == nil {
			t.Fatalf("invalid Tarski triple %v was accepted", query)
		}
	}
}
