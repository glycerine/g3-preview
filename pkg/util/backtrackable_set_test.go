// Source: src/util/backtrackable_set.h:25-111.
package util

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

type backtrackableSetEventRecord struct {
	value    int
	inserted bool
}

func TestBacktrackableSetNestedUndoAndEvents(t *testing.T) {
	var events []backtrackableSetEventRecord
	set := NewBacktrackableSet(func(value any, inserted bool) {
		events = append(events, backtrackableSetEventRecord{value: value.(int), inserted: inserted})
	})
	set.Insert(1)
	set.PushScope()
	set.Insert(2)
	set.Insert(2)
	set.Erase(1)
	set.Erase(9)
	set.PushScope()
	set.Insert(3)
	set.Erase(2)

	if set.Contains(1) || set.Contains(2) || !set.Contains(3) {
		t.Fatal("inner state", set.Values())
	}
	if err := set.PopScope(); err != nil {
		t.Fatal(err)
	}
	if set.Contains(1) || !set.Contains(2) || set.Contains(3) {
		t.Fatal("inner undo", set.Values())
	}
	if err := set.PopScope(); err != nil {
		t.Fatal(err)
	}
	if !set.Contains(1) || set.Contains(2) || set.Contains(3) {
		t.Fatal("outer undo", set.Values())
	}
	want := []backtrackableSetEventRecord{
		{value: 2, inserted: false},
		{value: 3, inserted: true},
		{value: 1, inserted: false},
		{value: 2, inserted: true},
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatal("event order", events, want)
	}
}

func TestBacktrackableSetLevelZeroEraseAndReset(t *testing.T) {
	called := false
	set := NewBacktrackableSet(func(any, bool) { called = true })
	set.Insert("a")
	set.Insert("b")
	set.Erase("a")
	if set.Contains("a") || !set.Contains("b") || set.Empty() {
		t.Fatal("level-zero erase", set.Values())
	}
	set.PushScope()
	set.Insert("c")
	set.Reset()
	if !set.Empty() || len(set.trail) != 0 || len(set.scopes) != 0 || called {
		t.Fatal("reset", set.Values(), len(set.trail), len(set.scopes), called)
	}
	if err := set.PopScope(); !errors.Is(err, ErrBacktrackableSetScope) {
		t.Fatal("scope underflow", err)
	}
}

func TestBacktrackableSetPointerIdentityAndValues(t *testing.T) {
	type object struct{ value int }
	a := &object{value: 1}
	b := &object{value: 1}
	set := NewBacktrackableSet(nil)
	set.Insert(a)
	set.Insert(b)
	if !set.Contains(a) || !set.Contains(b) || len(set.Values()) != 2 {
		t.Fatal("pointer identity", set.Values())
	}

	ints := NewBacktrackableSet(nil)
	ints.Insert(4)
	ints.Insert(2)
	ints.Insert(3)
	values := ints.Values()
	got := make([]int, len(values))
	for i, value := range values {
		got[i] = value.(int)
	}
	sort.Ints(got)
	if want := []int{2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Fatal("values", got, want)
	}
}
