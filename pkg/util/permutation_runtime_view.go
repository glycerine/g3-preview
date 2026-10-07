// Source: src/util/permutation.h:56-91; src/util/vector.h:449-451.
// Source author: Leonardo de Moura, 2011-06-10.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"math"
)

// RuntimeUint32View is a borrowed captured raw pointer in the checked Runtime
// token domain. It retains no owner, size, shadow payload or allocation charge.
// Same-capacity mutations are visible; actual realloc/release invalidates its
// captured token. Pointer formation does not dereference provider or payload.
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
		return 0, ErrBitBounds
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

// Concrete source cycle specialization: no scratch allocation or payload clone.
func ApplyPermutationCoreRuntimeUint32(size uint32, data, p RuntimeUint32View) error {
	for i := int32(0); i < int32(size); i++ {
		v, e := p.Read(uint32(i))
		if e != nil {
			return e
		}
		if int32(v) < 0 {
			continue
		}
		j := i
		for {
			if sourceDebug && j < 0 {
				return ErrBitAssertion
			}
			v, e = p.Read(uint32(j))
			if e != nil {
				return e
			}
			pj := int32(v)
			if sourceDebug && (pj < 0 || pj >= int32(size)) {
				return ErrBitAssertion
			}
			// Native signed negation of INT_MIN is undefined. Refuse only at
			// that reached operation, before its nonexistent defined result.
			if pj == math.MinInt32 {
				return ErrBitBounds
			}
			if e = p.Write(uint32(j), uint32(-pj-1)); e != nil {
				return e
			}
			if pj == i {
				break
			}
			a, e := data.Read(uint32(j))
			if e != nil {
				return e
			}
			b, e := data.Read(uint32(pj))
			if e != nil {
				return e
			}
			if e = data.Write(uint32(j), b); e != nil {
				return e
			}
			if e = data.Write(uint32(pj), a); e != nil {
				return e
			}
			j = pj
		}
	}
	return nil
}

// ApplyPermutationRuntimeUint32 restores markers only after its destructive child
// completes. A stopped child preserves actual completed marks and swaps.
func ApplyPermutationRuntimeUint32(size uint32, data, p RuntimeUint32View) error {
	if e := ApplyPermutationCoreRuntimeUint32(size, data, p); e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		v, e := p.Read(i)
		if e != nil {
			return e
		}
		if int32(v) == math.MinInt32 {
			return ErrBitBounds
		}
		if e = p.Write(i, uint32(-int32(v)-1)); e != nil {
			return e
		}
	}
	return nil
}
