package sat

import (
	"bytes"
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
