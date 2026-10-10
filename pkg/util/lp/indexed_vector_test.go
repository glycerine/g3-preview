package lp

import (
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestLPIndexedFloatVectorContracts(t *testing.T) {
	v := NewIndexedFloatVector(5)
	if v.DataSize() != 5 || v.Size() != 0 || !v.IsOK() {
		t.Fatal(v)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(v.SetValue(3, 3))
	must(v.SetValue(1, 1))
	if !reflect.DeepEqual(v.Index, []uint32{3, 1}) {
		t.Fatal(v.Index)
	}
	// Setting zero still inserts an index; it is not sparse-map assignment.
	must(v.SetValue(0, 2))
	if v.Size() != 3 {
		t.Fatal(v.Index)
	}
	err := v.SetValue(7, 1)
	if sourceDebugTests {
		if err != ErrAssertion || v.Size() != 3 {
			t.Fatal(err, v.Index)
		}
	} else if err != nil || v.Size() != 4 {
		t.Fatal(err, v.Index)
	}
	if v.Data[1] != 7 {
		t.Fatal("duplicate assertion must follow data publication")
	}
	must(v.Clear())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	for _, x := range v.Data {
		if x != 0 {
			t.Fatal(v.Data)
		}
	}
	must(v.SetAsDictionary(2))
	must(v.SetAsDictionary(2))
	if v.Data[2] != 1 || !reflect.DeepEqual(v.Index, []uint32{2}) {
		t.Fatal(v)
	}
	v.EraseFromIndex(3)
	v.EraseFromIndex(2)
	if v.Size() != 0 || v.Data[2] != 1 {
		t.Fatal(v)
	}
	v.RestoreIndex()
	if !reflect.DeepEqual(v.Index, []uint32{2}) {
		t.Fatal(v)
	}
	// Assignment preserves unindexed destination entries in its retained extent.
	v.Data[0] = 8
	y := NewIndexedFloatVector(3)
	must(y.SetValue(2, 1))
	must(v.Assign(y))
	if !reflect.DeepEqual(v.Data, []float64{8, 2, 0}) || !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal(v)
	}
	must(v.Assign(v))
	if !reflect.DeepEqual(v.Data, []float64{8, 0, 0}) || !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal("source self assignment", v)
	}
	var z IndexedFloatVector
	must(z.Assign(y))
	if !reflect.DeepEqual(z.Data, y.Data) {
		t.Fatal(z)
	}
	z.Data[1] = 99
	z.Data = append(z.Data, 55)
	if !z.Equal(y) {
		t.Fatal("source ignores other data values and extents")
	}
	z.Index = append(z.Index, 0)
	if z.Equal(y) || y.Equal(&z) {
		t.Fatal("different index sets")
	}
	z.Index = []uint32{1, 1}
	if !z.Equal(y) {
		t.Fatal("source index set ignores duplicates")
	}
	must(v.Resize(7))
	if v.DataSize() != 7 || v.Size() != 0 || v.Data[0] != 8 || v.Data[6] != 0 {
		t.Fatal(v)
	}
	must(v.Resize(2))
	if v.DataSize() != 2 {
		t.Fatal(v)
	}
	v.ClearAll()
	if !reflect.DeepEqual(v.Data, []float64{0, 0}) {
		t.Fatal(v)
	}
	// Add/drop paths with exact cancellation, zero no-ops and nonzero insertions.
	must(v.Add(0, 0))
	must(v.Add(0, 2))
	must(v.Add(0, 1))
	must(v.Add(0, float64(0)-3))
	if v.Size() != 0 || v.Data[0] != 0 {
		t.Fatal(v)
	}
	must(v.AddWithDropTolerance(0, 0))
	must(v.AddWithDropTolerance(0, 2))
	must(v.AddWithDropTolerance(0, 1))
	must(v.AddWithDropTolerance(0, float64(0)-3))
	if v.Size() != 0 || v.Data[0] != 0 {
		t.Fatal(v)
	}
	must(v.CleanUp())
	v.Index = []uint32{0, 1}
	v.Data[1] = 1
	must(v.CleanUp())
	if !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal(v)
	}
	for _, op := range []func() error{
		func() error { return v.SetValue(0, 99) }, func() error { return v.SetAsDictionary(99) },
		func() error { return v.Add(99, 1) }, func() error { return v.AddWithDropTolerance(99, 1) },
	} {
		if err := op(); err != ErrBounds {
			t.Fatal(err)
		}
	}
	v.Index = []uint32{1, 99}
	if err := v.Clear(); err != ErrBounds || v.Data[1] != 0 || len(v.Index) != 2 {
		t.Fatal(err, v)
	}
	if err := v.Resize(1); err != ErrBounds || len(v.Data) != 2 {
		t.Fatal(err, v)
	}
	if err := v.Assign(y); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.CleanUp(); err != ErrBounds || len(v.Index) != 2 {
		t.Fatal(err, v)
	}
	if !v.IsOK() {
		t.Fatal("upstream is_OK returns true unconditionally")
	}
	v.ClearAll()
	bad := NewIndexedFloatVector(1)
	bad.Index = []uint32{99}
	if err := v.Assign(bad); err != ErrBounds || !reflect.DeepEqual(v.Index, bad.Index) {
		t.Fatal(err, v)
	}
}

func TestLPIndexedFloatVectorIndependentSparseOracle(t *testing.T) {
	v := NewIndexedFloatVector(31)
	model := make(map[uint32]float64)
	r := rand.New(rand.NewPCG(14, 997))
	for step := 0; step < 6000; step++ {
		i := uint32(r.IntN(31))
		var err error
		switch r.IntN(7) {
		case 0, 1:
			x := float64(r.IntN(7))
			err = v.Add(i, x)
			model[i] += x
			if model[i] == 0 {
				delete(model, i)
			}
		case 2:
			x := model[i]
			err = v.Add(i, -x)
			delete(model, i)
		case 3:
			err = v.SetAsDictionary(i)
			if model[i] == 0 {
				model[i] = 1
			}
		case 4:
			x := float64(r.IntN(3))
			err = v.AddWithDropTolerance(i, x)
			model[i] += x
			if model[i] == 0 {
				delete(model, i)
			}
		case 5:
			err = v.CleanUp()
		case 6:
			if r.IntN(10) == 0 {
				err = v.Clear()
				clear(model)
			} else {
				v.RestoreIndex()
			}
		}
		if err != nil {
			t.Fatal(step, err)
		}
		if v.Size() != uint32(len(model)) {
			t.Fatal(step, v.Index, model)
		}
		seen := make(map[uint32]bool)
		for _, j := range v.Index {
			if seen[j] || model[j] == 0 {
				t.Fatal(step, v.Index, model)
			}
			seen[j] = true
		}
		for j, x := range v.Data {
			if x != model[uint32(j)] {
				t.Fatal(step, j, x, model)
			}
		}
	}
}

func TestLPIndexedUintVectorContracts(t *testing.T) {
	v := NewIndexedUintVector(5)
	if v.DataSize() != 5 || v.Size() != 0 || !v.IsOK() {
		t.Fatal(v)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(v.SetValue(3, 3))
	must(v.SetValue(1, 1))
	if !reflect.DeepEqual(v.Index, []uint32{3, 1}) {
		t.Fatal(v.Index)
	}
	// Setting zero still inserts an index; it is not sparse-map assignment.
	must(v.SetValue(0, 2))
	if v.Size() != 3 {
		t.Fatal(v.Index)
	}
	err := v.SetValue(7, 1)
	if sourceDebugTests {
		if err != ErrAssertion || v.Size() != 3 {
			t.Fatal(err, v.Index)
		}
	} else if err != nil || v.Size() != 4 {
		t.Fatal(err, v.Index)
	}
	if v.Data[1] != 7 {
		t.Fatal("duplicate assertion must follow data publication")
	}
	must(v.Clear())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	for _, x := range v.Data {
		if x != 0 {
			t.Fatal(v.Data)
		}
	}
	must(v.SetAsDictionary(2))
	must(v.SetAsDictionary(2))
	if v.Data[2] != 1 || !reflect.DeepEqual(v.Index, []uint32{2}) {
		t.Fatal(v)
	}
	v.EraseFromIndex(3)
	v.EraseFromIndex(2)
	if v.Size() != 0 || v.Data[2] != 1 {
		t.Fatal(v)
	}
	v.RestoreIndex()
	if !reflect.DeepEqual(v.Index, []uint32{2}) {
		t.Fatal(v)
	}
	// Assignment preserves unindexed destination entries in its retained extent.
	v.Data[0] = 8
	y := NewIndexedUintVector(3)
	must(y.SetValue(2, 1))
	must(v.Assign(y))
	if !reflect.DeepEqual(v.Data, []uint32{8, 2, 0}) || !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal(v)
	}
	must(v.Assign(v))
	if !reflect.DeepEqual(v.Data, []uint32{8, 0, 0}) || !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal("source self assignment", v)
	}
	var z IndexedUintVector
	must(z.Assign(y))
	if !reflect.DeepEqual(z.Data, y.Data) {
		t.Fatal(z)
	}
	z.Data[1] = 99
	z.Data = append(z.Data, 55)
	if !z.Equal(y) {
		t.Fatal("source ignores other data values and extents")
	}
	z.Index = append(z.Index, 0)
	if z.Equal(y) || y.Equal(&z) {
		t.Fatal("different index sets")
	}
	z.Index = []uint32{1, 1}
	if !z.Equal(y) {
		t.Fatal("source index set ignores duplicates")
	}
	must(v.Resize(7))
	if v.DataSize() != 7 || v.Size() != 0 || v.Data[0] != 8 || v.Data[6] != 0 {
		t.Fatal(v)
	}
	must(v.Resize(2))
	if v.DataSize() != 2 {
		t.Fatal(v)
	}
	v.ClearAll()
	if !reflect.DeepEqual(v.Data, []uint32{0, 0}) {
		t.Fatal(v)
	}
	// Add/drop paths with exact cancellation, zero no-ops and nonzero insertions.
	must(v.Add(0, 0))
	must(v.Add(0, 2))
	must(v.Add(0, 1))
	must(v.Add(0, ^uint32(2)))
	if v.Size() != 0 || v.Data[0] != 0 {
		t.Fatal(v)
	}
	must(v.AddWithDropTolerance(0, 0))
	must(v.AddWithDropTolerance(0, 2))
	must(v.AddWithDropTolerance(0, 1))
	must(v.AddWithDropTolerance(0, ^uint32(2)))
	if v.Size() != 0 || v.Data[0] != 0 {
		t.Fatal(v)
	}
	must(v.CleanUp())
	v.Index = []uint32{0, 1}
	v.Data[1] = 1
	must(v.CleanUp())
	if !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal(v)
	}
	for _, op := range []func() error{
		func() error { return v.SetValue(0, 99) }, func() error { return v.SetAsDictionary(99) },
		func() error { return v.Add(99, 1) }, func() error { return v.AddWithDropTolerance(99, 1) },
	} {
		if err := op(); err != ErrBounds {
			t.Fatal(err)
		}
	}
	v.Index = []uint32{1, 99}
	if err := v.Clear(); err != ErrBounds || v.Data[1] != 0 || len(v.Index) != 2 {
		t.Fatal(err, v)
	}
	if err := v.Resize(1); err != ErrBounds || len(v.Data) != 2 {
		t.Fatal(err, v)
	}
	if err := v.Assign(y); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.CleanUp(); err != ErrBounds || len(v.Index) != 2 {
		t.Fatal(err, v)
	}
	if !v.IsOK() {
		t.Fatal("upstream is_OK returns true unconditionally")
	}
	v.ClearAll()
	bad := NewIndexedUintVector(1)
	bad.Index = []uint32{99}
	if err := v.Assign(bad); err != ErrBounds || !reflect.DeepEqual(v.Index, bad.Index) {
		t.Fatal(err, v)
	}
}

func TestLPIndexedUintVectorIndependentSparseOracle(t *testing.T) {
	v := NewIndexedUintVector(31)
	model := make(map[uint32]uint32)
	r := rand.New(rand.NewPCG(14, 997))
	for step := 0; step < 6000; step++ {
		i := uint32(r.IntN(31))
		var err error
		switch r.IntN(7) {
		case 0, 1:
			x := uint32(r.IntN(7))
			err = v.Add(i, x)
			model[i] += x
			if model[i] == 0 {
				delete(model, i)
			}
		case 2:
			x := model[i]
			err = v.Add(i, 0-x)
			delete(model, i)
		case 3:
			err = v.SetAsDictionary(i)
			if model[i] == 0 {
				model[i] = 1
			}
		case 4:
			x := uint32(r.IntN(3))
			err = v.AddWithDropTolerance(i, x)
			model[i] += x
			if model[i] == 0 {
				delete(model, i)
			}
		case 5:
			err = v.CleanUp()
		case 6:
			if r.IntN(10) == 0 {
				err = v.Clear()
				clear(model)
			} else {
				v.RestoreIndex()
			}
		}
		if err != nil {
			t.Fatal(step, err)
		}
		if v.Size() != uint32(len(model)) {
			t.Fatal(step, v.Index, model)
		}
		seen := make(map[uint32]bool)
		for _, j := range v.Index {
			if seen[j] || model[j] == 0 {
				t.Fatal(step, v.Index, model)
			}
			seen[j] = true
		}
		for j, x := range v.Data {
			if x != model[uint32(j)] {
				t.Fatal(step, j, x, model)
			}
		}
	}
}

func TestLPIndexedFloatVectorDropToleranceEdges(t *testing.T) {
	v := NewIndexedFloatVector(9)
	values := []float64{0, 1e-15, -1e-15, 1e-14, -1e-14, 1, math.Inf(1), math.Inf(-1), math.NaN()}
	copy(v.Data, values)
	v.RestoreIndex()
	if !reflect.DeepEqual(v.Index, []uint32{3, 4, 5, 6, 7, 8}) || v.Data[1] != 0 || v.Data[2] != 0 {
		t.Fatal(v)
	}
	if !v.Equal(v) {
		t.Fatal("source self comparison tolerates NaN")
	}
	v.Index = []uint32{1, 3, 8, 2}
	v.Data[1] = 1e-15
	v.Data[2] = -1e-15
	if err := v.CleanUp(); err != nil || !reflect.DeepEqual(v.Index, []uint32{3, 8}) || v.Data[1] != 0 || v.Data[2] != 0 {
		t.Fatal(err, v)
	}
	v.ClearAll()
	if err := v.AddWithDropTolerance(0, 1e-15); err != nil || v.Size() != 0 {
		t.Fatal(err, v)
	}
	if err := v.AddWithDropTolerance(0, 1e-14); err != nil || v.Size() != 1 {
		t.Fatal(err, v)
	}
	if err := v.AddWithDropTolerance(0, -9e-15); err != nil || v.Size() != 0 || v.Data[0] != 0 {
		t.Fatal(err, v)
	}
	// Ordinary add tests exact zero, independently of drop tolerance.
	if err := v.Add(0, 1e-15); err != nil || v.Size() != 1 {
		t.Fatal(err, v)
	}
	if err := v.Add(0, -1e-15); err != nil || v.Size() != 0 {
		t.Fatal(err, v)
	}
	if err := v.AddWithDropTolerance(0, math.NaN()); err != nil || v.Size() != 1 || !math.IsNaN(v.Data[0]) {
		t.Fatal(err, v)
	}
	// Clear/CleanUp retain the reached source prefix on a bad public index.
	v.Data[1] = 1e-15
	v.Index = []uint32{1, 99}
	if err := v.CleanUp(); err != ErrBounds || v.Data[1] != 0 || len(v.Index) != 2 {
		t.Fatal(err, v)
	}
}
