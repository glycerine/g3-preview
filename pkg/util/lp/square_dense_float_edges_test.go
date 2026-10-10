package lp

import (
	"math"
	"reflect"
	"testing"
)

func lpFloatBlockFixture(t *testing.T, start uint32) *FloatSquareDenseSubmatrix {
	t.Helper()
	p := newFloatSparseMatrix(3)
	for i := start; i < 3; i++ {
		for j := start; j < 3; j++ {
			lpStaticMust(t, p.AddNewElement(i, j, float64(2+i+j)))
		}
	}
	m, err := NewFloatSquareDenseSubmatrix(p, start)
	lpStaticMust(t, err)
	return m
}

func TestLPFloatDenseBlockNewEntriesAndLiteralPivots(t *testing.T) {
	parent := newFloatSparseMatrix(2)
	lpStaticMust(t, parent.AddNewElement(1, 0, 9))
	m, err := NewFloatSquareDenseSubmatrix(parent, 0)
	lpStaticMust(t, err)
	m.Data = []float64{.5, 7, .2, .25}
	lpStaticMust(t, m.UpdateParent(&Settings{DropTolerance: 1e-14}))
	lpFloatSparseCheck(t, parent, map[IndexPair]float64{{0, 0}: 1, {0, 1}: 7, {1, 1}: 1})
	if !reflect.DeepEqual(m.Data, []float64{.5, 0, .2, .25}) || parent.Rows[0][0].Value != 7 {
		t.Fatal("new fill/max/L preservation", m.Data, parent.Rows)
	}
	// Row elimination drops only strict small results and writes positive zero.
	m.Data = []float64{1, 1, 1, 1 + 1e-15}
	lpStaticMust(t, m.PivotRowToRow(0, 1, &Settings{DropTolerance: 1e-14}))
	if m.Data[2] != -1 || m.Data[3] != 0 || math.Signbit(m.Data[3]) {
		t.Fatal("elimination cleanup", m.Data)
	}
	m.Data = []float64{2, 3, 4, 5}
	err = m.PivotRowToRow(0, 0, &Settings{})
	if sourceDebugTests {
		if err != ErrAssertion || !reflect.DeepEqual(m.Data, []float64{2, 3, 4, 5}) {
			t.Fatal("self pivot assertion", err, m.Data)
		}
	} else if err != nil || !reflect.DeepEqual(m.Data, []float64{-4, -3, 4, 5}) {
		t.Fatal("literal native self pivot", err, m.Data)
	}
	m.Data = []float64{0, 0, 0, 0}
	err = m.DivideRowByPivot(0)
	if sourceDebugTests {
		if err != ErrAssertion || m.Data[0] != 0 {
			t.Fatal("zero pivot assertion", err)
		}
	} else if err != nil || !math.IsInf(m.Data[0], 1) || !math.IsNaN(m.Data[1]) {
		t.Fatal("native zero pivot IEEE", err, m.Data)
	}
	m.Data = []float64{0, 1, 2, 3}
	err = m.PivotRowToRow(0, 1, &Settings{})
	if sourceDebugTests && err != ErrAssertion {
		t.Fatal("zero pivot row assertion", err)
	}
	// IEEE row products retain zero terms; no sparse-filtered dense shortcut.
	m.Data = []float64{1, 0, 0, 1}
	w := []float64{3, math.Inf(1)}
	lpStaticMust(t, m.ApplyLeft(w))
	if !math.IsNaN(w[0]) || !math.IsInf(w[1], 1) {
		t.Fatal("stored zero times infinity", w)
	}
}

func TestLPFloatDenseBlockConstructorAndProductGuards(t *testing.T) {
	for _, kind := range []string{"start", "row-map", "rows", "column-map", "earlier-column"} {
		t.Run(kind, func(t *testing.T) {
			p := newFloatSparseMatrix(2)
			lpStaticMust(t, p.AddNewElement(1, 1, 7))
			start := uint32(1)
			switch kind {
			case "start":
				start = 3
			case "row-map":
				p.RowPermutation.values = nil
			case "rows":
				p.Rows = p.Rows[:1]
			case "column-map":
				p.ColumnPermutation.values = nil
			case "earlier-column":
				lpStaticMust(t, p.AddNewElement(1, 0, 9))
			}
			m, err := NewFloatSquareDenseSubmatrix(p, start)
			want := ErrBounds
			if kind == "earlier-column" && sourceDebugTests {
				want = ErrAssertion
			}
			if err != want {
				t.Fatalf("constructor %v want %v", err, want)
			}
			if kind == "earlier-column" && m.Data[0] != 7 {
				t.Fatal("constructor prefix lost")
			}
		})
	}
	for _, kind := range []string{"cell", "find", "row-product", "column-product"} {
		m := lpFloatBlockFixture(t, 1)
		var err error
		switch kind {
		case "cell":
			_, err = m.cell(0, 1)
		case "find":
			_, err = m.FindPivotColumn(0)
		case "row-product":
			_, err = m.rowProduct(0, []float64{1, 2, 3})
		case "column-product":
			_, err = m.columnProduct(0, []float64{1, 2, 3})
		}
		want := ErrBounds
		if sourceDebugTests {
			want = ErrAssertion
		}
		if err != want {
			t.Fatalf("%s: %v want %v", kind, err, want)
		}
	}
	m := lpFloatBlockFixture(t, 0)
	m.Data = nil
	if _, err := m.FindPivotColumn(0); err != ErrBounds {
		t.Fatal("find data guard", err)
	}
	if err := m.DivideRowByPivot(0); err != ErrBounds {
		t.Fatal("divide data guard", err)
	}
	if err := m.PivotRowToRow(0, 1, &Settings{}); err != ErrBounds {
		t.Fatal("pivot data guard", err)
	}
	m = lpFloatBlockFixture(t, 0)
	before := append([]float64(nil), m.Data...)
	if err := m.ApplyLeftToT(&IndexedFloatVector{Data: []float64{1}, Index: []uint32{1}}, &Settings{}); err != ErrBounds {
		t.Fatal("left bad input", err)
	}
	if err := m.ApplyRightIndexed(&IndexedFloatVector{Data: []float64{1}, Index: []uint32{1}}); err != ErrBounds {
		t.Fatal("right bad input", err)
	}
	if !reflect.DeepEqual(before, m.Data) {
		t.Fatal("application mutated coefficients")
	}
	m.WorkVector.Index = []uint32{7}
	if err := m.ApplyRightIndexed(NewIndexedFloatVector(3)); err != ErrBounds {
		t.Fatal("old workspace clear failure", err)
	}
	m = lpFloatBlockFixture(t, 1)
	w := &IndexedFloatVector{Data: []float64{2, 3, 4}, Index: []uint32{0}}
	m.RowPermutation.values[0] = 9
	if err := m.ApplyRightIndexed(w); err != ErrBounds || w.Data[0] != 2 {
		t.Fatal("outside work assignment failure", err, w)
	}
	m = lpFloatBlockFixture(t, 1)
	m.ColumnPermutation.reverse[0] = 9
	if err := m.ApplyLeftToT(w, &Settings{}); err != ErrBounds {
		t.Fatal("outside temp assignment failure", err)
	}
	for _, apply := range []func([]float64) error{m.ApplyLeft, m.ApplyRight} {
		if err := apply(nil); err != ErrBounds {
			t.Fatal("outside dense guard", err)
		}
	}
}

func TestLPFloatDenseBlockParentAndPermutationErrorPrefixes(t *testing.T) {
	for _, kind := range []string{"parent-map", "parent-row", "parent-column-map", "cell", "other", "add-column", "add-row"} {
		t.Run(kind, func(t *testing.T) {
			m := lpFloatBlockFixture(t, 0)
			switch kind {
			case "parent-map":
				m.Parent.RowPermutation.values = nil
			case "parent-row":
				m.Parent.Rows = nil
			case "parent-column-map":
				m.Parent.ColumnPermutation.values = nil
			case "cell":
				m.Data = m.Data[:1]
			case "other":
				m.Parent.Columns[1].Values = nil
				m.Data[1] = 17
			case "add-column":
				m.Parent.Rows[0] = nil
				m.Parent.ColumnPermutation.reverse = nil
			case "add-row":
				m.Parent.Rows[0] = nil
				m.Parent.Columns = nil
			}
			if err := m.UpdateExistingParentRow(0, &Settings{DropTolerance: 1e-14}); err != ErrBounds {
				t.Fatal("update row guard", err)
			}
			if kind == "other" && m.Parent.Rows[0][1].Value != m.Data[1] {
				t.Fatal("upper row assignment not published before backlink failure")
			}
		})
	}
	m := lpFloatBlockFixture(t, 0)
	m.Parent.RowPermutation.values = nil
	if err := m.PushNewParentElements(&Settings{}); err != ErrBounds {
		t.Fatal("push row mapping guard", err)
	}
	if err := m.UpdateParent(&Settings{}); err != ErrBounds {
		t.Fatal("update propagated guard", err)
	}
	m = lpFloatBlockFixture(t, 0)
	m.Data = nil
	if err := m.PushNewParentElements(&Settings{}); err != ErrBounds {
		t.Fatal("push cell guard", err)
	}
	m = lpFloatBlockFixture(t, 0)
	m.Parent.ColumnPermutation.reverse = nil
	if err := m.PushNewParentElements(&Settings{}); err != ErrBounds {
		t.Fatal("push column mapping guard", err)
	}
	m = lpFloatBlockFixture(t, 0)
	m.Parent.Rows = nil
	if err := m.PushNewParentElements(&Settings{}); err != ErrBounds || m.Data[1] == 0 {
		t.Fatal("push add failure before coefficient clear", err)
	}
	m = lpFloatBlockFixture(t, 0)
	q := NewPermutation(3)
	q.values[2] = 3
	if err := m.Conjugate(q); err != ErrBounds {
		t.Fatal("first conjugation guard", err)
	}
	m = lpFloatBlockFixture(t, 0)
	q = NewPermutation(3)
	lpStaticMust(t, q.TransposeLeft(0, 1))
	m.ColumnPermutation = NewPermutation(4)
	err := m.Conjugate(q)
	if err == nil || !reflect.DeepEqual(m.RowPermutation.Values(), q.Values()) {
		t.Fatal("row composition prefix before column failure", err)
	}
}
