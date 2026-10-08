// Source: src/util/pool.h:25-46.
package util

import (
	"runtime"
	"testing"
)

type poolTestValue struct {
	id      int
	padding [256]byte
}

func TestPoolLIFOReuseAndFactoryOnlyWhenEmpty(t *testing.T) {
	nextID := 0
	pool := NewPool(func() any {
		nextID++
		return &poolTestValue{id: nextID}
	}, nil)
	defer pool.Close()

	first := pool.Mk().(*poolTestValue)
	second := pool.Mk().(*poolTestValue)
	if first.id != 1 || second.id != 2 || nextID != 2 {
		t.Fatalf("initial allocations = (%d, %d), factory calls = %d", first.id, second.id, nextID)
	}
	pool.Recycle(first)
	pool.Recycle(second)
	if got := pool.Cached(); got != 2 {
		t.Fatalf("cached = %d, want 2", got)
	}
	if got := pool.Mk(); got != second {
		t.Fatalf("first reuse = %p, want most recently recycled %p", got, second)
	}
	if got := pool.Mk(); got != first {
		t.Fatalf("second reuse = %p, want first recycled %p", got, first)
	}
	third := pool.Mk().(*poolTestValue)
	if third.id != 3 || nextID != 3 {
		t.Fatalf("new value id = %d, factory calls = %d; want 3, 3", third.id, nextID)
	}
}

func TestPoolRetainsCachedObjectsAcrossGarbageCollection(t *testing.T) {
	pool := NewPool(func() any { return &poolTestValue{id: 1} }, nil)
	value := pool.Mk().(*poolTestValue)
	pool.Recycle(value)
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	if got := pool.Mk(); got != value {
		t.Fatalf("reuse after garbage collection = %p, want retained value %p", got, value)
	}
	pool.Close()
}

func TestPoolCloseDestroysCachedObjectsForwardAndLeavesCheckedOutAlone(t *testing.T) {
	nextID := 0
	destroyed := make([]int, 0, 3)
	pool := NewPool(func() any {
		nextID++
		return &poolTestValue{id: nextID}
	}, func(value any) {
		destroyed = append(destroyed, value.(*poolTestValue).id)
	})
	first := pool.Mk().(*poolTestValue)
	second := pool.Mk().(*poolTestValue)
	third := pool.Mk().(*poolTestValue)
	checkedOut := pool.Mk().(*poolTestValue)
	pool.Recycle(first)
	pool.Recycle(second)
	pool.Recycle(third)
	pool.Close()

	if got, want := destroyed, []int{1, 2, 3}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("destroy order = %v, want %v", got, want)
	}
	if checkedOut.id != 4 {
		t.Fatalf("checked-out value changed during close: %#v", checkedOut)
	}
}

func TestPoolOwnerAndPointerPreconditions(t *testing.T) {
	poolAssertPanics(t, func() { NewPool(nil, nil) })

	pool := NewPool(func() any { return &poolTestValue{} }, nil)
	copyOfPool := *pool
	poolAssertPanics(t, func() { copyOfPool.Mk() })
	poolAssertPanics(t, func() { pool.Recycle(nil) })
	poolAssertPanics(t, func() { pool.Recycle(1) })
	var typedNil *poolTestValue
	poolAssertPanics(t, func() { pool.Recycle(typedNil) })
	pool.Close()
	poolAssertPanics(t, func() { pool.Mk() })
	poolAssertPanics(t, func() { pool.Recycle(&poolTestValue{}) })
	poolAssertPanics(t, func() { pool.Cached() })
	poolAssertPanics(t, func() { pool.Close() })
	var nilPool *Pool
	poolAssertPanics(t, func() { nilPool.Mk() })

	badFactory := NewPool(func() any { return 1 }, nil)
	poolAssertPanics(t, func() { badFactory.Mk() })
	badFactory.Close()
}

func poolAssertPanics(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	operation()
}
