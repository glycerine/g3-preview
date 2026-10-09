// Source: src/util/vector.h:45–201; src/sat/sat_solver.cpp:3614–3623.
package sat

import "github.com/glycerine/g3/pkg/util"

// satLiteralRowStorageView captures provenance without exposing a mutable owner.
// It can inspect the retained prefix or suffix of an interrupted source move.
type satLiteralRowStorageView struct {
	runtime *util.Runtime
	Block   util.Block
}

func (v satLiteralRowStorageView) Size() (uint32, error) {
	if v.Block == (util.Block{}) {
		return 0, nil
	}
	return v.runtime.ReadUint32(v.Block, 4)
}
func (v satLiteralRowStorageView) Read(i uint32) (uint32, error) {
	if satStorageSourceDebug {
		n, e := v.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrSATAssertion
		}
	}
	return v.runtime.ReadUint32(v.Block, 8+4*uint64(i))
}

// satLiteralRowsStorageState is a cold diagnostic copy. It does not read or
// repair source headers and remains available when a source operation stops.
type satLiteralRowsStorageState struct {
	Raw, Pending, OldRaw util.Block
	Rows, OldRows        []satLiteralRowStorageView
}

func (o *satSolverLiteralRows) storageState() satLiteralRowsStorageState {
	if o == nil {
		return satLiteralRowsStorageState{}
	}
	s := satLiteralRowsStorageState{Raw: o.raw, Pending: o.pending, OldRaw: o.oldRaw}
	for i := range o.rows {
		s.Rows = append(s.Rows, satLiteralRowStorageView{o.runtime, o.rows[i].Raw.Block})
	}
	for i := range o.oldRows {
		s.OldRows = append(s.OldRows, satLiteralRowStorageView{o.runtime, o.oldRows[i].Raw.Block})
	}
	return s
}
func satLiteralColumnOwningRoots(c *util.RuntimeUintColumn, roots []util.Block) []util.Block {
	if c.Raw.Pending != (util.Block{}) {
		roots = append(roots, c.Raw.Pending)
	}
	if !c.Raw.PendingReplaces && c.Raw.Block != (util.Block{}) {
		roots = append(roots, c.Raw.Block)
	}
	return roots
}

// OwningRoots reports physical roots even when an operation or destructor stops.
// The result is a cold diagnostic copy outside the source allocation protocol.
func (o *satSolverLiteralRows) owningRoots() []util.Block {
	if o == nil {
		return nil
	}
	var roots []util.Block
	for _, h := range [3]util.Block{o.raw, o.pending, o.oldRaw} {
		if h != (util.Block{}) {
			roots = append(roots, h)
		}
	}
	for i := range o.rows {
		roots = satLiteralColumnOwningRoots(&o.rows[i], roots)
	}
	for i := range o.oldRows {
		roots = satLiteralColumnOwningRoots(&o.oldRows[i], roots)
	}
	if o.temporaryActive {
		roots = satLiteralColumnOwningRoots(&o.temporary, roots)
	}
	return roots
}

// CheckRuntime validates entry and the caller's allocation authority.
func (o *satSolverLiteralRows) checkRuntime(r *util.Runtime) error {
	if e := o.open(); e != nil {
		return e
	}
	if r != o.runtime {
		return ErrSATOwner
	}
	return nil
}
