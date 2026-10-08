// Source: src/util/trail.h:25-357.
package util

import (
	"errors"
	"reflect"
	"testing"
)

func TestTrailStackReverseUndoAcrossScopesAndLevelZero(t *testing.T) {
	stack := NewTrailStack()
	var order []string
	trailMust(t, stack.PushUndo(func() { order = append(order, "level-zero") }))
	stack.PushScope()
	trailMust(t, stack.PushUndo(func() { order = append(order, "outer-first") }))
	trailMust(t, stack.PushUndo(func() { order = append(order, "outer-second") }))
	stack.PushScope()
	trailMust(t, stack.PushUndo(func() { order = append(order, "inner") }))

	if stack.GetNumScopes() != 2 {
		t.Fatal("scope count", stack.GetNumScopes())
	}
	trailMust(t, stack.PopScope(1))
	if want := []string{"inner"}; !reflect.DeepEqual(order, want) {
		t.Fatal("inner undo", order, want)
	}
	trailMust(t, stack.Reset())
	if want := []string{"inner", "outer-second", "outer-first", "level-zero"}; !reflect.DeepEqual(order, want) {
		t.Fatal("reset order", order, want)
	}
	if stack.GetNumScopes() != 0 || len(stack.trails) != 0 {
		t.Fatal("reset storage", stack.GetNumScopes(), len(stack.trails))
	}
}

func TestTrailStackClosureReplacementsForTypedTemplateTrails(t *testing.T) {
	stack := NewTrailStack()
	value := uint32(7)
	flag := false
	values := []int{1, 2}
	table := map[string]int{"keep": 1}
	stack.PushScope()

	oldValue := value
	trailMust(t, stack.PushUndo(func() { value = oldValue }))
	value = 19
	trailMust(t, stack.PushUndo(func() { flag = false }))
	flag = true
	oldLength := len(values)
	trailMust(t, stack.PushUndo(func() { values = values[:oldLength] }))
	values = append(values, 3, 4)
	trailMust(t, stack.PushUndo(func() { delete(table, "drop") }))
	table["drop"] = 2

	trailMust(t, stack.PopScope(1))
	if value != 7 || flag || !reflect.DeepEqual(values, []int{1, 2}) || !reflect.DeepEqual(table, map[string]int{"keep": 1}) {
		t.Fatal("typed closure restoration", value, flag, values, table)
	}
}

func TestUndoTrailStackPrefixAndTrailErrors(t *testing.T) {
	var order []int
	trails := []Trail{
		TrailFunc(func() { order = append(order, 0) }),
		TrailFunc(func() { order = append(order, 1) }),
		TrailFunc(func() { order = append(order, 2) }),
	}
	retained, err := UndoTrailStack(trails, 1)
	trailMust(t, err)
	if len(retained) != 1 || !reflect.DeepEqual(order, []int{2, 1}) {
		t.Fatal("prefix undo", len(retained), order)
	}
	if _, err = UndoTrailStack(retained, 2); !errors.Is(err, ErrTrailScopes) {
		t.Fatal("invalid prefix", err)
	}

	stack := NewTrailStack()
	if err = stack.Push(nil); !errors.Is(err, ErrTrailNil) {
		t.Fatal("nil interface trail", err)
	}
	var nilFunc TrailFunc
	if err = stack.Push(nilFunc); !errors.Is(err, ErrTrailNil) {
		t.Fatal("typed nil trail", err)
	}
	if err = stack.PopScope(1); !errors.Is(err, ErrTrailScopes) {
		t.Fatal("scope underflow", err)
	}
	trailMust(t, stack.PopScope(0))
}

func trailMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
