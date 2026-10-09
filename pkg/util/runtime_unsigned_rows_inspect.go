package util

// UnsignedRowStorageView captures provenance without exposing a mutable owner.
// It can inspect the retained prefix or suffix of an interrupted source move.
type UnsignedRowStorageView struct {
	runtime *Runtime
	Block   Block
}

func (v UnsignedRowStorageView) Size() (uint32, error) {
	if v.Block == (Block{}) {
		return 0, nil
	}
	return v.runtime.ReadUint32(v.Block, 4)
}
func (v UnsignedRowStorageView) Read(i uint32) (uint32, error) {
	if RuntimeVectorSourceDebug {
		n, e := v.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrRuntimeVectorAssertion
		}
	}
	return v.runtime.ReadUint32(v.Block, 8+4*uint64(i))
}

// UnsignedRowsStorageState is a cold diagnostic copy. It does not read or
// repair source headers and remains available when a source operation stops.
type UnsignedRowsStorageState struct {
	Raw, Pending, OldRaw Block
	Rows, OldRows        []UnsignedRowStorageView
}

func (o *RuntimeUnsignedRows) StorageState() UnsignedRowsStorageState {
	if o == nil {
		return UnsignedRowsStorageState{}
	}
	s := UnsignedRowsStorageState{Raw: o.raw, Pending: o.pending, OldRaw: o.oldRaw}
	for i := range o.rows {
		s.Rows = append(s.Rows, UnsignedRowStorageView{o.runtime, o.rows[i].Raw.Block})
	}
	for i := range o.oldRows {
		s.OldRows = append(s.OldRows, UnsignedRowStorageView{o.runtime, o.oldRows[i].Raw.Block})
	}
	return s
}
func columnOwningRoots(c *RuntimeUintColumn, roots []Block) []Block {
	if c.Raw.Pending != (Block{}) {
		roots = append(roots, c.Raw.Pending)
	}
	if !c.Raw.PendingReplaces && c.Raw.Block != (Block{}) {
		roots = append(roots, c.Raw.Block)
	}
	return roots
}

// OwningRoots reports physical roots even when an operation or destructor stops.
// The result is a cold diagnostic copy outside the source allocation protocol.
func (o *RuntimeUnsignedRows) OwningRoots() []Block {
	if o == nil {
		return nil
	}
	var roots []Block
	for _, h := range [3]Block{o.raw, o.pending, o.oldRaw} {
		if h != (Block{}) {
			roots = append(roots, h)
		}
	}
	for i := range o.rows {
		roots = columnOwningRoots(&o.rows[i], roots)
	}
	for i := range o.oldRows {
		roots = columnOwningRoots(&o.oldRows[i], roots)
	}
	if o.temporaryActive {
		roots = columnOwningRoots(&o.temporary, roots)
	}
	return roots
}

// CheckRuntime validates entry and the caller's allocation authority.
func (o *RuntimeUnsignedRows) CheckRuntime(r *Runtime) error {
	if e := o.open(); e != nil {
		return e
	}
	if r != o.runtime {
		return ErrRuntimeVectorOwner
	}
	return nil
}
