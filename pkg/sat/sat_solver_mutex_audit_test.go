// Source: src/util/max_cliques.h concrete named owners; src/sat/sat_solver.cpp:3119-3150.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func satMutexRowsRoots(o *satSolverLiteralRows) []util.Block {
	if o == nil {
		return nil
	}
	return o.storage().OwningRoots()
}
func satMutexColumnRoots(c *RuntimeUintColumn) []util.Block {
	r := []util.Block{c.Raw.Pending}
	if !c.Raw.PendingReplaces {
		r = append(r, c.Raw.Block)
	}
	return r
}
func satMaxCliquesRoots(g *util.NegatedLiteralCliques) []util.Block { return g.OwningRoots() }
func satMutexFrameRoots(f *satMutexFrame) []util.Block {
	r := satMaxCliquesRoots(f.graph)
	r = append(r, f.seen.raw, f.seen.pending)
	r = append(r, f.results.OwningRoots()...)
	if f.psUsed {
		r = append(r, satMutexColumnRoots(&f.ps)...)
	}
	if f.rowUsed {
		r = append(r, satMutexColumnRoots(&f.row)...)
	}
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
