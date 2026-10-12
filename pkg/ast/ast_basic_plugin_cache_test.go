// Source: src/util/util.h:318-322; src/ast/ast.cpp:988,1004.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBasicCacheNativeGrowthPreservesEntriesAndZeroGaps(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m := &ASTManager{BudgetField: b}
	cache := []astNodeRef{{slot: 17, generation: 23}}
	b.FailAfter(0)
	if e := m.resizeDeclCache(&cache, 12); e != nil {
		t.Fatal(e)
	}
	if len(cache) != 12 || cache[0] != (astNodeRef{slot: 17, generation: 23}) {
		t.Fatal("native resize lost old declaration or requested extent")
	}
	for i := 1; i < len(cache); i++ {
		if cache[i] != (astNodeRef{}) {
			t.Fatal("new cache slot not null", i)
		}
	}
	cache[11] = astNodeRef{slot: 29, generation: 31}
	if e := m.resizeDeclCache(&cache, 4); e != nil {
		t.Fatal(e)
	}
	if e := m.resizeDeclCache(&cache, 17); e != nil {
		t.Fatal(e)
	}
	if len(cache) != 17 || cache[11] != (astNodeRef{slot: 29, generation: 31}) || b.Used() != 0 {
		t.Fatal("native growth changed existing slots or charged arena budget")
	}
}
