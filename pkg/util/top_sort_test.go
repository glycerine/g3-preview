// Source: src/util/top_sort.h:28-98.
package util

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"
)

type topSortTestNode struct{ id int }

func topSortIDs(t *testing.T, values []any) []int {
	t.Helper()
	result := make([]int, len(values))
	for i, value := range values {
		node, ok := value.(*topSortTestNode)
		if !ok {
			t.Fatalf("unexpected top-sort value %T", value)
		}
		result[i] = node.id
	}
	return result
}

func TestTopSortDependencyFirstAndUnregisteredDependencies(t *testing.T) {
	a := &topSortTestNode{id: 1}
	b := &topSortTestNode{id: 2}
	c := &topSortTestNode{id: 3}
	unregistered := &topSortTestNode{id: 4}
	sort := NewTopSort()
	if err := sort.Insert(a, []any{b, unregistered}); err != nil {
		t.Fatal(err)
	}
	if err := sort.Insert(b, []any{c}); err != nil {
		t.Fatal(err)
	}
	if err := sort.Insert(c, nil); err != nil {
		t.Fatal(err)
	}
	sort.TopologicalSort()
	if got := topSortIDs(t, sort.TopSorted()); !reflect.DeepEqual(got, []int{3, 2, 1}) {
		t.Fatal("dependency order", got)
	}
	if got := sort.TopSorted(); len(got) != 0 {
		got[0] = unregistered
		if topSortIDs(t, sort.TopSorted())[0] != 3 {
			t.Fatal("caller mutated retained order")
		}
	}
}

func TestTopSortCyclesReplacementAndDeterministicRoots(t *testing.T) {
	a := &topSortTestNode{id: 1}
	b := &topSortTestNode{id: 2}
	c := &topSortTestNode{id: 3}
	d := &topSortTestNode{id: 4}
	sort := NewTopSort()
	for _, entry := range []struct {
		node *topSortTestNode
		deps []any
	}{{a, []any{b, b}}, {b, []any{c}}, {c, []any{a}}, {d, nil}} {
		if err := sort.Insert(entry.node, entry.deps); err != nil {
			t.Fatal(err)
		}
	}
	sort.TopologicalSort()
	if got := topSortIDs(t, sort.TopSorted()); !reflect.DeepEqual(got, []int{3, 2, 1, 4}) {
		t.Fatal("cycle completion order", got)
	}
	if err := sort.Insert(a, nil); err != nil {
		t.Fatal(err)
	}
	sort.TopologicalSort()
	if got := topSortIDs(t, sort.TopSorted()); !reflect.DeepEqual(got, []int{1, 3, 2, 4}) {
		t.Fatal("replacement or root insertion order", got)
	}
}

func TestTopSortPointerValidationIsAtomic(t *testing.T) {
	a := &topSortTestNode{id: 1}
	b := &topSortTestNode{id: 2}
	sort := NewTopSort()
	if err := sort.Insert(a, []any{b}); err != nil {
		t.Fatal(err)
	}
	var nilNode *topSortTestNode
	for _, call := range []func() error{
		func() error { return sort.Insert(nil, nil) },
		func() error { return sort.Insert(nilNode, nil) },
		func() error { return sort.Insert(1, nil) },
		func() error { return sort.Insert(a, []any{nilNode}) },
		func() error { return (*TopSort)(nil).Insert(a, nil) },
	} {
		if err := call(); !errors.Is(err, ErrTopSortNode) {
			t.Fatal("invalid pointer accepted", err)
		}
	}
	if err := sort.Insert(b, nil); err != nil {
		t.Fatal(err)
	}
	sort.TopologicalSort()
	if got := topSortIDs(t, sort.TopSorted()); !reflect.DeepEqual(got, []int{2, 1}) {
		t.Fatal("failed insert changed graph", got)
	}
}

func TestTopSortLargeChainUsesIterativeTraversal(t *testing.T) {
	const count = 100000
	nodes := make([]topSortTestNode, count)
	sort := NewTopSort()
	for i := range nodes {
		nodes[i].id = i
		var dependencies []any
		if i+1 < count {
			dependencies = []any{&nodes[i+1]}
		}
		if err := sort.Insert(&nodes[i], dependencies); err != nil {
			t.Fatal(err)
		}
	}
	sort.TopologicalSort()
	ordered := sort.TopSorted()
	if len(ordered) != count || ordered[0] != &nodes[count-1] || ordered[count-1] != &nodes[0] {
		t.Fatal("large chain order", len(ordered))
	}
}

func TestTopSortChaCha8DAGAgainstDependencyProperty(t *testing.T) {
	const count = 250
	nodes := make([]topSortTestNode, count)
	random := rand.New(rand.NewChaCha8([32]byte{28, 98, 31, 7}))
	for round := 0; round < 100; round++ {
		sort := NewTopSort()
		dependencies := make([][]any, count)
		for i := range nodes {
			nodes[i].id = i
			for j := i + 1; j < count; j++ {
				if random.Uint32()%97 == 0 {
					dependencies[i] = append(dependencies[i], &nodes[j])
				}
			}
			if err := sort.Insert(&nodes[i], dependencies[i]); err != nil {
				t.Fatal(err)
			}
		}
		sort.TopologicalSort()
		ordered := sort.TopSorted()
		if len(ordered) != count {
			t.Fatal("missing nodes", round, len(ordered))
		}
		position := make(map[any]int, count)
		for i, node := range ordered {
			position[node] = i
		}
		for i := range nodes {
			for _, dependency := range dependencies[i] {
				if position[dependency] >= position[&nodes[i]] {
					t.Fatal("dependency follows node", round, i, dependency.(*topSortTestNode).id)
				}
			}
		}
	}
}
