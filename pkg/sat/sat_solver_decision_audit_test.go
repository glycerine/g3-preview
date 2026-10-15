// Source: src/sat/sat_solver.cpp decision and GC member ownership; independent named physical census.
package sat

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func satDecisionAudit(t *testing.T, g *satSolverGC, decision *satSolverDecision) {
	t.Helper()
	c := g.conflict
	s := c.dynamic.probing
	analysis := c.analysis
	d := c.dynamic
	p := analysis.propagation
	v := p.variables
	l := s.lifecycle
	roots := []umemory.Block{}
	satScalarAuditRoots(t, v, roots...)
	want := decision.ownerBytes + g.ownerBytes + g.scratchBytes + g.cleaner.ownerBytes + c.ownerBytes + s.ownerBytes + analysis.ownerBytes + d.ownerBytes
	want += l.ownerBytes + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
	if want != analysis.budget.Used() {
		t.Fatal("exact named once-only graph", want, analysis.budget.Used())
	}

	refs := append([]StoredClauseRef(nil), l.Original...)
	addRefs := func(data []StoredClauseRef, lo, hi uint32) {
		for i := lo; i < hi; i++ {
			refs = append(refs, data[i])
		}
	}
	if g.sorting {
		ranges := [][2]uint32{{0, g.sortOut}, {g.sortLeft, g.sortMid}, {g.sortRight, g.sortEnd}, {g.sortEnd, g.sortN}}
		for stage, r := range ranges {
			if uint32(stage) < g.sortDrainStage {
				continue
			}
			lo := r[0]
			if uint32(stage) == g.sortDrainStage && g.sortDrainIndex > lo {
				lo = g.sortDrainIndex
			}
			data := g.sortSource()
			if stage == 0 {
				data = g.sortDestination()
			}
			addRefs(data, lo, r[1])
		}
	} else if g.interrupted {
		addRefs(l.learned, g.drainPrefix, g.out)
		lo := g.cursor
		if g.drainSuffix > lo {
			lo = g.drainSuffix
		}
		addRefs(l.learned, lo, g.end)
	} else {
		refs = append(refs, l.learned...)
	}
	if l.pendingCandidate != (StoredClauseRef{}) {
		refs = append(refs, l.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("stale named GC owner", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate GC owning ranges", r)
			}
		}
	}
	if small.Live != uint32(len(refs)) {
		t.Fatal("unnamed actual GC clause", small.Live, len(refs))
	}
}
