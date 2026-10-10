//go:build g3debug

// Source: src/util/lp/square_dense_submatrix.h:222-227;
// src/util/lp/square_dense_submatrix.hpp:354-363.
package lp

import "github.com/glycerine/g3/pkg/util/numeral"

func (m *RationalSquareDenseSubmatrix) RowCount() uint32    { return m.Parent.Dimension() }
func (m *RationalSquareDenseSubmatrix) ColumnCount() uint32 { return m.RowCount() }

// Element copies the source by-value debug result into caller-owned out.
func (m *RationalSquareDenseSubmatrix) Element(i, j uint32, out *numeral.Rational) (err error) {
	b, err := m.beginWork(0)
	if err != nil {
		return err
	}
	defer m.finishWork(b, &err)
	i, err = m.AdjustRow(i)
	if err != nil {
		return err
	}
	j, err = m.AdjustColumn(j)
	if err != nil {
		return err
	}
	if i < m.IndexStart || j < m.IndexStart {
		value := numeral.NewRational()
		if i == j {
			value.Num = numeral.NewInt(1)
		}
		return m.manager.Copy(out, &value)
	}
	offset := (i-m.IndexStart)*m.Dim + j - m.IndexStart
	if uint64(offset) >= uint64(len(m.Data)) {
		return ErrBounds
	}
	return m.manager.Copy(out, &m.Data[offset])
}
