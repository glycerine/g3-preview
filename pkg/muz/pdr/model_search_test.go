package pdr_test

// Source: src/muz/pdr/pdr_context.cpp model_node closure and model_search
// frontier ordering, cache, duplicate, reset, and backtracking invariants.

import (
	"testing"

	"github.com/glycerine/g3/pkg/muz/pdr"
)

func TestModelSearchFrontierOrderingClosureAndOwnership(t *testing.T) {
	for _, test := range []struct {
		name     string
		breadth  bool
		wantNext []int
	}{
		{"breadth", true, []int{0, 2, 1}},
		{"depth", false, []int{0, 1, 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			command, manager, owner := pdrManagerFixture(t, 2)
			boolean, err := manager.MkBoolSort()
			pdrMust(t, err)
			head := pdrFunction(t, manager, "search_"+test.name, nil, boolean)
			transformer, err := pdr.NewPredicateTransformer(owner, head, pdr.ReachableHashCache)
			pdrMust(t, err)
			truth, err := manager.MkTrue()
			pdrMust(t, err)
			pdrMust(t, transformer.InitializeSystem(truth, truth))

			rootState := pdrBool(t, manager, "root_"+test.name)
			root, err := pdr.NewModelNode(nil, rootState, transformer, 2)
			pdrMust(t, err)
			search := pdr.NewModelSearch(test.breadth)
			pdrMust(t, search.SetRoot(root))
			next, found, err := search.Next()
			pdrMust(t, err)
			if !found || next != root {
				t.Fatal("root frontier", next, found)
			}

			children := make([]*pdr.ModelNode, 3)
			for index := range children {
				state := pdrBool(t, manager, test.name+string(rune('a'+index)))
				children[index], err = pdr.NewModelNode(root, state, transformer, 1)
				pdrMust(t, err)
				pdrMust(t, search.AddLeaf(children[index]))
				if childIndex, x := children[index].Index(); x != nil || childIndex != uint32(index) {
					t.Fatal("child index", childIndex, x, index)
				}
			}
			if count, x := search.GoalCount(); x != nil || count != 3 {
				t.Fatal("frontier count", count, x)
			}
			pdrMust(t, search.WellFormed())
			for _, index := range test.wantNext {
				next, found, err = search.Next()
				pdrMust(t, err)
				if !found || next != children[index] {
					t.Fatal("frontier order", index, next, found)
				}
				if index == 0 {
					pdrMust(t, next.SetClosed())
					state, x := next.State()
					pdrMust(t, x)
					if cached, x := transformer.IsCachedReachable(state); x != nil || !cached {
						t.Fatal("closed node was not added to reachable cache", cached, x)
					}
				} else {
					pdrMust(t, next.SetPreClosed())
				}
			}
			pdrMust(t, root.CheckPreClosed())
			if !root.IsClosed() {
				t.Fatal("closed children did not pre-close root")
			}
			pdrMust(t, search.WellFormed())

			pdrMust(t, children[0].SetOpen())
			if root.IsClosed() {
				t.Fatal("opening a child did not open its ancestors")
			}
			pdrMust(t, search.BacktrackLevel(true, children[0]))
			if children[0].Level() != 2 {
				t.Fatal("level-sensitive backtrack", children[0].Level())
			}
			next, found, err = search.Next()
			pdrMust(t, err)
			if !found || next != children[0] {
				t.Fatal("backtracked node was not requeued", next, found)
			}
			pdrMust(t, next.SetPreClosed())

			pdrMust(t, search.Close())
			pdrMust(t, transformer.Close())
			pdrMust(t, owner.Close())
			pdrMust(t, command.Close())
		})
	}
}

func TestModelNodeRejectsInvalidParentLevel(t *testing.T) {
	command, manager, owner := pdrManagerFixture(t, 1)
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	head := pdrFunction(t, manager, "invalid_parent", nil, boolean)
	transformer, err := pdr.NewPredicateTransformer(owner, head, pdr.ReachableNoCache)
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	root, err := pdr.NewModelNode(nil, truth, transformer, 1)
	pdrMust(t, err)
	if _, err = pdr.NewModelNode(root, truth, transformer, 1); err == nil {
		t.Fatal("child at invalid level was accepted")
	}
	search := pdr.NewModelSearch(true)
	pdrMust(t, search.SetRoot(root))
	pdrMust(t, search.Close())
	pdrMust(t, transformer.Close())
	pdrMust(t, owner.Close())
	pdrMust(t, command.Close())
}
