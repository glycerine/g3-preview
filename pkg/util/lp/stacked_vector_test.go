package lp

import (
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

// Source: src/test/lp.cpp:2001-2027 (--smap). The upstream scenario
// had no value assertions; these expectations independently pin its snapshots.
func TestLPStackedVectorSource(t *testing.T) {
	var v StackedIntVector
	check := func(want []int32) {
		t.Helper()
		if !reflect.DeepEqual(v.Values(), want) {
			t.Fatalf("got %v, want %v", v.Values(), want)
		}
	}
	set := func(i uint32, x int32) {
		t.Helper()
		if err := v.Set(i, x); err != nil {
			t.Fatal(err)
		}
	}
	pop := func(k uint32) {
		t.Helper()
		if err := v.Pop(k); err != nil {
			t.Fatal(err)
		}
	}
	v.Push()
	v.Append(0)
	v.Append(1)
	v.Push()
	set(0, 3)
	set(0, 0)
	v.Append(2)
	v.Append(3)
	v.Append(34)
	v.Push()
	set(1, 3)
	set(2, 3)
	v.Push()
	set(0, 7)
	set(1, 9)
	pop(2)
	check([]int32{0, 1, 2, 3, 34})
	set(v.Size()-1, 7)
	v.Push()
	v.Append(33)
	set(0, 13)
	pop(1)
	check([]int32{0, 1, 2, 3, 7})
	pop(1)
	check([]int32{0, 1})
	pop(1)
	if v.Size() != 0 {
		t.Fatal(v.Values())
	}
}

// The reference model stores full copies, independently of the change journal.
func TestLPStackedUintVectorSnapshots(t *testing.T) {
	var v StackedUintVector
	if v.Size() != 0 || len(v.Values()) != 0 {
		t.Fatal("zero value")
	}
	if _, err := v.At(0); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Set(0, 0); err != ErrBounds {
		t.Fatal(err)
	}
	for _, k := range []uint32{0, 1, math.MaxUint32} {
		if _, err := v.PeekSize(k); err != ErrScope {
			t.Fatal(err)
		}
		if err := v.Pop(k); err != ErrScope {
			t.Fatal(err)
		}
	}
	// Source records changes even outside a scope, and ignores records for
	// elements appended after the oldest popped snapshot.
	v.Append(1)
	if err := v.Set(0, 2); err != nil {
		t.Fatal(err)
	}
	v.Push()
	v.Append(3)
	if err := v.Set(1, 4); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 4); err != nil {
		t.Fatal(err)
	}
	before := len(v.changes)
	if err := v.Set(0, 4); err != nil || len(v.changes) != before {
		t.Fatal("same-value replacement", err)
	}
	if size, err := v.PeekSize(1); err != nil || size != 1 {
		t.Fatal(size, err)
	}
	if err := v.Pop(0); err != ErrScope || v.Size() != 2 {
		t.Fatal("invalid pop mutation", err)
	}
	if err := v.Pop(math.MaxUint32); err != ErrScope || v.Size() != 2 {
		t.Fatal("excess pop mutation", err)
	}
	if err := v.Pop(1); err != nil {
		t.Fatal(err)
	}
	if got, err := v.At(0); err != nil || got != 2 || len(v.changes) != 1 {
		t.Fatal(got, err, len(v.changes))
	}
	model := []uint32{2}
	var snapshots [][]uint32
	r := rand.New(rand.NewPCG(77, 193))
	for step := 0; step < 4000; step++ {
		switch r.IntN(6) {
		case 0:
			x := uint32(r.IntN(200))
			v.Append(x)
			model = append(model, x)
		case 1:
			if len(model) > 0 {
				i := r.IntN(len(model))
				x := uint32(r.IntN(200))
				if err := v.Set(uint32(i), x); err != nil {
					t.Fatal(err)
				}
				model[i] = x
			}
		case 2:
			v.Push()
			snapshots = append(snapshots, append([]uint32(nil), model...))
		case 3:
			if len(snapshots) > 0 {
				k := r.IntN(len(snapshots)) + 1
				if err := v.Pop(uint32(k)); err != nil {
					t.Fatal(err)
				}
				model = append([]uint32(nil), snapshots[len(snapshots)-k]...)
				snapshots = snapshots[:len(snapshots)-k]
			}
		case 4:
			v.IncreaseSizeByOne()
			model = append(model, 0)
		case 5:
			if len(model) > 0 {
				i := r.IntN(len(model))
				if err := v.Set(uint32(i), model[i]); err != nil {
					t.Fatal(err)
				}
			}
		}
		if v.Size() != uint32(len(model)) || !reflect.DeepEqual(v.Values(), model) {
			t.Fatalf("step %d: got %v want %v", step, v.Values(), model)
		}
		for i, x := range model {
			got, err := v.At(uint32(i))
			if err != nil || got != x {
				t.Fatal(step, i, got, x, err)
			}
		}
		if _, err := v.At(v.Size()); err != ErrBounds {
			t.Fatal(step, err)
		}
		if err := v.Set(v.Size(), 0); err != ErrBounds {
			t.Fatal(step, err)
		}
		for k := 1; k <= len(snapshots); k++ {
			size, err := v.PeekSize(uint32(k))
			if err != nil || size != uint32(len(snapshots[len(snapshots)-k])) {
				t.Fatal(step, k, size, err)
			}
		}
		if _, err := v.PeekSize(uint32(len(snapshots) + 1)); err != ErrScope {
			t.Fatal(step, err)
		}
	}
}

// The reference model stores full copies, independently of the change journal.
func TestLPStackedIntVectorSnapshots(t *testing.T) {
	var v StackedIntVector
	if v.Size() != 0 || len(v.Values()) != 0 {
		t.Fatal("zero value")
	}
	if _, err := v.At(0); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Set(0, 0); err != ErrBounds {
		t.Fatal(err)
	}
	for _, k := range []uint32{0, 1, math.MaxUint32} {
		if _, err := v.PeekSize(k); err != ErrScope {
			t.Fatal(err)
		}
		if err := v.Pop(k); err != ErrScope {
			t.Fatal(err)
		}
	}
	// Source records changes even outside a scope, and ignores records for
	// elements appended after the oldest popped snapshot.
	v.Append(1)
	if err := v.Set(0, 2); err != nil {
		t.Fatal(err)
	}
	v.Push()
	v.Append(3)
	if err := v.Set(1, 4); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 4); err != nil {
		t.Fatal(err)
	}
	before := len(v.changes)
	if err := v.Set(0, 4); err != nil || len(v.changes) != before {
		t.Fatal("same-value replacement", err)
	}
	if size, err := v.PeekSize(1); err != nil || size != 1 {
		t.Fatal(size, err)
	}
	if err := v.Pop(0); err != ErrScope || v.Size() != 2 {
		t.Fatal("invalid pop mutation", err)
	}
	if err := v.Pop(math.MaxUint32); err != ErrScope || v.Size() != 2 {
		t.Fatal("excess pop mutation", err)
	}
	if err := v.Pop(1); err != nil {
		t.Fatal(err)
	}
	if got, err := v.At(0); err != nil || got != 2 || len(v.changes) != 1 {
		t.Fatal(got, err, len(v.changes))
	}
	model := []int32{2}
	var snapshots [][]int32
	r := rand.New(rand.NewPCG(77, 193))
	for step := 0; step < 4000; step++ {
		switch r.IntN(6) {
		case 0:
			x := int32(r.IntN(200) - 100)
			v.Append(x)
			model = append(model, x)
		case 1:
			if len(model) > 0 {
				i := r.IntN(len(model))
				x := int32(r.IntN(200) - 100)
				if err := v.Set(uint32(i), x); err != nil {
					t.Fatal(err)
				}
				model[i] = x
			}
		case 2:
			v.Push()
			snapshots = append(snapshots, append([]int32(nil), model...))
		case 3:
			if len(snapshots) > 0 {
				k := r.IntN(len(snapshots)) + 1
				if err := v.Pop(uint32(k)); err != nil {
					t.Fatal(err)
				}
				model = append([]int32(nil), snapshots[len(snapshots)-k]...)
				snapshots = snapshots[:len(snapshots)-k]
			}
		case 4:
			v.IncreaseSizeByOne()
			model = append(model, 0)
		case 5:
			if len(model) > 0 {
				i := r.IntN(len(model))
				if err := v.Set(uint32(i), model[i]); err != nil {
					t.Fatal(err)
				}
			}
		}
		if v.Size() != uint32(len(model)) || !reflect.DeepEqual(v.Values(), model) {
			t.Fatalf("step %d: got %v want %v", step, v.Values(), model)
		}
		for i, x := range model {
			got, err := v.At(uint32(i))
			if err != nil || got != x {
				t.Fatal(step, i, got, x, err)
			}
		}
		if _, err := v.At(v.Size()); err != ErrBounds {
			t.Fatal(step, err)
		}
		if err := v.Set(v.Size(), 0); err != ErrBounds {
			t.Fatal(step, err)
		}
		for k := 1; k <= len(snapshots); k++ {
			size, err := v.PeekSize(uint32(k))
			if err != nil || size != uint32(len(snapshots[len(snapshots)-k])) {
				t.Fatal(step, k, size, err)
			}
		}
		if _, err := v.PeekSize(uint32(len(snapshots) + 1)); err != ErrScope {
			t.Fatal(step, err)
		}
	}
}

// The reference model stores full copies, independently of the change journal.
func TestLPStackedColumnVectorSnapshots(t *testing.T) {
	var v StackedColumnVector
	if v.Size() != 0 || len(v.Values()) != 0 {
		t.Fatal("zero value")
	}
	if _, err := v.At(0); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Set(0, 0); err != ErrBounds {
		t.Fatal(err)
	}
	for _, k := range []uint32{0, 1, math.MaxUint32} {
		if _, err := v.PeekSize(k); err != ErrScope {
			t.Fatal(err)
		}
		if err := v.Pop(k); err != ErrScope {
			t.Fatal(err)
		}
	}
	// Source records changes even outside a scope, and ignores records for
	// elements appended after the oldest popped snapshot.
	v.Append(1)
	if err := v.Set(0, 2); err != nil {
		t.Fatal(err)
	}
	v.Push()
	v.Append(3)
	if err := v.Set(1, 4); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(0, 4); err != nil {
		t.Fatal(err)
	}
	before := len(v.changes)
	if err := v.Set(0, 4); err != nil || len(v.changes) != before {
		t.Fatal("same-value replacement", err)
	}
	if size, err := v.PeekSize(1); err != nil || size != 1 {
		t.Fatal(size, err)
	}
	if err := v.Pop(0); err != ErrScope || v.Size() != 2 {
		t.Fatal("invalid pop mutation", err)
	}
	if err := v.Pop(math.MaxUint32); err != ErrScope || v.Size() != 2 {
		t.Fatal("excess pop mutation", err)
	}
	if err := v.Pop(1); err != nil {
		t.Fatal(err)
	}
	if got, err := v.At(0); err != nil || got != 2 || len(v.changes) != 1 {
		t.Fatal(got, err, len(v.changes))
	}
	model := []ColumnType{2}
	var snapshots [][]ColumnType
	r := rand.New(rand.NewPCG(77, 193))
	for step := 0; step < 4000; step++ {
		switch r.IntN(6) {
		case 0:
			x := ColumnType(r.IntN(5))
			v.Append(x)
			model = append(model, x)
		case 1:
			if len(model) > 0 {
				i := r.IntN(len(model))
				x := ColumnType(r.IntN(5))
				if err := v.Set(uint32(i), x); err != nil {
					t.Fatal(err)
				}
				model[i] = x
			}
		case 2:
			v.Push()
			snapshots = append(snapshots, append([]ColumnType(nil), model...))
		case 3:
			if len(snapshots) > 0 {
				k := r.IntN(len(snapshots)) + 1
				if err := v.Pop(uint32(k)); err != nil {
					t.Fatal(err)
				}
				model = append([]ColumnType(nil), snapshots[len(snapshots)-k]...)
				snapshots = snapshots[:len(snapshots)-k]
			}
		case 4:
			v.IncreaseSizeByOne()
			model = append(model, 0)
		case 5:
			if len(model) > 0 {
				i := r.IntN(len(model))
				if err := v.Set(uint32(i), model[i]); err != nil {
					t.Fatal(err)
				}
			}
		}
		if v.Size() != uint32(len(model)) || !reflect.DeepEqual(v.Values(), model) {
			t.Fatalf("step %d: got %v want %v", step, v.Values(), model)
		}
		for i, x := range model {
			got, err := v.At(uint32(i))
			if err != nil || got != x {
				t.Fatal(step, i, got, x, err)
			}
		}
		if _, err := v.At(v.Size()); err != ErrBounds {
			t.Fatal(step, err)
		}
		if err := v.Set(v.Size(), 0); err != ErrBounds {
			t.Fatal(step, err)
		}
		for k := 1; k <= len(snapshots); k++ {
			size, err := v.PeekSize(uint32(k))
			if err != nil || size != uint32(len(snapshots[len(snapshots)-k])) {
				t.Fatal(step, k, size, err)
			}
		}
		if _, err := v.PeekSize(uint32(len(snapshots) + 1)); err != ErrScope {
			t.Fatal(step, err)
		}
	}
}
