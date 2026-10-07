// Source: src/util/scoped_ptr_vector.h; src/math/polynomial/polynomial.cpp:4071.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Runtime storage is the actual pointer-vector: header and scalar child IDs.
// The charged cold directory binds those IDs to actual constructed children.
// Published records remain named owning-abort contexts after source delete;
// unpublished caller pointers are never owned or discovered through a census.
type PolynomialSkeletonVector struct {
	self                                            *PolynomialSkeletonVector
	owner                                           *MultivariatePolynomialCore
	data                                            polynomialVector
	children                                        []*PolynomialSkeleton
	header                                          uint64
	sourceCause                                     error
	cursor, closeCursor                             uint32
	sourceStarted, sourceFinalized, closing, closed bool
}

func NewPolynomialSkeletonVector(m *MultivariatePolynomialCore) (*PolynomialSkeletonVector, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialSkeletonVector{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	v := &PolynomialSkeletonVector{owner: m, header: h}
	v.self = v
	v.data.width = 8
	return v, nil
}
func (v *PolynomialSkeletonVector) ready() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceStarted {
		return arena.ErrClosed
	}
	if v.sourceCause != nil {
		return v.sourceCause
	}
	return v.owner.check()
}
func (v *PolynomialSkeletonVector) Size() (uint32, error) {
	if e := v.ready(); e != nil {
		return 0, e
	}
	return v.data.size(v.owner)
}
func (v *PolynomialSkeletonVector) pointer(i uint32) (*PolynomialSkeleton, error) {
	if polynomialSourceDebug {
		n, e := v.data.size(v.owner)
		if e != nil {
			return nil, e
		}
		if i >= n {
			return nil, ErrPolynomialAssertion
		}
	}
	id, e := v.data.read64(v.owner, 8+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	if id == 0 {
		return nil, nil
	}
	if id > uint64(len(v.children)) {
		return nil, ErrPolynomialOwner
	}
	p := v.children[id-1]
	if p == nil {
		return nil, ErrPolynomialOwner
	}
	return p, nil
}
func (v *PolynomialSkeletonVector) Get(i uint32) (*PolynomialSkeleton, error) {
	if e := v.ready(); e != nil {
		return nil, e
	}
	return v.pointer(i)
}
func (v *PolynomialSkeletonVector) directory() error {
	if len(v.children) < cap(v.children) {
		return nil
	}
	old := cap(v.children)
	next := old + old/2 + 1
	stride := uint64(reflect.TypeOf((*PolynomialSkeleton)(nil)).Size())
	if uint64(next) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := v.owner.integer.Budget().Reserve(uint64(next) * stride); e != nil {
		return e
	}
	a := make([]*PolynomialSkeleton, len(v.children), next)
	copy(a, v.children)
	v.children = a
	v.owner.integer.Budget().Release(uint64(old) * stride)
	return nil
}
func (v *PolynomialSkeletonVector) prepare(p *PolynomialSkeleton) (uint64, error) {
	if p == nil {
		return 0, nil
	}
	if p.self != p || p.owner != v.owner {
		return 0, ErrPolynomialOwner
	}
	if e := p.ready(); e != nil {
		return 0, e
	}
	if e := v.directory(); e != nil {
		return 0, e
	}
	return uint64(len(v.children) + 1), nil
}
func (v *PolynomialSkeletonVector) publish(i uint32, p *PolynomialSkeleton, id uint64) error {
	if e := v.data.write64(v.owner, 8+uint64(i)*8, id); e != nil {
		return e
	}
	if p != nil {
		v.children = append(v.children, p)
	}
	if i >= v.data.initialized {
		v.data.initialized = i + 1
	}
	return nil
}
func (v *PolynomialSkeletonVector) Push(p *PolynomialSkeleton) error {
	if e := v.ready(); e != nil {
		return e
	}
	id, e := v.prepare(p)
	if e != nil {
		return e
	}
	i, e := v.data.appendSlot(v.owner)
	if e != nil {
		return e
	}
	if e = v.publish(i, p, id); e != nil {
		return e
	}
	return v.data.publishSize(v.owner, i+1)
}
func (v *PolynomialSkeletonVector) deleteChild(p *PolynomialSkeleton) error {
	if p == nil {
		return nil
	}
	if e := p.SourceFinalize(); e != nil {
		v.sourceCause = e
		return e
	}
	return nil
}
func (v *PolynomialSkeletonVector) Set(i uint32, p *PolynomialSkeleton) error {
	if e := v.ready(); e != nil {
		return e
	}
	old, e := v.pointer(i)
	if e != nil {
		return e
	}
	if old == p {
		return nil
	}
	// Source dealloc(old) precedes pointer assignment, including the cold
	// directory reservation needed to safely represent the new Go pointer.
	if e = v.deleteChild(old); e != nil {
		return e
	}
	id, e := v.prepare(p)
	if e != nil {
		v.sourceCause = e
		return e
	}
	if e = v.publish(i, p, id); e != nil {
		v.sourceCause = e
		return e
	}
	return nil
}
func (v *PolynomialSkeletonVector) Pop() error {
	if e := v.ready(); e != nil {
		return e
	}
	n, e := v.data.size(v.owner)
	if e != nil {
		return e
	}
	if n == 0 {
		if polynomialSourceDebug {
			return ErrPolynomialAssertion
		}
		return arena.ErrBounds
	}
	if e = v.Set(n-1, nil); e != nil {
		return e
	}
	return v.data.publishSize(v.owner, n-1)
}
func (v *PolynomialSkeletonVector) Resize(n uint32) error {
	if e := v.ready(); e != nil {
		return e
	}
	old, e := v.data.size(v.owner)
	if e != nil {
		return e
	}
	if n < old {
		for i := old; i > n; {
			i--
			p, e := v.pointer(i)
			if e != nil {
				return e
			}
			if e = v.deleteChild(p); e != nil {
				return e
			}
		}
		return v.data.publishSize(v.owner, n)
	}
	for i := old; i < n; i++ {
		if e = v.Push(nil); e != nil {
			return e
		}
	}
	return nil
}
func (v *PolynomialSkeletonVector) resetSource() error {
	n, e := v.data.size(v.owner)
	if e != nil {
		return e
	}
	for v.cursor < n {
		p, e := v.pointer(v.cursor)
		if e != nil {
			return e
		}
		if e = v.deleteChild(p); e != nil {
			return e
		}
		v.cursor++
	}
	if e = v.data.reset(v.owner); e != nil {
		return e
	}
	v.cursor = 0
	return nil
}
func (v *PolynomialSkeletonVector) Reset() error {
	if e := v.ready(); e != nil {
		return e
	}
	return v.resetSource()
}
func (v *PolynomialSkeletonVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	if v.sourceCause != nil {
		return v.sourceCause
	}
	if v.sourceFinalized {
		return nil
	}
	v.sourceStarted = true
	if e := v.resetSource(); e != nil {
		v.sourceCause = e
		return e
	}
	if v.data.raw != (util.Block{}) {
		if e := v.owner.memory.Deallocate(&v.data.raw); e != nil {
			v.sourceCause = e
			return e
		}
	}
	v.sourceFinalized = true
	return nil
}
func (v *PolynomialSkeletonVector) CloseChecked() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	// These are only actual pointer publications and retained stopped source
	// destructors. No caller registry or unpublished borrowed pointer is scanned.
	for v.closeCursor < uint32(len(v.children)) {
		p := v.children[v.closeCursor]
		if p != nil {
			if e := p.CloseChecked(); e != nil {
				return e
			}
		}
		v.children[v.closeCursor] = nil
		v.closeCursor++
	}
	if e := v.data.close(v.owner); e != nil {
		return e
	}
	v.owner.integer.Budget().Release(v.header + uint64(cap(v.children))*uint64(reflect.TypeOf((*PolynomialSkeleton)(nil)).Size()))
	v.children = nil
	v.header = 0
	v.closed = true
	return nil
}
