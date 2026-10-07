// Source: src/util/bit_vector.h; src/util/bit_vector.cpp; src/util/memory_manager.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// RuntimeBitVector owns the source payload-only unsigned allocation. Capacity
// and bit count are scalar cold fields; no svector header is stored in payload.
// The cold reflected wrapper is Go ownership metadata, not a source malloc.
type RuntimeBitVector struct {
	self                                           *RuntimeBitVector
	runtime                                        *Runtime
	budget                                         *arena.Budget
	block, pending                                 Block
	numBits, capacity                              uint32
	ownerBytes                                     uint64
	pendingReplaces, destroyed, sourceDone, closed bool
	constructorFailed                              bool
	stopped, destructorStop                        error
}

func bitVectorNumWords(n uint32) uint32 { return (n + 31) / 32 }
func bitVectorRequest(r *Runtime, n uint32) uint64 {
	v := uint64(n) * 4
	if r.SizeTBits() == 32 {
		v = uint64(uint32(v))
	}
	return v
}
func NewRuntimeBitVector(r *Runtime, b *arena.Budget) (*RuntimeBitVector, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(RuntimeBitVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &RuntimeBitVector{runtime: r, budget: b, ownerBytes: n}
	v.self = v
	return v, nil
}
func (v *RuntimeBitVector) identity() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.closed || v.sourceDone || v.destructorStop != nil {
		return arena.ErrClosed
	}
	return nil
}
func (v *RuntimeBitVector) open() error {
	if e := v.identity(); e != nil {
		return e
	}
	if v.stopped != nil {
		return v.stopped
	}
	if v.pending != (Block{}) {
		return ErrRuntimeVectorOwner
	}
	return nil
}

// scalarOpen preserves source cold-field queries after completed-object body
// refusal. Uncompleted constructors and stopped/destructed owners remain invalid.
func (v *RuntimeBitVector) scalarOpen() error {
	if e := v.identity(); e != nil {
		return e
	}
	if v.constructorFailed {
		return v.stopped
	}
	return nil
}
func (v *RuntimeBitVector) Size() (uint32, error) {
	if e := v.scalarOpen(); e != nil {
		return 0, e
	}
	return v.numBits, nil
}
func (v *RuntimeBitVector) Empty() (bool, error) { n, e := v.Size(); return n == 0, e }
func (v *RuntimeBitVector) NumWords() (uint32, error) {
	n, e := v.Size()
	return bitVectorNumWords(n), e
}
func (v *RuntimeBitVector) Word(i uint32) (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.read(i)
}
func (v *RuntimeBitVector) read(i uint32) (uint32, error) {
	return v.runtime.ReadUint32(v.block, uint64(i)*4)
}
func (v *RuntimeBitVector) write(i, x uint32) error {
	return v.runtime.WriteUint32(v.block, uint64(i)*4, x)
}
func (v *RuntimeBitVector) zero(begin, end uint32, x uint32) error {
	for i := begin; i < end; i++ {
		if e := v.write(i, x); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeBitVector) allocate(n uint32) error {
	h, e := v.runtime.Allocate(bitVectorRequest(v.runtime, n))
	if e != nil {
		v.pending = h
		return e
	}
	v.block = h
	v.destroyed = false
	return nil
}
func (v *RuntimeBitVector) expandTo(n uint32) error {
	if v.block == (Block{}) {
		if e := v.allocate(n); e != nil {
			return e
		}
	} else {
		h := v.block
		if e := v.runtime.Reallocate(&h, bitVectorRequest(v.runtime, n)); e != nil {
			if h != v.block {
				v.pending = h
				v.pendingReplaces = true
			}
			return e
		}
		v.block = h
	}
	// Source publishes m_data before memset, capacity only after memset.
	// Wrapped new-old may reach outside the physical allocation; bounds are checked
	// at actual writes, without inventing the svector overflow exception.
	count := n - v.capacity
	for i := uint32(0); i < count; i++ {
		if e := v.write(v.capacity+i, 0); e != nil {
			return e
		}
	}
	v.capacity = n
	return nil
}
func NewRuntimeBitVectorReserved(r *Runtime, b *arena.Budget, n uint32) (*RuntimeBitVector, error) {
	v, e := NewRuntimeBitVector(r, b)
	if e != nil {
		return nil, e
	}
	v.capacity = bitVectorNumWords(n)
	if e = v.allocate(v.capacity); e == nil {
		e = v.zero(0, v.capacity, 0)
	}
	if e != nil {
		v.stopped = e
		v.constructorFailed = true
	}
	return v, e
}
func NewRuntimeBitVectorWords(r *Runtime, b *arena.Budget, words []uint32, n int32) (*RuntimeBitVector, error) {
	v, e := NewRuntimeBitVector(r, b)
	if e != nil {
		return nil, e
	}
	v.numBits = uint32(n)
	v.capacity = bitVectorNumWords(uint32(n))
	if e = v.allocate(v.capacity); e != nil {
		v.stopped = e
		v.constructorFailed = true
		return v, e
	}
	for i := uint32(0); i < v.capacity; i++ {
		if uint64(i) >= uint64(len(words)) {
			e = arena.ErrBounds
			break
		}
		if e = v.write(i, words[i]); e != nil {
			break
		}
	}
	if e != nil {
		v.stopped = e
		v.constructorFailed = true
	}
	return v, e
}
func (v *RuntimeBitVector) Clone(b *arena.Budget) (*RuntimeBitVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeBitVector(v.runtime, b)
	if e != nil {
		return nil, e
	}
	e = out.copyConstruct(v)
	return out, e
}

// copyConstruct is the literal base constructor body, shared with the derived
// implicit copy constructor. No class destructor runs if this body refuses.
func (out *RuntimeBitVector) copyConstruct(v *RuntimeBitVector) error {
	out.numBits = v.numBits
	out.capacity = v.capacity
	if v.block == (Block{}) {
		return nil
	}
	e := out.allocate(out.capacity)
	if e == nil {
		for i := uint32(0); i < out.capacity; i++ {
			var x uint32
			x, e = v.read(i)
			if e != nil {
				break
			}
			e = out.write(i, x)
			if e != nil {
				break
			}
		}
	}
	if e != nil {
		out.stopped = e
		out.constructorFailed = true
	}
	return e
}

func (v *RuntimeBitVector) Assign(src *RuntimeBitVector) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	v.numBits = src.numBits
	if src.block == (Block{}) {
		return nil
	}
	if v.capacity < src.capacity {
		if v.block != (Block{}) {
			h := v.block
			e := v.runtime.Deallocate(&h)
			if h == (Block{}) {
				v.destroyed = true
			}
			if e != nil {
				v.stopped = e
				return e
			}
		}
		if e := v.allocate(src.capacity); e != nil {
			v.stopped = e
			return e
		}
		v.capacity = src.capacity
	}
	for i := uint32(0); i < src.capacity; i++ {
		x, e := src.read(i)
		if e != nil {
			return e
		}
		if e = v.write(i, x); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeBitVector) Swap(other *RuntimeBitVector) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	v.block, other.block = other.block, v.block
	v.runtime, other.runtime = other.runtime, v.runtime
	v.numBits, other.numBits = other.numBits, v.numBits
	v.capacity, other.capacity = other.capacity, v.capacity
	return nil
}
func (v *RuntimeBitVector) Fill0() error {
	if e := v.open(); e != nil {
		return e
	}
	return v.zero(0, v.capacity, 0)
}
func (v *RuntimeBitVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.block != (Block{}) {
		if e := v.zero(0, v.capacity, 0); e != nil {
			return e
		}
	}
	v.numBits = 0
	return nil
}
func (v *RuntimeBitVector) Get(i uint32) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	if sourceDebug && i >= v.numBits {
		return false, ErrRuntimeVectorAssertion
	}
	w, e := v.read(i / 32)
	return w&(uint32(1)<<(i%32)) != 0, e
}
func (v *RuntimeBitVector) SetBit(i uint32) error { return v.SetBool(i, true) }
func (v *RuntimeBitVector) Unset(i uint32) error  { return v.SetBool(i, false) }
func (v *RuntimeBitVector) SetBool(i uint32, val bool) error {
	if e := v.open(); e != nil {
		return e
	}
	if sourceDebug && i >= v.numBits {
		return ErrRuntimeVectorAssertion
	}
	w, e := v.read(i / 32)
	if e != nil {
		return e
	}
	mask := uint32(1) << (i % 32)
	if val {
		w |= mask
	} else {
		w &^= mask
	}
	return v.write(i/32, w)
}
func (v *RuntimeBitVector) Push(val bool) error {
	if e := v.open(); e != nil {
		return e
	}
	idx := v.numBits
	v.numBits++
	if bitVectorNumWords(v.numBits) > v.capacity {
		n := uint32(2)
		if v.capacity != 0 {
			n = (v.capacity*3 + 1) >> 1
		}
		if e := v.expandTo(n); e != nil {
			return e
		}
	}
	return v.SetBool(idx, val)
}
func (v *RuntimeBitVector) Pop() error {
	if e := v.open(); e != nil {
		return e
	}
	if sourceDebug && v.numBits == 0 {
		return ErrRuntimeVectorAssertion
	}
	v.numBits--
	return nil
}
func (v *RuntimeBitVector) Back() (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	if sourceDebug && v.numBits == 0 {
		return false, ErrRuntimeVectorAssertion
	}
	return v.Get(v.numBits - 1)
}
func (v *RuntimeBitVector) Shrink(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if sourceDebug && n > v.numBits {
		return ErrRuntimeVectorAssertion
	}
	v.numBits = n
	return nil
}
func (v *RuntimeBitVector) Reserve(n uint32, val bool) error {
	if e := v.open(); e != nil {
		return e
	}
	if n > v.numBits {
		return v.Resize(n, val)
	}
	return nil
}
func (v *RuntimeBitVector) Resize(n uint32, val bool) error {
	if e := v.open(); e != nil {
		return e
	}
	if n <= v.numBits {
		v.numBits = n
		return nil
	}
	ew := bitVectorNumWords(n)
	if ew > v.capacity {
		if e := v.expandTo((ew*3 + 1) >> 1); e != nil {
			return e
		}
	}
	bw := v.numBits / 32
	mask := (uint32(1) << (v.numBits % 32)) - 1
	w, e := v.read(bw)
	if e != nil {
		return e
	}
	x := uint32(0)
	if val {
		w |= ^mask
		x = ^uint32(0)
	} else {
		w &= mask
	}
	if e = v.write(bw, w); e != nil {
		return e
	}
	if bw < ew {
		if e = v.zero(bw+1, ew, x); e != nil {
			return e
		}
	}
	v.numBits = n
	return nil
}
func (v *RuntimeBitVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	if v.constructorFailed {
		return v.stopped
	}
	if v.destructorStop != nil {
		return v.destructorStop
	}
	if v.sourceDone {
		return nil
	}
	if v.block != (Block{}) {
		if e := v.runtime.Deallocate(&v.block); e != nil {
			v.destructorStop = e
			return e
		}
	}
	v.sourceDone = true
	return nil
}
func (v *RuntimeBitVector) Close() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.closed {
		return nil
	}
	if v.constructorFailed {
		return v.CleanupStopped()
	}
	if e := v.SourceFinalize(); e != nil {
		return e
	}
	return v.cleanup()
}
func (v *RuntimeBitVector) cleanup() error {
	if v.pending != (Block{}) {
		if e := v.releasePending(); e != nil {
			return e
		}
	}
	v.budget.Release(v.ownerBytes)
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *RuntimeBitVector) CleanupStopped() error {
	if v == nil || v.self != v {
		return ErrRuntimeVectorOwner
	}
	if v.closed {
		return nil
	}
	if v.destroyed || v.pendingReplaces {
		v.block = Block{}
	}
	if v.block != (Block{}) {
		if e := v.runtime.Deallocate(&v.block); e != nil {
			return errors.Join(v.stopped, e)
		}
	}
	v.sourceDone = true
	return v.cleanup()
}

// RuntimeBitVectorSpan is the explicit captured source unsigned-pointer borrow.
// Constructor reads validate physical extent/initialization/lifetime only at
// each reached memcpy word, after actual allocation and caller-count fields.
type RuntimeBitVectorSpan struct {
	Runtime *Runtime
	Block   Block
	Offset  uint64
}

func NewRuntimeBitVectorSpan(r *Runtime, b *arena.Budget, span RuntimeBitVectorSpan, n int32) (*RuntimeBitVector, error) {
	v, e := NewRuntimeBitVector(r, b)
	if e != nil {
		return nil, e
	}
	v.numBits = uint32(n)
	v.capacity = bitVectorNumWords(uint32(n))
	if e = v.allocate(v.capacity); e == nil {
		for i := uint32(0); i < v.capacity; i++ {
			if span.Runtime == nil {
				e = ErrRuntimeOwner
				break
			}
			off := uint64(i) * 4
			if span.Offset > ^uint64(0)-off {
				e = arena.ErrBounds
				break
			}
			var x uint32
			x, e = span.Runtime.ReadUint32(span.Block, span.Offset+off)
			if e != nil {
				break
			}
			e = v.write(i, x)
			if e != nil {
				break
			}
		}
	}
	if e != nil {
		v.stopped = e
		v.constructorFailed = true
	}
	return v, e
}

func (v *RuntimeBitVector) releasePending() error {
	if _, e := v.runtime.Size(v.pending); e != nil {
		return v.runtime.CleanupPartial(&v.pending)
	}
	// A TLS refusal can return an actual header-complete allocation before m_data
	// publication. Its owning abort performs the genuine source deallocation.
	return v.runtime.Deallocate(&v.pending)
}
