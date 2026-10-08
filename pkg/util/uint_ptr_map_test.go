// Source: src/util/uint_map.h:25-62.
package util

import "testing"

func TestUintPtrMapSparseInsertFindReplaceEraseAndReset(t *testing.T) {
	m := NewUintPtrMap()
	a, b := new(int), new(int)
	*a, *b = 10, 20
	if m.Contains(0) {
		t.Fatal("empty contains")
	}
	m.Insert(5, a)
	if len(m.values) != 6 || m.Contains(4) || !m.Contains(5) {
		t.Fatal("sparse insert", len(m.values), m.Contains(4), m.Contains(5))
	}
	if value, found := m.Find(5); !found || value != a || m.Get(5) != a {
		t.Fatal("find", value, found, m.Get(5))
	}
	m.Insert(5, b)
	if m.Get(5) != b {
		t.Fatal("replace", m.Get(5))
	}
	m.Erase(99)
	m.Erase(5)
	if m.Contains(5) {
		t.Fatal("erase")
	}
	if value, found := m.Find(5); found || value != nil {
		t.Fatal("absent find", value, found)
	}
	m.Insert(1, a)
	m.Reset()
	if len(m.values) != 0 || m.Contains(1) {
		t.Fatal("reset", len(m.values))
	}
}

func TestUintPtrMapMissingAndNilValuesPanic(t *testing.T) {
	m := NewUintPtrMap()
	assertUintPtrMapPanic(t, func() { m.Get(0) })
	assertUintPtrMapPanic(t, func() { m.Insert(0, nil) })
	assertUintPtrMapPanic(t, func() { m.Insert(0, 7) })
	var pointer *int
	assertUintPtrMapPanic(t, func() { m.Insert(0, pointer) })
	if len(m.values) != 0 {
		t.Fatal("failed insert mutated map", len(m.values))
	}
}

func TestUintPtrMapPointerIdentity(t *testing.T) {
	type object struct{ value int }
	a := &object{value: 1}
	b := &object{value: 1}
	m := NewUintPtrMap()
	m.Insert(0, a)
	m.Insert(1, b)
	if m.Get(0) != a || m.Get(1) != b || m.Get(0) == m.Get(1) {
		t.Fatal("pointer identity")
	}
}

func assertUintPtrMapPanic(t *testing.T, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	call()
}
