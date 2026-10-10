//go:build g3debug

// Source: src/util/lp/lu.h:66-82; src/util/lp/lu.hpp:96-108.
package lp

import "github.com/glycerine/g3/pkg/util/numeral"

func (m *FloatDiagonalMatrix) SetDimension(n uint32) { m.length = n }
func (m *FloatDiagonalMatrix) RowCount() uint32      { return m.length }
func (m *FloatDiagonalMatrix) ColumnCount() uint32   { return m.length }
func (m *FloatDiagonalMatrix) Element(i, j uint32) float64 {
	if i == j {
		if j == m.index {
			return m.inverse
		}
		return 1
	}
	return 0
}
func (m *RationalDiagonalMatrix) SetDimension(n uint32) { m.length = n }
func (m *RationalDiagonalMatrix) RowCount() uint32      { return m.length }
func (m *RationalDiagonalMatrix) ColumnCount() uint32   { return m.length }
func (m *RationalDiagonalMatrix) Element(i, j uint32, out *numeral.Rational) (err error) {
	b, err := m.beginWork(0)
	if err != nil {
		return err
	}
	defer m.finishWork(b, &err)
	if i == j && j == m.index {
		return m.manager.Copy(out, &m.inverse)
	}
	v := numeral.NewRational()
	if i == j {
		v.Num = numeral.NewInt(1)
	}
	return m.manager.Copy(out, &v)
}
