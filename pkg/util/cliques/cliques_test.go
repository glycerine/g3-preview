// Source: src/util/max_cliques.h:26-137.
package cliques

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestNegatedLiteralPartition(t *testing.T) {
	f := New()
	for _, edge := range [][2]uint32{{1, 3}, {1, 5}, {3, 5}} {
		if err := f.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatal(err)
		}
	}
	got, err := f.Cliques([]uint32{0, 2, 4})
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]uint32{{0, 2, 4}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSourceCachePersistsAcrossQueries(t *testing.T) {
	f := New()
	if err := f.AddEdge(1, 3); err != nil {
		t.Fatal(err)
	}
	if err := f.AddEdge(1, 5); err != nil {
		t.Fatal(err)
	}
	first, err := f.Cliques([]uint32{0, 2})
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.Cliques([]uint32{0, 4})
	if err != nil {
		t.Fatal(err)
	}
	fresh := New()
	_ = fresh.AddEdge(1, 3)
	_ = fresh.AddEdge(1, 5)
	independent, err := fresh.Cliques([]uint32{0, 4})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, [][]uint32{{0, 2}}) || len(second) != 0 || !reflect.DeepEqual(independent, [][]uint32{{0, 4}}) {
		t.Fatal(first, second, independent)
	}
}

func TestLiteralRangeGuard(t *testing.T) {
	f := New()
	if err := f.AddEdge(^uint32(0), 0); !errors.Is(err, ErrLiteralRange) {
		t.Fatal(err)
	}
}

func TestAllThreeVertexMutexGraphs(t *testing.T) {
	pairs := [3][2]uint32{{0, 1}, {0, 2}, {1, 2}}
	for mask := uint32(0); mask < 8; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			f := New()
			neighbors := [3]uint32{}
			for j, pair := range pairs {
				if mask&(1<<j) == 0 {
					continue
				}
				if err := f.AddEdge(2*pair[0]+1, 2*pair[1]+1); err != nil {
					t.Fatal(err)
				}
				neighbors[pair[0]] |= 1 << pair[1]
				neighbors[pair[1]] |= 1 << pair[0]
			}
			got, err := f.Cliques([]uint32{0, 2, 4})
			if err != nil {
				t.Fatal(err)
			}

			var want [][]uint32
			remaining := uint32(7)
			for remaining != 0 {
				candidate := remaining
				var clique []uint32
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
					want = append(want, clique)
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestMixedPolarityAndTransitiveBooleanOracle(t *testing.T) {
	edges := [][2]uint32{{1, 2}, {3, 5}}
	for _, goal := range [][]uint32{{0, 4}, {0, 3}, {0, 3, 4}, {1, 2, 5}} {
		t.Run(fmt.Sprint(goal), func(t *testing.T) {
			f := New()
			for _, edge := range edges {
				if err := f.AddEdge(edge[0], edge[1]); err != nil {
					t.Fatal(err)
				}
			}
			truth := func(assignment, literal uint32) bool {
				return assignment&(1<<(literal>>1)) != 0 != (literal&1 != 0)
			}
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
			remaining := append([]uint32(nil), goal...)
			var want [][]uint32
			for len(remaining) > 0 {
				candidate := append([]uint32(nil), remaining...)
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
					want = append(want, row)
				}
			}
			got, err := f.Cliques(goal)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
