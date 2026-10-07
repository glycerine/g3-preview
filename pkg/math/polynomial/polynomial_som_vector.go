// Source: src/math/polynomial/polynomial.cpp:2159-2214,3875,6865.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The real Runtime pointer vector contains scalar child IDs. The charged cold
// directory binds only actual source publications to actual class owners.
// pending is the genuine allocated child whose source push has not completed.
type PolynomialSOMVector struct {
	self                                            *PolynomialSOMVector
	storage, owner                                  *MultivariatePolynomialCore
	data                                            polynomialVector
	children                                        []*PolynomialSOMBuffer
	pending                                         *PolynomialSOMBuffer
	header                                          uint64
	cursor, closeCursor                             uint32
	sourceStage                                     uint8
	sourceCause                                     error
	sourceStarted, sourceFinalized, closing, closed bool
}

func NewPolynomialSOMVector(m *MultivariatePolynomialCore) (*PolynomialSOMVector, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialSOMVector{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	v := &PolynomialSOMVector{storage: m, header: h}
	v.self = v
	v.data.width = 8
	return v, nil
}
func (v *PolynomialSOMVector) ready() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing || v.sourceStarted {
		return arena.ErrClosed
	}
	if v.pending != nil {
		return ErrPolynomialPending
	}
	return v.storage.check()
}
func (v *PolynomialSOMVector) Size() (uint32, error) {
	if e := v.ready(); e != nil {
		return 0, e
	}
	return v.data.size(v.storage)
}
func (v *PolynomialSOMVector) child(i uint32) (*PolynomialSOMBuffer, error) {
	if polynomialSourceDebug {
		n, e := v.data.size(v.storage)
		if e != nil {
			return nil, e
		}
		if i >= n {
			return nil, ErrPolynomialAssertion
		}
	}
	id, e := v.data.read64(v.storage, 8+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	if id == 0 || id > uint64(len(v.children)) {
		return nil, ErrPolynomialOwner
	}
	b := v.children[id-1]
	if b == nil {
		return nil, ErrPolynomialOwner
	}
	return b, nil
}
func (v *PolynomialSOMVector) SetOwner(m *MultivariatePolynomialCore) error {
	if e := v.ready(); e != nil {
		return e
	}
	if m != v.storage {
		return ErrPolynomialOwner
	}
	if polynomialSourceDebug && v.owner != nil && v.owner != m {
		return ErrPolynomialAssertion
	}
	if v.owner == nil {
		v.owner = m
		n, e := v.data.size(v.storage)
		if e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			b, e := v.child(i)
			if e != nil {
				return e
			}
			if e = b.SetOwner(m); e != nil {
				return e
			}
		}
	}
	return nil
}
func (v *PolynomialSOMVector) directory() error {
	if len(v.children) < cap(v.children) {
		return nil
	}
	old := cap(v.children)
	n := old + old/2 + 1
	stride := uint64(reflect.TypeOf((*PolynomialSOMBuffer)(nil)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := v.storage.integer.Budget().Reserve(uint64(n) * stride); e != nil {
		return e
	}
	a := make([]*PolynomialSOMBuffer, len(v.children), n)
	copy(a, v.children)
	v.children = a
	v.storage.integer.Budget().Release(uint64(old) * stride)
	return nil
}
func (v *PolynomialSOMVector) At(i uint32) (*PolynomialSOMBuffer, error) {
	if e := v.ready(); e != nil {
		return nil, e
	}
	n, e := v.data.size(v.storage)
	if e != nil {
		return nil, e
	}
	requested := i + 1 // source unsigned ensure_capacity(idx+1) wraps first.
	for n < requested {
		// Actual alloc(child) and owner binding precede pointer-vector push.
		b, e := NewPolynomialSOMBuffer(v.storage)
		if b != nil {
			v.pending = b
		}
		if e != nil {
			return nil, e
		}
		if v.owner != nil {
			if e = b.SetOwner(v.owner); e != nil {
				return nil, e
			}
		}
		if e = v.directory(); e != nil {
			return nil, e
		}
		slot, e := v.data.appendSlot(v.storage)
		if e != nil {
			return nil, e
		}
		if e = v.data.write64(v.storage, 8+uint64(slot)*8, uint64(len(v.children)+1)); e != nil {
			return nil, e
		}
		v.children = append(v.children, b)
		v.pending = nil
		v.data.initialized = slot + 1
		if e = v.data.publishSize(v.storage, slot+1); e != nil {
			return nil, e
		}
		n = slot + 1
	}
	if polynomialSourceDebug && n < requested {
		return nil, ErrPolynomialAssertion
	}
	return v.child(i)
}
func (v *PolynomialSOMVector) ResetN(n uint32) error {
	if e := v.ready(); e != nil {
		return e
	}
	old, e := v.data.size(v.storage)
	if e != nil {
		return e
	}
	if n > old {
		n = old
	}
	for i := uint32(0); i < n; i++ {
		b, e := v.child(i)
		if e != nil {
			return e
		}
		if e = b.Reset(); e != nil {
			return e
		}
	}
	return nil
}
func (v *PolynomialSOMVector) Reset() error {
	if e := v.ready(); e != nil {
		return e
	}
	n, e := v.data.size(v.storage)
	if e != nil {
		return e
	}
	return v.ResetN(n)
}
func (v *PolynomialSOMVector) SourceFinalize() error {
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
	fail := func(e error) error { v.sourceCause = e; return e }
	if v.sourceStage == 0 {
		n, e := v.data.size(v.storage)
		if e != nil {
			return fail(e)
		}
		for v.cursor < n {
			b, e := v.child(v.cursor)
			if e != nil {
				return fail(e)
			}
			if e = b.SourceFinalize(); e != nil {
				return fail(e)
			}
			v.cursor++
		}
		v.sourceStage++
	}
	if v.sourceStage == 1 {
		if v.data.raw != (util.Block{}) {
			if e := v.storage.memory.Deallocate(&v.data.raw); e != nil {
				return fail(e)
			}
		}
		v.sourceStage++
	}
	v.sourceFinalized = true
	return nil
}
func (v *PolynomialSOMVector) CloseChecked() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	for v.closeCursor < uint32(len(v.children)) {
		b := v.children[v.closeCursor]
		if b != nil {
			if e := b.CloseChecked(); e != nil {
				return e
			}
			v.children[v.closeCursor] = nil
		}
		v.closeCursor++
	}
	if v.pending != nil {
		if e := v.pending.CloseChecked(); e != nil {
			return e
		}
		v.pending = nil
	}
	if e := v.data.close(v.storage); e != nil {
		return e
	}
	v.storage.integer.Budget().Release(v.header + uint64(cap(v.children))*uint64(reflect.TypeOf((*PolynomialSOMBuffer)(nil)).Size()))
	v.header = 0
	v.children = nil
	v.closed = true
	return nil
}
