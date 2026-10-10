package lp

import (
	"math"
	"reflect"
	"testing"
)

func TestLPFloatEtaSparseToleranceAndOrderContracts(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	eta := NewFloatEtaMatrix(1, 4)
	eta.SetDiagonal(1)
	row := NewFloatRowEtaMatrix(0, 1, 4)
	for _, entry := range []FloatSparseEntry{{0, -1}, {2, 2}, {3, 0}} {
		if err := eta.Append(entry.Index, entry.Value); err != nil {
			t.Fatal(err)
		}
		if err := row.Append(entry.Index, entry.Value); err != nil {
			t.Fatal(err)
		}
	}
	for _, world := range []struct {
		tail  FloatTailMatrix
		right bool
		want  []float64
		index []uint32
	}{
		{eta, false, []float64{0, 1, 2, 0}, []uint32{1, 2}},
		{eta, true, []float64{1, 0, 0, 0}, []uint32{0}},
		{row, false, []float64{1, 0, 0, 0}, []uint32{0}},
		{row, true, []float64{0, 1, 2, 0}, []uint32{1, 2}},
	} {
		w := &IndexedFloatVector{Data: []float64{1, 1, 0, 0}, Index: []uint32{0, 1}}
		if world.right {
			err = world.tail.ApplyRightIndexed(w)
		} else {
			err = world.tail.ApplyLeftToT(w, s)
		}
		if err != nil || !reflect.DeepEqual(w.Index, world.index) {
			t.Fatal(err, w.Index, world.index)
		}
		lpFloatCheckVector(t, w.Data, world.want)
	}
	// Configured left tolerance and the fixed right tolerance are distinct.
	s.DropTolerance = 1
	eta = NewFloatEtaMatrix(0, 2)
	eta.SetDiagonal(2)
	if err := eta.Append(1, 1e-8); err != nil {
		t.Fatal(err)
	}
	left, right := lpFloatSparseFromDense([]float64{1, 0}), lpFloatSparseFromDense([]float64{1, 0})
	if err := eta.ApplyLeftToT(left, s); err != nil {
		t.Fatal(err)
	}
	if err := eta.ApplyRightIndexed(right); err != nil {
		t.Fatal(err)
	}
	lpFloatCheckVector(t, left.Data, []float64{0, 0})
	lpFloatCheckVector(t, right.Data, []float64{0.5, 0})
	if len(left.Index) != 0 || !reflect.DeepEqual(right.Index, []uint32{0}) {
		t.Fatal(left.Index, right.Index)
	}
	row = NewFloatRowEtaMatrix(0, 0, 2)
	if err := row.Append(1, 1e-8); err != nil {
		t.Fatal(err)
	}
	left = lpFloatSparseFromDense([]float64{0, 1})
	right = lpFloatSparseFromDense([]float64{1, 0})
	if err := row.ApplyLeftToT(left, s); err != nil {
		t.Fatal(err)
	}
	if err := row.ApplyRightIndexed(right); err != nil {
		t.Fatal(err)
	}
	lpFloatCheckVector(t, left.Data, []float64{0, 1})
	lpFloatCheckVector(t, right.Data, []float64{1, 1e-8})
	if !reflect.DeepEqual(right.Index, []uint32{0, 1}) {
		t.Fatal(right.Index)
	}
	// Strict boundary: exactly eps survives, its immediate neighbor is dropped.
	s.DropTolerance = 1e-14
	for _, value := range []float64{1e-14, -1e-14, math.Nextafter(1e-14, 0), -math.Nextafter(1e-14, 0)} {
		eta = NewFloatEtaMatrix(0, 2)
		eta.SetDiagonal(1)
		if err := eta.Append(1, value); err != nil {
			t.Fatal(err)
		}
		row = NewFloatRowEtaMatrix(0, 0, 2)
		if err := row.Append(1, value); err != nil {
			t.Fatal(err)
		}
		w := lpFloatSparseFromDense([]float64{1, 0})
		if err := eta.ApplyLeftToT(w, s); err != nil {
			t.Fatal(err)
		}
		want := value
		if math.Abs(value) < 1e-14 {
			want = 0
		}
		if w.Data[1] != want {
			t.Fatal(value, w.Data)
		}
		w = lpFloatSparseFromDense([]float64{1, 0})
		if err := row.ApplyRightIndexed(w); err != nil || w.Data[1] != want {
			t.Fatal(value, err, w.Data)
		}
	}
	// Zero epsilon admits explicit sparse zeros through the source strict test.
	s.DropTolerance = 0
	eta = NewFloatEtaMatrix(0, 2)
	eta.SetDiagonal(1)
	if err := eta.Append(1, 0); err != nil {
		t.Fatal(err)
	}
	w := lpFloatSparseFromDense([]float64{1, 0})
	if err := eta.ApplyLeftToT(w, s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Index, []uint32{0, 1}) || w.Data[1] != 0 {
		t.Fatal(w)
	}
	row = NewFloatRowEtaMatrix(0, 0, 2)
	if err := row.Append(1, 0); err != nil {
		t.Fatal(err)
	}
	w = lpFloatSparseFromDense([]float64{0, 1})
	if err := row.ApplyLeftToT(w, s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Index, []uint32{1, 0}) || w.Data[0] != 0 {
		t.Fatal(w)
	}
	// was_in_index is measured after division, not from the actual index set.
	eta = NewFloatEtaMatrix(1, 2)
	eta.SetDiagonal(math.MaxFloat64)
	if err := eta.Append(0, 1); err != nil {
		t.Fatal(err)
	}
	w = &IndexedFloatVector{Data: []float64{1, 1e-300}, Index: []uint32{1, 0}}
	if err := eta.ApplyRightIndexed(w); err != nil {
		t.Fatal(err)
	}
	if w.Data[1] != 1 || !reflect.DeepEqual(w.Index, []uint32{1, 0, 1}) {
		t.Fatal(w)
	}
	// Row sparse-left publishes zero before an unsupported erase(end).
	s.DropTolerance = 1e-14
	row = NewFloatRowEtaMatrix(0, 0, 2)
	if err := row.Append(1, -1); err != nil {
		t.Fatal(err)
	}
	w = &IndexedFloatVector{Data: []float64{1, 1}, Index: []uint32{1}}
	if err := row.ApplyLeftToT(w, s); err != ErrBounds || w.Data[0] != 0 || !reflect.DeepEqual(w.Index, []uint32{1}) {
		t.Fatal(err, w)
	}
}
func TestLPFloatEtaBoundsAndConjugationPrefixes(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	eta := NewFloatEtaMatrix(1, 3)
	eta.SetDiagonal(2)
	row := NewFloatRowEtaMatrix(0, 1, 3)
	for _, entry := range []FloatSparseEntry{{0, 3}, {99, 4}} {
		if err := eta.Append(entry.Index, entry.Value); err != nil {
			t.Fatal(err)
		}
		if err := row.Append(entry.Index, entry.Value); err != nil {
			t.Fatal(err)
		}
	}
	for _, world := range []struct {
		tail          FloatTailMatrix
		right, sparse bool
		want          []float64
	}{
		{eta, false, false, []float64{7, 2, 0}}, {eta, false, true, []float64{7, 1, 0}},
		{eta, true, false, []float64{1, 2, 0}}, {eta, true, true, []float64{1, 4, 0}},
		{row, false, false, []float64{1, 5, 0}}, {row, false, true, []float64{1, 2, 0}},
		{row, true, false, []float64{7, 2, 0}}, {row, true, true, []float64{7, 2, 0}},
	} {
		w := &IndexedFloatVector{Data: []float64{1, 2, 0}, Index: []uint32{0, 1}}
		if world.right {
			if world.sparse {
				err = world.tail.ApplyRightIndexed(w)
			} else {
				err = world.tail.ApplyRight(w.Data)
			}
		} else if world.sparse {
			err = world.tail.ApplyLeftToT(w, s)
		} else {
			err = world.tail.ApplyLeft(w.Data)
		}
		if err != ErrBounds {
			t.Fatal(err)
		}
		lpFloatCheckVector(t, w.Data, world.want)
		if !reflect.DeepEqual(w.Index, []uint32{0, 1}) {
			t.Fatal(w.Index)
		}
	}
	for _, tail := range []FloatTailMatrix{NewFloatEtaMatrix(99, 2), NewFloatRowEtaMatrix(0, 99, 2)} {
		if err := tail.ApplyLeft([]float64{1, 2}); err != ErrBounds {
			t.Fatal(err)
		}
		if err := tail.ApplyRight([]float64{1, 2}); err != ErrBounds {
			t.Fatal(err)
		}
		if err := tail.ApplyLeftToT(lpFloatSparseFromDense([]float64{1, 2}), s); err != ErrBounds {
			t.Fatal(err)
		}
		if err := tail.ApplyRightIndexed(lpFloatSparseFromDense([]float64{1, 2})); err != ErrBounds {
			t.Fatal(err)
		}
	}
	p, err := PermutationFromValues(2, []uint32{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	eta = NewFloatEtaMatrix(0, 2)
	row = NewFloatRowEtaMatrix(0, 0, 2)
	for _, entry := range []FloatSparseEntry{{1, 1}, {99, 2}, {1, 3}} {
		eta.Column.Append(entry.Index, entry.Value)
		if err := row.Append(entry.Index, entry.Value); err != nil {
			t.Fatal(err)
		}
	}
	if err := eta.Conjugate(p); err != ErrBounds || eta.ColumnIndex() != 1 || !reflect.DeepEqual(eta.Column.Data, []FloatSparseEntry{{0, 1}, {99, 2}, {1, 3}}) {
		t.Fatal(err, eta)
	}
	if err := row.Conjugate(p); err != ErrBounds || row.RowIndex() != 1 || !reflect.DeepEqual(row.Row.Data, []FloatSparseEntry{{1, 1}, {99, 2}, {0, 3}}) {
		t.Fatal(err, row)
	}
	eta = NewFloatEtaMatrix(99, 2)
	row = NewFloatRowEtaMatrix(0, 99, 2)
	if err := eta.Conjugate(p); err != ErrBounds || eta.ColumnIndex() != 99 {
		t.Fatal(err, eta)
	}
	if err := row.Conjugate(p); err != ErrBounds || row.RowIndex() != 99 {
		t.Fatal(err, row)
	}
}
func TestLPFloatEtaIEEEAndAssertionProfiles(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	var coefficients SparseFloatVector
	coefficients.Append(0, 2)
	coefficients.Append(1, 0)
	err = coefficients.Divide(0)
	if sourceDebugTests {
		if err != ErrAssertion || coefficients.At(0) != 2 || coefficients.At(1) != 0 {
			t.Fatal(err, coefficients)
		}
	} else if err != nil || !math.IsInf(coefficients.At(0), 1) || !math.IsNaN(coefficients.At(1)) {
		t.Fatal(err, coefficients)
	}
	eta := NewFloatEtaMatrix(0, 2)
	eta.SetDiagonal(2)
	err = eta.Append(0, 1)
	if sourceDebugTests {
		if err != ErrAssertion || eta.Column.Size() != 0 {
			t.Fatal(err, eta)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	if err := eta.Append(1, 3); err != nil {
		t.Fatal(err)
	}
	if !sourceDebugTests {
		dense, sparse := []float64{2, 0}, lpFloatSparseFromDense([]float64{2, 0})
		if err := eta.ApplyLeft(dense); err != nil {
			t.Fatal(err)
		}
		if err := eta.ApplyLeftToT(sparse, s); err != nil {
			t.Fatal(err)
		}
		lpFloatCheckVector(t, dense, []float64{2, 12})
		lpFloatCheckVector(t, sparse.Data, []float64{3, 6})
		dense = []float64{2, 0}
		sparse = lpFloatSparseFromDense(dense)
		if err := eta.ApplyRight(dense); err != nil {
			t.Fatal(err)
		}
		if err := eta.ApplyRightIndexed(sparse); err != nil {
			t.Fatal(err)
		}
		lpFloatCheckVector(t, dense, []float64{3, 0})
		lpFloatCheckVector(t, sparse.Data, []float64{2, 0})
	}
	row := NewFloatRowEtaMatrix(0, 0, 2)
	err = row.Append(0, 1)
	if sourceDebugTests {
		if err != ErrAssertion || row.Row.Size() != 0 {
			t.Fatal(err, row)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	if !sourceDebugTests {
		if err := row.Append(0, 1); err != nil {
			t.Fatal(err)
		}
		if err := row.Append(1, 3); err != nil {
			t.Fatal(err)
		}
		dense, sparse := []float64{2, 4}, lpFloatSparseFromDense([]float64{2, 4})
		if err := row.ApplyLeft(dense); err != nil {
			t.Fatal(err)
		}
		if err := row.ApplyLeftToT(sparse, s); err != nil {
			t.Fatal(err)
		}
		lpFloatCheckVector(t, dense, []float64{20, 4})
		lpFloatCheckVector(t, sparse.Data, []float64{18, 4})
	}
	eta = NewFloatEtaMatrix(99, 2)
	w := &IndexedFloatVector{Data: []float64{7, 8}}
	if err := eta.ApplyRightIndexed(w); err != nil || w.Data[0] != 7 {
		t.Fatal(err, w)
	}
	row = NewFloatRowEtaMatrix(0, 0, 2)
	if err := row.Append(1, 2); err != nil {
		t.Fatal(err)
	}
	w = &IndexedFloatVector{Data: []float64{1, 0}}
	if err := row.ApplyRightIndexed(w); err != nil || w.Data[1] != 2 || !reflect.DeepEqual(w.Index, []uint32{1}) {
		t.Fatal(err, w)
	}
	// Signed-zero early return retains the original payload and literal index.
	eta = NewFloatEtaMatrix(0, 2)
	eta.SetDiagonal(1)
	w = &IndexedFloatVector{Data: []float64{math.Copysign(0, -1), 0}, Index: []uint32{0}}
	if err := eta.ApplyLeftToT(w, s); err != nil || !math.Signbit(w.Data[0]) || len(w.Index) != 1 {
		t.Fatal(err, w)
	}
	row = NewFloatRowEtaMatrix(0, 0, 2)
	if err := row.ApplyLeftToT(w, s); err != nil || !math.Signbit(w.Data[0]) {
		t.Fatal(err, w)
	}
	eta = NewFloatEtaMatrix(0, 2)
	eta.SetDiagonal(math.NaN())
	if err := eta.Append(1, 1); err != nil {
		t.Fatal(err)
	}
	w = lpFloatSparseFromDense([]float64{1, 0})
	if err := eta.ApplyLeftToT(w, s); err != nil || !math.IsNaN(w.Data[0]) || w.Data[1] != 1 {
		t.Fatal(err, w)
	}
}
