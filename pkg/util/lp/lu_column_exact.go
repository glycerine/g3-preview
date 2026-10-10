// Source: src/util/lp/lu.hpp:193-229,260-276,503-523; exact T columns.
package lp

// The source multiplication uses unsigned arithmetic before comparing sizes.
func luSparseColumnRoute(indexSize, dataSize, ratio uint32) bool {
	return indexSize*ratio < dataSize
}

// assignDenseData implements the live whole-data assignment in solve_Bd.
// It deep-copies every rational, leaving the caller's index untouched until
// the LU algorithm clears it. Each copy's numerical receipts belong to d.
// Native Go storage replaces source vector allocation/layout mechanics.
func (d *IndexedRationalVector) assignDenseData(w *IndexedRationalVector) error {
	if d.manager != w.manager {
		return ErrExactVectorOwner
	}
	if d.calls.pending() || w.calls.pending() {
		return ErrExactVectorPending
	}
	if d == w {
		return nil
	}
	if err := d.resizeData(w.DataSize()); err != nil {
		return err
	}
	for i := range w.Data {
		if err := d.copySlot(&d.Data[i], &w.Data[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *RationalLU) PrepareEntering(entering uint32, w *IndexedRationalVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = w.Clear(); err != nil {
		return err
	}
	if err = m.A.CopyColumnToIndexedVector(entering, w); err != nil {
		return err
	}
	for _, tail := range m.Tail {
		if err = tail.ApplyLeftToT(w, m.Settings); err != nil {
			return err
		}
	}
	return m.Q.ApplyReverseLeftIndexed(w)
}

func (m *RationalLU) solveColumnIndexed(d *IndexedRationalVector) error {
	var active []uint32
	if err := m.U.SolveUYIndexed(d, m.Settings, &active); err != nil {
		return err
	}
	return m.R.ApplyReverseLeftIndexed(d)
}

func (m *RationalLU) solveColumnDense(d *IndexedRationalVector) error {
	if err := m.U.SolveUY(d.Data); err != nil {
		return err
	}
	if err := m.R.ApplyReverseLeftToT(d.Data); err != nil {
		return err
	}
	return m.appendColumnIndex(d)
}

func (m *exactLUState) appendColumnIndex(d *IndexedRationalVector) error {
	for i := m.Dim; i > 0; {
		i--
		if uint64(i) >= uint64(len(d.Data)) {
			return ErrBounds
		}
		zero, err := m.A.manager.IsZero(&d.Data[i])
		if err != nil {
			return err
		}
		if !zero {
			d.Index = append(d.Index, i)
		}
	}
	return nil
}

func (m *RationalLU) SolveBd(entering uint32, d, w *IndexedRationalVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = m.PrepareEntering(entering, w); err != nil {
		return err
	}
	if luSparseColumnRoute(w.Size(), d.DataSize(), 10) {
		if err = d.Assign(w); err != nil {
			return err
		}
		return m.solveColumnIndexed(d)
	}
	if err = d.assignDenseData(w); err != nil {
		return err
	}
	d.Index = d.Index[:0]
	return m.solveColumnDense(d)
}

func (m *PairLU) PrepareEntering(entering uint32, w *IndexedRationalVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = w.Clear(); err != nil {
		return err
	}
	if err = m.A.CopyColumnToIndexedVector(entering, w); err != nil {
		return err
	}
	for _, tail := range m.Tail {
		if err = tail.ApplyLeftToT(w, m.Settings); err != nil {
			return err
		}
	}
	return m.Q.ApplyReverseLeftIndexed(w)
}

func (m *PairLU) solveColumnIndexed(d *IndexedRationalVector) error {
	var active []uint32
	if err := m.U.SolveUYIndexed(d, m.Settings, &active); err != nil {
		return err
	}
	return m.R.ApplyReverseLeftIndexed(d)
}

func (m *PairLU) solveColumnDense(d *IndexedRationalVector) error {
	if err := m.U.SolveUY(d.Data); err != nil {
		return err
	}
	if err := m.R.ApplyReverseLeftToT(d.Data); err != nil {
		return err
	}
	return m.appendColumnIndex(d)
}

func (m *PairLU) SolveBd(entering uint32, d, w *IndexedRationalVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = m.PrepareEntering(entering, w); err != nil {
		return err
	}
	if luSparseColumnRoute(w.Size(), d.DataSize(), 10) {
		if err = d.Assign(w); err != nil {
			return err
		}
		return m.solveColumnIndexed(d)
	}
	if err = d.assignDenseData(w); err != nil {
		return err
	}
	d.Index = d.Index[:0]
	return m.solveColumnDense(d)
}

// SolveBdFaster is the retained mpq/numeric_pair caller's single-buffer route.
func (m *PairLU) SolveBdFaster(entering uint32, d *IndexedRationalVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = m.PrepareEntering(entering, d); err != nil {
		return err
	}
	return m.solveColumnIndexed(d)
}
