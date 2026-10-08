// Source: src/util/scoped_ptr_vector.h:24-53.
package util

import (
	"reflect"
	"testing"
)

type scopedPtrVectorValue struct{ id int }

func TestScopedPtrVectorResetForwardAndResizeReverseDestruction(t *testing.T) {
	destroyed := make([]int, 0)
	vector := NewScopedPtrVector(func(value any) {
		destroyed = append(destroyed, value.(*scopedPtrVectorValue).id)
	})
	values := []*scopedPtrVectorValue{{id: 1}, {id: 2}, {id: 3}, {id: 4}}
	for _, value := range values {
		vector.PushBack(value)
	}
	vector.Resize(2)
	if !reflect.DeepEqual(destroyed, []int{4, 3}) || vector.Size() != 2 {
		t.Fatal("reverse shrink", destroyed, vector.Size())
	}
	vector.Reset()
	if !reflect.DeepEqual(destroyed, []int{4, 3, 1, 2}) || !vector.Empty() {
		t.Fatal("forward reset", destroyed)
	}
	vector.Close()
}

func TestScopedPtrVectorSetPopGrowAndIdentity(t *testing.T) {
	destroyed := make([]int, 0)
	vector := NewScopedPtrVector(func(value any) {
		destroyed = append(destroyed, value.(*scopedPtrVectorValue).id)
	})
	first, second := &scopedPtrVectorValue{id: 1}, &scopedPtrVectorValue{id: 2}
	vector.PushBack(first)
	vector.Set(0, first)
	if len(destroyed) != 0 || vector.At(0) != first {
		t.Fatal("same-pointer set")
	}
	vector.Set(0, second)
	if !reflect.DeepEqual(destroyed, []int{1}) || vector.At(0) != second {
		t.Fatal("replacement order", destroyed)
	}
	vector.Resize(3)
	if vector.Size() != 3 || vector.At(1) != nil || vector.At(2) != nil {
		t.Fatal("nil growth")
	}
	vector.PopBack()
	vector.PopBack()
	vector.PopBack()
	if !reflect.DeepEqual(destroyed, []int{1, 2}) || !vector.Empty() {
		t.Fatal("pop destruction", destroyed)
	}
	vector.Close()
}

func TestScopedPtrVectorGCOnlyAndTypedNil(t *testing.T) {
	vector := NewScopedPtrVector(nil)
	var typedNil *scopedPtrVectorValue
	vector.PushBack(typedNil)
	vector.PushBack(&scopedPtrVectorValue{id: 1})
	vector.Set(0, typedNil)
	vector.Reset()
	if !vector.Empty() {
		t.Fatal("GC-only reset")
	}
	vector.Close()
}

func TestScopedPtrVectorOwnerAndPreconditionPanics(t *testing.T) {
	vector := NewScopedPtrVector(nil)
	vector.PushBack(&scopedPtrVectorValue{id: 1})
	copyOwner := *vector
	for name, operation := range map[string]func(){
		"copy":       copyOwner.Reset,
		"nonpointer": func() { vector.PushBack(1) },
		"set-bounds": func() { vector.Set(1, nil) },
		"at-bounds":  func() { vector.At(1) },
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
	vector.Close()
	for name, operation := range map[string]func(){
		"closed": vector.Reset,
		"nil":    func() { (*ScopedPtrVector)(nil).Reset() },
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
