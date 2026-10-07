// Source: src/util/max_cliques.h; independent bounded canonical mutex CNFs.
package util

import (
	"fmt"
	"testing"
)

func TestNegatedLiteralCliquesIndependentAllThreeVertexMutexGraphs(t *testing.T) {
	pairs := [3][2]uint32{{0, 1}, {0, 2}, {1, 2}}
	for mask := uint32(0); mask < 8; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			m, b := memoryRuntimeFixture(t, false, 64)
			g, e := NewNegatedLiteralCliques(m, b)
			memoryRuntimeRequire(t, e)
			t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
			neighbors := [3]uint32{}
			for j, p := range pairs {
				if mask&(1<<j) != 0 {
					memoryRuntimeRequire(t, g.AddEdge(2*p[0]+1, 2*p[1]+1))
					neighbors[p[0]] |= 1 << p[1]
					neighbors[p[1]] |= 1 << p[0]
				}
			}
			ps := cliqueTestInput(t, m)
			for _, v := range []uint32{0, 2, 4} {
				memoryRuntimeRequire(t, ps.Push(v))
			}
			out, e := NewRuntimeUnsignedRows(m, b)
			memoryRuntimeRequire(t, e)
			t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
			memoryRuntimeRequire(t, g.Cliques(ps, out))
			cliqueTestAudit(t, m, g, ps, out)
			// Independent greedy intersection oracle for the canonical negative-pair
			// domain. Not a replacement algorithm; source traversal/caches execute above.
			expected := [][]uint32{}
			remaining := uint32(7)
			for remaining != 0 {
				candidate := remaining
				clique := []uint32{}
				for candidate != 0 {
					v := uint32(0)
					for candidate&(1<<v) == 0 {
						v++
					}
					candidate &^= 1 << v
					remaining &^= 1 << v
					clique = append(clique, v*2)
					candidate &= neighbors[v]
				}
				if len(clique) > 1 {
					expected = append(expected, clique)
				}
			}
			n, e := out.RowCount()
			memoryRuntimeRequire(t, e)
			if int(n) != len(expected) {
				t.Fatal(n, expected)
			}
			for i, row := range expected {
				got, e := cliqueTestCapture(out, uint32(i))
				memoryRuntimeRequire(t, e)
				if int(got.count) != len(row) {
					t.Fatal(got.count, row)
				}
				for j, v := range row {
					x, e := got.read(uint32(j))
					memoryRuntimeRequire(t, e)
					if x != v {
						t.Fatal(x, v)
					}
					for k := 0; k < j; k++ {
						for assignment := uint32(0); assignment < 8; assignment++ {
							valid := true
							for bit, p := range pairs {
								if mask&(1<<bit) != 0 && assignment&(1<<p[0]) != 0 && assignment&(1<<p[1]) != 0 {
									valid = false
								}
							}
							if valid && assignment&(1<<(v/2)) != 0 && assignment&(1<<(row[k]/2)) != 0 {
								t.Fatal("returned pair not mutually exclusive", assignment, row)
							}
						}
					}
				}
			}
		})
	}
}

// The oracle enumerates satisfying assignments, independently of graph walks.
// Selected goals cover opposite signs and an implication through a non-goal.
func TestNegatedLiteralCliquesMixedPolarityAndTransitiveBooleanOracle(t *testing.T) {
	edges := [][2]uint32{{1, 2}, {3, 5}}
	for _, goal := range [][]uint32{{0, 4}, {0, 3}, {0, 3, 4}, {1, 2, 5}} {
		t.Run(fmt.Sprint(goal), func(t *testing.T) {
			m, b := memoryRuntimeFixture(t, false, 64)
			g, e := NewNegatedLiteralCliques(m, b)
			memoryRuntimeRequire(t, e)
			t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
			for _, edge := range edges {
				memoryRuntimeRequire(t, g.AddEdge(edge[0], edge[1]))
			}
			ps := cliqueTestInput(t, m)
			for _, v := range goal {
				memoryRuntimeRequire(t, ps.Push(v))
			}
			out, e := NewRuntimeUnsignedRows(m, b)
			memoryRuntimeRequire(t, e)
			t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
			truth := func(assignment, l uint32) bool { return (assignment&(1<<(l>>1)) != 0) != (l&1 != 0) }
			mutex := func(a, b uint32) bool {
				for assignment := uint32(0); assignment < 8; assignment++ {
					valid := true
					for _, edge := range edges {
						if !truth(assignment, edge[0]) && !truth(assignment, edge[1]) {
							valid = false
						}
					}
					if valid && truth(assignment, a) && truth(assignment, b) {
						return false
					}
				}
				return true
			}
			remaining := append([]uint32{}, goal...)
			var expected [][]uint32
			for len(remaining) > 0 {
				candidate := append([]uint32{}, remaining...)
				var row []uint32
				for len(candidate) > 0 {
					v := candidate[0]
					row = append(row, v)
					next := remaining[:0]
					for _, x := range remaining {
						if x != v {
							next = append(next, x)
						}
					}
					remaining = next
					nextCandidates := candidate[:0]
					for _, x := range candidate[1:] {
						if mutex(v, x) {
							nextCandidates = append(nextCandidates, x)
						}
					}
					candidate = nextCandidates
				}
				if len(row) > 1 {
					expected = append(expected, row)
				}
			}
			memoryRuntimeRequire(t, g.Cliques(ps, out))
			cliqueTestAudit(t, m, g, ps, out)
			n, e := out.RowCount()
			memoryRuntimeRequire(t, e)
			if int(n) != len(expected) {
				t.Fatal(n, expected)
			}
			for i, want := range expected {
				got, e := cliqueTestCapture(out, uint32(i))
				memoryRuntimeRequire(t, e)
				if int(got.count) != len(want) {
					t.Fatal(got.count, want)
				}
				for j, v := range want {
					x, e := got.read(uint32(j))
					memoryRuntimeRequire(t, e)
					if x != v {
						t.Fatal(x, v)
					}
				}
			}
		})
	}
}
