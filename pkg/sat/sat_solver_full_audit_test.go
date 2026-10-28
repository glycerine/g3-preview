package sat

// Source: src/sat/sat_solver.h owning members; independent actual Runtime root census.

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func satFullSolverRuntimeAudit(t *testing.T, s *Solver, outputs ...*satSolverLiteralRows) {
	t.Helper()
	satScalarAuditRoots(t, s.Variables, satFullSolverRuntimeRoots(s, outputs...)...)
}

// Native SAT containers own no Runtime payload blocks.
func satFullSolverRuntimeRoots(_ *Solver, _ ...*satSolverLiteralRows) []umemory.Block { return nil }
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
