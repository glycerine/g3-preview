// Source: src/util/lp/lu.hpp:193-229,260-276,503-523.
package lp

// PrepareEntering builds Q^-1*L_k*...*L_0*A[:,entering], the column
// consumed by the source's triangular solve and Suhl update algorithms.
func (m *FloatLU) PrepareEntering(entering uint32, w *IndexedFloatVector) (err error) {
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

func (m *FloatLU) solveColumnIndexed(d *IndexedFloatVector) error {
	if err := m.U.DoubleSolveUYIndexed(d, m.Settings); err != nil {
		return err
	}
	return m.R.ApplyReverseLeftIndexed(d)
}

func (m *FloatLU) solveColumnDense(d *IndexedFloatVector) error {
	if err := m.U.DoubleSolveUY(d.Data); err != nil {
		return err
	}
	if err := m.R.ApplyReverseLeftToT(d.Data); err != nil {
		return err
	}
	for i := m.Dim; i > 0; {
		i--
		if uint64(i) >= uint64(len(d.Data)) {
			return ErrBounds
		}
		if d.Data[i] == 0 {
			continue
		}
		if m.Settings.FloatDropSmall(d.Data[i]) {
			d.Data[i] = 0
		} else {
			d.Index = append(d.Index, i)
		}
	}
	return nil
}

// SolveBd solves B*d=A[:,entering], retaining the transformed column in w.
// The source's strict density test selects sparse assignment or a complete
// dense data copy. These differ for unindexed values and self-assignment.
func (m *FloatLU) SolveBd(entering uint32, d, w *IndexedFloatVector) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer m.finish(&err)
	if err = m.PrepareEntering(entering, w); err != nil {
		return err
	}
	if luSparseColumnRoute(w.Size(), d.DataSize(), 120) {
		if err = d.Assign(w); err != nil {
			return err
		}
		return m.solveColumnIndexed(d)
	}
	d.Data = append(d.Data[:0], w.Data...)
	d.Index = d.Index[:0]
	return m.solveColumnDense(d)
}
