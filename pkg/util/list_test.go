// Source: src/test/list.cpp:24-42; src/util/list.h:25-92.
package util

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func listInts(t *testing.T, list *List) []int {
	t.Helper()
	values := make([]int, 0, ListLength(list))
	for it, end := list.Begin(), list.End(); it.NotEqual(end); it.Advance() {
		value, ok := it.Value().(int)
		if !ok {
			t.Fatalf("non-int list value %T", it.Value())
		}
		values = append(values, value)
	}
	return values
}

func TestListRegisteredAppendAssertions(t *testing.T) {
	l1 := NewList(10, nil)
	l2 := NewList(20, l1)
	l3 := NewList(30, nil)
	l4 := NewList(40, l3)
	if AppendList(l1, nil) != l1 {
		t.Fatal("append(l1, nil) did not return l1")
	}
	if AppendList(l2, nil) != l2 {
		t.Fatal("append(l2, nil) did not return l2")
	}
	if AppendList(nil, l2) != l2 {
		t.Fatal("append(nil, l2) did not return l2")
	}
	if AppendList(nil, nil) != nil {
		t.Fatal("append(nil, nil) was non-nil")
	}
	l5 := AppendList(l4, l2)
	if got := listInts(t, l5); !reflect.DeepEqual(got, []int{40, 30, 20, 10}) {
		t.Fatal("registered append values", got)
	}
	l6 := AppendList(l5, l5)
	if got := listInts(t, l6); !reflect.DeepEqual(got, []int{40, 30, 20, 10, 40, 30, 20, 10}) {
		t.Fatal("registered self append values", got)
	}
}

func TestListAppendCopiesPrefixAndSharesSuffix(t *testing.T) {
	suffixTail := NewList("tail", nil)
	suffix := NewList("suffix", suffixTail)
	firstTail := NewList("b", nil)
	first := NewList("a", firstTail)
	joined := AppendList(first, suffix)

	if joined == first || joined.Tail() == firstTail {
		t.Fatal("append reused a first-list node")
	}
	if joined.Tail().Tail() != suffix {
		t.Fatal("append did not share the second list")
	}
	joined.SetHead("changed-copy")
	joined.Tail().SetHead("changed-copy-tail")
	if first.Head() != "a" || firstTail.Head() != "b" {
		t.Fatal("copied prefix aliases source prefix")
	}
	suffix.SetHead("shared-change")
	if joined.Tail().Tail().Head() != "shared-change" {
		t.Fatal("second-list sharing was lost")
	}
	if ListLength(first) != 2 || ListLength(suffix) != 2 || ListLength(joined) != 4 {
		t.Fatal("length changed by append")
	}
}

func TestListIteratorMutationNilValueAndPostAdvance(t *testing.T) {
	tail := NewList(2, nil)
	list := NewList(nil, tail)
	it := list.Begin()
	end := list.End()
	if it.Done() || !it.NotEqual(end) || it.Value() != nil {
		t.Fatal("nil head was confused with end")
	}
	previous := it.PostAdvance()
	if previous.Value() != nil || it.Value() != 2 || !previous.NotEqual(it) {
		t.Fatal("post advance", previous.Value(), it.Value())
	}
	it.Advance()
	if !it.Done() || !it.Equal(end) || it.NotEqual(end) {
		t.Fatal("end iterator comparison")
	}
	list.SetHead(1)
	list.SetTail(nil)
	if list.Head() != 1 || list.Tail() != nil || ListLength(list) != 1 {
		t.Fatal("mutable accessors")
	}
}

func TestListAppendChaCha8AgainstSliceOracle(t *testing.T) {
	random := rand.New(rand.NewChaCha8([32]byte{25, 7, 10, 92}))
	for round := 0; round < 2000; round++ {
		firstValues := make([]int, random.IntN(80))
		secondValues := make([]int, random.IntN(80))
		for i := range firstValues {
			firstValues[i] = random.IntN(1 << 20)
		}
		for i := range secondValues {
			secondValues[i] = random.IntN(1 << 20)
		}
		first := listFromInts(firstValues)
		second := listFromInts(secondValues)
		joined := AppendList(first, second)
		want := append(append([]int(nil), firstValues...), secondValues...)
		if got := listInts(t, joined); !reflect.DeepEqual(got, want) {
			t.Fatal("oracle mismatch", round, got, want)
		}
		if len(secondValues) == 0 {
			if joined != first {
				t.Fatal("nil suffix identity", round)
			}
		} else {
			shared := joined
			for range firstValues {
				shared = shared.Tail()
			}
			if shared != second {
				t.Fatal("suffix identity", round)
			}
		}
	}
}

func listFromInts(values []int) *List {
	var result *List
	for i := len(values) - 1; i >= 0; i-- {
		result = NewList(values[i], result)
	}
	return result
}
