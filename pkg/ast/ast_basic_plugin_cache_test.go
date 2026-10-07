// Licensed under the MIT license.
package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBasicCacheResizeRetainsEachSourceExpansion(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m := &ASTManager{BudgetField: b}
	var cache []astNodeRef
	width := uint64(reflect.TypeOf(astNodeRef{}).Size())
	defer func() {
		b.Release(uint64(cap(cache)) * width)
		if b.Used() != 0 {
			t.Fatal("retained cache backing", b.Used())
		}
	}()
	// vector::resize(12) first expands to 2, then attempts 3. Its
	// logical size is not published until every expansion succeeds.
	b.FailAfter(1)
	err := m.resizeDeclCache(&cache, 12)
	if !errors.Is(err, arena.ErrInjectedFailure) || len(cache) != 0 || cap(cache) != 2 || b.Used() != 2*width {
		t.Fatalf("first completed expansion: len=%d cap=%d bytes=%d err=%v", len(cache), cap(cache), b.Used(), err)
	}
	b.FailAfter(-1)
	if err := m.resizeDeclCache(&cache, 1); err != nil {
		t.Fatal(err)
	}
	cache[0] = astNodeRef{slot: 17, generation: 23}
	b.FailAfter(1)
	err = m.resizeDeclCache(&cache, 12)
	if !errors.Is(err, arena.ErrInjectedFailure) || len(cache) != 1 || cap(cache) != 3 || cache[0] != (astNodeRef{slot: 17, generation: 23}) || b.Used() != 3*width {
		t.Fatalf("second completed expansion: len=%d cap=%d bytes=%d err=%v", len(cache), cap(cache), b.Used(), err)
	}
	b.FailAfter(-1)
	if err := m.resizeDeclCache(&cache, 12); err != nil {
		t.Fatal(err)
	}
	if len(cache) != 12 || cap(cache) != 12 || b.Used() != 12*width {
		t.Fatal("source 2,3,5,8,12 capacities", len(cache), cap(cache), b.Used())
	}
	for i := 1; i < len(cache); i++ {
		if cache[i] != (astNodeRef{}) {
			t.Fatal("value initialized slot", i)
		}
	}
}
