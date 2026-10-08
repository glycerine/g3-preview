// Source: src/util/ref_vector.h:27-359.
package util

import (
	"reflect"
	"testing"
)

type refVectorValue struct {
	id   uint32
	refs int
}

type refVectorManager struct{ events []string }

func (manager *refVectorManager) inc(value any) {
	manager.events = append(manager.events, "i"+refVectorValueName(value))
}

func (manager *refVectorManager) dec(value any) {
	manager.events = append(manager.events, "d"+refVectorValueName(value))
}

func refVectorValueName(value any) string {
	if nilScopedPointer(value) {
		return "nil"
	}
	return string(rune('0' + value.(*refVectorValue).id))
}

func TestRefVectorCoreCallbacksResizeReserveAndErase(t *testing.T) {
	manager := &refVectorManager{}
	vector := NewRefVector(manager, manager.inc, manager.dec)
	a, b := &refVectorValue{id: 1}, &refVectorValue{id: 2}
	vector.PushBack(a).PushBack(b)
	vector.Set(0, a)
	vector.Reserve(4)
	vector.ResizeFill(6, b)
	if vector.Size() != 6 || vector.GetDefault(9, a) != a || vector.Back() != b {
		t.Fatal("growth or access mismatch")
	}
	vector.EraseIndex(1)
	if !vector.EraseValue(a) || vector.EraseValue(a) || !vector.Contains(b) {
		t.Fatal("erase or contains mismatch")
	}
	vector.Shrink(1)
	vector.PopBack()
	want := []string{"i1", "i2", "i1", "d1", "i2", "i2", "d2", "d1", "dnil", "d2", "d2", "dnil"}
	if !reflect.DeepEqual(manager.events, want) {
		t.Fatalf("events = %v, want %v", manager.events, want)
	}
	vector.Close()
}

func TestRefVectorCopyMoveAssignSwapAndOrderOperations(t *testing.T) {
	manager := &refVectorManager{}
	a, b, c := &refVectorValue{id: 1}, &refVectorValue{id: 2}, &refVectorValue{id: 3}
	vector := NewRefVectorFrom(manager, manager.inc, manager.dec, []any{a, b, c})
	clone := vector.Clone()
	if clone.Manager() != manager || !RefVectorPtrEqual(vector, clone) {
		t.Fatal("clone mismatch")
	}
	clone.SwapIndices(0, 2)
	clone.Reverse()
	if !RefVectorPtrEqual(vector, clone) {
		t.Fatal("swap plus reverse should restore order")
	}
	moved := clone.Move()
	if !clone.Empty() || !RefVectorPtrEqual(vector, moved) {
		t.Fatal("move mismatch")
	}
	other := NewRefVector(manager, manager.inc, manager.dec)
	other.PushBack(c)
	other.Assign(vector)
	other.Assign(other)
	if !RefVectorPtrEqual(vector, other) {
		t.Fatal("assignment mismatch")
	}
	vector.Swap(clone)
	if !vector.Empty() || clone.Size() != 3 {
		t.Fatal("whole-vector swap mismatch")
	}
	vector.Close()
	clone.Close()
	moved.Close()
	other.Close()
}

func TestRefVectorElementReferenceAndStructuralInvalidation(t *testing.T) {
	manager := &refVectorManager{}
	a, b := &refVectorValue{id: 1}, &refVectorValue{id: 2}
	vector := NewRefVectorFrom(manager, manager.inc, manager.dec, []any{a, b})
	first := vector.Element(0)
	second := vector.Element(1)
	iterator := vector.Begin()
	if iterator.Get() != a || iterator.Next().Get() != b || !iterator.Next().Next().Equal(vector.End()) {
		t.Fatal("iterator mismatch")
	}
	if first.Get() != a || !first.Equal(a) {
		t.Fatal("element get/equality mismatch")
	}
	first.Assign(second)
	if first.Get() != b || vector.Get(0) != b {
		t.Fatal("element assignment mismatch")
	}
	refVectorAssertPanics(t, func() { first.Set(nil) })
	vector.PushBack(a)
	refVectorAssertPanics(t, func() { first.Get() })
	refVectorAssertPanics(t, func() { iterator.Get() })
	vector.Close()
}

func TestSRefVectorAndPointerHashEquality(t *testing.T) {
	a, b := &refVectorValue{id: 3}, &refVectorValue{id: 7}
	self := NewSRefVector()
	self.PushBack(a).PushBack(nil).PushBack(b)
	if a.refs != 1 || b.refs != 1 {
		t.Fatalf("self counts after push = %d, %d", a.refs, b.refs)
	}
	movedSelf := self.RefVectorCore.Move()
	if !self.Empty() || movedSelf.Size() != 3 {
		t.Fatal("self-managed core move mismatch")
	}
	movedSelf.Close()
	self.Close()
	if a.refs != 0 || b.refs != 0 {
		t.Fatalf("self counts after close = %d, %d", a.refs, b.refs)
	}

	manager := &refVectorManager{}
	left := NewRefVectorFrom(manager, manager.inc, manager.dec, []any{a, b})
	right := left.Clone()
	wantHash := Composite(17, 2, func(index uint32) uint32 { return []uint32{3, 7}[index] })
	hasher := RefVectorPtrHasher{GetID: func(value any) uint32 { return value.(*refVectorValue).id }}
	if got := hasher.Hash(left); got != wantHash {
		t.Fatalf("hash = %d, want %d", got, wantHash)
	}
	equality := RefVectorPtrEquality{}
	if RefVectorPtrHash(nil, nil) != 0 || !equality.Equal(nil, nil) || equality.Equal(left, nil) || !equality.Equal(left, right) {
		t.Fatal("pointer hash/equality edge mismatch")
	}
	right.Set(1, a)
	if RefVectorPtrEqual(left, right) {
		t.Fatal("different pointer sequence compares equal")
	}
	left.Close()
	right.Close()
}

func (value *refVectorValue) IncRef() { value.refs++ }
func (value *refVectorValue) DecRef() { value.refs-- }

func TestRefVectorPreconditions(t *testing.T) {
	manager := &refVectorManager{}
	otherManager := &refVectorManager{}
	refVectorAssertPanics(t, func() { NewRefVector(nil, manager.inc, manager.dec) })
	refVectorAssertPanics(t, func() { NewRefVector(manager, nil, manager.dec) })
	vector := NewRefVector(manager, manager.inc, manager.dec)
	other := NewRefVector(otherManager, otherManager.inc, otherManager.dec)
	refVectorAssertPanics(t, func() { vector.PushBack(1) })
	refVectorAssertPanics(t, func() { vector.Get(0) })
	refVectorAssertPanics(t, func() { vector.PopBack() })
	refVectorAssertPanics(t, func() { vector.Shrink(1) })
	refVectorAssertPanics(t, func() { vector.Swap(other) })
	refVectorAssertPanics(t, func() { vector.Append(vector.RefVectorCore) })
	empty := NewRefVector(manager, manager.inc, manager.dec)
	if RefVectorPtrHash(empty, nil) != 0 {
		t.Fatal("empty vector hash is nonzero")
	}
	empty.Close()
	vector.Close()
	refVectorAssertPanics(t, func() { vector.Size() })
	other.Close()
}

func refVectorAssertPanics(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	operation()
}
