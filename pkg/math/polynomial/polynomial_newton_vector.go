// Source: src/math/polynomial/polynomial.cpp:2914-2952. Licensed under the MIT license.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The actual source pointer-vector backing is encoded as Runtime uint64 child
// IDs; the separate, fully charged cold directory owns genuine child objects.
type PolynomialNewtonVector struct {
	self                              *PolynomialNewtonVector
	owner, storage                    *MultivariatePolynomialCore
	data                              polynomialVector
	children                          []*PolynomialNewtonInterpolator
	pending                           *PolynomialNewtonInterpolator
	header                            uint64
	cursor                            uint32
	closing, closed                   bool
	sourceOrphans                     bool
	sourceFinalizing, sourceFinalized bool
	sourceStage                       uint8
	sourceCause                       error
}

func NewPolynomialNewtonVector(m *MultivariatePolynomialCore) (*PolynomialNewtonVector, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialNewtonVector{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	v := &PolynomialNewtonVector{storage: m, header: h}
	v.self = v
	v.data.width = 8
	return v, nil
}
func (v *PolynomialNewtonVector) Size() (uint32, error) {
	if v == nil || v.self != v {
		return 0, ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceFinalizing {
		return 0, arena.ErrClosed
	}
	return v.data.size(v.storage)
}
func (v *PolynomialNewtonVector) directory() error {
	if len(v.children) < cap(v.children) {
		return nil
	}
	old := cap(v.children)
	c := old + old/2 + 1
	stride := uint64(reflect.TypeOf((*PolynomialNewtonInterpolator)(nil)).Size())
	if uint64(c) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := v.storage.integer.Budget().Reserve(uint64(c) * stride); e != nil {
		return e
	}
	a := make([]*PolynomialNewtonInterpolator, len(v.children), c)
	copy(a, v.children)
	v.children = a
	v.storage.integer.Budget().Release(uint64(old) * stride)
	return nil
}
func (v *PolynomialNewtonVector) At(i uint32) (*PolynomialNewtonInterpolator, error) {
	if v == nil || v.self != v {
		return nil, ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceFinalizing {
		return nil, arena.ErrClosed
	}
	if e := v.storage.check(); e != nil {
		return nil, e
	}
	if v.pending != nil || v.cursor != 0 || v.sourceOrphans {
		return nil, ErrPolynomialPending
	}
	if v.owner == nil {
		if polynomialSourceDebug {
			return nil, ErrPolynomialAssertion
		}
		return nil, ErrPolynomialOwner
	}
	for {
		size, e := v.data.size(v.storage)
		if e != nil {
			return nil, e
		}
		if i < size {
			break
		}
		child, e := NewPolynomialNewtonInterpolator(v.owner)
		if child != nil {
			v.pending = child
		}
		if e != nil {
			return nil, e
		}
		indexCount, e := v.data.size(v.storage)
		if e != nil {
			return nil, e
		}
		if uint64(indexCount) >= uint64(len(v.children)) {
			if e = v.directory(); e != nil {
				return nil, e
			}
			v.children = append(v.children, child)
		} else {
			v.children[indexCount] = child
		}
		index, e := v.data.appendSlot(v.storage)
		if e != nil {
			return nil, e
		}
		if e = v.data.write64(v.storage, 8+uint64(index)*8, uint64(indexCount+1)); e != nil {
			return nil, e
		}
		if index >= v.data.initialized {
			v.data.initialized = index + 1
		}
		if e = v.data.publishSize(v.storage, index+1); e != nil {
			return nil, e
		}
		v.pending = nil
	}
	id, e := v.data.read64(v.storage, 8+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	if id == 0 || id > uint64(len(v.children)) {
		return nil, arena.ErrBounds
	}
	return v.children[id-1], nil
}
func (v *PolynomialNewtonVector) Flush() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceFinalizing {
		return arena.ErrClosed
	}
	return v.flushMode(false)
}
func (v *PolynomialNewtonVector) flushMode(owning bool) error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	n, e := v.data.size(v.storage)
	if e != nil {
		return e
	}
	for v.cursor < n {
		id, e := v.data.read64(v.storage, 8+uint64(v.cursor)*8)
		if e != nil {
			return e
		}
		if id == 0 || id > uint64(len(v.children)) {
			return arena.ErrBounds
		}
		if e = v.children[id-1].closeMode(owning, false); e != nil {
			return e
		}
		if v.children[id-1].closed {
			v.children[id-1] = nil
		} else {
			v.sourceOrphans = true
		}
		v.cursor++
	}
	if e = v.data.reset(v.storage); e != nil {
		return e
	}
	v.cursor = 0
	return nil
}
func (v *PolynomialNewtonVector) Close() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	if e := v.flushMode(true); e != nil {
		return e
	}
	// The source pointer-vector member dies after source child destruction.
	// Retained Go-only unjoined roots are aborted only after that boundary.
	if v.data.raw != (util.Block{}) {
		if e := v.storage.memory.Deallocate(&v.data.raw); e != nil {
			return e
		}
	}
	for i, child := range v.children {
		if child != nil {
			if e := child.Close(); e != nil {
				return e
			}
			v.children[i] = nil
		}
	}
	if v.pending != nil {
		if e := v.pending.Close(); e != nil {
			return e
		}
		v.pending = nil
	}
	if e := v.data.close(v.storage); e != nil {
		return e
	}
	v.sourceOrphans = false
	v.storage.integer.Budget().Release(uint64(cap(v.children)) * uint64(reflect.TypeOf((*PolynomialNewtonInterpolator)(nil)).Size()))
	v.children = nil
	v.storage.integer.Budget().Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}

// SetOwner follows the source nullable owner assignment. The Go allocation
// context is independent and remains the actual owner of the vector backing.
func (v *PolynomialNewtonVector) SetOwner(m *MultivariatePolynomialCore) error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceFinalizing {
		return arena.ErrClosed
	}
	if polynomialSourceDebug && v.owner != nil && v.owner != m {
		return ErrPolynomialAssertion
	}
	if m != nil {
		if e := m.check(); e != nil {
			return e
		}
	}
	v.owner = m
	return nil
}
