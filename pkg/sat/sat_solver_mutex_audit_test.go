package sat

// Source: src/util/max_cliques.h concrete named owners; src/sat/sat_solver.cpp:3119-3150.

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func satMutexColumnRoots(_ *satUintColumn) []umemory.Block { return nil }
func satMutexNonzero(roots []umemory.Block) []umemory.Block {
	n := 0
	for _, r := range roots {
		if r != (umemory.Block{}) {
			roots[n] = r
			n++
		}
	}
	return roots[:n]
}
