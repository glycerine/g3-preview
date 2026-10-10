package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
)

// Remaining arithmetic/BV plugin caches use this concrete storage.
// Formatting traversal uses native slices and a map directly.
type astFormatRefVector struct{ words []uint32 }

func (v *astFormatRefVector) size() uint32 {
	if len(v.words) == 0 {
		return 0
	}
	return v.words[1]
}
func (v *astFormatRefVector) bytes() uint64 { return uint64(len(v.words)) * 4 }
func (v *astFormatRefVector) at(i uint32) (astNodeRef, error) {
	if len(v.words) == 0 || i >= v.words[1] {
		return astNodeRef{}, arena.ErrBounds
	}
	return astNodeRef{v.words[2+i*2], v.words[3+i*2]}, nil
}
func (v *astFormatRefVector) push(b *arena.Budget, r astNodeRef) error {
	size := v.size()
	var capacity uint32
	if len(v.words) != 0 {
		capacity = v.words[0]
	}
	if size == capacity {
		n := (uint64(capacity)*3 + 1) / 2
		if n < 2 {
			n = 2
		}
		if n > math.MaxUint32 || n*2+2 > uint64(^uint(0)>>1)/4 {
			return arena.ErrBounds
		}
		bytes := (n*2 + 2) * 4
		if e := b.Reserve(bytes); e != nil {
			return e
		}
		a := make([]uint32, int(n)*2+2)
		a[0] = uint32(n)
		a[1] = size
		for i := uint32(0); i < size*2; i++ {
			a[2+i] = v.words[2+i]
		}
		b.Release(v.bytes())
		v.words = a
	}
	v.words[2+size*2] = r.slot
	v.words[3+size*2] = r.generation
	v.words[1]++
	return nil
}
func (v *astFormatRefVector) pop() { v.words[1]-- }
func (v *astFormatRefVector) reset() {
	if len(v.words) != 0 {
		v.words[1] = 0
	}
}
func (v *astFormatRefVector) close(b *arena.Budget) { b.Release(v.bytes()); v.words = nil }
