// Source: src/sat/sat_scc.cpp; src/util/vector.h. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Actual scalar Runtime backing; widths replace host pointer layout explicitly.
// Header/capacity arithmetic retains selected unsigned32 source geometry.
type satSCCVector struct {
	raw, pending memory.Block
	width        uint32
	stopped      bool
}

func (v *satSCCVector) size(r *memory.Runtime) (uint32, error) {
	if v.raw == (memory.Block{}) {
		return 0, nil
	}
	return r.ReadUint32(v.raw, 4)
}
func (v *satSCCVector) cap(r *memory.Runtime) (uint32, error) {
	if v.raw == (memory.Block{}) {
		return 0, nil
	}
	return r.ReadUint32(v.raw, 0)
}
func (v *satSCCVector) grow(r *memory.Runtime) error {
	if v.stopped || v.pending != (memory.Block{}) {
		return ErrSATOwner
	}
	old, e := v.cap(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && v.raw != (memory.Block{}) && old == 0 {
		return ErrSATAssertion
	}
	n := uint32(2)
	if v.raw != (memory.Block{}) {
		n = (3*old + 1) >> 1
	}
	bytes := n*v.width + 8
	if v.raw != (memory.Block{}) && (n <= old || bytes <= old*v.width+8) {
		return NewDefaultException("Overflow encountered when expanding vector")
	}
	if v.raw == (memory.Block{}) {
		v.pending, e = r.Allocate(uint64(bytes))
		if e != nil {
			return e
		}
		if e = r.WriteUint32(v.pending, 4, 0); e != nil {
			return e
		}
		if e = r.WriteUint32(v.pending, 0, n); e != nil {
			return e
		}
		v.raw = v.pending
		v.pending = memory.Block{}
		return nil
	}
	before := v.raw
	e = r.Reallocate(&v.raw, uint64(bytes))
	if e != nil {
		v.stopped = v.raw != before
		return e
	}
	return r.WriteUint32(v.raw, 0, n)
}
func (v *satSCCVector) get(r *memory.Runtime, i, k uint32) (uint32, error) {
	if satStorageSourceDebug {
		n, e := v.size(r)
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrSATAssertion
		}
	}
	if v.width == 1 {
		x, e := r.Read(v.raw, 8+uint64(i))
		return uint32(x), e
	}
	return r.ReadUint32(v.raw, 8+uint64(i)*uint64(v.width)+uint64(k)*4)
}
func (v *satSCCVector) put(r *memory.Runtime, i, k, x uint32) error {
	if v.width == 1 {
		return r.Write(v.raw, 8+uint64(i), byte(x))
	}
	return r.WriteUint32(v.raw, 8+uint64(i)*uint64(v.width)+uint64(k)*4, x)
}
func (v *satSCCVector) push(r *memory.Runtime, x uint32) error {
	n, e := v.size(r)
	if e != nil {
		return e
	}
	c, e := v.cap(r)
	if e != nil {
		return e
	}
	if n >= c {
		if e = v.grow(r); e != nil {
			return e
		}
	}
	if e = v.put(r, n, 0, x); e != nil {
		return e
	}
	return r.WriteUint32(v.raw, 4, n+1)
}
func (v *satSCCVector) resize(r *memory.Runtime, n, x uint32) error {
	old, e := v.size(r)
	if e != nil {
		return e
	}
	if n <= old {
		if v.raw == (memory.Block{}) {
			return nil
		}
		return r.WriteUint32(v.raw, 4, n)
	}
	for {
		capacity, e := v.cap(r)
		if e != nil {
			return e
		}
		if n <= capacity {
			break
		}
		if e = v.grow(r); e != nil {
			return e
		}
	}
	// vector::resize publishes the entire requested size before constructors.
	if e = r.WriteUint32(v.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = v.put(r, i, 0, x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satSCCVector) pop(r *memory.Runtime) error {
	n, e := v.size(r)
	if e != nil {
		return e
	}
	if n == 0 {
		return arena.ErrBounds
	}
	return r.WriteUint32(v.raw, 4, n-1)
}
func (v *satSCCVector) close(r *memory.Runtime) error {
	if v.raw != (memory.Block{}) {
		if e := r.Deallocate(&v.raw); e != nil {
			return e
		}
	}
	if v.pending != (memory.Block{}) {
		if e := r.Deallocate(&v.pending); e != nil {
			return e
		}
	}
	v.stopped = false
	return nil
}
