// Source: src/sat/sat_solver.h owning members; independent actual Runtime root census.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func satFullSolverRuntimeAudit(t *testing.T, s *satSolver) {
	t.Helper()
	roots := []memory.Block{}
	add := func(b memory.Block) {
		if b != (memory.Block{}) {
			roots = append(roots, b)
		}
	}
	raw := func(c *satSolverAssignmentColumn) {
		if !c.pendingReplaces {
			add(c.block)
		}
		add(c.pending)
	}
	col := func(c *satSolverRuntimeUintColumn) { raw(&c.raw) }
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
	for _, v := range []*satSolverRuntimeUintColumn{&p.toAssert, &p.cacheWork, &p.assigned.values, &c.assumptions, &c.userScopeLiterals, &c.core, &c.minCore, &c.assumptionSet.values, &a.lemma, &a.extAntecedents, &a.minStack, &a.unmark, &s.auxLiterals, &s.todoAntecedents, &s.model, &s.mus.core, &s.mus.mus, &s.mus.model, &s.simplifier.elimination.vars} {
		col(v)
	}
	for f := s.mus.frames; f != nil; f = f.next {
		if f.vectorUsed {
			col(&f.vector)
		}
		for i := uint32(0); i < f.setsUsed; i++ {
			raw(&f.sets[i].membership)
			col(&f.sets[i].values)
		}
	}
	raw(&p.assigned.membership)
	raw(&c.assumptionSet.membership)
	raw(&s.asymm.savedPhase)
	for _, v := range []*satSCCVector{&s.scc.index, &s.scc.lowlink, &s.scc.stack, &s.scc.inStack, &s.scc.roots, &s.scc.frames, &s.scc.toElim} {
		add(v.raw)
		add(v.pending)
	}
	if s.antecedents != nil {
		add(s.antecedents.raw)
		add(s.antecedents.pending)
		for _, entry := range s.antecedents.entries {
			add(entry.raw)
		}
	}
	for _, b := range s.clauses.runtimeIDs.RawRoots() {
		add(b)
	}
	for i := range s.binaryClauseGraph {
		col(&s.binaryClauseGraph[i])
	}
	satScalarAuditRoots(t, s.variables, roots...)
}
func satFullSolverUserClauseAudit(t *testing.T, s *satSolver) {
	t.Helper()
	refs := []satStoredClauseRef{}
	for _, learned := range [2]bool{false, true} {
		vec := s.lifecycle.original
		if learned {
			vec = s.lifecycle.learned
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
	if s.lifecycle.pendingCandidate != (satStoredClauseRef{}) {
		refs = append(refs, s.lifecycle.pendingCandidate)
	}
	for i, r := range refs {
		if !s.clauses.IsLive(r) {
			t.Fatal("stale actual named clause", r)
		}
		for _, other := range refs[:i] {
			if r == other {
				t.Fatal("duplicate actual owning range", r)
			}
		}
	}
	large := uint32(0)
	for _, v := range s.clauses.large {
		if v.live {
			large++
		}
	}
	if uint32(len(refs)) != s.clauses.storage.Stats().Live+large {
		t.Fatal("anonymous actual clause", len(refs), s.clauses.storage.Stats().Live, large)
	}
}
