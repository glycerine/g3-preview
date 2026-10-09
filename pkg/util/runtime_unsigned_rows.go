// Source: src/util/vector.h:45–201,389–435; src/sat/sat_solver.cpp:3614–3623.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Actual vector<unsigned_vector>. The outer source8-byte pointer is a scalar
// Block slot/generation under one fixed Runtime. Cold provenance/cleanup state
// is explicitly charged, distinct from the real source raw allocation request.
// Inner payloads are actual same-Runtime unsigned vectors, never Go slices.
type RuntimeUnsignedRows struct {
	self                                                *RuntimeUnsignedRows
	runtime                                             *Runtime
	budget                                              *arena.Budget
	raw, pending, oldRaw                                Block
	rows, oldRows                                       []RuntimeUintColumn
	metadataBytes, oldMetadataBytes                     uint64
	ownerBytes                                          uint64
	appendActive                                        bool
	appendIndex, drain                                  uint32
	temporary                                           RuntimeUintColumn
	temporaryActive                                     bool
	closing, closed, stopped                            bool
	sourceDrain, sourceEnd                              uint32
	sourceCaptured, sourceFinalized, sourceAcknowledged bool
	sourceCause                                         error
}

func NewRuntimeUnsignedRows(r *Runtime, b *arena.Budget) (*RuntimeUnsignedRows, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrRuntimeVectorOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(RuntimeUnsignedRows{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	o := &RuntimeUnsignedRows{runtime: r, budget: b, ownerBytes: bytes}
	o.self = o
	return o, nil
}
func (o *RuntimeUnsignedRows) open() error {
	if o == nil || o.self != o {
		return ErrRuntimeVectorOwner
	}
	if o.closing || o.closed || o.stopped || o.appendActive || o.oldRaw != (Block{}) || o.pending != (Block{}) {
		return arena.ErrClosed
	}
	return o.runtime.CheckOpen()
}
func (o *RuntimeUnsignedRows) RowCount() (uint32, error) {
	if o == nil || o.self != o {
		return 0, ErrRuntimeVectorOwner
	}
	if o.closed {
		return 0, arena.ErrClosed
	}
	if o.raw == (Block{}) {
		return 0, nil
	}
	return o.runtime.ReadUint32(o.raw, 4)
}

// Row borrows an element for reading. The outer rows owner remains responsible
// for growing and finalizing the returned column.
func (o *RuntimeUnsignedRows) Row(i uint32) (*RuntimeUintColumn, error) {
	if e := o.open(); e != nil {
		return nil, e
	}
	if RuntimeVectorSourceDebug {
		n, e := o.RowCount()
		if e != nil {
			return nil, e
		}
		if i >= n {
			return nil, ErrRuntimeVectorAssertion
		}
	}
	slot, e := o.runtime.ReadUint32(o.raw, 8+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	gen, e := o.runtime.ReadUint32(o.raw, 12+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	if uint64(i) >= uint64(len(o.rows)) {
		return nil, arena.ErrBounds
	}
	row := &o.rows[i]
	if row.Raw.Self != &row.Raw || slot != row.Raw.Block.Slot() || gen != row.Raw.Block.Generation() {
		return nil, ErrRuntimeVectorOwner
	}
	return row, nil
}
func (o *RuntimeUnsignedRows) capacity() (uint32, error) {
	if o.raw == (Block{}) {
		return 0, nil
	}
	return o.runtime.ReadUint32(o.raw, 0)
}
func unsignedRowWrite(r *Runtime, b Block, i uint32, h Block) error {
	if e := r.WriteUint32(b, 8+8*uint64(i), h.Slot()); e != nil {
		return e
	}
	return r.WriteUint32(b, 12+8*uint64(i), h.Generation())
}

func (o *RuntimeUnsignedRows) expand() error {
	old, e := o.capacity()
	if e != nil {
		return e
	}
	if o.raw != (Block{}) && RuntimeVectorSourceDebug && old == 0 {
		return ErrRuntimeVectorAssertion
	}
	if o.raw != (Block{}) && old == 0 {
		return z3exception.NewDefaultException("Overflow encountered when expanding vector")
	}
	var n, request uint32
	if o.raw == (Block{}) {
		n, request = 2, 24
	} else {
		n, request, e = RuntimeScalarColumnNext(old, 8)
		if e != nil {
			return e
		}
		if n <= old {
			return z3exception.NewDefaultException("Overflow encountered when expanding vector")
		}
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(RuntimeUintColumn{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(RuntimeUintColumn{}).Size()) {
		return arena.ErrBounds
	}
	if e = o.budget.Reserve(bytes); e != nil {
		return e
	}
	o.stopped = true
	next := make([]RuntimeUintColumn, n)
	// Cold pending metadata is retained even if the source raw request refuses.
	o.oldRows, o.oldMetadataBytes = o.rows, o.metadataBytes
	o.rows, o.metadataBytes = next, bytes
	h, e := o.runtime.Allocate(uint64(request))
	o.pending = h
	if e != nil {
		return e
	}
	size, e := o.RowCount()
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
		o.raw, o.pending = h, Block{}
		o.sourceCaptured, o.sourceFinalized = false, false
		o.sourceDrain, o.sourceEnd = 0, 0
	} else {
		if e = o.runtime.WriteUint32(h, 4, size); e != nil {
			return e
		}
		o.oldRaw = o.raw
		o.raw, o.pending = h, Block{}
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
			if slot != src.Raw.Block.Slot() || gen != src.Raw.Block.Generation() {
				return ErrRuntimeVectorOwner
			}
			// Source move constructor takes the old inner pointer, followed by the
			// now-empty old object's destructor. No coefficient cloning occurs.
			if e = unsignedRowWrite(o.runtime, h, i, src.Raw.Block); e != nil {
				return e
			}
			o.rows[i].Raw = src.Raw
			o.rows[i].Raw.Self = &o.rows[i].Raw
			src.Raw.Block = Block{}
			src.Raw.Pending = Block{}
			if e = unsignedRowWrite(o.runtime, o.oldRaw, i, Block{}); e != nil {
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
func (o *RuntimeUnsignedRows) AppendColumn(src *RuntimeUintColumn) error {
	if e := o.open(); e != nil {
		return e
	}
	if src == nil || src.Raw.Runtime != o.runtime {
		return ErrRuntimeVectorOwner
	}
	n, e := o.RowCount()
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
	dst.Bind(o.runtime)
	if e = CopyRuntimeVectorColumn(&dst.Raw, &src.Raw, 4); e != nil {
		return e
	}
	if e = unsignedRowWrite(o.runtime, o.raw, n, dst.Raw.Block); e != nil {
		return e
	}
	if e = o.runtime.WriteUint32(o.raw, 4, n+1); e != nil {
		return e
	}
	o.appendActive = false
	return nil
}

// BeginRow retains the caller adapter's temporary unsigned vector until EndRow.
func (o *RuntimeUnsignedRows) BeginRow() (*RuntimeUintColumn, error) {
	if e := o.open(); e != nil {
		return nil, e
	}
	if o.temporaryActive {
		return nil, arena.ErrClosed
	}
	o.temporary = RuntimeUintColumn{}
	o.temporary.Bind(o.runtime)
	o.temporaryActive = true
	return &o.temporary, nil
}

// EndRow copies a completed temporary into the output and destroys the temporary.
// On failure its named allocation roots remain available to checked cleanup.
func (o *RuntimeUnsignedRows) EndRow(cause error) error {
	if o == nil || o.self != o || !o.temporaryActive {
		return ErrRuntimeVectorOwner
	}
	if cause == nil {
		cause = o.AppendColumn(&o.temporary)
	}
	if e := o.temporary.Finalize(); e != nil {
		if cause == nil {
			cause = e
		}
		return cause
	}
	o.temporaryActive = false
	return cause
}

// Source-only vector::finalize. Only actually published, constructed elements
// are destroyed; a failed last element constructor has no source destructor.
func (o *RuntimeUnsignedRows) SourceFinalize() error {
	if o == nil || o.self != o {
		return ErrRuntimeVectorOwner
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
func (o *RuntimeUnsignedRows) sourceFinalize() error {
	if o.sourceFinalized {
		return nil
	}
	if o.raw == (Block{}) {
		o.sourceFinalized = true
		return nil
	}
	if !o.sourceCaptured {
		n, e := o.RowCount()
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
		var row *RuntimeUintColumn
		if uint64(i) < uint64(len(o.rows)) {
			v := &o.rows[i]
			if v.Raw.Self != nil && v.Raw.Block.Slot() == slot && v.Raw.Block.Generation() == gen {
				row = v
			}
		}
		if row == nil && uint64(i) < uint64(len(o.oldRows)) {
			v := &o.oldRows[i]
			if v.Raw.Self != nil && v.Raw.Block.Slot() == slot && v.Raw.Block.Generation() == gen {
				row = v
			}
		}
		if row == nil {
			return ErrRuntimeVectorOwner
		}
		if e = row.Finalize(); e != nil {
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
func (o *RuntimeUnsignedRows) CloseChecked() error {
	if o == nil || o.self != o {
		return ErrRuntimeVectorOwner
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
func (o *RuntimeUnsignedRows) CleanupStopped() error {
	if o == nil || o.self != o {
		return ErrRuntimeVectorOwner
	}
	if !o.closing || o.sourceCause == nil {
		return ErrRuntimeVectorOwner
	}
	o.sourceAcknowledged = true
	return o.owningAbort()
}
func (o *RuntimeUnsignedRows) owningAbort() error {
	if o.temporaryActive {
		if e := o.temporary.Finalize(); e != nil {
			return e
		}
		o.temporaryActive = false
	}
	// After source finalization, drain only retained failed-copy/move owners.
	for o.drain < uint32(len(o.rows)) {
		if o.rows[o.drain].Raw.Self != nil {
			if e := o.rows[o.drain].Finalize(); e != nil {
				return e
			}
		}
		o.drain++
	}
	for i := range o.oldRows {
		if o.oldRows[i].Raw.Self != nil {
			if e := o.oldRows[i].Finalize(); e != nil {
				return e
			}
		}
	}
	for _, p := range [3]*Block{&o.oldRaw, &o.raw, &o.pending} {
		if *p != (Block{}) {
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
