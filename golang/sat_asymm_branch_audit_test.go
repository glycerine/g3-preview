// Source: src/sat/sat_asymm_branch.cpp owning source roots. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func satAsymmAudit(t *testing.T, c *satAsymmBranch) {
	t.Helper()
	l := c.lifecycle
	p := l.propagation
	v := p.variables
	a := p.clauses
	refs := []satStoredClauseRef{}
	refs = append(refs, l.learned...)
	if c.sorting {
		for stage, span := range [4][2]uint32{{0, c.sortOut}, {c.sortLeft, c.sortMid}, {c.sortRight, c.sortEnd}, {c.sortEnd, c.sortN}} {
			if uint32(stage) < c.sortDrainStage {
				continue
			}
			lo := span[0]
			if uint32(stage) == c.sortDrainStage && c.sortDrainIndex > lo {
				lo = c.sortDrainIndex
			}
			data := c.sortSource()
			if stage == 0 {
				data = c.sortDestination()
			}
			refs = append(refs, data[lo:span[1]]...)
		}
	} else if c.interrupted {
		refs = append(refs, l.original[c.drainPrefix:c.out]...)
		lo := c.cursor
		if c.drainSuffix > lo {
			lo = c.drainSuffix
		}
		refs = append(refs, l.original[lo:c.end]...)
	} else {
		refs = append(refs, l.original...)
	}
	if c.discardedClause != (satStoredClauseRef{}) {
		refs = append(refs, c.discardedClause)
	}
	if l.pendingCandidate != (satStoredClauseRef{}) {
		refs = append(refs, l.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("named source owner stale", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate scalar owner", r)
			}
		}
	}
	small := a.storage.Stats()
	if small.Live != uint32(len(refs)) {
		t.Fatal("unnamed physical clause", small.Live, len(refs))
	}
	want := c.ownerBytes + c.scratchBytes + l.ownerBytes + uint64(cap(l.original)+cap(l.learned))*uint64(reflect.TypeOf(satStoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	want += a.ownerBytes + 4*uint64(cap(a.ids.free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(utilReslimit{}).Size()) + 8*uint64(cap(r.limits)) + uint64(reflect.TypeOf((*utilReslimit)(nil)).Size())*uint64(cap(r.children)) + p.memory.Stats().TotalBytes
	if want != l.budget.Used() {
		t.Fatal("whole named once-only physical graph", want, l.budget.Used())
	}
	roots := []memory.Block{}
	for _, h := range [2]memory.Block{c.savedPhase.block, c.savedPhase.pending} {
		if h != (memory.Block{}) {
			roots = append(roots, h)
		}
	}
	satScalarAuditRoots(t, v, roots...)
}
