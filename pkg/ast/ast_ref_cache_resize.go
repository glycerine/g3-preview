// Source: src/util/util.h:318-322; src/util/vector.h:407-421.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
)

// Keep the old logical size through all geometric expansions. The backing
// stores Go manager-local references; this does not claim native pointer ABI.
func (v *astFormatRefVector) resizeNull(b *arena.Budget, needed uint32) error {
	size := v.size()
	if needed <= size {
		return nil
	}
	var capacity uint32
	if len(v.words) != 0 {
		capacity = v.words[0]
	}
	for capacity < needed {
		n := (uint64(capacity)*3 + 1) / 2
		if n < 2 {
			n = 2
		}
		if n > math.MaxUint32 || n*2+2 > uint64(^uint(0)>>1)/4 {
			return arena.ErrBounds
		}
		if e := b.Reserve((n*2 + 2) * 4); e != nil {
			return e
		}
		words := make([]uint32, int(n)*2+2)
		words[0], words[1] = uint32(n), size
		if size != 0 {
			copy(words[2:2+int(size)*2], v.words[2:2+int(size)*2])
		}
		b.Release(v.bytes())
		v.words, capacity = words, uint32(n)
	}
	for i := size; i < needed; i++ {
		v.words[2+i*2], v.words[3+i*2] = 0, 0
	}
	v.words[1] = needed
	return nil
}
