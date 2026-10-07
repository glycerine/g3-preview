// Source: src/sat/sat_scc.cpp; src/util/vector.h.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Actual scalar Runtime backing; widths replace host pointer layout explicitly.
// Header/capacity arithmetic retains selected unsigned32 source geometry.
type satSCCVector struct {
	raw, pending util.Block
	width        uint32
	stopped      bool
}

func (v *satSCCVector) size(r *util.Runtime) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return r.ReadUint32(v.raw, 4)
}
func (v *satSCCVector) cap(r *util.Runtime) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return r.ReadUint32(v.raw, 0)
}
func (v *satSCCVector) grow(r *util.Runtime) error {
	if v.stopped || v.pending != (util.Block{}) {
		return ErrSATOwner
	}
	old, e := v.cap(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && v.raw != (util.Block{}) && old == 0 {
		return ErrSATAssertion
	}
	n := uint32(2)
	if v.raw != (util.Block{}) {
		n = (3*old + 1) >> 1
	}
	bytes := n*v.width + 8
	if v.raw != (util.Block{}) && (n <= old || bytes <= old*v.width+8) {
		return util.NewDefaultException("Overflow encountered when expanding vector")
	}
	if v.raw == (util.Block{}) {
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
		v.pending = util.Block{}
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
func (v *satSCCVector) get(r *util.Runtime, i, k uint32) (uint32, error) {
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
func (v *satSCCVector) put(r *util.Runtime, i, k, x uint32) error {
	if v.width == 1 {
		return r.Write(v.raw, 8+uint64(i), byte(x))
	}
	return r.WriteUint32(v.raw, 8+uint64(i)*uint64(v.width)+uint64(k)*4, x)
}
func (v *satSCCVector) push(r *util.Runtime, x uint32) error {
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
func (v *satSCCVector) resize(r *util.Runtime, n, x uint32) error {
	old, e := v.size(r)
	if e != nil {
		return e
	}
	if n <= old {
		if v.raw == (util.Block{}) {
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
func (v *satSCCVector) pop(r *util.Runtime) error {
	n, e := v.size(r)
	if e != nil {
		return e
	}
	if n == 0 {
		return arena.ErrBounds
	}
	return r.WriteUint32(v.raw, 4, n-1)
}
func (v *satSCCVector) close(r *util.Runtime) error {
	if v.raw != (util.Block{}) {
		if e := r.Deallocate(&v.raw); e != nil {
			return e
		}
	}
	if v.pending != (util.Block{}) {
		if e := r.Deallocate(&v.pending); e != nil {
			return e
		}
	}
	v.stopped = false
	return nil
}
