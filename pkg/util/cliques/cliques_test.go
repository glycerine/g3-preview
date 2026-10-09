// Source: src/util/max_cliques.h:26-137.
package cliques

import (
	"errors"
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
