// Source: src/sat/sat_solver.h owning members; independent actual Runtime root census.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func satFullSolverRuntimeAudit(t *testing.T, s *Solver, outputs ...*satSolverLiteralRows) {
	t.Helper()
	satScalarAuditRoots(t, s.Variables, satFullSolverRuntimeRoots(s, outputs...)...)
}
func satFullSolverRuntimeRoots(s *Solver, outputs ...*satSolverLiteralRows) []util.Block {
	roots := []util.Block{}
	add := func(b util.Block) {
		if b != (util.Block{}) {
			roots = append(roots, b)
		}
	}
	raw := func(c *RuntimeVectorColumn) {
		if !c.PendingReplaces {
			add(c.Block)
		}
		add(c.Pending)
	}
	col := func(c *RuntimeUintColumn) { raw(&c.Raw) }
	if q := s.simplifier.blocked.queue; q != nil && q.runtimeBacked() {
		raw(&q.runtimeValues.Raw)
		raw(&q.runtimeIndices.Raw)
	}
	p := s.probing
	c := s.conflict
	a := s.analysis
	add(p.cache.block)
	add(p.cache.pending)
	add(p.cache.retired)
	for _, row := range p.cache.rows {
		if !row.pendingReplaces {
			add(row.block)
		}
		add(row.pending)
	}
	for _, v := range []*RuntimeUintColumn{&p.toAssert, &p.cacheWork, &c.assumptions, &c.userScopeLiterals, &c.core, &c.minCore, &a.lemma, &a.extAntecedents, &a.minStack, &a.unmark, &s.auxLiterals, &s.todoAntecedents, &s.model, &s.mus.core, &s.mus.mus, &s.mus.model, &s.simplifier.elimination.vars} {
		col(v)
	}
	for f := s.mus.frames; f != nil; f = f.next {
		if f.vectorUsed {
			col(&f.vector)
		}
	}
	raw(&s.asymm.savedPhase)
	for _, v := range []*satSCCVector{&s.scc.index, &s.scc.lowlink, &s.scc.stack, &s.scc.inStack, &s.scc.roots, &s.scc.frames, &s.scc.toElim} {
		add(v.raw)
		add(v.pending)
	}
	for f := s.consequenceFrames; f != nil; f = f.next {
		for i := uint32(0); i < f.used; i++ {
			tag := f.order[i]
			if tag >= 3 && tag < 6 {
				col(&f.vectors[tag-3])
			} else if tag == 6 {
				add(f.index.raw)
				add(f.index.pending)
			}
		}
	}
	if m := s.antecedents; m != nil {
		add(m.raw)
		add(m.pending)
		for _, entry := range m.entries {
			add(entry.raw)
			add(entry.pending)
		}
		for _, entry := range m.extraEntries {
			add(entry.raw)
			add(entry.pending)
		}
		for _, entry := range m.pendingEntries {
			add(entry.raw)
			add(entry.pending)
		}
		for i := uint32(0); i < m.tempUsed; i++ {
			add(m.temps[i].raw)
			add(m.temps[i].pending)
		}
	}
	for _, out := range outputs {
		for _, h := range out.owningRoots() {
			add(h)
		}
	}

	for f := s.mutexFrames; f != nil; f = f.next {
		roots = append(roots, satMutexFrameRoots(f)...)
	}

	for i := range s.binaryClauseGraph {
		col(&s.binaryClauseGraph[i])
	}
	return roots
}
func satFullSolverUserClauseAudit(t *testing.T, s *Solver) {
	t.Helper()
	refs := []StoredClauseRef{}
	for _, learned := range [2]bool{false, true} {
		vec := s.Lifecycle.Original
		if learned {
			vec = s.Lifecycle.learned
		}
		if s.userGC.active && s.userGC.learned == learned {
			g := &s.userGC
			refs = append(refs, vec[g.drainPrefix:g.out]...)
			lo := g.in
			if g.drainSuffix > lo {
				lo = g.drainSuffix
			}
			refs = append(refs, vec[lo:g.end]...)
		} else {
			refs = append(refs, vec...)
		}
	}
	if s.Lifecycle.pendingCandidate != (StoredClauseRef{}) {
		refs = append(refs, s.Lifecycle.pendingCandidate)
	}
	for i, r := range refs {
		if !s.Clauses.IsLive(r) {
			t.Fatal("stale actual named clause", r)
		}
		for _, other := range refs[:i] {
			if r == other {
				t.Fatal("duplicate actual owning range", r)
			}
		}
	}
	large := uint32(0)
	for _, v := range s.Clauses.large {
		if v.live {
			large++
		}
	}
	if uint32(len(refs)) != s.Clauses.storage.Stats().Live+large {
		t.Fatal("anonymous actual clause", len(refs), s.Clauses.storage.Stats().Live, large)
	}
}
