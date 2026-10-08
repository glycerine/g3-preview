// Source: src/util/scoped_numeral_vector.h:24-68.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestGenericScopedNumeralVectorCompleteLifecycle(t *testing.T) {
	var deleted []int
	manager := testScopedNumeralManager(&deleted)
	vector, err := NewScopedNumeralVector(manager)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := vector.Manager(); got != manager || vector.Size() != 0 || vector.Capacity() != 0 {
		t.Fatal("constructor state")
	}
	for _, value := range []int{2, 4, 6} {
		if err = vector.PushBack(value); err != nil {
			t.Fatal(err)
		}
	}
	slot, _ := vector.Get(1)
	*slot = 5
	view, _ := vector.CPtr()
	if view[1] != 5 {
		t.Fatal("live access")
	}
	if err = vector.PopBack(); err != nil {
		t.Fatal(err)
	}
	if err = vector.Resize(5); err != nil {
		t.Fatal(err)
	}
	if err = vector.Shrink(1); err != nil {
		t.Fatal(err)
	}
	if want := []int{6, 5, 0, 0, 0}; !reflect.DeepEqual(deleted, want) {
		t.Fatalf("delete order = %v, want %v", deleted, want)
	}
	if err = vector.Reset(); err != nil || !vector.Empty() {
		t.Fatal("reset", err)
	}
	if want := []int{6, 5, 0, 0, 0, 2}; !reflect.DeepEqual(deleted, want) {
		t.Fatalf("reset order = %v, want %v", deleted, want)
	}
	if err = vector.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGenericScopedNumeralVectorFailureAndOwnerPreconditions(t *testing.T) {
	if vector, err := NewScopedNumeralVector(nil); vector != nil || !errors.Is(err, ErrScopedOwner) {
		t.Fatalf("nil manager = %#v, %v", vector, err)
	}
	manager := testScopedNumeralManager(new([]int))
	setFailure := errors.New("set failure")
	manager.Set = func(*any, any) error { return setFailure }
	vector, _ := NewScopedNumeralVector(manager)
	if err := vector.PushBack(3); !errors.Is(err, setFailure) || vector.Size() != 1 {
		t.Fatalf("push prefix = %v, size %d", err, vector.Size())
	}
	manager.Set = func(out *any, value any) error { *out = value; return nil }
	copyOfVector := *vector
	if _, err := copyOfVector.Get(0); !errors.Is(err, ErrScopedCopy) {
		t.Fatal("copied owner", err)
	}
	if err := vector.Shrink(2); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("growing shrink", err)
	}
	if err := vector.Close(); err != nil {
		t.Fatal(err)
	}
	if err := vector.PopBack(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
	empty, _ := NewScopedNumeralVector(testScopedNumeralManager(new([]int)))
	if err := empty.PopBack(); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("empty pop", err)
	}
	_ = empty.Close()
}
