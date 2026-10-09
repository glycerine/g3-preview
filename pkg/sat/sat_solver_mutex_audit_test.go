// Source: src/util/max_cliques.h concrete named owners; src/sat/sat_solver.cpp:3119-3150.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func satMutexRowsRoots(o *satSolverLiteralRows) []util.Block {
	if o == nil {
		return nil
	}
	return o.owningRoots()
}
func satMutexColumnRoots(c *RuntimeUintColumn) []util.Block {
	r := []util.Block{c.Raw.Pending}
	if !c.Raw.PendingReplaces {
		r = append(r, c.Raw.Block)
	}
	return r
}
func satMutexFrameRoots(f *satMutexFrame) []util.Block {
	r := []util.Block{f.seen.raw, f.seen.pending}
	return satMutexNonzero(r)
}
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
