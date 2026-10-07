// Source: src/util/max_cliques.h; src/util/vector.h:342-387.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
)

// reserve(n) on vector<unsigned_vector> default-constructs actual empty rows.
func (o *satSolverLiteralRows) reserveEmptyRows(n uint32) error {
	if e := o.open(); e != nil {
		return e
	}
	old, e := o.rowCount()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		cap, e := o.capacity()
		if e != nil {
			return e
		}
		if cap >= n {
			break
		}
		if e = o.expand(); e != nil {
			return e
		}
	}
	// Source resize publishes requested size once, then placement-constructs the
	// ascending suffix. The cold actual row prefix names completed constructors.
	o.stopped = true
	if e = o.runtime.WriteUint32(o.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if uint64(i) >= uint64(len(o.rows)) {
			return arena.ErrBounds
		}
		o.rows[i] = RuntimeUintColumn{}
		o.rows[i].Bind(o.runtime)
		if e = satLiteralRowWrite(o.runtime, o.raw, i, o.rows[i].Raw.Block); e != nil {
			return e
		}
	}
	o.stopped = false
	return nil
}

// Concrete outer-element reference: validate scalar provenance without an
// eager inner size read. The consumer reads inner headers at its source site.
func (o *satSolverLiteralRows) unsignedRow(i uint32) (*RuntimeUintColumn, error) {
	if e := o.open(); e != nil {
		return nil, e
	}
	if satStorageSourceDebug {
		n, e := o.rowCount()
		if e != nil {
			return nil, e
		}
		if i >= n {
			return nil, ErrSATAssertion
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
		return nil, ErrSATOwner
	}
	return row, nil
}
func satCliqueAppend(dst, src *RuntimeUintColumn) error {
	for i := uint32(0); ; i++ {
		n, e := src.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := src.Read(i)
		if e != nil {
			return e
		}
		if e = dst.Push(v); e != nil {
			return e
		}
	}
}
func satCliqueRowPush(o *satSolverLiteralRows, i, v uint32) error {
	row, e := o.unsignedRow(i)
	if e != nil {
		return e
	}
	before := row.Raw.Block
	cause := row.Push(v)
	if row.Raw.Block != before {
		if e = satLiteralRowWrite(o.runtime, o.raw, i, row.Raw.Block); e != nil {
			if cause == nil {
				return e
			}
		}
	}
	return cause
}
func satCliqueRowAppend(o *satSolverLiteralRows, index uint32, src *RuntimeUintColumn) error {
	for i := uint32(0); ; i++ {
		n, e := src.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := src.Read(i)
		if e != nil {
			return e
		}
		if e = satCliqueRowPush(o, index, v); e != nil {
			return e
		}
	}
}
