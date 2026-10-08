// Source: src/util/top_sort.h:28-98.
package util

import (
	"errors"
	"reflect"
)

var ErrTopSortNode = errors.New("g3: top-sort nodes must be non-nil pointers")

// TopSort orders project objects after all registered dependencies reachable
// from them. The source class is templated over T*; this non-generic port keeps
// those pointers as dynamically typed Go values and validates them at Insert.
//
// Node order is recorded separately from the lookup map. This makes unrelated
// output deterministic without relying on Go map iteration.
type TopSort struct {
	dependencies map[any][]any
	nodes        []any
	topSorted    []any
}

// NewTopSort constructs an empty dependency graph.
func NewTopSort() *TopSort {
	return &TopSort{dependencies: make(map[any][]any)}
}

func validTopSortNode(node any) bool {
	if node == nil {
		return false
	}
	value := reflect.ValueOf(node)
	return value.Kind() == reflect.Pointer && !value.IsNil()
}

// Insert registers node and replaces its dependency set. Dependencies are
// copied and duplicate pointers are removed in first-occurrence order.
func (sort *TopSort) Insert(node any, dependencies []any) error {
	if sort == nil || !validTopSortNode(node) {
		return ErrTopSortNode
	}
	for _, dependency := range dependencies {
		if !validTopSortNode(dependency) {
			return ErrTopSortNode
		}
	}
	if sort.dependencies == nil {
		sort.dependencies = make(map[any][]any)
	}
	if _, exists := sort.dependencies[node]; !exists {
		sort.nodes = append(sort.nodes, node)
	}
	unique := make([]any, 0, len(dependencies))
	seen := make(map[any]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if _, exists := seen[dependency]; exists {
			continue
		}
		seen[dependency] = struct{}{}
		unique = append(unique, dependency)
	}
	sort.dependencies[node] = unique
	return nil
}

type topSortFrame struct {
	node any
	next int
}

// TopologicalSort recomputes the dependency-first order. A dependency that was
// not itself inserted is ignored, matching the source traverse early return.
// Cycles are emitted once in DFS completion order.
func (sort *TopSort) TopologicalSort() {
	if sort == nil {
		return
	}
	sort.topSorted = sort.topSorted[:0]
	state := make(map[any]uint8, len(sort.nodes))
	stack := make([]topSortFrame, 0, len(sort.nodes))
	for _, root := range sort.nodes {
		if state[root] != 0 {
			continue
		}
		state[root] = 1
		stack = append(stack, topSortFrame{node: root})
		for len(stack) != 0 {
			frame := &stack[len(stack)-1]
			dependencies := sort.dependencies[frame.node]
			if frame.next < len(dependencies) {
				dependency := dependencies[frame.next]
				frame.next++
				if _, registered := sort.dependencies[dependency]; !registered || state[dependency] != 0 {
					continue
				}
				state[dependency] = 1
				stack = append(stack, topSortFrame{node: dependency})
				continue
			}
			state[frame.node] = 2
			sort.topSorted = append(sort.topSorted, frame.node)
			stack = stack[:len(stack)-1]
		}
	}
}

// TopSorted returns a copy of the most recently computed order.
func (sort *TopSort) TopSorted() []any {
	if sort == nil {
		return nil
	}
	return append([]any(nil), sort.topSorted...)
}
