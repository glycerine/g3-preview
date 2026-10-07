// Source: src/util/vector.h: source 3/2 capacity growth.
package util

import "github.com/glycerine/g3/pkg/arena"

func SourceVectorCapacity(old int, stride uint64) (int, error) {
	if stride == 0 {
		return 0, arena.ErrConfig
	}
	if old < 0 || uint64(old) > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	n := uint64(2)
	if old != 0 {
		n = (3*uint64(old) + 1) >> 1
	}
	if n <= uint64(old) || n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/stride {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
