// Source: src/util/vector.h:449-451.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// RuntimeUint32View is a borrowed captured pointer in the checked Runtime
// token domain. Same-capacity mutations are visible; reallocating or releasing
// the backing block invalidates the captured token.
type RuntimeUint32View struct {
	runtime  *Runtime
	block    Block
	offset   uint64
	overflow bool
}

func BorrowRuntimeUint32(r *Runtime, h Block, offset uint64) RuntimeUint32View {
	return RuntimeUint32View{runtime: r, block: h, offset: offset}
}

func (v RuntimeUint32View) Offset(words uint32) RuntimeUint32View {
	delta := uint64(words) * 4
	if v.offset > math.MaxUint64-delta {
		v.overflow = true
	}
	v.offset += delta
	return v
}

func (v RuntimeUint32View) wordOffset(i uint32) (uint64, error) {
	if v.runtime == nil {
		return 0, ErrRuntimeOwner
	}
	delta := uint64(i) * 4
	if v.overflow || v.offset > math.MaxUint64-delta {
		return 0, arena.ErrBounds
	}
	return v.offset + delta, nil
}

func (v RuntimeUint32View) Read(i uint32) (uint32, error) {
	off, e := v.wordOffset(i)
	if e != nil {
		return 0, e
	}
	return v.runtime.ReadUint32(v.block, off)
}

func (v RuntimeUint32View) Write(i, x uint32) error {
	off, e := v.wordOffset(i)
	if e != nil {
		return e
	}
	return v.runtime.WriteUint32(v.block, off, x)
}
