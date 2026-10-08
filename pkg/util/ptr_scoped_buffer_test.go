// Source: src/test/buffer.cpp:25-44; src/util/ptr_scoped_buffer.h:23-52.
package util

import (
	"reflect"
	"testing"
)

type ptrScopedPoint struct{ x, y int }

func TestPtrScopedBufferRegisteredAssertions(t *testing.T) {
	destroyed := make([]*ptrScopedPoint, 0)
	buffer := NewPtrScopedBuffer(func(value any) {
		destroyed = append(destroyed, value.(*ptrScopedPoint))
	})
	if !buffer.Empty() {
		t.Fatal("new buffer is not empty")
	}
	first := &ptrScopedPoint{10, 20}
	buffer.PushBack(first)
	if buffer.Empty() {
		t.Fatal("push left buffer empty")
	}
	p1 := &ptrScopedPoint{30, 20}
	buffer.PushBack(p1)
	if buffer.Get(1) != p1 {
		t.Fatal("pointer identity")
	}
	third := &ptrScopedPoint{40, 20}
	buffer.PushBack(third)
	if buffer.Size() != 3 {
		t.Fatal("source size", buffer.Size())
	}
	buffer.PopBack()
	if buffer.Get(0) == p1 || buffer.Get(1) != p1 {
		t.Fatal("source positions")
	}
	p2 := &ptrScopedPoint{30, 20}
	if buffer.Get(0) == p2 {
		t.Fatal("equal content confused with pointer identity")
	}
	buffer.Set(0, p2)
	if buffer.Get(0) != p2 || buffer.Size() != 2 {
		t.Fatal("source replacement", buffer.Size())
	}
	last := &ptrScopedPoint{40, 40}
	buffer.PushBack(last)
	buffer.Close()
	if !reflect.DeepEqual(destroyed, []*ptrScopedPoint{third, first, p2, p1, last}) {
		t.Fatal("source destruction sequence", destroyed)
	}
}

func TestPtrScopedBufferResetFinalizeReleaseAppendAndIterator(t *testing.T) {
	destroyed := make([]int, 0)
	buffer := NewPtrScopedBuffer(func(value any) {
		destroyed = append(destroyed, value.(*ptrScopedPoint).x)
	})
	a, b, c := &ptrScopedPoint{x: 1}, &ptrScopedPoint{x: 2}, &ptrScopedPoint{x: 3}
	buffer.Append([]any{a, b, c})
	if buffer.Back() != c {
		t.Fatal("back")
	}
	seen := make([]int, 0, 3)
	for iterator, end := buffer.Begin(), buffer.End(); iterator.NotEqual(end); iterator.Advance() {
		seen = append(seen, iterator.Value().(*ptrScopedPoint).x)
	}
	if !reflect.DeepEqual(seen, []int{1, 2, 3}) {
		t.Fatal("iterator", seen)
	}
	buffer.Release()
	if !buffer.Empty() || len(destroyed) != 0 {
		t.Fatal("release destroyed values", destroyed)
	}
	buffer.Append([]any{a, b})
	buffer.Reset()
	if !reflect.DeepEqual(destroyed, []int{1, 2}) || !buffer.Empty() {
		t.Fatal("reset", destroyed)
	}
	buffer.PushBack(c)
	buffer.Finalize()
	if !reflect.DeepEqual(destroyed, []int{1, 2, 3}) || buffer.values != nil {
		t.Fatal("finalize", destroyed, buffer.values)
	}
	buffer.Close()
}

func TestPtrScopedBufferIdentityTypedNilAndAtomicAppend(t *testing.T) {
	destroyed := 0
	buffer := NewPtrScopedBuffer(func(any) { destroyed++ })
	a := &ptrScopedPoint{x: 1}
	buffer.PushBack(a)
	buffer.Set(0, a)
	if destroyed != 0 {
		t.Fatal("same pointer destroyed")
	}
	var nilPoint *ptrScopedPoint
	buffer.Set(0, nilPoint)
	if destroyed != 1 || buffer.Get(0) != nilPoint {
		t.Fatal("typed nil replacement", destroyed)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("nonpointer append did not panic")
			}
		}()
		buffer.Append([]any{a, 1, a})
	}()
	if buffer.Size() != 1 {
		t.Fatal("invalid append was not atomic", buffer.Size())
	}
	buffer.Close()
	if destroyed != 1 {
		t.Fatal("typed nil destroyed", destroyed)
	}
}

func TestPtrScopedBufferOwnerAndBoundsPanics(t *testing.T) {
	buffer := NewPtrScopedBuffer(nil)
	copyOwner := *buffer
	for name, operation := range map[string]func(){
		"copy":      copyOwner.Reset,
		"back":      func() { buffer.Back() },
		"pop":       buffer.PopBack,
		"get":       func() { buffer.Get(0) },
		"set":       func() { buffer.Set(0, nil) },
		"end-value": func() { buffer.End().Value() },
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
	buffer.Close()
}
