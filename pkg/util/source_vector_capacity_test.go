// Source: src/util/vector.h:59-99.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSourceVectorCapacityInvalidWidthsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		old    int
		stride uint64
		want   error
	}{
		{"zero stride", 0, 0, arena.ErrConfig},
		{"negative capacity", -1, 4, arena.ErrBounds},
		{"element too wide", 0, ^uint64(0), arena.ErrBounds},
		{"capacity overflow", int(^uint(0) >> 1), 4, arena.ErrBounds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SourceVectorCapacity(tc.old, tc.stride)
			if got != 0 || !errors.Is(err, tc.want) {
				t.Fatalf("capacity = %d, error = %v; want 0, %v", got, err, tc.want)
			}
		})
	}
}
