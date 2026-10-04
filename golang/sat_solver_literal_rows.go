// Source: src/util/vector.h:45–201,389–435; src/sat/sat_solver.cpp:3614–3623.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Actual vector<literal_vector>. The outer source8-byte pointer is a scalar
// Block slot/generation under one fixed Runtime. Cold provenance/cleanup state
// is explicitly charged, distinct from the real source raw allocation request.
// Inner payloads are actual same-Runtime unsigned vectors, never Go slices.
type satSolverLiteralRows struct {
	self                                                *satSolverLiteralRows
	runtime                                             *memory.Runtime
	budget                                              *arena.Budget
	raw, pending, oldRaw                                memory.Block
	rows, oldRows                                       []satSolverRuntimeUintColumn
	metadataBytes, oldMetadataBytes                     uint64
	ownerBytes                                          uint64
	appendActive                                        bool
	appendIndex, drain                                  uint32
	temporary                                           satSolverRuntimeUintColumn
	temporaryActive                                     bool
	closing, closed, stopped                            bool
	sourceDrain, sourceEnd                              uint32
	sourceCaptured, sourceFinalized, sourceAcknowledged bool
	sourceCause                                         error
}

func newSATSolverLiteralRows(r *memory.Runtime, b *arena.Budget) (*satSolverLiteralRows, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(satSolverLiteralRows{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	o := &satSolverLiteralRows{runtime: r, budget: b, ownerBytes: bytes}
	o.self = o
	return o, nil
}
func (o *satSolverLiteralRows) open() error {
	if o == nil || o.self != o {
		return ErrSATOwner
	}
	if o.closing || o.closed || o.stopped || o.appendActive || o.oldRaw != (memory.Block{}) || o.pending != (memory.Block{}) {
		return arena.ErrClosed
	}
	return o.runtime.CheckOpen()
}
func (o *satSolverLiteralRows) rowCount() (uint32, error) {
	if o == nil || o.self != o {
		return 0, ErrSATOwner
	}
	if o.closed {
		return 0, arena.ErrClosed
	}
	if o.raw == (memory.Block{}) {
		return 0, nil
	}
	return o.runtime.ReadUint32(o.raw, 4)
}
func (o *satSolverLiteralRows) capacity() (uint32, error) {
	if o.raw == (memory.Block{}) {
		return 0, nil
	}
	return o.runtime.ReadUint32(o.raw, 0)
}
func satLiteralRowWrite(r *memory.Runtime, b memory.Block, i uint32, h memory.Block) error {
	if e := r.WriteUint32(b, 8+8*uint64(i), h.Slot()); e != nil {
		return e
	}
	return r.WriteUint32(b, 12+8*uint64(i), h.Generation())
}

type satSolverLiteralRowSpan struct {
	span  satSolverLiteralSpan
	count uint32
}

func (v satSolverLiteralRowSpan) read(i uint32) (satLiteral, error) {
	if satStorageSourceDebug && i >= v.count {
		return 0, ErrSATAssertion
	}
	return v.span.literal(i)
}
func (o *satSolverLiteralRows) captureRow(i uint32) (satSolverLiteralRowSpan, error) {
	if e := o.open(); e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	if satStorageSourceDebug {
		n, e := o.rowCount()
		if e != nil {
			return satSolverLiteralRowSpan{}, e
		}
		if i >= n {
			return satSolverLiteralRowSpan{}, ErrSATAssertion
		}
	}
	// Verify the reached pointer words before accessing the concrete cold owner.
	slot, e := o.runtime.ReadUint32(o.raw, 8+8*uint64(i))
	if e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	gen, e := o.runtime.ReadUint32(o.raw, 12+8*uint64(i))
	if e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	if uint64(i) >= uint64(len(o.rows)) {
		return satSolverLiteralRowSpan{}, arena.ErrBounds
	}
	row := &o.rows[i]
	if row.raw.block.Slot() != slot || row.raw.block.Generation() != gen {
		return satSolverLiteralRowSpan{}, ErrSATOwner
	}
	n, e := row.size()
	if e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	return satSolverLiteralRowSpan{satConflictRuntimeSpan(row), n}, nil
}
func (o *satSolverLiteralRows) expand() error {
	old, e := o.capacity()
	if e != nil {
		return e
	}
	if o.raw != (memory.Block{}) && satStorageSourceDebug && old == 0 {
		return ErrSATAssertion
	}
	if o.raw != (memory.Block{}) && old == 0 {
		return NewDefaultException("Overflow encountered when expanding vector")
	}
	var n, request uint32
	if o.raw == (memory.Block{}) {
		n, request = 2, 24
	} else {
		n, request, e = satScalarColumnNext(old, 8)
		if e != nil {
			return e
		}
		if n <= old {
			return NewDefaultException("Overflow encountered when expanding vector")
		}
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(satSolverRuntimeUintColumn{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(satSolverRuntimeUintColumn{}).Size()) {
		return arena.ErrBounds
	}
	if e = o.budget.Reserve(bytes); e != nil {
		return e
	}
	o.stopped = true
	next := make([]satSolverRuntimeUintColumn, n)
	// Cold pending metadata is retained even if the source raw request refuses.
	o.oldRows, o.oldMetadataBytes = o.rows, o.metadataBytes
	o.rows, o.metadataBytes = next, bytes
	h, e := o.runtime.Allocate(uint64(request))
	o.pending = h
	if e != nil {
		return e
	}
	size, e := o.rowCount()
	if e != nil {
		return e
	}
	if old == 0 {
		if e = o.runtime.WriteUint32(h, 0, n); e != nil {
			return e
		}
		if e = o.runtime.WriteUint32(h, 4, 0); e != nil {
			return e
		}
		o.raw, o.pending = h, memory.Block{}
		o.sourceCaptured, o.sourceFinalized = false, false
		o.sourceDrain, o.sourceEnd = 0, 0
	} else {
		if e = o.runtime.WriteUint32(h, 4, size); e != nil {
			return e
		}
		o.oldRaw = o.raw
		o.raw, o.pending = h, memory.Block{}
		for i := uint32(0); i < size; i++ {
			if uint64(i) >= uint64(len(o.oldRows)) {
				return arena.ErrBounds
			}
			src := &o.oldRows[i]
			// std::swap reads the actual old element pointer at this move, after
			// new outer size/pointer publication and any completed earlier moves.
			slot, e := o.runtime.ReadUint32(o.oldRaw, 8+uint64(i)*8)
			if e != nil {
				return e
			}
			gen, e := o.runtime.ReadUint32(o.oldRaw, 12+uint64(i)*8)
			if e != nil {
				return e
			}
			if slot != src.raw.block.Slot() || gen != src.raw.block.Generation() {
				return ErrSATOwner
			}
			// Source move constructor takes the old inner pointer, followed by the
			// now-empty old object's destructor. No coefficient cloning occurs.
			if e = satLiteralRowWrite(o.runtime, h, i, src.raw.block); e != nil {
				return e
			}
			o.rows[i].raw = src.raw
			o.rows[i].raw.self = &o.rows[i].raw
			src.raw.block = memory.Block{}
			src.raw.pending = memory.Block{}
			if e = satLiteralRowWrite(o.runtime, o.oldRaw, i, memory.Block{}); e != nil {
				return e
			}
		}
		if e = o.runtime.Deallocate(&o.oldRaw); e != nil {
			return e
		}
		if e = o.runtime.WriteUint32(o.raw, 0, n); e != nil {
			return e
		}
	}
	if o.oldMetadataBytes != 0 {
		o.budget.Release(o.oldMetadataBytes)
	}
	o.oldRows = nil
	o.oldMetadataBytes = 0
	o.stopped = false
	return nil
}
func (o *satSolverLiteralRows) appendColumn(src *satSolverRuntimeUintColumn) error {
	if e := o.open(); e != nil {
		return e
	}
	n, e := o.rowCount()
	if e != nil {
		return e
	}
	c, e := o.capacity()
	if e != nil {
		return e
	}
	if n == c {
		if e = o.expand(); e != nil {
			return e
		}
	}
	if uint64(n) >= uint64(len(o.rows)) {
		return arena.ErrBounds
	}
	o.appendActive, o.appendIndex = true, n
	dst := &o.rows[n]
	dst.bind(o.runtime)
	if e = satMUSCopyColumn(&dst.raw, &src.raw, 4); e != nil {
		return e
	}
	if e = satLiteralRowWrite(o.runtime, o.raw, n, dst.raw.block); e != nil {
		return e
	}
	if e = o.runtime.WriteUint32(o.raw, 4, n+1); e != nil {
		return e
	}
	o.appendActive = false
	return nil
}
func (o *satSolverLiteralRows) appendLiterals(input []satLiteral) (err error) {
	if e := o.open(); e != nil {
		return e
	}
	o.temporary = satSolverRuntimeUintColumn{}
	o.temporary.bind(o.runtime)
	o.temporaryActive = true
	// Explicit caller-array constructor adapter to the actual literal_vector.
	defer func() {
		if e := o.temporary.finalize(); e != nil {
			if err == nil {
				err = e
			}
			return
		}
		o.temporaryActive = false
	}()
	for _, v := range input {
		if err = o.temporary.push(v.index()); err != nil {
			return err
		}
	}
	return o.appendColumn(&o.temporary)
}

// Source-only vector::finalize. Only actually published, constructed elements
// are destroyed; a failed last element constructor has no source destructor.
func (o *satSolverLiteralRows) SourceFinalize() error {
	if o == nil || o.self != o {
		return ErrSATOwner
	}
	if o.closed || o.closing {
		return arena.ErrClosed
	}
	if o.sourceCause != nil {
		return o.sourceCause
	}
	e := o.sourceFinalize()
	if e != nil {
		o.stopped = true
		o.sourceCause = e
	}
	return e
}
func (o *satSolverLiteralRows) sourceFinalize() error {
	if o.sourceFinalized {
		return nil
	}
	if o.raw == (memory.Block{}) {
		o.sourceFinalized = true
		return nil
	}
	if !o.sourceCaptured {
		n, e := o.rowCount()
		if e != nil {
			return e
		}
		o.sourceEnd = n
		o.sourceCaptured = true
	}
	for o.sourceDrain < o.sourceEnd {
		i := o.sourceDrain
		slot, e := o.runtime.ReadUint32(o.raw, 8+uint64(i)*8)
		if e != nil {
			return e
		}
		gen, e := o.runtime.ReadUint32(o.raw, 12+uint64(i)*8)
		if e != nil {
			return e
		}
		var row *satSolverRuntimeUintColumn
		if uint64(i) < uint64(len(o.rows)) {
			v := &o.rows[i]
			if v.raw.self != nil && v.raw.block.Slot() == slot && v.raw.block.Generation() == gen {
				row = v
			}
		}
		if row == nil && uint64(i) < uint64(len(o.oldRows)) {
			v := &o.oldRows[i]
			if v.raw.self != nil && v.raw.block.Slot() == slot && v.raw.block.Generation() == gen {
				row = v
			}
		}
		if row == nil {
			return ErrSATOwner
		}
		if e = row.finalize(); e != nil {
			return e
		}
		o.sourceDrain++
	}
	if e := o.runtime.Deallocate(&o.raw); e != nil {
		return e
	}
	o.sourceFinalized = true
	return nil
}
func (o *satSolverLiteralRows) CloseChecked() error {
	if o == nil || o.self != o {
		return ErrSATOwner
	}
	if o.closed {
		return nil
	}
	o.closing = true
	if o.sourceCause != nil && !o.sourceAcknowledged {
		return o.sourceCause
	}
	if !o.sourceAcknowledged {
		if e := o.sourceFinalize(); e != nil {
			o.sourceCause = e
			return e
		}
	}
	return o.owningAbort()
}

// Explicit acknowledgement of a reached source destructor guard. Only actual
// remaining named roots are consumed; no header/object construction is repaired.
func (o *satSolverLiteralRows) CleanupStopped() error {
	if o == nil || o.self != o {
		return ErrSATOwner
	}
	if !o.closing || o.sourceCause == nil {
		return ErrSATOwner
	}
	o.sourceAcknowledged = true
	return o.owningAbort()
}
func (o *satSolverLiteralRows) owningAbort() error {
	if o.temporaryActive {
		if e := o.temporary.finalize(); e != nil {
			return e
		}
		o.temporaryActive = false
	}
	// After source finalization, drain only retained failed-copy/move owners.
	for o.drain < uint32(len(o.rows)) {
		if o.rows[o.drain].raw.self != nil {
			if e := o.rows[o.drain].finalize(); e != nil {
				return e
			}
		}
		o.drain++
	}
	for i := range o.oldRows {
		if o.oldRows[i].raw.self != nil {
			if e := o.oldRows[i].finalize(); e != nil {
				return e
			}
		}
	}
	for _, p := range [3]*memory.Block{&o.oldRaw, &o.raw, &o.pending} {
		if *p != (memory.Block{}) {
			if e := o.runtime.Deallocate(p); e != nil {
				return e
			}
		}
	}
	o.budget.Release(o.metadataBytes + o.oldMetadataBytes + o.ownerBytes)
	o.metadataBytes, o.oldMetadataBytes, o.ownerBytes = 0, 0, 0
	o.rows, o.oldRows = nil, nil
	o.closed = true
	return nil
}
