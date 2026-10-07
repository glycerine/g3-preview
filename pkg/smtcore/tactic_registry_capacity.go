// Source: src/util/vector.h. Licensed under the MIT license.
package smtcore

import (
	"github.com/glycerine/g3/pkg/arena"
)

func tacticRegistryCapacity(old int, stride uint64) (int, error) {
	n := uint64(2)
	if old != 0 {
		n = (3*uint64(old) + 1) >> 1
	}
	if n <= uint64(old) || n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/stride {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
