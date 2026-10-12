package ast

import (
	"testing"
)

func TestSortDependencyOrderRetainedSourceTraversal(t *testing.T) {
	// Handles stand in for distinct live sort objects. The omitted dependency
	// is a built-in sort that decl_collector did not add to m_sorts.
	a := ASTHandle{origin: 1, slot: 1, generation: 1}
	b := ASTHandle{origin: 1, slot: 2, generation: 1}
	c := ASTHandle{origin: 1, slot: 3, generation: 1}
	d := ASTHandle{origin: 1, slot: 4, generation: 1}
	builtin := ASTHandle{origin: 1, slot: 5, generation: 1}
	var order sortDependencyOrder
	order.insert(a, []ASTHandle{b, builtin})
	order.insert(b, []ASTHandle{c})
	order.insert(c, nil)
	order.insert(d, []ASTHandle{b})
	order.insert(a, []ASTHandle{b, builtin}) // repeated collected sort
	want := []ASTHandle{c, b, a, d}
	if got := order.order(); !equalSortHandles(got, want) {
		t.Fatalf("dependency-first order = %v, want %v", got, want)
	}
	if got := order.order(); !equalSortHandles(got, want) {
		t.Fatalf("repeat order = %v, want %v", got, want)
	}
}

func TestSortDependencyOrderCycleAndUncollectedRoot(t *testing.T) {
	a := ASTHandle{origin: 2, slot: 1, generation: 1}
	b := ASTHandle{origin: 2, slot: 2, generation: 1}
	c := ASTHandle{origin: 2, slot: 3, generation: 1}
	var order sortDependencyOrder
	order.insert(a, []ASTHandle{b})
	order.insert(b, []ASTHandle{a, c})
	// c is not inserted and must not appear in the result.
	if got, want := order.order(), []ASTHandle{b, a}; !equalSortHandles(got, want) {
		t.Fatalf("source cycle traversal = %v, want %v", got, want)
	}
}

func equalSortHandles(a, b []ASTHandle) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
