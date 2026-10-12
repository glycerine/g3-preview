package productiter

import (
	"math"
	"reflect"
	"testing"
)

func TestSourceProductIteratorOrderCarryAndWrap(t *testing.T) {
	sizes := []uint32{2, 3}
	coordinates := []uint32{0, 0}
	want := [][2]uint32{{1, 0}, {0, 1}, {1, 1}, {0, 2}, {1, 2}}
	for _, next := range want {
		if !Next(sizes, coordinates) || !reflect.DeepEqual(coordinates, next[:]) {
			t.Fatalf("next coordinate: got %v, want %v", coordinates, next)
		}
	}
	if Next(sizes, coordinates) || !reflect.DeepEqual(coordinates, []uint32{0, 0}) {
		t.Fatalf("terminal carry: %v", coordinates)
	}
	wrapped := []uint32{math.MaxUint32}
	if !Next([]uint32{3}, wrapped) || wrapped[0] != 0 {
		t.Fatalf("unsigned wrap: %v", wrapped)
	}
	if Next(nil, nil) {
		t.Fatal("empty product has no next coordinate")
	}
}
