// Source: src/util/max_cliques.h concrete named owners; src/sat/sat_solver.cpp:3119-3150.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func satMutexRowsRoots(o *satSolverLiteralRows) []util.Block {
	if o == nil {
		return nil
	}
	r := []util.Block{o.raw, o.pending, o.oldRaw}
	for i := range o.rows {
		r = append(r, satMutexColumnRoots(&o.rows[i])...)
	}
	for i := range o.oldRows {
		r = append(r, satMutexColumnRoots(&o.oldRows[i])...)
	}
	if o.temporaryActive {
		r = append(r, satMutexColumnRoots(&o.temporary)...)
	}
	return r
}
func satMutexColumnRoots(c *RuntimeUintColumn) []util.Block {
	r := []util.Block{c.Raw.Pending}
	if !c.Raw.PendingReplaces {
		r = append(r, c.Raw.Block)
	}
	return r
}
func satMaxCliquesRoots(g *satMaxCliques) []util.Block {
	if g == nil {
		return nil
	}
	r := satMutexRowsRoots(g.next)
	r = append(r, satMutexRowsRoots(g.tc)...)
	for _, s := range []*satCliqueUintSet{&g.reachable[0], &g.reachable[1], &g.seen1, &g.seen2, &g.vars} {
		r = append(r, satMutexColumnRoots(&s.words)...)
	}
	r = append(r, satMutexColumnRoots(&g.todo)...)
	return satMutexNonzero(append(r, satMutexColumnRoots(&g.clique)...))
}
func satMutexFrameRoots(f *satMutexFrame) []util.Block {
	r := satMaxCliquesRoots(f.graph)
	r = append(r, f.seen.raw, f.seen.pending)
	r = append(r, satMutexRowsRoots(f.results)...)
	if f.psUsed {
		r = append(r, satMutexColumnRoots(&f.ps)...)
	}
	if f.rowUsed {
		r = append(r, satMutexColumnRoots(&f.row)...)
	}
	return satMutexNonzero(r)
}
func satMutexGraphAudit(t *testing.T, s *Solver, g *satMaxCliques, outputs ...*satSolverLiteralRows) {
	t.Helper()
	r := satFullSolverRuntimeRoots(s, outputs...)
	r = append(r, satMaxCliquesRoots(g)...)
	satScalarAuditRoots(t, s.Variables, r...)
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
