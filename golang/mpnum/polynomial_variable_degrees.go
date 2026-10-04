// Source: src/math/polynomial/polynomial.h:62-69; src/util/vector.h.
// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilid"
	"reflect"
)

// One real unsigned_vector with source eight-byte cap/size header. Runtime
// bytes are authoritative; there is no shadow slice or per-variable owner.
// The extra charged cold header and stopped-release cursor are Go adapters.
type PolynomialVariableDegrees struct {
	self                                                *PolynomialVariableDegrees
	runtime                                             *memory.Runtime
	constructed, rawDestroyed, copyBroken               bool
	bodyCause                                           error
	raw, pending                                        memory.Block
	header                                              uint64
	interrupted, partial, finalizing, finalized, closed bool
	sourceCause                                         error
}

func NewPolynomialVariableDegrees(r *memory.Runtime) (*PolynomialVariableDegrees, error) {
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialVariableDegrees{}).Size())
	if e := r.Budget().Reserve(h); e != nil {
		return nil, e
	}
	d := &PolynomialVariableDegrees{runtime: r, header: h, constructed: true}
	d.self = d
	return d, nil
}
func (d *PolynomialVariableDegrees) identity() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return d.runtime.CheckOpen()
}
func (d *PolynomialVariableDegrees) open() error {
	if e := d.identity(); e != nil {
		return e
	}
	// A returned failed-constructor token is an owning cleanup carrier, not
	// a completed source object. Ordinary source methods cannot use it.
	if !d.constructed {
		return ErrPolynomialPending
	}
	if d.finalizing {
		return arena.ErrClosed
	}
	return nil
}
func (d *PolynomialVariableDegrees) size() (uint32, error) {
	if d.raw == (memory.Block{}) {
		return 0, nil
	}
	return d.runtime.ReadUint32(d.raw, 4)
}
func (d *PolynomialVariableDegrees) capacity() (uint32, error) {
	if d.raw == (memory.Block{}) {
		return 0, nil
	}
	return d.runtime.ReadUint32(d.raw, 0)
}
func (d *PolynomialVariableDegrees) expand() error {
	if d.pending != (memory.Block{}) || d.interrupted {
		return ErrPolynomialPending
	}
	old, e := d.capacity()
	if e != nil {
		return e
	}
	if d.raw != (memory.Block{}) && old == 0 {
		if modularSourceDebug {
			return ErrPolynomialAssertion
		}
		return memory.Failure{Kind: memory.FailureVectorOverflow}
	}
	nc, nb, e := utilid.VectorGrowth(old)
	if e != nil {
		return e
	}
	if d.raw == (memory.Block{}) {
		d.pending, e = d.runtime.Allocate(uint64(nb))
		if e != nil {
			return e
		}
		if e = d.runtime.WriteUint32(d.pending, 0, nc); e != nil {
			return e
		}
		if e = d.runtime.WriteUint32(d.pending, 4, 0); e != nil {
			return e
		}
		d.raw = d.pending
		d.pending = memory.Block{}
		return nil
	}
	before := d.raw
	if e = d.runtime.Reallocate(&d.raw, uint64(nb)); e != nil {
		if before != d.raw {
			d.interrupted = true
			d.partial = d.runtime.SizeTBits() == 32 && uint32(nb+4) < 4
		}
		return e
	}
	return d.runtime.WriteUint32(d.raw, 0, nc)
}
func (d *PolynomialVariableDegrees) resize(n uint32) error {
	old, e := d.size()
	if e != nil {
		return e
	}
	if n <= old {
		if d.raw == (memory.Block{}) {
			return nil
		}
		return d.runtime.WriteUint32(d.raw, 4, n)
	}
	for {
		cap, e := d.capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = d.expand(); e != nil {
			return e
		}
	}
	if e = d.runtime.WriteUint32(d.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = d.runtime.WriteUint32(d.raw, uint64(8)+uint64(i)*4, 0); e != nil {
			d.interrupted = true
			return e
		}
	}
	return nil
}
func (d *PolynomialVariableDegrees) SetDegree(x, degree uint32) error {
	if e := d.open(); e != nil {
		return e
	}
	if d.interrupted || d.pending != (memory.Block{}) {
		return ErrPolynomialPending
	}
	n, e := d.size()
	if e != nil {
		return e
	}
	if x >= n {
		if e = d.resize(x + 1); e != nil {
			return e
		}
	}
	// Source setx uses m_data[x] directly. The physical guard occurs here,
	// after any defined wrapped resize/shrink publication, in both profiles.
	return d.runtime.WriteUint32(d.raw, uint64(8)+uint64(x)*4, degree)
}
func (d *PolynomialVariableDegrees) Degree(x uint32) (uint32, error) {
	if e := d.open(); e != nil {
		return 0, e
	}
	n, e := d.size()
	if e != nil {
		return 0, e
	}
	if x >= n {
		return 0, nil
	}
	return d.runtime.ReadUint32(d.raw, uint64(8)+uint64(x)*4)
}

// The implicit source member destructor sees only published m_data. Failed
// initial allocation roots belong to explicit owning abort, not source unwind.
func (d *PolynomialVariableDegrees) SourceFinalize() error {
	if e := d.identity(); e != nil {
		return e
	}
	if !d.constructed {
		return nil
	}
	if d.sourceCause != nil {
		return d.sourceCause
	}
	if d.finalized {
		return nil
	}
	d.finalizing = true
	if d.raw != (memory.Block{}) {
		if e := d.runtime.Deallocate(&d.raw); e != nil {
			d.sourceCause = e
			return e
		}
	}
	d.finalized = true
	return nil
}
func (d *PolynomialVariableDegrees) CloseChecked() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return nil
	}
	d.finalizing = true
	if d.rawDestroyed {
		d.raw = memory.Block{}
		d.rawDestroyed = false
	}
	if d.raw != (memory.Block{}) {
		var e error
		if d.partial {
			e = d.runtime.CleanupPartial(&d.raw)
		} else {
			e = d.runtime.Deallocate(&d.raw)
		}
		if e != nil {
			return e
		}
		d.partial = false
	}
	if d.pending != (memory.Block{}) {
		if e := d.runtime.Deallocate(&d.pending); e != nil {
			return e
		}
	}
	d.runtime.Budget().Release(d.header)
	d.header = 0
	d.closed = true
	return nil
}
