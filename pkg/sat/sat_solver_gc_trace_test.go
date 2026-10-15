package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satGCTraceObserver struct {
	output       bytes.Buffer
	gc           *satSolverGC
	clause       StoredClauseRef
	entrySeen    bool
	entryLive    bool
	entryCount   int
	entryStarted bool
}

type satGCClauseTraceObserver struct {
	output                                    bytes.Buffer
	gc                                        *satSolverGC
	clause                                    StoredClauseRef
	freezeSeen, freezeWasActive               bool
	reactivationSeen, reactivationHadOriginal bool
	cleanupSeen, cleanupHadPrefix             bool
}

func (w *satGCClauseTraceObserver) Write(p []byte) (int, error) {
	text := string(p)
	a := w.gc.lifecycle().propagation.clauses
	if strings.HasPrefix(text, "freezing size:") {
		w.freezeSeen = true
		frozen, err := a.Frozen(w.clause)
		w.freezeWasActive = err == nil && !frozen
	}
	if strings.HasPrefix(text, "reactivating:\n") {
		w.reactivationSeen = true
		first, err := a.Literal(w.clause, 0)
		w.reactivationHadOriginal = err == nil && first == 1
	}
	if strings.HasPrefix(text, "after cleanup:\n") {
		w.cleanupSeen = true
		first, err := a.Literal(w.clause, 0)
		w.cleanupHadPrefix = err == nil && first == 2
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:1569,1612,1630; records bracket actual
// freeze and cleanup state changes and use the source clause/literal printers.
func TestSATGCFrozenClauseSourceTraceBoundaries(t *testing.T) {
	g, s := satGCFixture(t, 4)
	p := s.analysis.propagation
	cfg := &p.config.state
	cfg.GCSmallLBD = 1
	cfg.GCK = 3
	cfg.Initialized |= satConfigGCSmallLBDMask | satConfigGCKMask
	r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
	for i := uint32(0); i < 4; i++ {
		satOK(t, p.variables.setPhase(BoolVar(i), 0))
		satOK(t, p.variables.setPreviousPhase(BoolVar(i), 0))
		satOK(t, p.variables.setGCAssigned(BoolVar(i), true))
	}
	w := &satGCClauseTraceObserver{gc: g, clause: r}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_frozen")
	g.traceTags = tags
	satOK(t, g.gcDynPSM())
	frozen, err := p.clauses.Frozen(r)
	satOK(t, err)
	if !frozen {
		t.Fatal("clause did not freeze")
	}
	if !tags.ShouldTrace("sat_frozen") {
		if w.output.Len() != 0 || w.freezeSeen {
			t.Fatal("erased freeze TRACE emitted")
		}
		return
	}
	want := "-------- [sat_frozen] gc_dyn_psm src/sat/sat_solver.cpp:1540 ---------\n" +
		"m_min_d_tk: 0\n------------------------------------------------\n" +
		"-------- [sat_frozen] gc_dyn_psm src/sat/sat_solver.cpp:1569 ---------\n" +
		"freezing size: 3 psm: 3 (0 1 2)+*\n------------------------------------------------\n"
	if got := w.output.String(); got != want || !w.freezeSeen || !w.freezeWasActive {
		t.Fatalf("freeze trace got %q, want %q; seen=%v active=%v", got, want, w.freezeSeen, w.freezeWasActive)
	}
}

func TestSATGCReactivationCleanupSourceTraceBoundaries(t *testing.T) {
	g, s := satGCFixture(t, 4)
	p := s.analysis.propagation
	r := satGCClause(t, g, []Literal{1, 2, 4}, 5)
	satOK(t, g.lifecycle().detachClause(r))
	satOK(t, p.clauses.Freeze(r))
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.clauses.Unfreeze(r))
	w := &satGCClauseTraceObserver{gc: g, clause: r}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_gc")
	tags.Enable("sat")
	g.traceTags = tags
	keep, err := g.activateFrozenClause(r)
	satOK(t, err)
	if keep {
		t.Fatal("binary reduction retained ternary clause")
	}
	if !tags.ShouldTrace("sat_gc") {
		if w.output.Len() != 0 || w.reactivationSeen || w.cleanupSeen {
			t.Fatal("erased reactivation TRACE emitted")
		}
		return
	}
	want := "-------- [sat_gc] activate_frozen_clause src/sat/sat_solver.cpp:1612 ---------\n" +
		"reactivating:\n(-0 1 2)+*\n------------------------------------------------\n" +
		"-------- [sat] activate_frozen_clause src/sat/sat_solver.cpp:1630 ---------\n" +
		"after cleanup:\n1 2\n------------------------------------------------\n"
	if got := w.output.String(); got != want || !w.reactivationSeen || !w.reactivationHadOriginal || !w.cleanupSeen || !w.cleanupHadPrefix {
		t.Fatalf("reactivation trace got %q, want %q; original=%v prefix=%v", got, want, w.reactivationHadOriginal, w.cleanupHadPrefix)
	}
}

func (w *satGCTraceObserver) Write(p []byte) (int, error) {
	if bytes.Equal(p, []byte("gc\n")) {
		w.entrySeen = true
		w.entryLive = w.gc.lifecycle().propagation.clauses.IsLive(w.clause)
		w.entryCount = len(w.gc.lifecycle().learned)
		w.entryStarted = w.gc.interrupted
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:1494,1519; each TRACE precedes its first
// learned-clause traversal or dynamic-GC counter update.
func TestSATGCHalfAndDynamicSourceEntryTrace(t *testing.T) {
	for _, tc := range []struct {
		name, function, line string
		run                  func(*satSolverGC) error
	}{
		{"half", "gc_half", "1494", (*satSolverGC).gcHalf},
		{"dynamic", "gc_dyn_psm", "1519", (*satSolverGC).gcDynPSM},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := satGCFixture(t, 4)
			if tc.name == "dynamic" {
				cfg := &g.lifecycle().propagation.config.state
				cfg.GCSmallLBD = 1
				cfg.GCK = 3
				cfg.Initialized |= satConfigGCSmallLBDMask | satConfigGCKMask
			}
			clause := satGCClause(t, g, []Literal{0, 2, 4}, 3)
			w := &satGCTraceObserver{gc: g, clause: clause}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("sat")
			if tc.name == "dynamic" {
				tags.Enable("sat_frozen")
			}
			g.traceTags = tags
			satOK(t, tc.run(g))
			if !tags.ShouldTrace("sat") {
				if w.output.Len() != 0 || w.entrySeen {
					t.Fatal("erased GC TRACE emitted", w.output.String())
				}
				return
			}
			want := "-------- [sat] " + tc.function +
				" src/sat/sat_solver.cpp:" + tc.line + " ---------\n" +
				"gc\n------------------------------------------------\n"
			if tc.name == "dynamic" {
				want += "-------- [sat_frozen] gc_dyn_psm src/sat/sat_solver.cpp:1540 ---------\n" +
					"m_min_d_tk: 1\n------------------------------------------------\n"
			}
			if got := w.output.String(); got != want || !w.entrySeen || !w.entryLive || w.entryCount != 1 || w.entryStarted {
				t.Fatalf("source GC entry: got %q, want %q, live=%v count=%d started=%v", got, want, w.entryLive, w.entryCount, w.entryStarted)
			}
		})
	}
}
