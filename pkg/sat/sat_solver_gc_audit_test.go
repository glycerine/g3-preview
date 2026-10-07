// Source: src/sat/sat_solver.cpp GC member ownership; independent named physical census.
package sat

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func satGCAudit(t *testing.T, g *satSolverGC) {
	t.Helper()
	c := g.conflict
	s := c.dynamic.probing
	analysis := c.analysis
	d := c.dynamic
	p := analysis.propagation
	v := p.variables
	l := s.lifecycle
	roots := []util.Block{}
	add := func(h util.Block) {
		if h != (util.Block{}) {
			roots = append(roots, h)
		}
	}
	col := func(c *RuntimeUintColumn) {
		if !c.Raw.PendingReplaces {
			add(c.Raw.Block)
		}
		add(c.Raw.Pending)
	}
	add(s.cache.block)
	add(s.cache.pending)
	add(s.cache.retired)
	for _, x := range []*RuntimeUintColumn{&s.toAssert, &s.cacheWork, &s.assigned.values, &analysis.lemma, &analysis.extAntecedents, &analysis.unmark, &analysis.minStack, &c.assumptions, &c.userScopeLiterals, &c.core, &c.minCore, &c.assumptionSet.values} {
		col(x)
	}
	for _, x := range []*RuntimeVectorColumn{&s.assigned.membership, &c.assumptionSet.membership} {
		if !x.PendingReplaces {
			add(x.Block)
		}
		add(x.Pending)
	}
	for _, row := range s.cache.rows {
		if !row.pendingReplaces {
			add(row.block)
		}
		add(row.pending)
	}
	satScalarAuditRoots(t, v, roots...)
	want := g.ownerBytes + g.scratchBytes + g.cleaner.ownerBytes + c.ownerBytes + s.ownerBytes + analysis.ownerBytes + d.ownerBytes + uint64(cap(s.cache.rows))*uint64(reflect.TypeOf(satProbingCacheRow{}).Size())
	want += l.ownerBytes + uint64(cap(l.Original)+cap(l.learned))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(cap(a.ids.Free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(util.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*util.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
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
