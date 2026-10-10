package lp

import (
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

// The standalone source test_init_U fixture is strengthened here. Its placement
// after the upstream driver's unconditional return is not credited as execution.
func TestLPFloatSparseOriginalInitU(t *testing.T) {
	a := NewFloatStaticMatrix(3, 7)
	for row := uint32(0); row < 3; row++ {
		for column := uint32(0); column < 4; column++ {
			lpStaticMust(t, a.Set(row, column, float64(10*(row+1)+column)))
		}
		lpStaticMust(t, a.Set(row, row+4, float64(10*(row+1)+4)))
	}
	m, err := NewFloatSparseBasisMatrix(a, []uint32{1, 2, 4})
	lpStaticMust(t, err)
	want := [][]float64{{11, 12, 14}, {21, 22, 0}, {31, 32, 0}}
	for i := range want {
		for j, value := range want[i] {
			got, err := m.Get(uint32(i), uint32(j))
			if err != nil || got != value {
				t.Fatal(i, j, got, value, err)
			}
		}
	}
	if m.ActiveElements() != 7 || m.Dimension() != 3 || m.pivotQueue.Size() != 0 {
		t.Fatal("initial counts")
	}
}

func TestLPFloatSparseBasisIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(154, 2026))
	type entry struct {
		row   uint32
		value float64
	}
	for world := 0; world < 600; world++ {
		n := uint32(1 + rng.IntN(8))
		a := NewFloatStaticMatrix(n, n+3)
		input := make([][]entry, n+3)
		for k := 0; k < 60; k++ {
			i, j := rng.Uint32N(n), rng.Uint32N(n+3)
			value := float64(rng.IntN(17)-8) / 4
			// AddNewElement retains explicit zero and duplicate coordinates.
			lpStaticMust(t, a.AddNewElement(i, j, value))
			input[j] = append(input[j], entry{i, value})
		}
		basis := make([]uint32, n)
		for i := range basis {
			basis[i] = rng.Uint32N(n + 3)
		}
		m, err := NewFloatSparseBasisMatrix(a, basis)
		lpStaticMust(t, err)
		rows := make([][]FloatIndexedValue, n)
		columns := make([][]FloatIndexedValue, n)
		var count uint32
		for j := int(n) - 1; j >= 0; j-- {
			for co, x := range input[basis[j]] {
				columns[j] = append(columns[j], FloatIndexedValue{x.value, x.row, uint32(len(rows[x.row]))})
				rows[x.row] = append(rows[x.row], FloatIndexedValue{x.value, uint32(j), uint32(co)})
				count++
			}
		}
		if !reflect.DeepEqual(rows, m.Rows) || m.ActiveElements() != count {
			t.Fatal("row/order oracle", world)
		}
		for j := range columns {
			if !reflect.DeepEqual(columns[j], m.Columns[j].Values) || m.Columns[j].ShortenedMarkovitz != 0 {
				t.Fatal("column oracle", world, j)
			}
			if m.WorkPivot[j] != -1 || m.Processed[j] {
				t.Fatal("workspace initialization")
			}
		}
		// Logical dense values use the first stored duplicate. Independent maps
		// track each permutation without reading either permutation's storage.
		logicalRows, logicalColumns := make([]uint32, n), make([]uint32, n)
		for i := range logicalRows {
			logicalRows[i] = uint32(i)
			logicalColumns[i] = uint32(i)
		}
		for step := 0; step < 16; step++ {
			if n > 1 {
				x := rng.Uint32N(n)
				y := (x + 1 + rng.Uint32N(n-1)) % n
				if step%2 == 0 {
					lpStaticMust(t, m.SwapRows(x, y))
					logicalRows[x], logicalRows[y] = logicalRows[y], logicalRows[x]
				} else {
					lpStaticMust(t, m.SwapColumns(x, y))
					logicalColumns[x], logicalColumns[y] = logicalColumns[y], logicalColumns[x]
				}
			}
			for i := uint32(0); i < n; i++ {
				physical, err := m.AdjustRow(i)
				if err != nil || physical != logicalRows[i] {
					t.Fatal("row mapping")
				}
				inverse, err := m.AdjustRowInverse(physical)
				if err != nil || inverse != i {
					t.Fatal("inverse row")
				}
				physical, err = m.AdjustColumn(i)
				if err != nil || physical != logicalColumns[i] {
					t.Fatal("column mapping")
				}
				inverse, err = m.AdjustColumnInverse(physical)
				if err != nil || inverse != i {
					t.Fatal("inverse column")
				}
				var lowest uint32
				for logicalRow, sourceRow := range logicalRows {
					for _, x := range input[basis[logicalColumns[i]]] {
						if x.row == sourceRow && uint32(logicalRow) > lowest {
							lowest = uint32(logicalRow)
						}
					}
				}
				got, err := m.LowestRowInColumn(i)
				if err != nil || got != lowest {
					t.Fatal("lowest", got, lowest, err)
				}
				for j := uint32(0); j < n; j++ {
					var want float64
					for _, x := range input[basis[logicalColumns[j]]] {
						if x.row == logicalRows[i] {
							want = x.value
							break
						}
					}
					got, err := m.Get(i, j)
					if err != nil || got != want {
						t.Fatal("logical value", world, i, j, got, want, err)
					}
				}
			}
		}
		// Mutating either owner does not alias the other direction or source.
		if count > 0 {
			for i := range m.Rows {
				if len(m.Rows[i]) > 0 {
					v := &m.Rows[i][0]
					original := m.Columns[v.Index].Values[v.Other].Value
					v.Value = 12345
					if m.Columns[v.Index].Values[v.Other].Value != original {
						t.Fatal("row/column value alias")
					}
					break
				}
			}
		}
	}
}

func TestLPFloatSparseBasisLiteralStops(t *testing.T) {
	empty, err := NewFloatSparseBasisMatrix(NewFloatStaticMatrix(0, 0), nil)
	if err != nil || empty.Dimension() != 0 || empty.ActiveElements() != 0 {
		t.Fatal(empty, err)
	}
	a := NewFloatStaticMatrix(2, 3)
	lpStaticMust(t, a.Set(0, 2, 7))
	m, err := NewFloatSparseBasisMatrix(a, []uint32{9, 2})
	if !errors.Is(err, ErrBounds) || m.ActiveElements() != 1 || len(m.Rows[0]) != 1 {
		t.Fatal("reverse completed prefix", m, err)
	}
	m, err = NewFloatSparseBasisMatrix(a, []uint32{2})
	if !errors.Is(err, ErrBounds) || m.ActiveElements() != 0 {
		t.Fatal("short basis", err)
	}
	// Extra basis elements are ignored, not interpreted as a dimension change.
	m, err = NewFloatSparseBasisMatrix(a, []uint32{0, 2, 99})
	lpStaticMust(t, err)
	if got, err := m.LowestRowInColumn(0); err != nil || got != 0 {
		t.Fatal(got, err)
	}
	for _, f := range []func() error{
		func() error { _, e := m.Get(2, 0); return e }, func() error { _, e := m.Get(0, 2); return e },
		func() error { _, e := m.GetNotAdjusted(2, 0); return e }, func() error { _, e := m.LowestRowInColumn(2); return e },
		func() error { return m.SwapRows(0, 2) }, func() error { return m.SwapColumns(2, 0) },
		func() error { return m.copyColumnFromStatic(2, a, 2) },
	} {
		if !errors.Is(f(), ErrBounds) {
			t.Fatal("missing bounds error")
		}
	}
	if got, err := m.GetNotAdjusted(0, 99); err != nil || got != 0 {
		t.Fatal("unadjusted absent column", got, err)
	}
	a.Columns[2] = append(a.Columns[2], ColumnCell{2, 0})
	m, err = NewFloatSparseBasisMatrix(a, []uint32{0, 2})
	if !errors.Is(err, ErrBounds) || m.ActiveElements() != 1 {
		t.Fatal("invalid row prefix", err)
	}
	a.Columns[2][1] = ColumnCell{0, 99}
	m, err = NewFloatSparseBasisMatrix(a, []uint32{0, 2})
	if !errors.Is(err, ErrBounds) || m.ActiveElements() != 1 {
		t.Fatal("invalid offset prefix", err)
	}
	m.Columns = nil
	if _, err = m.LowestRowInColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	m.Columns = make([]FloatSparseColumn, 2)
	m.Columns[0].Values = []FloatIndexedValue{{Index: 2}}
	if _, err = m.LowestRowInColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	// IEEE payloads and signed zero are copied without numerical filtering.
	a = NewFloatStaticMatrix(1, 1)
	for _, v := range []float64{math.Copysign(0, -1), math.Inf(1), math.NaN()} {
		lpStaticMust(t, a.AddNewElement(0, 0, v))
	}
	m, err = NewFloatSparseBasisMatrix(a, []uint32{0})
	lpStaticMust(t, err)
	if !math.Signbit(m.Rows[0][0].Value) || !math.IsInf(m.Rows[0][1].Value, 1) || !math.IsNaN(m.Rows[0][2].Value) {
		t.Fatal("IEEE copies")
	}
}
