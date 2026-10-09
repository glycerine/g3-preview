// Source: src/util/parray.h:26-137.
package ast

import (
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
)

const (
	astPArraySet uint32 = iota
	astPArrayPush
	astPArrayPop
	astPArrayRoot
)

const (
	astPArrayMaxRC          uint32 = (1 << 30) - 1
	astPArrayCellBytes      uint32 = 24
	astPArrayCapacityHeader uint32 = strconv.IntSize / 8
)

func astPArrayNull(ref arena.SmallBytesRef) bool {
	return ref == (arena.SmallBytesRef{})
}
