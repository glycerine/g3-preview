// Source: src/sat/sat_solver.cpp decision and GC member ownership; independent named physical census.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
)

func satSimplifierAudit(t *testing.T, simplifier *satSimplifier) {
	decision := simplifier.decision
	g := simplifier.cleaner.lifecycle.integrityGC
	if g == nil {
		t.Fatal("audit requires named real GC binding")
	}
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
	for _, x := range []*RuntimeUintColumn{&s.toAssert, &s.cacheWork, &analysis.lemma, &analysis.extAntecedents, &analysis.unmark, &analysis.minStack, &c.assumptions, &c.userScopeLiterals, &c.core, &c.minCore} {
		col(x)
	}
	for _, row := range s.cache.rows {
		if !row.pendingReplaces {
			add(row.block)
		}
		add(row.pending)
	}
	if !simplifier.elimination.vars.Raw.PendingReplaces {
		add(simplifier.elimination.vars.Raw.Block)
	}
	add(simplifier.elimination.vars.Raw.Pending)
	if q := simplifier.blocked.queue; q != nil && q.runtimeBacked() {
		col(&q.runtimeValues)
		col(&q.runtimeIndices)
	}
	satScalarAuditRoots(t, v, roots...)
	want := decision.ownerBytes + g.ownerBytes + g.scratchBytes + g.cleaner.ownerBytes + c.ownerBytes + s.ownerBytes + analysis.ownerBytes + d.ownerBytes + uint64(cap(s.cache.rows))*uint64(reflect.TypeOf(satProbingCacheRow{}).Size())
	want += l.ownerBytes + uint64(cap(l.Original)+cap(l.learned))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
	want += simplifier.ownerBytes + satOuterUseOwnedBytes(simplifier.useList) + simplifier.kernel.ownerBytes + simplifier.kernel.backingBytes
	set := simplifier.subTodo
	want += set.ownerBytes + 4*uint64(cap(set.ids)) + uint64(cap(set.refs))*uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	for _, stats := range []satStorageVectorStats{simplifier.subBinTodo.Stats(), simplifier.bsCS.Stats(), simplifier.posCls.Stats(), simplifier.negCls.Stats()} {
		want += stats.OwnerBytes + stats.BackingBytes + stats.BindingBytes
	}
	ls := simplifier.literalStore
	want += ls.ownerBytes + ls.directoryBytes + satWatchGraphBytes(ls.words.Stats())
	dummy := simplifier.dummy
	da := dummy.owner
	ds := da.storage.Stats()
	want += dummy.ownerBytes + da.ownerBytes + 4*uint64(da.ids.FreeCapacity()) + da.largeBytes + da.largeDirectory + ds.OwnerBytes + ds.PayloadBytes + ds.MetadataBytes + ds.DirectoryBytes
	want += simplifier.clauseSort.bytes + 8*uint64(cap(simplifier.elimination.costs)+cap(simplifier.elimination.sortScratch))
	if q := simplifier.blocked.queue; q != nil {
		want += q.ownerBytes + 4*uint64(cap(q.values)+cap(q.indices))
	}
	if cs := simplifier.blocked.toRemove; cs != nil {
		stats := cs.Stats()
		want += stats.OwnerBytes + stats.BackingBytes + stats.BindingBytes
	}
	if want != analysis.budget.Used() {
		t.Fatal("exact named once-only graph", want, analysis.budget.Used())
	}

	refs := []StoredClauseRef{}
	addRefs := func(data []StoredClauseRef, lo, hi uint32) {
		for i := lo; i < hi; i++ {
			refs = append(refs, data[i])
		}
	}
	for _, learned := range []bool{false, true} {
		vec := simplifier.clauseVector(learned)
		if simplifier.clauseSort.active && simplifier.clauseSort.learned == learned {
			st := &simplifier.clauseSort
			ranges := [4][2]uint32{{0, st.out}, {st.left, st.mid}, {st.right, st.end}, {st.end, st.n}}
			for stage, r := range ranges {
				if uint32(stage) < st.drainStage {
					continue
				}
				lo := r[0]
				if uint32(stage) == st.drainStage && st.drainIndex > lo {
					lo = st.drainIndex
				}
				data := simplifier.sortSource()
				if stage == 0 {
					data = simplifier.sortDestination()
				}
				addRefs(data, lo, r[1])
			}
		} else if simplifier.cleanupInterrupted && simplifier.cleanupLearned == learned {
			addRefs(vec, simplifier.drainPrefix, simplifier.cleanupOut)
			lo := simplifier.cleanupIndex
			if simplifier.drainSuffix > lo {
				lo = simplifier.drainSuffix
			}
			addRefs(vec, lo, simplifier.cleanupEnd)
		} else {
			addRefs(vec, 0, uint32(len(vec)))
		}
	}
	if l.pendingCandidate != (StoredClauseRef{}) {
		refs = append(refs, l.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("stale named simplifier owner", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate simplifier owning ranges", r)
			}
		}
	}
	if small.Live != uint32(len(refs)) {
		t.Fatal("unnamed actual simplifier clause", small.Live, len(refs))
	}
}

func TestSATSimplifierAllActualRefusalPrefixes(t *testing.T) {
	for _, route := range []uint32{0, 1, 2} {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			failed, success := 0, 0
			for fault := int64(0); fault < 512; fault++ {
				done := false
				t.Run(fmt.Sprint(fault), func(t *testing.T) {
					s, l := satSimplifierFixture(t, 4)
					p := l.propagation
					// Named GC owner is a borrow, not a second solver or copied random stream.
					// This points to the actual previously constructed decision GC cohort.
					if l.integrityGC == nil {
						t.Fatal("actual fixture GC binding missing")
					}
					if route == 0 {
						for v := uint32(1); v < 4; v++ {
							satOK(t, p.variables.external.Write(v, 1))
						}
					}
					clauses := [][]Literal{{0, 2, 4}, {1, 2, 4}, {0, 2, 4, 6}, {1, 3, 6}}
					if route == 1 {
						clauses = [][]Literal{{0, 2, 4}, {1, 3, 6}}
						s.numCalls = 2
					}
					for _, ls := range clauses {
						c, e := l.MkClauseCore(uint32(len(ls)), ls, false)
						satOK(t, e)
						satOK(t, p.clauses.MarkStrengthened(c))
					}
					satOK(t, p.variables.elimTodo.Reset())
					for v := uint32(0); v < 4; v++ {
						satOK(t, p.variables.elimTodo.Insert(v))
					}
					s.lifecycle.budget.FailAfter(fault)
					e := s.Run(route == 2)
					satSimplifierAudit(t, s)
					if e != nil {
						failed++
						satProbingExpectedRefusal(t, p, e)
					} else {
						success++
						done = true
					}
					s.lifecycle.budget.FailAfter(-1)
				})
				if done {
					break
				}
			}
			if failed == 0 || success != 1 {
				t.Fatal("refusal campaign horizon", failed, success)
			}
			t.Log("all genuine refusal prefixes", failed)
		})
	}
}

func TestSATSimplifierActualCancellationRetainsNamedSourcePrefix(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	for _, ls := range [][]Literal{{0, 2, 4}, {1, 3, 6}} {
		_, e := l.MkClauseCore(3, ls, false)
		satOK(t, e)
	}
	s.numCalls = 2 // actual default blocked round: report/queue precede checkpoint.
	l.propagation.limit.Cancel()
	e := s.Run(false)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal("actual source checkpoint cancellation", e)
	}
	satSimplifierAudit(t, s)
	if s.blocked.reportActive || s.blocked.queue != nil || s.blocked.toRemove != nil || s.scopeActive || s.numCalls != 2 {
		t.Fatal("cancelled before outer scoped constructor and call increment")
	}
}
