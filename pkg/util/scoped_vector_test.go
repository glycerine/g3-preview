// Source: src/util/scoped_vector.h:23-164.
package util

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func scopedVectorInts(t *testing.T, vector *ScopedVector) []int {
	t.Helper()
	values := make([]int, 0, vector.Size())
	for iterator, end := vector.Begin(), vector.End(); iterator.NotEqual(end); iterator.Advance() {
		value, ok := iterator.Value().(int)
		if !ok {
			t.Fatalf("unexpected scoped-vector value %T", iterator.Value())
		}
		values = append(values, value)
	}
	return values
}

func TestScopedVectorLevelZeroSetPopAndEraseSwap(t *testing.T) {
	vector := NewScopedVector()
	for _, value := range []int{10, 20, 30, 40} {
		vector.PushBack(value)
	}
	physical := len(vector.elems)
	vector.Set(1, 21)
	if len(vector.elems) != physical || vector.At(1) != 21 {
		t.Fatal("level-zero set copied", len(vector.elems), vector.At(1))
	}
	vector.EraseAndSwap(1)
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{10, 40, 30}) {
		t.Fatal("erase and swap", got)
	}
	vector.PopBack()
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{10, 40}) || !vector.CheckInvariant() {
		t.Fatal("pop", got)
	}
}

func TestScopedVectorNestedCopyOnWriteAndMultiPop(t *testing.T) {
	vector := NewScopedVector()
	for _, value := range []int{1, 2, 3} {
		vector.PushBack(value)
	}
	vector.PushScope()
	vector.Set(0, 10)
	physical := len(vector.elems)
	vector.Set(0, 11)
	if len(vector.elems) != physical {
		t.Fatal("same-scope set copied twice")
	}
	vector.PushBack(4)
	vector.PushScope()
	vector.Set(1, 20)
	vector.EraseAndSwap(2)
	vector.PushBack(5)
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{11, 20, 4, 5}) {
		t.Fatal("nested state", got)
	}
	vector.PopScope(1)
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{11, 2, 3, 4}) {
		t.Fatal("inner rollback", got)
	}
	vector.PopScope(1)
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{1, 2, 3}) || !vector.CheckInvariant() {
		t.Fatal("outer rollback", got)
	}

	vector.PushScope()
	vector.Set(0, 100)
	vector.PushScope()
	vector.Set(1, 200)
	vector.PopScope(2)
	if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatal("multi-pop rollback", got)
	}
}

func TestScopedVectorIteratorOwnerAndDirectRef(t *testing.T) {
	first, second := NewScopedVector(), NewScopedVector()
	first.PushBack(1)
	first.PushBack(2)
	second.PushBack(1)
	iterator := first.Begin()
	if iterator.Equal(second.Begin()) || iterator.Value() != 1 {
		t.Fatal("iterator owner identity")
	}
	previous := iterator.PostAdvance()
	if previous.Value() != 1 || iterator.Value() != 2 {
		t.Fatal("post advance")
	}
	iterator.Advance()
	if !iterator.Equal(first.End()) {
		t.Fatal("end iterator")
	}
	*first.Ref(0) = 9
	if first.At(0) != 9 {
		t.Fatal("direct ref")
	}
}

func TestScopedVectorChaCha8AgainstSnapshotOracle(t *testing.T) {
	random := rand.New(rand.NewChaCha8([32]byte{18, 77, 4, 111}))
	vector := NewScopedVector()
	model := make([]int, 0)
	scopes := make([][]int, 0)
	for step := 0; step < 50000; step++ {
		switch random.IntN(7) {
		case 0:
			scopes = append(scopes, append([]int(nil), model...))
			vector.PushScope()
		case 1:
			if len(scopes) != 0 {
				count := 1 + random.IntN(len(scopes))
				model = append(model[:0], scopes[len(scopes)-count]...)
				scopes = scopes[:len(scopes)-count]
				vector.PopScope(uint32(count))
			}
		case 2:
			value := random.IntN(1 << 20)
			model = append(model, value)
			vector.PushBack(value)
		case 3:
			if len(model) != 0 {
				model = model[:len(model)-1]
				vector.PopBack()
			}
		case 4, 5:
			if len(model) != 0 {
				index := random.IntN(len(model))
				value := random.IntN(1 << 20)
				model[index] = value
				vector.Set(uint32(index), value)
			}
		case 6:
			if len(model) != 0 {
				index := random.IntN(len(model))
				model[index] = model[len(model)-1]
				model = model[:len(model)-1]
				vector.EraseAndSwap(uint32(index))
			}
		}
		if got := scopedVectorInts(t, vector); !reflect.DeepEqual(got, model) || !vector.CheckInvariant() {
			t.Fatal("snapshot mismatch", step, got, model)
		}
	}
}

func TestScopedVectorPreconditionPanics(t *testing.T) {
	vector := NewScopedVector()
	for name, operation := range map[string]func(){
		"at":        func() { vector.At(0) },
		"ref":       func() { vector.Ref(0) },
		"set":       func() { vector.Set(0, 1) },
		"pop":       vector.PopBack,
		"erase":     func() { vector.EraseAndSwap(0) },
		"scope":     func() { vector.PopScope(1) },
		"nil-size":  func() { (*ScopedVector)(nil).Size() },
		"end-value": func() { vector.End().Value() },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing panic")
				}
			}()
			operation()
		})
	}
}
