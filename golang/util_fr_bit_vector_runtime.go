// Source: src/util/bit_vector.h:226-252; src/util/bit_vector.cpp:237-247; src/util/vector.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// The fast-reset derived owner embeds its base and actual unsigned-vector
// member; the reflected header charges both once, without a separate base heap.
type RuntimeFastResetBitVector struct {
	self                            *RuntimeFastResetBitVector
	base                            RuntimeBitVector
	ones                            satSolverRuntimeUintColumn
	budget                          *arena.Budget
	ownerBytes                      uint64
	closed                          bool
	sourceStage                     uint8
	stop, destructorStop            error
	constructorFailed, oneDestroyed bool
}

func NewRuntimeFastResetBitVector(r *memory.Runtime, b *arena.Budget) (*RuntimeFastResetBitVector, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(RuntimeFastResetBitVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	f := &RuntimeFastResetBitVector{budget: b, ownerBytes: n}
	f.self = f
	f.base.self = &f.base
	f.base.runtime = r
	f.base.budget = b
	f.ones.bind(r)
	return f, nil
}
func (f *RuntimeFastResetBitVector) open() error {
	if f == nil || f.self != f {
		return ErrSATOwner
	}
	if f.closed || f.sourceStage != 0 || f.destructorStop != nil {
		return arena.ErrClosed
	}
	if f.stop != nil {
		return f.stop
	}
	return f.base.open()
}
func (f *RuntimeFastResetBitVector) SetBit(i uint32) error {
	if e := f.open(); e != nil {
		return e
	}
	if e := f.ones.push(i); e != nil {
		return e
	}
	return f.base.SetBit(i)
}
func (f *RuntimeFastResetBitVector) SetBool(i uint32, val bool) error {
	if e := f.open(); e != nil {
		return e
	}
	if val {
		if e := f.ones.push(i); e != nil {
			return e
		}
	}
	return f.base.SetBool(i, val)
}
func (f *RuntimeFastResetBitVector) Push(val bool) error {
	if e := f.open(); e != nil {
		return e
	}
	if val {
		if e := f.ones.push(f.base.numBits); e != nil {
			return e
		}
	}
	return f.base.Push(val)
}
func (f *RuntimeFastResetBitVector) Fill0() error {
	if e := f.open(); e != nil {
		return e
	}
	if e := f.base.Fill0(); e != nil {
		return e
	}
	return f.ones.shrink(0)
}
func (f *RuntimeFastResetBitVector) Reset() error {
	if e := f.open(); e != nil {
		return e
	}
	sz := f.base.numBits
	n, e := f.ones.size()
	if e != nil {
		return e
	}
	raw := f.ones.raw.block
	r := f.ones.raw.runtime
	for i := uint32(0); i < n; i++ {
		idx, e := r.ReadUint32(raw, 8+uint64(i)*4)
		if e != nil {
			return e
		}
		if idx < sz {
			if e = f.base.Unset(idx); e != nil {
				return e
			}
		}
	}
	return f.ones.shrink(0)
}
func (f *RuntimeFastResetBitVector) SourceFinalize() error {
	if f == nil || f.self != f {
		return ErrSATOwner
	}
	if f.closed {
		return arena.ErrClosed
	}
	if f.constructorFailed {
		return f.stop
	}
	if f.destructorStop != nil {
		return f.destructorStop
	}
	if f.sourceStage == 0 {
		if f.ones.raw.block != (memory.Block{}) {
			if e := f.ones.raw.runtime.Deallocate(&f.ones.raw.block); e != nil {
				f.destructorStop = e
				return e
			}
		}
		f.sourceStage = 1
	}
	if f.sourceStage == 1 {
		if e := f.base.SourceFinalize(); e != nil {
			f.destructorStop = e
			return e
		}
		f.sourceStage = 2
	}
	return nil
}
func (f *RuntimeFastResetBitVector) Close() error {
	if f == nil || f.self != f {
		return ErrSATOwner
	}
	if f.closed {
		return nil
	}
	if f.constructorFailed {
		return f.CleanupStopped()
	}
	if e := f.SourceFinalize(); e != nil {
		return e
	}
	if e := f.ones.raw.finalize(); e != nil {
		return e
	}
	if e := f.base.cleanup(); e != nil {
		return e
	}
	f.budget.Release(f.ownerBytes)
	f.ownerBytes = 0
	f.closed = true
	return nil
}

func (f *RuntimeFastResetBitVector) Clone(r *memory.Runtime, b *arena.Budget) (*RuntimeFastResetBitVector, error) {
	if e := f.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeFastResetBitVector(r, b)
	if e != nil {
		return nil, e
	}
	if e = out.base.copyConstruct(&f.base); e != nil {
		out.constructorFailed = true
		out.stop = e
		return out, e
	}
	if e = intHeapCopyColumn(&out.ones.raw, &f.ones.raw); e != nil {
		// The base completed; the member copy constructor did not. Only the base
		// destructor is part of source constructor unwind, before owning abort.
		out.constructorFailed = true
		out.stop = e
		if cleanup := out.base.SourceFinalize(); cleanup != nil {
			out.stop = errors.Join(e, cleanup)
		}
		return out, out.stop
	}
	return out, nil
}
func (f *RuntimeFastResetBitVector) Assign(src *RuntimeFastResetBitVector) error {
	if e := f.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if e := f.base.Assign(&src.base); e != nil {
		f.stop = e
		return e
	}
	// vector::operator= has its own self guard after the base assignment.
	if f == src {
		return nil
	}
	if f.ones.raw.block != (memory.Block{}) {
		h := f.ones.raw.block
		e := f.ones.raw.runtime.Deallocate(&h)
		if h == (memory.Block{}) {
			f.oneDestroyed = true
		}
		if e != nil {
			f.stop = e
			return e
		}
	}
	if e := intHeapCopyColumn(&f.ones.raw, &src.ones.raw); e != nil {
		f.stop = e
		return e
	}
	f.oneDestroyed = false
	return nil
}
func (f *RuntimeFastResetBitVector) CleanupStopped() error {
	if f == nil || f.self != f {
		return ErrSATOwner
	}
	if f.closed {
		return nil
	}
	if f.oneDestroyed {
		f.ones.raw.block = memory.Block{}
	}
	if e := f.ones.raw.finalize(); e != nil {
		return e
	}
	if e := f.base.CleanupStopped(); e != nil {
		return e
	}
	f.budget.Release(f.ownerBytes)
	f.ownerBytes = 0
	f.closed = true
	return nil
}
