// Source: src/sat/sat_solver.cpp:60–74; src/sat/sat_solver.h:84–132,344–483.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

import "testing"

func TestSATSolverRealDestructorStagesKeepLaterSourceMembers(t *testing.T) {
	l, p, v, _ := satLifecycleFixture(t, 4)
	o, e := p.clauses.MkClause([]satLiteral{0, 2, 4, 6}, false)
	satOK(t, e)
	q, e := p.clauses.MkClause([]satLiteral{1, 3, 5, 7}, true)
	satOK(t, e)
	satOK(t, l.pushClause(false, o))
	satOK(t, l.pushClause(true, q))
	satOK(t, l.push())
	originalCap, learnedCap := cap(l.original), cap(l.learned)
	trail := v.trail
	queue := v.queue
	watches := v.watches
	generation := l.nonLearnedGeneration
	satOK(t, l.deleteClausesBody())
	if p.clauses.IsLive(o) || p.clauses.IsLive(q) || l.nonLearnedGeneration != generation+2 || cap(l.original) != originalCap || cap(l.learned) != learnedCap {
		t.Fatal("source dtor body changed member storage")
	}
	satOK(t, l.deleteClausesBody())
	if l.nonLearnedGeneration != generation+2 {
		t.Fatal("completed clause deletion replayed")
	}
	satOK(t, l.finalizeScopesReinit())
	if l.scopes != nil || l.reinit != nil || v.trail != trail || v.queue != queue || v.watches != watches {
		t.Fatal("source scopes/reinit stage consumed later members")
	}
	satOK(t, v.finalizeTrail())
	if !trail.closed || v.trail != nil || v.queue != queue || v.watches != watches {
		t.Fatal("source trail stage")
	}
	satOK(t, v.finalizeQueue())
	if v.queue != nil || v.watches != watches {
		t.Fatal("source queue stage")
	}
	satOK(t, v.finalizeColumnsWatches())
	if v.watches != nil || cap(l.original) != originalCap || cap(l.learned) != learnedCap {
		t.Fatal("clause vector freed before watches")
	}
	satOK(t, l.finalizeClauseVectors())
	if l.original != nil || l.learned != nil {
		t.Fatal("source clause vector stage")
	}
	// Standalone owning adapters acknowledge already completed actual stages;
	// their final metadata release must not replay clause generations or frees.
	satOK(t, l.Close())
	satOK(t, v.Close())
	if l.nonLearnedGeneration != generation+2 {
		t.Fatal("owning acknowledgment replay")
	}
}
