// Source: src/util/max_cliques.h concrete named owners; src/sat/sat_solver.cpp:3119-3150.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func satMutexColumnRoots(_ *satUintColumn) []util.Block { return nil }
func satMutexNonzero(roots []util.Block) []util.Block {
	n := 0
	for _, r := range roots {
		if r != (util.Block{}) {
			roots[n] = r
			n++
		}
	}
	return roots[:n]
}
